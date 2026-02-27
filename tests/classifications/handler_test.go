package classifications_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JaimeStill/herald/internal/classifications"
	"github.com/JaimeStill/herald/pkg/pagination"
)

type mockSystem struct {
	listFn           func(ctx context.Context, page pagination.PageRequest, filters classifications.Filters) (*pagination.PageResult[classifications.Classification], error)
	findFn           func(ctx context.Context, id uuid.UUID) (*classifications.Classification, error)
	findByDocumentFn func(ctx context.Context, documentID uuid.UUID) (*classifications.Classification, error)
	classifyFn       func(ctx context.Context, documentID uuid.UUID) (*classifications.Classification, error)
	validateFn       func(ctx context.Context, id uuid.UUID, cmd classifications.ValidateCommand) (*classifications.Classification, error)
	updateFn         func(ctx context.Context, id uuid.UUID, cmd classifications.UpdateCommand) (*classifications.Classification, error)
	deleteFn         func(ctx context.Context, id uuid.UUID) error
}

func (m *mockSystem) Handler() *classifications.Handler {
	return classifications.NewHandler(m, slog.New(slog.NewTextHandler(io.Discard, nil)), pagination.Config{DefaultPageSize: 20, MaxPageSize: 100})
}

func (m *mockSystem) List(ctx context.Context, page pagination.PageRequest, filters classifications.Filters) (*pagination.PageResult[classifications.Classification], error) {
	return m.listFn(ctx, page, filters)
}

func (m *mockSystem) Find(ctx context.Context, id uuid.UUID) (*classifications.Classification, error) {
	return m.findFn(ctx, id)
}

func (m *mockSystem) FindByDocument(ctx context.Context, documentID uuid.UUID) (*classifications.Classification, error) {
	return m.findByDocumentFn(ctx, documentID)
}

func (m *mockSystem) Classify(ctx context.Context, documentID uuid.UUID) (*classifications.Classification, error) {
	return m.classifyFn(ctx, documentID)
}

func (m *mockSystem) Validate(ctx context.Context, id uuid.UUID, cmd classifications.ValidateCommand) (*classifications.Classification, error) {
	return m.validateFn(ctx, id, cmd)
}

func (m *mockSystem) Update(ctx context.Context, id uuid.UUID, cmd classifications.UpdateCommand) (*classifications.Classification, error) {
	return m.updateFn(ctx, id, cmd)
}

func (m *mockSystem) Delete(ctx context.Context, id uuid.UUID) error {
	return m.deleteFn(ctx, id)
}

func newTestHandler(sys *mockSystem) *classifications.Handler {
	return classifications.NewHandler(
		sys,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		pagination.Config{DefaultPageSize: 20, MaxPageSize: 100},
	)
}

func setupMux(h *classifications.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	group := h.Routes()
	for _, route := range group.Routes {
		pattern := route.Method + " " + group.Prefix + route.Pattern
		mux.HandleFunc(pattern, route.Handler)
	}
	return mux
}

func sampleClassification() classifications.Classification {
	now := time.Now().Truncate(time.Second)
	return classifications.Classification{
		ID:             uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		DocumentID:     uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
		Classification: "SECRET",
		Confidence:     "HIGH",
		MarkingsFound:  []string{"SECRET", "NOFORN"},
		Rationale:      "Banner markings indicate SECRET//NOFORN.",
		ClassifiedAt:   now,
		ModelName:      "gpt-5-mini",
		ProviderName:   "azure",
	}
}

func TestHandlerList(t *testing.T) {
	c := sampleClassification()
	sys := &mockSystem{
		listFn: func(_ context.Context, _ pagination.PageRequest, _ classifications.Filters) (*pagination.PageResult[classifications.Classification], error) {
			result := pagination.NewPageResult([]classifications.Classification{c}, 1, 1, 20)
			return &result, nil
		},
	}

	mux := setupMux(newTestHandler(sys))

	t.Run("returns paginated list", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/classifications", nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}

		var result pagination.PageResult[classifications.Classification]
		if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
			t.Fatalf("decode: %v", err)
		}

		if result.Total != 1 {
			t.Errorf("total = %d, want 1", result.Total)
		}
		if len(result.Data) != 1 {
			t.Fatalf("data length = %d, want 1", len(result.Data))
		}
		if result.Data[0].ID != c.ID {
			t.Errorf("id = %v, want %v", result.Data[0].ID, c.ID)
		}
	})

	t.Run("passes query filters", func(t *testing.T) {
		var captured classifications.Filters
		sys.listFn = func(_ context.Context, _ pagination.PageRequest, f classifications.Filters) (*pagination.PageResult[classifications.Classification], error) {
			captured = f
			result := pagination.NewPageResult([]classifications.Classification{}, 0, 1, 20)
			return &result, nil
		}

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/classifications?classification=SECRET&confidence=HIGH", nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if captured.Classification == nil || *captured.Classification != "SECRET" {
			t.Errorf("classification filter = %v, want SECRET", captured.Classification)
		}
		if captured.Confidence == nil || *captured.Confidence != "HIGH" {
			t.Errorf("confidence filter = %v, want HIGH", captured.Confidence)
		}
	})
}

func TestHandlerFind(t *testing.T) {
	c := sampleClassification()

	t.Run("returns classification by id", func(t *testing.T) {
		sys := &mockSystem{
			findFn: func(_ context.Context, id uuid.UUID) (*classifications.Classification, error) {
				if id != c.ID {
					return nil, classifications.ErrNotFound
				}
				return &c, nil
			},
		}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/classifications/"+c.ID.String(), nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}

		var got classifications.Classification
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.ID != c.ID {
			t.Errorf("id = %v, want %v", got.ID, c.ID)
		}
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		sys := &mockSystem{}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/classifications/not-a-uuid", nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("not found returns 404", func(t *testing.T) {
		sys := &mockSystem{
			findFn: func(_ context.Context, _ uuid.UUID) (*classifications.Classification, error) {
				return nil, classifications.ErrNotFound
			},
		}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/classifications/"+uuid.New().String(), nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})
}

func TestHandlerFindByDocument(t *testing.T) {
	c := sampleClassification()

	t.Run("returns classification by document id", func(t *testing.T) {
		sys := &mockSystem{
			findByDocumentFn: func(_ context.Context, docID uuid.UUID) (*classifications.Classification, error) {
				if docID != c.DocumentID {
					return nil, classifications.ErrNotFound
				}
				return &c, nil
			},
		}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/classifications/document/"+c.DocumentID.String(), nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}

		var got classifications.Classification
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.DocumentID != c.DocumentID {
			t.Errorf("document_id = %v, want %v", got.DocumentID, c.DocumentID)
		}
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		sys := &mockSystem{}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/classifications/document/not-a-uuid", nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("not found returns 404", func(t *testing.T) {
		sys := &mockSystem{
			findByDocumentFn: func(_ context.Context, _ uuid.UUID) (*classifications.Classification, error) {
				return nil, classifications.ErrNotFound
			},
		}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/classifications/document/"+uuid.New().String(), nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})
}

func TestHandlerSearch(t *testing.T) {
	c := sampleClassification()

	t.Run("returns search results", func(t *testing.T) {
		sys := &mockSystem{
			listFn: func(_ context.Context, _ pagination.PageRequest, _ classifications.Filters) (*pagination.PageResult[classifications.Classification], error) {
				result := pagination.NewPageResult([]classifications.Classification{c}, 1, 1, 20)
				return &result, nil
			},
		}
		mux := setupMux(newTestHandler(sys))

		body, _ := json.Marshal(classifications.SearchRequest{
			PageRequest: pagination.PageRequest{Page: 1, PageSize: 20},
		})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/classifications/search", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}

		var result pagination.PageResult[classifications.Classification]
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
		req := httptest.NewRequest("POST", "/classifications/search", bytes.NewReader([]byte("not json")))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("normalizes pagination", func(t *testing.T) {
		var capturedPage pagination.PageRequest
		sys := &mockSystem{
			listFn: func(_ context.Context, page pagination.PageRequest, _ classifications.Filters) (*pagination.PageResult[classifications.Classification], error) {
				capturedPage = page
				result := pagination.NewPageResult([]classifications.Classification{}, 0, page.Page, page.PageSize)
				return &result, nil
			},
		}
		mux := setupMux(newTestHandler(sys))

		body, _ := json.Marshal(classifications.SearchRequest{
			PageRequest: pagination.PageRequest{Page: 0, PageSize: 0},
		})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/classifications/search", bytes.NewReader(body))
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

func TestHandlerClassify(t *testing.T) {
	c := sampleClassification()

	t.Run("classifies document", func(t *testing.T) {
		var capturedDocID uuid.UUID
		sys := &mockSystem{
			classifyFn: func(_ context.Context, docID uuid.UUID) (*classifications.Classification, error) {
				capturedDocID = docID
				return &c, nil
			},
		}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/classifications/"+c.DocumentID.String(), nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201", rec.Code)
		}
		if capturedDocID != c.DocumentID {
			t.Errorf("documentId = %v, want %v", capturedDocID, c.DocumentID)
		}

		var got classifications.Classification
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.Classification != "SECRET" {
			t.Errorf("classification = %q, want SECRET", got.Classification)
		}
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		sys := &mockSystem{}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/classifications/not-a-uuid", nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("system error maps to status", func(t *testing.T) {
		sys := &mockSystem{
			classifyFn: func(_ context.Context, _ uuid.UUID) (*classifications.Classification, error) {
				return nil, classifications.ErrNotFound
			},
		}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/classifications/"+uuid.New().String(), nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})
}

func TestHandlerValidate(t *testing.T) {
	c := sampleClassification()
	validatedBy := "admin"
	c.ValidatedBy = &validatedBy
	now := time.Now()
	c.ValidatedAt = &now

	t.Run("validates classification", func(t *testing.T) {
		var capturedID uuid.UUID
		var capturedCmd classifications.ValidateCommand
		sys := &mockSystem{
			validateFn: func(_ context.Context, id uuid.UUID, cmd classifications.ValidateCommand) (*classifications.Classification, error) {
				capturedID = id
				capturedCmd = cmd
				return &c, nil
			},
		}
		mux := setupMux(newTestHandler(sys))

		body, _ := json.Marshal(classifications.ValidateCommand{ValidatedBy: "admin"})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/classifications/"+c.ID.String()+"/validate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if capturedID != c.ID {
			t.Errorf("id = %v, want %v", capturedID, c.ID)
		}
		if capturedCmd.ValidatedBy != "admin" {
			t.Errorf("validated_by = %q, want admin", capturedCmd.ValidatedBy)
		}
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		sys := &mockSystem{}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/classifications/not-a-uuid/validate", bytes.NewReader([]byte(`{}`)))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("invalid json returns 400", func(t *testing.T) {
		sys := &mockSystem{}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/classifications/"+c.ID.String()+"/validate", bytes.NewReader([]byte("not json")))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("invalid status returns 409", func(t *testing.T) {
		sys := &mockSystem{
			validateFn: func(_ context.Context, _ uuid.UUID, _ classifications.ValidateCommand) (*classifications.Classification, error) {
				return nil, classifications.ErrInvalidStatus
			},
		}
		mux := setupMux(newTestHandler(sys))

		body, _ := json.Marshal(classifications.ValidateCommand{ValidatedBy: "admin"})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/classifications/"+uuid.New().String()+"/validate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusConflict {
			t.Errorf("status = %d, want 409", rec.Code)
		}
	})
}

func TestHandlerUpdate(t *testing.T) {
	c := sampleClassification()
	c.Classification = "TOP SECRET"
	c.Rationale = "Updated rationale."
	updatedBy := "reviewer"
	c.ValidatedBy = &updatedBy

	t.Run("updates classification", func(t *testing.T) {
		var capturedID uuid.UUID
		var capturedCmd classifications.UpdateCommand
		sys := &mockSystem{
			updateFn: func(_ context.Context, id uuid.UUID, cmd classifications.UpdateCommand) (*classifications.Classification, error) {
				capturedID = id
				capturedCmd = cmd
				return &c, nil
			},
		}
		mux := setupMux(newTestHandler(sys))

		body, _ := json.Marshal(classifications.UpdateCommand{
			Classification: "TOP SECRET",
			Rationale:      "Updated rationale.",
			UpdatedBy:      "reviewer",
		})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/classifications/"+c.ID.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if capturedID != c.ID {
			t.Errorf("id = %v, want %v", capturedID, c.ID)
		}
		if capturedCmd.Classification != "TOP SECRET" {
			t.Errorf("classification = %q, want TOP SECRET", capturedCmd.Classification)
		}
		if capturedCmd.UpdatedBy != "reviewer" {
			t.Errorf("updated_by = %q, want reviewer", capturedCmd.UpdatedBy)
		}
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		sys := &mockSystem{}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/classifications/not-a-uuid", bytes.NewReader([]byte(`{}`)))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("invalid json returns 400", func(t *testing.T) {
		sys := &mockSystem{}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/classifications/"+c.ID.String(), bytes.NewReader([]byte("not json")))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("invalid status returns 409", func(t *testing.T) {
		sys := &mockSystem{
			updateFn: func(_ context.Context, _ uuid.UUID, _ classifications.UpdateCommand) (*classifications.Classification, error) {
				return nil, classifications.ErrInvalidStatus
			},
		}
		mux := setupMux(newTestHandler(sys))

		body, _ := json.Marshal(classifications.UpdateCommand{
			Classification: "SECRET",
			Rationale:      "test",
			UpdatedBy:      "admin",
		})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/classifications/"+uuid.New().String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusConflict {
			t.Errorf("status = %d, want 409", rec.Code)
		}
	})
}

func TestHandlerDelete(t *testing.T) {
	classificationID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	t.Run("deletes classification", func(t *testing.T) {
		var capturedID uuid.UUID
		sys := &mockSystem{
			deleteFn: func(_ context.Context, id uuid.UUID) error {
				capturedID = id
				return nil
			},
		}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("DELETE", "/classifications/"+classificationID.String(), nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", rec.Code)
		}
		if capturedID != classificationID {
			t.Errorf("id = %v, want %v", capturedID, classificationID)
		}
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		sys := &mockSystem{}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("DELETE", "/classifications/not-a-uuid", nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("not found returns 404", func(t *testing.T) {
		sys := &mockSystem{
			deleteFn: func(_ context.Context, _ uuid.UUID) error {
				return classifications.ErrNotFound
			},
		}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("DELETE", "/classifications/"+uuid.New().String(), nil)
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

	if group.Prefix != "/classifications" {
		t.Errorf("prefix = %q, want /classifications", group.Prefix)
	}

	want := []struct {
		method  string
		pattern string
	}{
		{"GET", ""},
		{"GET", "/{id}"},
		{"GET", "/document/{id}"},
		{"POST", "/search"},
		{"POST", "/{documentId}"},
		{"POST", "/{id}/validate"},
		{"PUT", "/{id}"},
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
