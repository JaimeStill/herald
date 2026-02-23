# 17 - Document HTTP Handlers, OpenAPI Specs, and API Wiring

## Problem Context

Issue #16 delivered the document domain core (types, System interface, repository, mapping, errors). Herald needs an HTTP presentation layer to expose document operations — upload, list, search, find, and delete — through a REST API. This task adds the Handler struct with 5 endpoints, multipart upload processing with pdfcpu page count extraction, a reusable byte-size formatting package, and wires everything into the API module.

## Architecture Approach

Follows agent-lab's handler pattern: the `System` interface exposes a `Handler()` factory method, and the handler defines a `Routes()` method returning a `routes.Group`. The group carries both HTTP route registrations and OpenAPI schemas, so a single `routes.Register()` call wires the mux and spec simultaneously.

Upload size configuration lives on `APIConfig` (not storage config) since it's an HTTP concern. A new `pkg/formatting` package provides reusable bidirectional byte-size conversion (B through YB, base-1024).

The OpenAPI spec file (`internal/documents/openapi.go`) has already been created.

## Implementation

### Step 1: Create `pkg/formatting/bytes.go`

New file — reusable bidirectional byte size formatting.

```go
package formatting

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

var units = []string{"B", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"}

var bytesPattern = regexp.MustCompile(`^(\d+\.?\d*)\s*([A-Za-z]*)$`)

func ParseBytes(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty byte size string")
	}

	matches := bytesPattern.FindStringSubmatch(s)
	if matches == nil {
		return 0, fmt.Errorf("invalid byte size: %q", s)
	}

	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid byte size number: %w", err)
	}

	unit := strings.ToUpper(matches[2])

	// bare number with no unit means bytes
	if unit == "" {
		return int64(value), nil
	}

	idx := slices.Index(units, unit)
	if idx == -1 {
		return 0, fmt.Errorf("unknown byte size unit: %q", unit)
	}

	return int64(value * math.Pow(1024, float64(idx))), nil
}

// FormatBytes converts a byte count into a human-readable string using
// base-1024 units (B through YB). Precision controls the number of
// decimal places in the output.
func FormatBytes(n int64, precision int) string {
	if n == 0 {
		return "0 B"
	}

	if precision < 0 {
		precision = 0
	}

	f := float64(n)
	k := 1024.0
	i := int(math.Floor(math.Log(f) / math.Log(k)))

	if i >= len(units) {
		i = len(units) - 1
	}

	size := f / math.Pow(k, float64(i))
	formatted := strconv.FormatFloat(size, 'f', precision, 64)

	return formatted + " " + units[i]
}
```

### Step 2: Add `MaxUploadSize` to `internal/config/api.go`

Add the field, env var loading, default, merge, and accessor. The following shows incremental changes to the existing file.

**Add import** for the formatting package alongside existing imports:

```go
import (
	"fmt"
	"os"

	"github.com/JaimeStill/herald/pkg/formatting"
	"github.com/JaimeStill/herald/pkg/middleware"
	"github.com/JaimeStill/herald/pkg/openapi"
	"github.com/JaimeStill/herald/pkg/pagination"
)
```

**Add field to struct:**

```go
type APIConfig struct {
	BasePath      string                `toml:"base_path"`
	MaxUploadSize string                `toml:"max_upload_size"`
	CORS          middleware.CORSConfig `toml:"cors"`
	OpenAPI       openapi.Config        `toml:"openapi"`
	Pagination    pagination.Config     `toml:"pagination"`
}
```

**Add accessor method** (after the struct):

```go
func (c *APIConfig) MaxUploadSizeBytes() int64 {
	size, err := formatting.ParseBytes(c.MaxUploadSize)
	if err != nil {
		return 50 * 1024 * 1024 // 50MB fallback
	}
	return size
}
```

**Update `loadDefaults()`** — add the MaxUploadSize default:

```go
func (c *APIConfig) loadDefaults() {
	if c.BasePath == "" {
		c.BasePath = "/api"
	}
	if c.MaxUploadSize == "" {
		c.MaxUploadSize = "50MB"
	}
}
```

**Update `loadEnv()`** — add the env var override:

```go
func (c *APIConfig) loadEnv() {
	if v := os.Getenv("HERALD_API_BASE_PATH"); v != "" {
		c.BasePath = v
	}
	if v := os.Getenv("HERALD_API_MAX_UPLOAD_SIZE"); v != "" {
		c.MaxUploadSize = v
	}
}
```

**Update `Merge()`** — add the MaxUploadSize merge:

```go
func (c *APIConfig) Merge(overlay *APIConfig) {
	if overlay.BasePath != "" {
		c.BasePath = overlay.BasePath
	}
	if overlay.MaxUploadSize != "" {
		c.MaxUploadSize = overlay.MaxUploadSize
	}
	c.CORS.Merge(&overlay.CORS)
	c.OpenAPI.Merge(&overlay.OpenAPI)
	c.Pagination.Merge(&overlay.Pagination)
}
```

### Step 3: Add `pdfcpu` dependency

```bash
go get github.com/pdfcpu/pdfcpu
```

### Step 4: Create `internal/documents/handler.go`

New file — Handler struct with all 6 endpoint methods and helpers.

```go
package documents

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/pdfcpu/pdfcpu/pkg/api"

	"github.com/JaimeStill/herald/pkg/handlers"
	"github.com/JaimeStill/herald/pkg/pagination"
	"github.com/JaimeStill/herald/pkg/routes"
)

type Handler struct {
	sys           System
	logger        *slog.Logger
	pagination    pagination.Config
	maxUploadSize int64
}

func NewHandler(
	sys System,
	logger *slog.Logger,
	pagination pagination.Config,
	maxUploadSize int64,
) *Handler {
	return &Handler{
		sys:           sys,
		logger:        logger.With("handler", "documents"),
		pagination:    pagination,
		maxUploadSize: maxUploadSize,
	}
}

func (h *Handler) Routes() routes.Group {
	return routes.Group{
		Prefix:      "/documents",
		Tags:        []string{"Documents"},
		Description: "Document upload and management",
		Schemas:     Spec.Schemas(),
		Routes: []routes.Route{
			{Method: "GET", Pattern: "", Handler: h.List, OpenAPI: Spec.List},
			{Method: "GET", Pattern: "/{id}", Handler: h.Find, OpenAPI: Spec.Find},
			{Method: "POST", Pattern: "", Handler: h.Upload, OpenAPI: Spec.Upload},
			{Method: "POST", Pattern: "/search", Handler: h.Search, OpenAPI: Spec.Search},
			{Method: "DELETE", Pattern: "/{id}", Handler: h.Delete, OpenAPI: Spec.Delete},
		},
	}
}

// SearchRequest combines pagination with document filters for the search endpoint.
type SearchRequest struct {
	pagination.PageRequest
	Filters
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	page := pagination.PageRequestFromQuery(r.URL.Query(), h.pagination)
	filters := FiltersFromQuery(r.URL.Query())

	result, err := h.sys.List(r.Context(), page, filters)
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusInternalServerError, err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, result)
}

func (h *Handler) Find(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, ErrInvalidFile)
		return
	}

	doc, err := h.sys.Find(r.Context(), id)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, doc)
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, ErrInvalidFile)
		return
	}

	req.PageRequest.Normalize(h.pagination)

	result, err := h.sys.List(r.Context(), req.PageRequest, req.Filters)
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusInternalServerError, err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, result)
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(h.maxUploadSize); err != nil {
		handlers.RespondError(w, h.logger, http.StatusRequestEntityTooLarge, ErrFileTooLarge)
		return
	}

	// validate metadata before reading file data
	externalID, err := strconv.Atoi(r.FormValue("external_id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, ErrInvalidFile)
		return
	}

	externalPlatform := r.FormValue("external_platform")
	if externalPlatform == "" {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, ErrInvalidFile)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, ErrInvalidFile)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, ErrInvalidFile)
		return
	}

	contentType := detectContentType(header.Header.Get("Content-Type"), data)
	pageCount := extractPDFPageCount(h.logger, data, contentType)

	cmd := CreateCommand{
		Data:             data,
		Filename:         header.Filename,
		ContentType:      contentType,
		ExternalID:       externalID,
		ExternalPlatform: externalPlatform,
		PageCount:        pageCount,
	}

	doc, err := h.sys.Create(r.Context(), cmd)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusCreated, doc)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, ErrInvalidFile)
		return
	}

	if err := h.sys.Delete(r.Context(), id); err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// detectContentType returns the best content type for uploaded file data.
// Prefers the multipart header value unless it is empty or the generic
// application/octet-stream, in which case it falls back to http.DetectContentType.
func detectContentType(header string, data []byte) string {
	header = strings.TrimSpace(header)
	if header != "" && header != "application/octet-stream" {
		return header
	}
	return http.DetectContentType(data)
}

// extractPDFPageCount returns the page count if data is a PDF, or nil otherwise.
// Failures are non-fatal: a warning is logged and nil is returned.
func extractPDFPageCount(logger *slog.Logger, data []byte, contentType string) *int {
	if contentType != "application/pdf" {
		return nil
	}

	count, err := api.PageCount(bytes.NewReader(data), nil)
	if err != nil {
		logger.Warn("failed to extract PDF page count", "error", err)
		return nil
	}

	return &count
}
```

### Step 5: Add `Handler()` to System interface and wire API routes

**File:** `internal/documents/system.go` — add the method to the interface:

```go
type System interface {
	Handler(maxUploadSize int64) *Handler

	List(
		ctx context.Context,
		page pagination.PageRequest,
		filters Filters,
	) (*pagination.PageResult[Document], error)

	Find(ctx context.Context, id uuid.UUID) (*Document, error)
	Create(ctx context.Context, cmd CreateCommand) (*Document, error)
	CreateBatch(ctx context.Context, cmds []CreateCommand) []BatchResult
	Delete(ctx context.Context, id uuid.UUID) error
}
```

**File:** `internal/documents/repository.go` — add method on `repo`:

```go
func (r *repo) Handler(maxUploadSize int64) *Handler {
	return NewHandler(r, r.logger, r.pagination, maxUploadSize)
}
```

Add this method anywhere within the existing methods (before `List` is a natural spot).

**File:** `internal/api/routes.go` — implement `registerRoutes`:

```go
package api

import (
	"net/http"

	"github.com/JaimeStill/herald/internal/config"
	"github.com/JaimeStill/herald/pkg/openapi"
	"github.com/JaimeStill/herald/pkg/routes"
)

func registerRoutes(
	mux *http.ServeMux,
	spec *openapi.Spec,
	domain *Domain,
	cfg *config.Config,
) {
	routes.Register(
		mux,
		cfg.API.BasePath,
		spec,
		domain.Documents.Handler(cfg.API.MaxUploadSizeBytes()).Routes(),
	)
}
```

### Step 6: Run dependency and build commands

```bash
go get github.com/pdfcpu/pdfcpu
go mod tidy
go vet ./...
go build ./cmd/server/
```

## Validation Criteria

- [ ] `go vet ./...` passes
- [ ] `go build ./cmd/server/` compiles cleanly
- [ ] MaxUploadSize configurable via `HERALD_API_MAX_UPLOAD_SIZE` env var
- [ ] All 5 endpoints defined with correct HTTP methods and patterns
- [ ] Upload endpoint requires `external_id` and `external_platform` form fields
- [ ] PDF page count extraction uses pdfcpu (non-fatal on failure)
- [ ] Batch upload handles multiple files with partial success semantics
- [ ] Search endpoint accepts JSON body with pagination + filters
- [ ] `formatting.ParseBytes` / `formatting.FormatBytes` round-trip correctly
- [ ] OpenAPI spec at `/api/openapi.json` includes all document endpoints and schemas
