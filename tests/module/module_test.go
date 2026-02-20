package module_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JaimeStill/herald/pkg/module"
)

func TestNewValidPrefix(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
	}{
		{"api", "/api"},
		{"scalar", "/scalar"},
		{"docs", "/docs"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := module.New(tt.prefix, http.NewServeMux())
			if m.Prefix() != tt.prefix {
				t.Errorf("prefix: got %s, want %s", m.Prefix(), tt.prefix)
			}
		})
	}
}

func TestNewInvalidPrefixPanics(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
	}{
		{"empty", ""},
		{"no leading slash", "api"},
		{"nested path", "/api/v1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Error("expected panic for invalid prefix")
				}
			}()
			module.New(tt.prefix, http.NewServeMux())
		})
	}
}

func TestServePrefixStripping(t *testing.T) {
	mux := http.NewServeMux()

	var receivedPath string
	mux.HandleFunc("GET /items", func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	})

	m := module.New("/api", mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/items", nil)
	m.Serve(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", rec.Code)
	}
	if receivedPath != "/items" {
		t.Errorf("inner path: got %s, want /items", receivedPath)
	}
}

func TestServeRootPath(t *testing.T) {
	mux := http.NewServeMux()

	var receivedPath string
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	})

	m := module.New("/api", mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api", nil)
	m.Serve(rec, req)

	if receivedPath != "/" {
		t.Errorf("root path: got %s, want /", receivedPath)
	}
}

func TestModuleMiddleware(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	m := module.New("/api", mux)

	var middlewareCalled bool
	m.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareCalled = true
			next.ServeHTTP(w, r)
		})
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api", nil)
	m.Serve(rec, req)

	if !middlewareCalled {
		t.Error("module middleware should have been called")
	}
}

func TestRouterDispatch(t *testing.T) {
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("api"))
	})

	scalarMux := http.NewServeMux()
	scalarMux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("scalar"))
	})

	router := module.NewRouter()
	router.Mount(module.New("/api", apiMux))
	router.Mount(module.New("/scalar", scalarMux))

	tests := []struct {
		name     string
		path     string
		wantBody string
	}{
		{"api module", "/api/health", "api"},
		{"scalar module", "/scalar", "scalar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("GET", tt.path, nil)
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("status: got %d, want 200", rec.Code)
			}
			if body := rec.Body.String(); body != tt.wantBody {
				t.Errorf("body: got %s, want %s", body, tt.wantBody)
			}
		})
	}
}

func TestRouterNativeFallback(t *testing.T) {
	router := module.NewRouter()
	router.HandleNative("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/healthz", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", rec.Code)
	}
	if body := rec.Body.String(); body != "ok" {
		t.Errorf("body: got %s, want ok", body)
	}
}

func TestRouterTrailingSlashNormalization(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /items", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router := module.NewRouter()
	router.Mount(module.New("/api", mux))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/items/", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("trailing slash normalization: got %d, want 200", rec.Code)
	}
}
