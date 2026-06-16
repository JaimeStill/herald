package cli_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JaimeStill/herald/internal/cli"
	"github.com/JaimeStill/herald/pkg/auth"
)

// testClient builds an unauthenticated client pointed at url.
func testClient(t *testing.T, url string) *cli.Client {
	t.Helper()
	c, err := cli.NewClient(&cli.Settings{
		API:     url,
		Output:  cli.OutputJSON,
		Timeout: "30s",
		Auth:    auth.Config{Mode: auth.ModeNone},
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	return c
}

func TestUploadDocument(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: got %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/documents" {
			t.Errorf("path: got %s, want /api/documents", r.URL.Path)
		}
		if h := r.Header.Get("Authorization"); h != "" {
			t.Errorf("unexpected Authorization header: %q", h)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}
		if got := r.FormValue("external_id"); got != "7" {
			t.Errorf("external_id: got %q, want 7", got)
		}
		if got := r.FormValue("external_platform"); got != "PROG" {
			t.Errorf("external_platform: got %q, want PROG", got)
		}
		f, hdr, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("form file: %v", err)
		}
		f.Close()
		if hdr.Filename != "sample.pdf" {
			t.Errorf("filename: got %q, want sample.pdf", hdr.Filename)
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id":                "11111111-1111-1111-1111-111111111111",
			"external_id":       7,
			"external_platform": "PROG",
			"status":            "pending",
		})
	}))
	defer srv.Close()

	file := filepath.Join(t.TempDir(), "sample.pdf")
	writeFile(t, file, "%PDF-1.4 fake content")

	doc, err := testClient(t, srv.URL).UploadDocument(context.Background(), file, 7, "PROG")
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if doc.ExternalID != 7 {
		t.Errorf("external_id: got %d, want 7", doc.ExternalID)
	}
	if doc.Status != "pending" {
		t.Errorf("status: got %q, want pending", doc.Status)
	}
}

func TestListDocuments(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("status"); got != "review" {
			t.Errorf("status query: got %q, want review", got)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data":        []map[string]any{{"id": "11111111-1111-1111-1111-111111111111", "status": "review"}},
			"total":       1,
			"page":        1,
			"page_size":   20,
			"total_pages": 1,
		})
	}))
	defer srv.Close()

	result, err := testClient(t, srv.URL).ListDocuments(context.Background(), url.Values{"status": {"review"}})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("total: got %d, want 1", result.Total)
	}
	if len(result.Data) != 1 {
		t.Fatalf("data: got %d items, want 1", len(result.Data))
	}
	if result.Data[0].Status != "review" {
		t.Errorf("data[0].status: got %q, want review", result.Data[0].Status)
	}
}

func TestGetDocument(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/documents/abc" {
			t.Errorf("path: got %s, want /api/documents/abc", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"id":     "11111111-1111-1111-1111-111111111111",
			"status": "complete",
		})
	}))
	defer srv.Close()

	doc, err := testClient(t, srv.URL).GetDocument(context.Background(), "abc")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if doc.Status != "complete" {
		t.Errorf("status: got %q, want complete", doc.Status)
	}
}

// sseHandler writes each event to the response and flushes between them.
func sseHandler(events ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)
		for _, e := range events {
			w.Write([]byte(e))
			if flusher != nil {
				flusher.Flush()
			}
		}
	}
}

func TestClassifyStreamComplete(t *testing.T) {
	// The complete event carries the full ExecutionEvent envelope; the
	// classification lives in the nested data field (regression for the
	// envelope-decode bug).
	srv := httptest.NewServer(sseHandler(
		"event: node.start\ndata: {\"type\":\"node.start\",\"data\":{\"node\":\"init\"}}\n\n",
		"event: complete\ndata: {\"type\":\"complete\",\"data\":{\"id\":\"11111111-1111-1111-1111-111111111111\",\"document_id\":\"22222222-2222-2222-2222-222222222222\",\"classification\":\"SECRET//NOFORN\",\"confidence\":\"HIGH\"}}\n\n",
	))
	defer srv.Close()

	cl, err := testClient(t, srv.URL).Classify(context.Background(), "22222222-2222-2222-2222-222222222222")
	if err != nil {
		t.Fatalf("classify: %v", err)
	}
	if cl.Classification != "SECRET//NOFORN" {
		t.Errorf("classification: got %q, want SECRET//NOFORN", cl.Classification)
	}
	if cl.Confidence != "HIGH" {
		t.Errorf("confidence: got %q, want HIGH", cl.Confidence)
	}
}

func TestClassifyStreamError(t *testing.T) {
	srv := httptest.NewServer(sseHandler(
		"event: error\ndata: {\"type\":\"error\",\"data\":{\"message\":\"page render failed\"}}\n\n",
	))
	defer srv.Close()

	_, err := testClient(t, srv.URL).Classify(context.Background(), "doc")
	if err == nil {
		t.Fatal("expected error from error event")
	}
	if !strings.Contains(err.Error(), "page render failed") {
		t.Errorf("error %q does not contain %q", err.Error(), "page render failed")
	}
}

func TestClassifyStreamTruncated(t *testing.T) {
	srv := httptest.NewServer(sseHandler(
		"event: node.start\ndata: {\"type\":\"node.start\",\"data\":{\"node\":\"init\"}}\n\n",
	))
	defer srv.Close()

	_, err := testClient(t, srv.URL).Classify(context.Background(), "doc")
	if err == nil {
		t.Fatal("expected error for stream with no terminal event")
	}
	if !strings.Contains(err.Error(), "ended without a result") {
		t.Errorf("error %q does not contain %q", err.Error(), "ended without a result")
	}
}

func TestDecodeErrorEnvelope(t *testing.T) {
	t.Run("json error envelope", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "document not found"})
		}))
		defer srv.Close()

		_, err := testClient(t, srv.URL).GetDocument(context.Background(), "missing")
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "404") || !strings.Contains(err.Error(), "document not found") {
			t.Errorf("error %q should contain status and message", err.Error())
		}
	})

	t.Run("plain body fallback", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte("upstream exploded"))
		}))
		defer srv.Close()

		_, err := testClient(t, srv.URL).GetDocument(context.Background(), "x")
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "upstream exploded") {
			t.Errorf("error %q should contain the raw body", err.Error())
		}
	})
}

func TestClassifyPreStreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid document id"})
	}))
	defer srv.Close()

	_, err := testClient(t, srv.URL).Classify(context.Background(), "bad")
	if err == nil {
		t.Fatal("expected pre-stream error")
	}
	if !strings.Contains(err.Error(), "invalid document id") {
		t.Errorf("error %q does not contain %q", err.Error(), "invalid document id")
	}
}
