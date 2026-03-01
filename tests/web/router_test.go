package web_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JaimeStill/herald/pkg/web"
)

func TestRouterRegisteredRoute(t *testing.T) {
	r := web.NewRouter()
	r.HandleFunc("GET /hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/hello", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("registered route: got %d, want 200", rec.Code)
	}
}

func TestRouterFallback(t *testing.T) {
	r := web.NewRouter()
	r.HandleFunc("GET /known", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r.SetFallback(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/unknown", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusTeapot {
		t.Errorf("fallback: got %d, want %d", rec.Code, http.StatusTeapot)
	}
}

func TestRouterNoFallback(t *testing.T) {
	r := web.NewRouter()
	r.HandleFunc("GET /known", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/unknown", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("no fallback: got %d, want 404", rec.Code)
	}
}

func TestRouterHandle(t *testing.T) {
	r := web.NewRouter()
	r.Handle("GET /mux", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/mux", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Errorf("Handle: got %d, want %d", rec.Code, http.StatusAccepted)
	}
}
