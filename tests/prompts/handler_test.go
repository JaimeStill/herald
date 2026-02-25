package prompts_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/JaimeStill/herald/internal/prompts"
	"github.com/JaimeStill/herald/pkg/pagination"
)

type mockSystem struct {
	listFn         func(ctx context.Context, page pagination.PageRequest, filters prompts.Filters) (*pagination.PageResult[prompts.Prompt], error)
	findFn         func(ctx context.Context, id uuid.UUID) (*prompts.Prompt, error)
	instructionsFn func(ctx context.Context, stage prompts.Stage) (string, error)
	specFn         func(ctx context.Context, stage prompts.Stage) (string, error)
	createFn       func(ctx context.Context, cmd prompts.CreateCommand) (*prompts.Prompt, error)
	updateFn       func(ctx context.Context, id uuid.UUID, cmd prompts.UpdateCommand) (*prompts.Prompt, error)
	deleteFn       func(ctx context.Context, id uuid.UUID) error
	activateFn     func(ctx context.Context, id uuid.UUID) (*prompts.Prompt, error)
	deactivateFn   func(ctx context.Context, id uuid.UUID) (*prompts.Prompt, error)
}

func (m *mockSystem) Handler() *prompts.Handler {
	return prompts.NewHandler(m, slog.New(slog.NewTextHandler(io.Discard, nil)), pagination.Config{DefaultPageSize: 20, MaxPageSize: 100})
}

func (m *mockSystem) List(ctx context.Context, page pagination.PageRequest, filters prompts.Filters) (*pagination.PageResult[prompts.Prompt], error) {
	return m.listFn(ctx, page, filters)
}

func (m *mockSystem) Find(ctx context.Context, id uuid.UUID) (*prompts.Prompt, error) {
	return m.findFn(ctx, id)
}

func (m *mockSystem) Instructions(ctx context.Context, stage prompts.Stage) (string, error) {
	return m.instructionsFn(ctx, stage)
}

func (m *mockSystem) Spec(ctx context.Context, stage prompts.Stage) (string, error) {
	return m.specFn(ctx, stage)
}

func (m *mockSystem) Create(ctx context.Context, cmd prompts.CreateCommand) (*prompts.Prompt, error) {
	return m.createFn(ctx, cmd)
}

func (m *mockSystem) Update(ctx context.Context, id uuid.UUID, cmd prompts.UpdateCommand) (*prompts.Prompt, error) {
	return m.updateFn(ctx, id, cmd)
}

func (m *mockSystem) Delete(ctx context.Context, id uuid.UUID) error {
	return m.deleteFn(ctx, id)
}

func (m *mockSystem) Activate(ctx context.Context, id uuid.UUID) (*prompts.Prompt, error) {
	return m.activateFn(ctx, id)
}

func (m *mockSystem) Deactivate(ctx context.Context, id uuid.UUID) (*prompts.Prompt, error) {
	return m.deactivateFn(ctx, id)
}

func newTestHandler(sys *mockSystem) *prompts.Handler {
	return prompts.NewHandler(
		sys,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		pagination.Config{DefaultPageSize: 20, MaxPageSize: 100},
	)
}

func setupMux(h *prompts.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	group := h.Routes()
	for _, route := range group.Routes {
		pattern := route.Method + " " + group.Prefix + route.Pattern
		mux.HandleFunc(pattern, route.Handler)
	}
	return mux
}

func samplePrompt() prompts.Prompt {
	return prompts.Prompt{
		ID:           uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		Name:         "detailed-classify",
		Stage:        prompts.StageClassify,
		Instructions: "Analyze each page thoroughly.",
		Description:  ptr("Detailed classification instructions"),
		Active:       false,
	}
}

func TestHandlerList(t *testing.T) {
	p := samplePrompt()
	sys := &mockSystem{
		listFn: func(_ context.Context, _ pagination.PageRequest, _ prompts.Filters) (*pagination.PageResult[prompts.Prompt], error) {
			result := pagination.NewPageResult([]prompts.Prompt{p}, 1, 1, 20)
			return &result, nil
		},
	}

	mux := setupMux(newTestHandler(sys))

	t.Run("returns paginated list", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/prompts", nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}

		var result pagination.PageResult[prompts.Prompt]
		if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
			t.Fatalf("decode: %v", err)
		}

		if result.Total != 1 {
			t.Errorf("total = %d, want 1", result.Total)
		}
		if len(result.Data) != 1 {
			t.Fatalf("data length = %d, want 1", len(result.Data))
		}
		if result.Data[0].ID != p.ID {
			t.Errorf("id = %v, want %v", result.Data[0].ID, p.ID)
		}
	})

	t.Run("passes query filters", func(t *testing.T) {
		var captured prompts.Filters
		sys.listFn = func(_ context.Context, _ pagination.PageRequest, f prompts.Filters) (*pagination.PageResult[prompts.Prompt], error) {
			captured = f
			result := pagination.NewPageResult([]prompts.Prompt{}, 0, 1, 20)
			return &result, nil
		}

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/prompts?stage=classify&name=detailed", nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if captured.Stage == nil || *captured.Stage != prompts.StageClassify {
			t.Errorf("stage filter = %v, want classify", captured.Stage)
		}
		if captured.Name == nil || *captured.Name != "detailed" {
			t.Errorf("name filter = %v, want detailed", captured.Name)
		}
	})
}

func TestHandlerStages(t *testing.T) {
	sys := &mockSystem{}
	mux := setupMux(newTestHandler(sys))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/prompts/stages", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var stages []prompts.Stage
	if err := json.NewDecoder(rec.Body).Decode(&stages); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(stages) != 2 {
		t.Fatalf("stages length = %d, want 2", len(stages))
	}

	want := []prompts.Stage{prompts.StageClassify, prompts.StageEnhance}
	for i, s := range stages {
		if s != want[i] {
			t.Errorf("stages[%d] = %q, want %q", i, s, want[i])
		}
	}
}

func TestHandlerFind(t *testing.T) {
	p := samplePrompt()

	t.Run("returns prompt by id", func(t *testing.T) {
		sys := &mockSystem{
			findFn: func(_ context.Context, id uuid.UUID) (*prompts.Prompt, error) {
				if id != p.ID {
					return nil, prompts.ErrNotFound
				}
				return &p, nil
			},
		}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/prompts/"+p.ID.String(), nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}

		var got prompts.Prompt
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.ID != p.ID {
			t.Errorf("id = %v, want %v", got.ID, p.ID)
		}
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		sys := &mockSystem{}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/prompts/not-a-uuid", nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("not found returns 404", func(t *testing.T) {
		sys := &mockSystem{
			findFn: func(_ context.Context, _ uuid.UUID) (*prompts.Prompt, error) {
				return nil, prompts.ErrNotFound
			},
		}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/prompts/"+uuid.New().String(), nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})
}

func TestHandlerInstructions(t *testing.T) {
	t.Run("returns stage content", func(t *testing.T) {
		sys := &mockSystem{
			instructionsFn: func(_ context.Context, stage prompts.Stage) (string, error) {
				return "test instructions for " + string(stage), nil
			},
		}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/prompts/classify/instructions", nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}

		var got prompts.StageContent
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.Stage != prompts.StageClassify {
			t.Errorf("stage = %q, want classify", got.Stage)
		}
		if got.Content != "test instructions for classify" {
			t.Errorf("content = %q, want %q", got.Content, "test instructions for classify")
		}
	})

	t.Run("invalid stage returns 400", func(t *testing.T) {
		sys := &mockSystem{}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/prompts/banana/instructions", nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("system error maps to status", func(t *testing.T) {
		sys := &mockSystem{
			instructionsFn: func(_ context.Context, _ prompts.Stage) (string, error) {
				return "", prompts.ErrInvalidStage
			},
		}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/prompts/classify/instructions", nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})
}

func TestHandlerSpec(t *testing.T) {
	t.Run("returns stage content", func(t *testing.T) {
		sys := &mockSystem{
			specFn: func(_ context.Context, stage prompts.Stage) (string, error) {
				return "test spec for " + string(stage), nil
			},
		}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/prompts/enhance/spec", nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}

		var got prompts.StageContent
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.Stage != prompts.StageEnhance {
			t.Errorf("stage = %q, want enhance", got.Stage)
		}
		if got.Content != "test spec for enhance" {
			t.Errorf("content = %q, want %q", got.Content, "test spec for enhance")
		}
	})

	t.Run("invalid stage returns 400", func(t *testing.T) {
		sys := &mockSystem{}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/prompts/init/spec", nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})
}

func TestHandlerSearch(t *testing.T) {
	p := samplePrompt()

	t.Run("returns search results", func(t *testing.T) {
		sys := &mockSystem{
			listFn: func(_ context.Context, _ pagination.PageRequest, _ prompts.Filters) (*pagination.PageResult[prompts.Prompt], error) {
				result := pagination.NewPageResult([]prompts.Prompt{p}, 1, 1, 20)
				return &result, nil
			},
		}
		mux := setupMux(newTestHandler(sys))

		body, _ := json.Marshal(prompts.SearchRequest{
			PageRequest: pagination.PageRequest{Page: 1, PageSize: 20},
		})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/prompts/search", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}

		var result pagination.PageResult[prompts.Prompt]
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
		req := httptest.NewRequest("POST", "/prompts/search", bytes.NewReader([]byte("not json")))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("normalizes pagination", func(t *testing.T) {
		var capturedPage pagination.PageRequest
		sys := &mockSystem{
			listFn: func(_ context.Context, page pagination.PageRequest, _ prompts.Filters) (*pagination.PageResult[prompts.Prompt], error) {
				capturedPage = page
				result := pagination.NewPageResult([]prompts.Prompt{}, 0, page.Page, page.PageSize)
				return &result, nil
			},
		}
		mux := setupMux(newTestHandler(sys))

		body, _ := json.Marshal(prompts.SearchRequest{
			PageRequest: pagination.PageRequest{Page: 0, PageSize: 0},
		})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/prompts/search", bytes.NewReader(body))
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

func TestHandlerCreate(t *testing.T) {
	p := samplePrompt()

	t.Run("creates prompt from json body", func(t *testing.T) {
		var capturedCmd prompts.CreateCommand
		sys := &mockSystem{
			createFn: func(_ context.Context, cmd prompts.CreateCommand) (*prompts.Prompt, error) {
				capturedCmd = cmd
				return &p, nil
			},
		}
		mux := setupMux(newTestHandler(sys))

		body, _ := json.Marshal(prompts.CreateCommand{
			Name:         "detailed-classify",
			Stage:        prompts.StageClassify,
			Instructions: "Analyze each page thoroughly.",
			Description:  ptr("Detailed classification instructions"),
		})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/prompts", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201", rec.Code)
		}
		if capturedCmd.Name != "detailed-classify" {
			t.Errorf("name = %q, want detailed-classify", capturedCmd.Name)
		}
		if capturedCmd.Stage != prompts.StageClassify {
			t.Errorf("stage = %q, want classify", capturedCmd.Stage)
		}
	})

	t.Run("invalid json returns 400", func(t *testing.T) {
		sys := &mockSystem{}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/prompts", bytes.NewReader([]byte("not json")))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("invalid stage returns 400", func(t *testing.T) {
		sys := &mockSystem{}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/prompts", bytes.NewReader([]byte(`{"name":"test","stage":"banana","instructions":"test"}`)))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("duplicate name returns 409", func(t *testing.T) {
		sys := &mockSystem{
			createFn: func(_ context.Context, _ prompts.CreateCommand) (*prompts.Prompt, error) {
				return nil, prompts.ErrDuplicate
			},
		}
		mux := setupMux(newTestHandler(sys))

		body, _ := json.Marshal(prompts.CreateCommand{
			Name:         "detailed-classify",
			Stage:        prompts.StageClassify,
			Instructions: "test",
		})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/prompts", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusConflict {
			t.Errorf("status = %d, want 409", rec.Code)
		}
	})
}

func TestHandlerUpdate(t *testing.T) {
	p := samplePrompt()
	p.Name = "updated-classify"

	t.Run("updates prompt", func(t *testing.T) {
		var capturedID uuid.UUID
		var capturedCmd prompts.UpdateCommand
		sys := &mockSystem{
			updateFn: func(_ context.Context, id uuid.UUID, cmd prompts.UpdateCommand) (*prompts.Prompt, error) {
				capturedID = id
				capturedCmd = cmd
				return &p, nil
			},
		}
		mux := setupMux(newTestHandler(sys))

		body, _ := json.Marshal(prompts.UpdateCommand{
			Name:         "updated-classify",
			Stage:        prompts.StageClassify,
			Instructions: "Updated instructions.",
		})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/prompts/"+p.ID.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if capturedID != p.ID {
			t.Errorf("id = %v, want %v", capturedID, p.ID)
		}
		if capturedCmd.Name != "updated-classify" {
			t.Errorf("name = %q, want updated-classify", capturedCmd.Name)
		}
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		sys := &mockSystem{}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/prompts/not-a-uuid", bytes.NewReader([]byte(`{}`)))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("invalid stage returns 400", func(t *testing.T) {
		sys := &mockSystem{}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/prompts/"+p.ID.String(), bytes.NewReader([]byte(`{"name":"test","stage":"invalid","instructions":"test"}`)))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("not found returns 404", func(t *testing.T) {
		sys := &mockSystem{
			updateFn: func(_ context.Context, _ uuid.UUID, _ prompts.UpdateCommand) (*prompts.Prompt, error) {
				return nil, prompts.ErrNotFound
			},
		}
		mux := setupMux(newTestHandler(sys))

		body, _ := json.Marshal(prompts.UpdateCommand{
			Name:         "test",
			Stage:        prompts.StageClassify,
			Instructions: "test",
		})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/prompts/"+uuid.New().String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})
}

func TestHandlerDelete(t *testing.T) {
	promptID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	t.Run("deletes prompt", func(t *testing.T) {
		var capturedID uuid.UUID
		sys := &mockSystem{
			deleteFn: func(_ context.Context, id uuid.UUID) error {
				capturedID = id
				return nil
			},
		}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("DELETE", "/prompts/"+promptID.String(), nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", rec.Code)
		}
		if capturedID != promptID {
			t.Errorf("id = %v, want %v", capturedID, promptID)
		}
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		sys := &mockSystem{}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("DELETE", "/prompts/not-a-uuid", nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("not found returns 404", func(t *testing.T) {
		sys := &mockSystem{
			deleteFn: func(_ context.Context, _ uuid.UUID) error {
				return prompts.ErrNotFound
			},
		}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("DELETE", "/prompts/"+uuid.New().String(), nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})
}

func TestHandlerActivate(t *testing.T) {
	p := samplePrompt()
	p.Active = true

	t.Run("activates prompt", func(t *testing.T) {
		var capturedID uuid.UUID
		sys := &mockSystem{
			activateFn: func(_ context.Context, id uuid.UUID) (*prompts.Prompt, error) {
				capturedID = id
				return &p, nil
			},
		}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/prompts/"+p.ID.String()+"/activate", nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if capturedID != p.ID {
			t.Errorf("id = %v, want %v", capturedID, p.ID)
		}

		var got prompts.Prompt
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if !got.Active {
			t.Error("active = false, want true")
		}
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		sys := &mockSystem{}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/prompts/not-a-uuid/activate", nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("not found returns 404", func(t *testing.T) {
		sys := &mockSystem{
			activateFn: func(_ context.Context, _ uuid.UUID) (*prompts.Prompt, error) {
				return nil, prompts.ErrNotFound
			},
		}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/prompts/"+uuid.New().String()+"/activate", nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})
}

func TestHandlerDeactivate(t *testing.T) {
	p := samplePrompt()

	t.Run("deactivates prompt", func(t *testing.T) {
		var capturedID uuid.UUID
		sys := &mockSystem{
			deactivateFn: func(_ context.Context, id uuid.UUID) (*prompts.Prompt, error) {
				capturedID = id
				return &p, nil
			},
		}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/prompts/"+p.ID.String()+"/deactivate", nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if capturedID != p.ID {
			t.Errorf("id = %v, want %v", capturedID, p.ID)
		}

		var got prompts.Prompt
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.Active {
			t.Error("active = true, want false")
		}
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		sys := &mockSystem{}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/prompts/not-a-uuid/deactivate", nil)
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("not found returns 404", func(t *testing.T) {
		sys := &mockSystem{
			deactivateFn: func(_ context.Context, _ uuid.UUID) (*prompts.Prompt, error) {
				return nil, prompts.ErrNotFound
			},
		}
		mux := setupMux(newTestHandler(sys))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/prompts/"+uuid.New().String()+"/deactivate", nil)
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

	if group.Prefix != "/prompts" {
		t.Errorf("prefix = %q, want /prompts", group.Prefix)
	}

	want := []struct {
		method  string
		pattern string
	}{
		{"GET", ""},
		{"GET", "/stages"},
		{"GET", "/{id}"},
		{"GET", "/{stage}/instructions"},
		{"GET", "/{stage}/spec"},
		{"POST", ""},
		{"PUT", "/{id}"},
		{"DELETE", "/{id}"},
		{"POST", "/search"},
		{"POST", "/{id}/activate"},
		{"POST", "/{id}/deactivate"},
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
