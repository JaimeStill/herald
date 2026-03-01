package web_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JaimeStill/herald/pkg/web"
)

func TestServeEmbeddedFile(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		contentType string
	}{
		{"json", []byte(`{"ok":true}`), "application/json"},
		{"html", []byte(`<h1>hello</h1>`), "text/html"},
		{"plain", []byte("hello"), "text/plain"},
		{"empty", []byte{}, "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := web.ServeEmbeddedFile(tt.data, tt.contentType)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/file", nil)
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("status: got %d, want 200", rec.Code)
			}

			ct := rec.Header().Get("Content-Type")
			if ct != tt.contentType {
				t.Errorf("content-type: got %q, want %q", ct, tt.contentType)
			}

			if rec.Body.String() != string(tt.data) {
				t.Errorf("body: got %q, want %q", rec.Body.String(), string(tt.data))
			}
		})
	}
}

func TestServeEmbeddedFileHeaders(t *testing.T) {
	data := []byte("test content")
	handler := web.ServeEmbeddedFile(data, "text/css")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/style.css", nil)
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Type") != "text/css" {
		t.Errorf("content-type: got %q, want %q", rec.Header().Get("Content-Type"), "text/css")
	}

	if rec.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", rec.Code)
	}
}
