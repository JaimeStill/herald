package pagination_test

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"github.com/JaimeStill/herald/pkg/pagination"
	"github.com/JaimeStill/herald/pkg/query"
)

func defaultConfig() pagination.Config {
	return pagination.Config{DefaultPageSize: 20, MaxPageSize: 100}
}

func TestConfigFinalizeDefaults(t *testing.T) {
	cfg := pagination.Config{}
	if err := cfg.Finalize(nil); err != nil {
		t.Fatalf("finalize failed: %v", err)
	}

	if cfg.DefaultPageSize != 20 {
		t.Errorf("DefaultPageSize = %d, want 20", cfg.DefaultPageSize)
	}
	if cfg.MaxPageSize != 100 {
		t.Errorf("MaxPageSize = %d, want 100", cfg.MaxPageSize)
	}
}

func TestConfigFinalizeEnvOverrides(t *testing.T) {
	t.Setenv("TEST_PAGE_SIZE", "50")
	t.Setenv("TEST_MAX_PAGE", "200")

	env := &pagination.ConfigEnv{
		DefaultPageSize: "TEST_PAGE_SIZE",
		MaxPageSize:     "TEST_MAX_PAGE",
	}

	cfg := pagination.Config{}
	if err := cfg.Finalize(env); err != nil {
		t.Fatalf("finalize failed: %v", err)
	}

	if cfg.DefaultPageSize != 50 {
		t.Errorf("DefaultPageSize = %d, want 50", cfg.DefaultPageSize)
	}
	if cfg.MaxPageSize != 200 {
		t.Errorf("MaxPageSize = %d, want 200", cfg.MaxPageSize)
	}
}

func TestConfigFinalizeValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     pagination.Config
		wantErr string
	}{
		{
			name:    "default exceeds max",
			cfg:     pagination.Config{DefaultPageSize: 200, MaxPageSize: 100},
			wantErr: "default_page_size cannot exceed max_page_size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Finalize(nil)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestConfigMerge(t *testing.T) {
	base := pagination.Config{DefaultPageSize: 20, MaxPageSize: 100}
	overlay := pagination.Config{DefaultPageSize: 50}
	base.Merge(&overlay)

	if base.DefaultPageSize != 50 {
		t.Errorf("DefaultPageSize = %d, want 50", base.DefaultPageSize)
	}
	if base.MaxPageSize != 100 {
		t.Errorf("MaxPageSize = %d, want 100 (unchanged)", base.MaxPageSize)
	}
}

func TestPageRequestNormalize(t *testing.T) {
	cfg := defaultConfig()

	tests := []struct {
		name         string
		req          pagination.PageRequest
		wantPage     int
		wantPageSize int
	}{
		{
			name:         "zero values get defaults",
			req:          pagination.PageRequest{},
			wantPage:     1,
			wantPageSize: 20,
		},
		{
			name:         "negative page corrected",
			req:          pagination.PageRequest{Page: -1, PageSize: 10},
			wantPage:     1,
			wantPageSize: 10,
		},
		{
			name:         "page size clamped to max",
			req:          pagination.PageRequest{Page: 1, PageSize: 500},
			wantPage:     1,
			wantPageSize: 100,
		},
		{
			name:         "valid values preserved",
			req:          pagination.PageRequest{Page: 3, PageSize: 25},
			wantPage:     3,
			wantPageSize: 25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.req.Normalize(cfg)
			if tt.req.Page != tt.wantPage {
				t.Errorf("Page = %d, want %d", tt.req.Page, tt.wantPage)
			}
			if tt.req.PageSize != tt.wantPageSize {
				t.Errorf("PageSize = %d, want %d", tt.req.PageSize, tt.wantPageSize)
			}
		})
	}
}

func TestPageRequestOffset(t *testing.T) {
	tests := []struct {
		name       string
		page       int
		pageSize   int
		wantOffset int
	}{
		{"page 1", 1, 20, 0},
		{"page 2", 2, 20, 20},
		{"page 3 size 10", 3, 10, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := pagination.PageRequest{Page: tt.page, PageSize: tt.pageSize}
			if got := req.Offset(); got != tt.wantOffset {
				t.Errorf("Offset() = %d, want %d", got, tt.wantOffset)
			}
		})
	}
}

func TestPageRequestFromQuery(t *testing.T) {
	cfg := defaultConfig()

	t.Run("all params present", func(t *testing.T) {
		values := url.Values{
			"page":      {"2"},
			"page_size": {"15"},
			"search":    {"test"},
			"sort":      {"name,-createdAt"},
		}

		req := pagination.PageRequestFromQuery(values, cfg)

		if req.Page != 2 {
			t.Errorf("Page = %d, want 2", req.Page)
		}
		if req.PageSize != 15 {
			t.Errorf("PageSize = %d, want 15", req.PageSize)
		}
		if req.Search == nil || *req.Search != "test" {
			t.Errorf("Search = %v, want 'test'", req.Search)
		}
		if len(req.Sort) != 2 {
			t.Fatalf("Sort length = %d, want 2", len(req.Sort))
		}
		if req.Sort[0].Field != "name" || req.Sort[0].Descending {
			t.Errorf("Sort[0] = %v, want {name false}", req.Sort[0])
		}
		if req.Sort[1].Field != "createdAt" || !req.Sort[1].Descending {
			t.Errorf("Sort[1] = %v, want {createdAt true}", req.Sort[1])
		}
	})

	t.Run("empty params get defaults", func(t *testing.T) {
		values := url.Values{}
		req := pagination.PageRequestFromQuery(values, cfg)

		if req.Page != 1 {
			t.Errorf("Page = %d, want 1", req.Page)
		}
		if req.PageSize != 20 {
			t.Errorf("PageSize = %d, want 20", req.PageSize)
		}
		if req.Search != nil {
			t.Errorf("Search = %v, want nil", req.Search)
		}
	})
}

func TestNewPageResult(t *testing.T) {
	tests := []struct {
		name           string
		total          int
		page           int
		pageSize       int
		wantTotalPages int
	}{
		{"exact division", 100, 1, 20, 5},
		{"remainder", 101, 1, 20, 6},
		{"single page", 5, 1, 20, 1},
		{"empty result", 0, 1, 20, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pagination.NewPageResult([]string{"a"}, tt.total, tt.page, tt.pageSize)
			if result.TotalPages != tt.wantTotalPages {
				t.Errorf("TotalPages = %d, want %d", result.TotalPages, tt.wantTotalPages)
			}
			if result.Total != tt.total {
				t.Errorf("Total = %d, want %d", result.Total, tt.total)
			}
			if result.Page != tt.page {
				t.Errorf("Page = %d, want %d", result.Page, tt.page)
			}
			if result.PageSize != tt.pageSize {
				t.Errorf("PageSize = %d, want %d", result.PageSize, tt.pageSize)
			}
		})
	}
}

func TestNewPageResultNilDataBecomesEmpty(t *testing.T) {
	result := pagination.NewPageResult[string](nil, 0, 1, 20)
	if result.Data == nil {
		t.Error("Data should be empty slice, not nil")
	}
	if len(result.Data) != 0 {
		t.Errorf("Data length = %d, want 0", len(result.Data))
	}
}

func TestSortFieldsUnmarshalString(t *testing.T) {
	input := `"name,-createdAt"`
	var sf pagination.SortFields
	if err := json.Unmarshal([]byte(input), &sf); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if len(sf) != 2 {
		t.Fatalf("length = %d, want 2", len(sf))
	}
	if sf[0] != (query.SortField{Field: "name", Descending: false}) {
		t.Errorf("sf[0] = %v, want {name false}", sf[0])
	}
	if sf[1] != (query.SortField{Field: "createdAt", Descending: true}) {
		t.Errorf("sf[1] = %v, want {createdAt true}", sf[1])
	}
}

func TestSortFieldsUnmarshalArray(t *testing.T) {
	input := `[{"Field":"name","Descending":false},{"Field":"createdAt","Descending":true}]`
	var sf pagination.SortFields
	if err := json.Unmarshal([]byte(input), &sf); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if len(sf) != 2 {
		t.Fatalf("length = %d, want 2", len(sf))
	}
	if sf[0] != (query.SortField{Field: "name", Descending: false}) {
		t.Errorf("sf[0] = %v, want {name false}", sf[0])
	}
	if sf[1] != (query.SortField{Field: "createdAt", Descending: true}) {
		t.Errorf("sf[1] = %v, want {createdAt true}", sf[1])
	}
}
