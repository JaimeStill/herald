package documents_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JaimeStill/herald/internal/documents"
	"github.com/JaimeStill/herald/pkg/pagination"
)

type mockSystem struct {
	listFn   func(ctx context.Context, page pagination.PageRequest, filters documents.Filters) (*pagination.PageResult[documents.Document], error)
	findFn   func(ctx context.Context, id uuid.UUID) (*documents.Document, error)
	createFn func(ctx context.Context, cmd documents.CreateCommand) (*documents.Document, error)
	deleteFn func(ctx context.Context, id uuid.UUID) error
}

func (m *mockSystem) Handler(maxUploadSize int64) *documents.Handler {
	return documents.NewHandler(m, slog.New(slog.NewTextHandler(io.Discard, nil)), pagination.Config{DefaultPageSize: 20, MaxPageSize: 100}, maxUploadSize)
}

func (m *mockSystem) List(ctx context.Context, page pagination.PageRequest, filters documents.Filters) (*pagination.PageResult[documents.Document], error) {
	return m.listFn(ctx, page, filters)
}

func (m *mockSystem) Find(ctx context.Context, id uuid.UUID) (*documents.Document, error) {
	return m.findFn(ctx, id)
}

func (m *mockSystem) Create(ctx context.Context, cmd documents.CreateCommand) (*documents.Document, error) {
	return m.createFn(ctx, cmd)
}

func (m *mockSystem) Delete(ctx context.Context, id uuid.UUID) error {
	return m.deleteFn(ctx, id)
}

func newTestHandler(sys *mockSystem) *documents.Handler {
	return documents.NewHandler(
		sys,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		pagination.Config{DefaultPageSize: 20, MaxPageSize: 100},
		50*1024*1024,
	)
}

func setupMux(h *documents.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	group := h.Routes()
	for _, route := range group.Routes {
		pattern := route.Method + " " + group.Prefix + route.Pattern
		mux.HandleFunc(pattern, route.Handler)
	}
	return mux
}

func sampleDoc() documents.Document {
	return documents.Document{
		ID:               uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		ExternalID:       12345,
		ExternalPlatform: "HQ",
		Filename:         "report.pdf",
		ContentType:      "application/pdf",
		SizeBytes:        1024,
		PageCount:        ptr(5),
		StorageKey:       "documents/550e8400-e29b-41d4-a716-446655440000",
		Status:           "pending",
		UploadedAt:       time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
		UpdatedAt:        time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
	}
}

func TestHandlerList(t *testing.T) {
	doc := sampleDoc()
	sys := &mockSystem{
		listFn: func(_ context.Context, _ pagination.PageRequest, _ documents.Filters) (*pagination.PageResult[documents.Document], error) {
			result := pagination.NewPageResult([]documents.Document{doc}, 1, 1, 20)
			return &result, nil
		},
	}

	mux := setupMux(newTestHandler(sys))

	t.Run("returns paginated list", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/documents", nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}

		var result pagination.PageResult[documents.Document]
		if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
			t.Fatalf("decode: %v", err)
		}

		if result.Total != 1 {
			t.Errorf("total = %d, want 1", result.Total)
		}
		if len(result.Data) != 1 {
			t.Fatalf("data length = %d, want 1", len(result.Data))
		}
		if result.Data[0].ID != doc.ID {
			t.Errorf("id = %v, want %v", result.Data[0].ID, doc.ID)
		}
	})

	t.Run("passes query filters", func(t *testing.T) {
		var captured documents.Filters
		sys.listFn = func(_ context.Context, _ pagination.PageRequest, f documents.Filters) (*pagination.PageResult[documents.Document], error) {
			captured = f
			result := pagination.NewPageResult([]documents.Document{}, 0, 1, 20)
			return &result, nil
		}

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/documents?status=pending&filename=report", nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if captured.Status == nil || *captured.Status != "pending" {
			t.Errorf("status filter = %v, want pending", captured.Status)
		}
		if captured.Filename == nil || *captured.Filename != "report" {
			t.Errorf("filename filter = %v, want report", captured.Filename)
		}
	})
}

func TestHandlerFind(t *testing.T) {
	doc := sampleDoc()

	t.Run("returns document by id", func(t *testing.T) {
		sys := &mockSystem{
			findFn: func(_ context.Context, id uuid.UUID) (*documents.Document, error) {
				if id != doc.ID {
					return nil, documents.ErrNotFound
				}
				return &doc, nil
			},
		}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/documents/"+doc.ID.String(), nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}

		var got documents.Document
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.ID != doc.ID {
			t.Errorf("id = %v, want %v", got.ID, doc.ID)
		}
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		sys := &mockSystem{}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/documents/not-a-uuid", nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("not found returns 404", func(t *testing.T) {
		sys := &mockSystem{
			findFn: func(_ context.Context, _ uuid.UUID) (*documents.Document, error) {
				return nil, documents.ErrNotFound
			},
		}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/documents/"+uuid.New().String(), nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})
}

func TestHandlerSearch(t *testing.T) {
	doc := sampleDoc()

	t.Run("returns search results", func(t *testing.T) {
		sys := &mockSystem{
			listFn: func(_ context.Context, _ pagination.PageRequest, _ documents.Filters) (*pagination.PageResult[documents.Document], error) {
				result := pagination.NewPageResult([]documents.Document{doc}, 1, 1, 20)
				return &result, nil
			},
		}
		mux := setupMux(newTestHandler(sys))

		body, _ := json.Marshal(documents.SearchRequest{
			PageRequest: pagination.PageRequest{Page: 1, PageSize: 20},
		})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/documents/search", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}

		var result pagination.PageResult[documents.Document]
		if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if result.Total != 1 {
			t.Errorf("total = %d, want 1", result.Total)
		}
	})

	t.Run("invalid json returns 400", func(t *testing.T) {
		sys := &mockSystem{}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/documents/search", bytes.NewReader([]byte("not json")))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("normalizes pagination", func(t *testing.T) {
		var capturedPage pagination.PageRequest
		sys := &mockSystem{
			listFn: func(_ context.Context, page pagination.PageRequest, _ documents.Filters) (*pagination.PageResult[documents.Document], error) {
				capturedPage = page
				result := pagination.NewPageResult([]documents.Document{}, 0, page.Page, page.PageSize)
				return &result, nil
			},
		}
		mux := setupMux(newTestHandler(sys))

		body, _ := json.Marshal(documents.SearchRequest{
			PageRequest: pagination.PageRequest{Page: 0, PageSize: 0},
		})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/documents/search", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if capturedPage.Page != 1 {
			t.Errorf("page = %d, want 1 (normalized)", capturedPage.Page)
		}
		if capturedPage.PageSize != 20 {
			t.Errorf("page_size = %d, want 20 (default)", capturedPage.PageSize)
		}
	})
}

func TestHandlerUpload(t *testing.T) {
	doc := sampleDoc()

	t.Run("creates document from multipart form", func(t *testing.T) {
		var capturedCmd documents.CreateCommand
		sys := &mockSystem{
			createFn: func(_ context.Context, cmd documents.CreateCommand) (*documents.Document, error) {
				capturedCmd = cmd
				return &doc, nil
			},
		}
		mux := setupMux(newTestHandler(sys))

		body, contentType := createMultipartForm(t, "report.pdf", []byte("fake pdf content"), "12345", "HQ")

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/documents", body)
		req.Header.Set("Content-Type", contentType)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201", rec.Code)
		}
		if capturedCmd.Filename != "report.pdf" {
			t.Errorf("filename = %q, want report.pdf", capturedCmd.Filename)
		}
		if capturedCmd.ExternalID != 12345 {
			t.Errorf("external_id = %d, want 12345", capturedCmd.ExternalID)
		}
		if capturedCmd.ExternalPlatform != "HQ" {
			t.Errorf("external_platform = %q, want HQ", capturedCmd.ExternalPlatform)
		}
	})

	t.Run("missing external_id returns 400", func(t *testing.T) {
		sys := &mockSystem{}
		mux := setupMux(newTestHandler(sys))

		body, contentType := createMultipartForm(t, "report.pdf", []byte("content"), "", "HQ")

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/documents", body)
		req.Header.Set("Content-Type", contentType)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("missing external_platform returns 400", func(t *testing.T) {
		sys := &mockSystem{}
		mux := setupMux(newTestHandler(sys))

		body, contentType := createMultipartForm(t, "report.pdf", []byte("content"), "12345", "")

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/documents", body)
		req.Header.Set("Content-Type", contentType)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("missing file returns 400", func(t *testing.T) {
		sys := &mockSystem{}
		mux := setupMux(newTestHandler(sys))

		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		writer.WriteField("external_id", "12345")
		writer.WriteField("external_platform", "HQ")
		writer.Close()

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/documents", &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("system create error maps status", func(t *testing.T) {
		sys := &mockSystem{
			createFn: func(_ context.Context, _ documents.CreateCommand) (*documents.Document, error) {
				return nil, documents.ErrDuplicate
			},
		}
		mux := setupMux(newTestHandler(sys))

		body, contentType := createMultipartForm(t, "report.pdf", []byte("content"), "12345", "HQ")

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/documents", body)
		req.Header.Set("Content-Type", contentType)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusConflict {
			t.Errorf("status = %d, want 409", rec.Code)
		}
	})
}

func TestHandlerDelete(t *testing.T) {
	docID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	t.Run("deletes document", func(t *testing.T) {
		var capturedID uuid.UUID
		sys := &mockSystem{
			deleteFn: func(_ context.Context, id uuid.UUID) error {
				capturedID = id
				return nil
			},
		}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("DELETE", "/documents/"+docID.String(), nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", rec.Code)
		}
		if capturedID != docID {
			t.Errorf("id = %v, want %v", capturedID, docID)
		}
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		sys := &mockSystem{}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("DELETE", "/documents/not-a-uuid", nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("not found returns 404", func(t *testing.T) {
		sys := &mockSystem{
			deleteFn: func(_ context.Context, _ uuid.UUID) error {
				return documents.ErrNotFound
			},
		}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("DELETE", "/documents/"+uuid.New().String(), nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})
}

func TestHandlerRoutes(t *testing.T) {
	sys := &mockSystem{}
	h := newTestHandler(sys)
	group := h.Routes()

	if group.Prefix != "/documents" {
		t.Errorf("prefix = %q, want /documents", group.Prefix)
	}

	want := []struct {
		method  string
		pattern string
	}{
		{"GET", ""},
		{"GET", "/{id}"},
		{"POST", ""},
		{"POST", "/search"},
		{"DELETE", "/{id}"},
	}

	if len(group.Routes) != len(want) {
		t.Fatalf("route count = %d, want %d", len(group.Routes), len(want))
	}

	for i, w := range want {
		r := group.Routes[i]
		if r.Method != w.method || r.Pattern != w.pattern {
			t.Errorf("route[%d] = %s %s, want %s %s", i, r.Method, r.Pattern, w.method, w.pattern)
		}
	}
}

func createMultipartForm(t *testing.T, filename string, content []byte, externalID, externalPlatform string) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	if len(content) > 0 {
		part, err := writer.CreateFormFile("file", filename)
		if err != nil {
			t.Fatalf("create form file: %v", err)
		}
		part.Write(content)
	}

	if externalID != "" {
		writer.WriteField("external_id", externalID)
	}
	if externalPlatform != "" {
		writer.WriteField("external_platform", externalPlatform)
	}

	writer.Close()
	return &buf, writer.FormDataContentType()
}
