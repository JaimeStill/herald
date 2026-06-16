package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

// Client is an authenticated HTTP client for the Herald API. Each call makes a
// single attempt (no retry): the CLI is a primitive over the API, so resilience
// and orchestration are the caller's concern. It is safe for concurrent use.
type Client struct {
	base    string
	client  *http.Client
	cred    azcore.TokenCredential // nil when auth is disabled
	scope   string
	timeout time.Duration
}

// NewClient builds a Client from settings. The credential is derived from the
// auth config and is nil when auth_mode is none, in which case no Authorization
// header is sent. Tokens are acquired via the credential per request and served
// from azidentity's internal cache; the Client adds no caching of its own.
func NewClient(s *Settings) (*Client, error) {
	cred, err := s.Auth.TokenCredential()
	if err != nil {
		return nil, fmt.Errorf("create credential: %w", err)
	}

	return &Client{
		base:    strings.TrimRight(s.API, "/"),
		client:  &http.Client{},
		cred:    cred,
		scope:   s.Scope,
		timeout: s.TimeoutDuration(),
	}, nil
}

// getJSON issues a GET, bounded by the client timeout, and decodes a 2xx JSON
// body into out.
func (c *Client) getJSON(ctx context.Context, path string, query url.Values, out any) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.requestURL(path, query), nil)
	if err != nil {
		return err
	}
	resp, err := c.send(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	return decodeResponse(resp, out)
}

// postMultipart uploads a single file with form fields, bounded by the client
// timeout, and decodes the 2xx JSON body into out.
func (c *Client) postMultipart(
	ctx context.Context,
	path string,
	fields map[string]string,
	fileField, filename string,
	data []byte,
	out any,
) error {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, v := range fields {
		if err := mw.WriteField(k, v); err != nil {
			return fmt.Errorf("write field %s: %w", k, err)
		}
	}
	fw, err := mw.CreateFormFile(fileField, filename)
	if err != nil {
		return fmt.Errorf("create form file: %w", err)
	}
	if _, err := fw.Write(data); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	if err := mw.Close(); err != nil {
		return fmt.Errorf("finalize multipart: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.requestURL(path, nil), &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := c.send(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	return decodeResponse(resp, out)
}

// postStream issues a POST and returns the streaming response on a 2xx status.
// Unlike the buffered helpers it does not bound the request with the client
// timeout — the caller owns both the deadline (it must wrap ctx) and the body
// (it must close it), since the stream outlives this call. Non-2xx responses are
// decoded as errors before returning.
func (c *Client) postStream(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.requestURL(path, nil), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.send(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer func() { _ = resp.Body.Close() }()
		return nil, decodeError(resp)
	}
	return resp, nil
}

// send attaches auth and executes a single request attempt.
func (c *Client) send(req *http.Request) (*http.Response, error) {
	if err := c.authorize(req.Context(), req); err != nil {
		return nil, err
	}
	return c.client.Do(req)
}

// authorize attaches a bearer token when auth is enabled. The token is acquired
// from the credential (azidentity serves it from its internal cache); it is a
// no-op when no credential is configured.
func (c *Client) authorize(ctx context.Context, req *http.Request) error {
	if c.cred == nil {
		return nil
	}
	tk, err := c.cred.GetToken(ctx, policy.TokenRequestOptions{Scopes: []string{c.scope}})
	if err != nil {
		return fmt.Errorf("acquire token for scope %q: %w", c.scope, err)
	}
	req.Header.Set("Authorization", "Bearer "+tk.Token)
	return nil
}

func (c *Client) requestURL(path string, query url.Values) string {
	u := c.base + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	return u
}

// decodeResponse decodes a 2xx body into out (out may be nil to ignore the body)
// and maps any non-2xx response to an error.
func decodeResponse(resp *http.Response, out any) error {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return decodeError(resp)
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

// decodeError builds an error from a non-2xx response, preferring the API's
// {"error": "..."} envelope and falling back to the raw body or status line.
func decodeError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))

	var env struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &env) == nil && env.Error != "" {
		return fmt.Errorf("%s: %s", resp.Status, env.Error)
	}

	if msg := strings.TrimSpace(string(body)); msg != "" {
		return fmt.Errorf("%s: %s", resp.Status, msg)
	}
	return fmt.Errorf("%s", resp.Status)
}
