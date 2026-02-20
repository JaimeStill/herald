package middleware_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JaimeStill/herald/pkg/middleware"
)

func TestApplyOrder(t *testing.T) {
	var order []string
	mw := middleware.New()

	mw.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "first")
			next.ServeHTTP(w, r)
		})
	})

	mw.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "second")
			next.ServeHTTP(w, r)
		})
	})

	handler := mw.Apply(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(rec, req)

	if len(order) != 3 {
		t.Fatalf("execution count: got %d, want 3", len(order))
	}
	if order[0] != "first" || order[1] != "second" || order[2] != "handler" {
		t.Errorf("order: got %v, want [first second handler]", order)
	}
}

func TestCORSDisabled(t *testing.T) {
	cfg := &middleware.CORSConfig{Enabled: false}
	handler := middleware.CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "http://example.com")
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("CORS headers should not be set when disabled")
	}
}

func TestCORSAllowedOrigin(t *testing.T) {
	cfg := &middleware.CORSConfig{
		Enabled:        true,
		Origins:        []string{"http://example.com"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type"},
		MaxAge:         3600,
	}

	handler := middleware.CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "http://example.com")
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://example.com" {
		t.Errorf("allow-origin: got %s, want http://example.com", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST" {
		t.Errorf("allow-methods: got %s", got)
	}
	if got := rec.Header().Get("Access-Control-Max-Age"); got != "3600" {
		t.Errorf("max-age: got %s, want 3600", got)
	}
}

func TestCORSDisallowedOrigin(t *testing.T) {
	cfg := &middleware.CORSConfig{
		Enabled: true,
		Origins: []string{"http://allowed.com"},
	}

	handler := middleware.CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "http://denied.com")
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("should not set allow-origin for disallowed origin")
	}
}

func TestCORSPreflight(t *testing.T) {
	cfg := &middleware.CORSConfig{
		Enabled:        true,
		Origins:        []string{"http://example.com"},
		AllowedMethods: []string{"GET", "POST"},
	}

	var handlerCalled bool
	handler := middleware.CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "http://example.com")
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("preflight status: got %d, want 200", rec.Code)
	}
	if handlerCalled {
		t.Error("handler should not be called for preflight")
	}
}

func TestCORSCredentials(t *testing.T) {
	cfg := &middleware.CORSConfig{
		Enabled:          true,
		Origins:          []string{"http://example.com"},
		AllowCredentials: true,
	}

	handler := middleware.CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "http://example.com")
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("allow-credentials: got %s, want true", got)
	}
}

func TestLogger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	var handlerCalled bool
	handler := middleware.Logger(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	handler.ServeHTTP(rec, req)

	if !handlerCalled {
		t.Error("inner handler should have been called")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", rec.Code)
	}
}

func TestCORSConfigFinalizeDefaults(t *testing.T) {
	cfg := middleware.CORSConfig{}
	if err := cfg.Finalize(nil); err != nil {
		t.Fatalf("finalize failed: %v", err)
	}

	if len(cfg.AllowedMethods) != 5 {
		t.Errorf("allowed_methods: got %d, want 5", len(cfg.AllowedMethods))
	}
	if len(cfg.AllowedHeaders) != 2 {
		t.Errorf("allowed_headers: got %d, want 2", len(cfg.AllowedHeaders))
	}
	if cfg.MaxAge != 3600 {
		t.Errorf("max_age: got %d, want 3600", cfg.MaxAge)
	}
}

func TestCORSConfigFinalizeEnv(t *testing.T) {
	t.Setenv("TEST_CORS_ENABLED", "true")
	t.Setenv("TEST_CORS_ORIGINS", "http://a.com, http://b.com")
	t.Setenv("TEST_CORS_CREDS", "true")

	env := &middleware.CORSEnv{
		Enabled:          "TEST_CORS_ENABLED",
		Origins:          "TEST_CORS_ORIGINS",
		AllowCredentials: "TEST_CORS_CREDS",
	}

	cfg := middleware.CORSConfig{}
	if err := cfg.Finalize(env); err != nil {
		t.Fatalf("finalize failed: %v", err)
	}

	if !cfg.Enabled {
		t.Error("enabled should be true")
	}
	if len(cfg.Origins) != 2 {
		t.Fatalf("origins: got %d, want 2", len(cfg.Origins))
	}
	if cfg.Origins[0] != "http://a.com" || cfg.Origins[1] != "http://b.com" {
		t.Errorf("origins: got %v", cfg.Origins)
	}
	if !cfg.AllowCredentials {
		t.Error("allow_credentials should be true")
	}
}

func TestCORSConfigMerge(t *testing.T) {
	base := middleware.CORSConfig{
		Enabled:        false,
		Origins:        []string{"http://base.com"},
		AllowedMethods: []string{"GET"},
		MaxAge:         3600,
	}

	overlay := middleware.CORSConfig{
		Enabled: true,
		Origins: []string{"http://overlay.com"},
		MaxAge:  7200,
	}

	base.Merge(&overlay)

	if !base.Enabled {
		t.Error("enabled should be true after merge")
	}
	if len(base.Origins) != 1 || base.Origins[0] != "http://overlay.com" {
		t.Errorf("origins: got %v", base.Origins)
	}
	if base.MaxAge != 7200 {
		t.Errorf("max_age: got %d, want 7200", base.MaxAge)
	}
}
