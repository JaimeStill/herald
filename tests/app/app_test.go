package app_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JaimeStill/herald/app"
)

func TestNewModule(t *testing.T) {
	m, err := app.NewModule("/app", nil)
	if err != nil {
		t.Fatalf("NewModule: %v", err)
	}
	if m.Prefix() != "/app" {
		t.Errorf("prefix: got %q, want %q", m.Prefix(), "/app")
	}
}

func TestShellTemplate(t *testing.T) {
	m, err := app.NewModule("/app", nil)
	if err != nil {
		t.Fatalf("NewModule: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/app/", nil)
	m.Serve(rec, req)

	body := rec.Body.String()

	checks := []struct {
		name    string
		content string
	}{
		{"base href", `<base href="/app/"`},
		{"css link", `dist/app.css`},
		{"js script", `dist/app.js`},
		{"title", `Herald`},
		{"app content", `id="app-content"`},
	}

	for _, check := range checks {
		if !strings.Contains(body, check.content) {
			t.Errorf("shell template missing %s: want %q in body", check.name, check.content)
		}
	}
}

func TestDistAssetServing(t *testing.T) {
	m, err := app.NewModule("/app", nil)
	if err != nil {
		t.Fatalf("NewModule: %v", err)
	}

	tests := []struct {
		name string
		path string
	}{
		{"javascript", "/app/dist/app.js"},
		{"css", "/app/dist/app.css"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("GET", tt.path, nil)
			m.Serve(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("status: got %d, want 200", rec.Code)
			}
			if rec.Body.Len() == 0 {
				t.Error("expected non-empty response body")
			}
		})
	}
}

func TestAuthConfigInjection(t *testing.T) {
	authCfg := &app.ClientAuthConfig{
		TenantID:      "test-tenant-id",
		ClientID:      "test-client-id",
		Authority:     "https://login.microsoftonline.com/test-tenant-id/v2.0",
		CacheLocation: "localStorage",
	}

	m, err := app.NewModule("/app", authCfg)
	if err != nil {
		t.Fatalf("NewModule: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/app/", nil)
	m.Serve(rec, req)

	body := rec.Body.String()

	checks := []struct {
		name    string
		content string
	}{
		{"config script tag", `<script id="herald-config" type="application/json">`},
		{"tenant_id", `"tenant_id":"test-tenant-id"`},
		{"client_id", `"client_id":"test-client-id"`},
		{"redirect_uri", `"redirect_uri":"/app/"`},
		{"authority", `"authority":"https://login.microsoftonline.com/test-tenant-id/v2.0"`},
		{"cache_location", `"cache_location":"localStorage"`},
		{"user-menu", `id="user-menu"`},
	}

	for _, check := range checks {
		if !strings.Contains(body, check.content) {
			t.Errorf("auth config missing %s: want %q in body", check.name, check.content)
		}
	}
}

func TestNoAuthConfigWhenNil(t *testing.T) {
	m, err := app.NewModule("/app", nil)
	if err != nil {
		t.Fatalf("NewModule: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/app/", nil)
	m.Serve(rec, req)

	body := rec.Body.String()

	absent := []struct {
		name    string
		content string
	}{
		{"config script tag", `herald-config`},
		{"user-menu", `user-menu`},
	}

	for _, check := range absent {
		if strings.Contains(body, check.content) {
			t.Errorf("should not contain %s when auth is nil: found %q in body", check.name, check.content)
		}
	}
}

func TestSPAFallback(t *testing.T) {
	m, err := app.NewModule("/app", nil)
	if err != nil {
		t.Fatalf("NewModule: %v", err)
	}

	paths := []string{
		"/app/",
		"/app/prompts",
		"/app/review/some-doc-id",
		"/app/unknown/nested/path",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("GET", path, nil)
			m.Serve(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("status: got %d, want 200", rec.Code)
			}

			body := rec.Body.String()
			if !strings.Contains(body, `<base href="/app/"`) {
				t.Error("SPA fallback did not render shell template")
			}
		})
	}
}
