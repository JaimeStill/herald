package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

// Client is an authenticated, retrying HTTP client for the Herald API. It is safe
// for concurrent use and intended to be shared across a batch; the retry policy
// then applies across all in-flight requests. Throughput is governed by the
// caller's concurrency cap (see RunBatch), not by a client-side rate limiter:
// when the server pushes back with 429/5xx the client honors Retry-After.
type Client struct {
	base       string
	client     *http.Client
	cred       azcore.TokenCredential // nil when auth is disabled
	scope      string
	maxRetries int
	baseDelay  time.Duration
	maxDelay   time.Duration
	timeout    time.Duration
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
		base:       strings.TrimRight(s.API, "/"),
		client:     &http.Client{},
		cred:       cred,
		scope:      s.Scope,
		maxRetries: s.MaxRetries,
		baseDelay:  s.RetryBaseDelayDuration(),
		maxDelay:   s.RetryMaxDelayDuration(),
		timeout:    s.TimeoutDuration(),
	}, nil
}

// getJSON issues a GET, bounded by the client timeout, and decodes a 2xx JSON
// body into out.
func (c *Client) getJSON(ctx context.Context, path string, query url.Values, out any) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.do(ctx, func(rc context.Context) (*http.Request, error) {
		return http.NewRequestWithContext(rc, http.MethodGet, c.requestURL(path, query), nil)
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decodeResponse(resp, out)
}

// postMultipart uploads a single file with form fields, bounded by the client
// timeout, and decodes the 2xx JSON body into out. The multipart body is built
// once and replayed on each retry.
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

	body := buf.Bytes()
	contentType := mw.FormDataContentType()

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.do(ctx, func(rc context.Context) (*http.Request, error) {
		req, err := http.NewRequestWithContext(rc, http.MethodPost, c.requestURL(path, nil), bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", contentType)
		return req, nil
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decodeResponse(resp, out)
}

// postStream issues a POST and returns the streaming response on a 2xx status.
// Unlike the buffered helpers it does not bound the request with the client
// timeout — the caller owns both the deadline (it must wrap ctx) and the body
// (it must close it), since the stream outlives this call. Non-2xx responses are
// decoded as errors before returning.
func (c *Client) postStream(ctx context.Context, path string) (*http.Response, error) {
	resp, err := c.do(ctx, func(rc context.Context) (*http.Request, error) {
		req, err := http.NewRequestWithContext(rc, http.MethodPost, c.requestURL(path, nil), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "text/event-stream")
		return req, nil
	})
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		return nil, decodeError(resp)
	}
	return resp, nil
}

// do executes a request with retry. newRequest is called once per attempt so the
// request (and its body) is rebuilt fresh each time. Requests are retried on 429
// and 5xx (honoring Retry-After) and on transport errors, up to maxRetries; other
// 4xx responses are returned without retry for the caller to decode. The caller
// owns ctx (including its deadline) and must close the returned body.
func (c *Client) do(ctx context.Context, newRequest func(context.Context) (*http.Request, error)) (*http.Response, error) {
	var (
		lastErr    error
		retryAfter time.Duration
	)

	for attempt := 0; ; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(c.backoff(attempt, retryAfter)):
			}
		}
		retryAfter = 0

		req, err := newRequest(ctx)
		if err != nil {
			return nil, err
		}
		if err := c.authorize(ctx, req); err != nil {
			return nil, err
		}

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = err
			if attempt >= c.maxRetries {
				return nil, fmt.Errorf("request failed after %d attempts: %w", attempt+1, lastErr)
			}
			continue
		}

		if isRetryable(resp.StatusCode) {
			retryAfter = parseRetryAfter(resp.Header.Get("Retry-After"))
			drain(resp.Body)
			lastErr = fmt.Errorf("server responded %s", resp.Status)
			if attempt >= c.maxRetries {
				return nil, fmt.Errorf("after %d attempts: %w", attempt+1, lastErr)
			}
			continue
		}

		return resp, nil
	}
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

func isRetryable(status int) bool {
	return status == http.StatusTooManyRequests || status >= 500
}

// backoff returns the wait before the given attempt (1-based for retries).
// A positive Retry-After is honored verbatim; otherwise it is exponential from
// the configured base delay, capped at the configured max delay, with full
// jitter.
func (c *Client) backoff(attempt int, retryAfter time.Duration) time.Duration {
	if retryAfter > 0 {
		return retryAfter
	}
	exp := c.baseDelay << (attempt - 1)
	if exp <= 0 || exp > c.maxDelay {
		exp = c.maxDelay
	}
	if exp <= 0 {
		return 0
	}
	return time.Duration(rand.Int64N(int64(exp)))
}

// parseRetryAfter interprets a Retry-After header value, which may be a number of
// seconds or an HTTP date. It returns 0 when absent or unparseable.
func parseRetryAfter(v string) time.Duration {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil {
		if secs <= 0 {
			return 0
		}
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return 0
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

// drain discards and closes a response body so the connection can be reused.
func drain(rc io.ReadCloser) {
	_, _ = io.Copy(io.Discard, io.LimitReader(rc, 64*1024))
	_ = rc.Close()
}
