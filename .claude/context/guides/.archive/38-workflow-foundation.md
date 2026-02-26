# 38 — Workflow Foundation: Types, Runtime, Errors, and Parsing

## Problem Context

Herald's `workflow/` package contains only a placeholder `doc.go`. All subsequent workflow sub-issues (#39 init node, #40 classify node, #41 enhance/graph assembly) depend on foundational types, a runtime dependency struct, sentinel errors, generic JSON parsing, and prompt composition. This issue establishes those shared building blocks.

## Architecture Approach

- **Two core types** — `ClassificationState` holds the overall document assessment, `ClassificationPage` holds per-page data (image path, markings, enhancement flags). Methods on `ClassificationState` derive enhancement decisions from page data
- **Runtime bundles workflow dependencies** as a simple struct with exported fields, following the `internal/infrastructure/infrastructure.go` pattern but scoped to what workflow nodes need
- **Generic JSON parser** lives in `pkg/formatting/` (alongside existing `bytes.go`) since it's a reusable utility, not workflow-specific
- **Prompt composition** calls the prompts domain's `Instructions()` and `Spec()` methods, combining both layers with serialized running state for context accumulation

## Implementation

### Step 1: Add Dependencies

```bash
go get github.com/JaimeStill/go-agents-orchestration
go get github.com/JaimeStill/document-context
```

Then delete the placeholder:

```bash
rm workflow/doc.go
```

### Step 2: `workflow/errors.go`

New file — four sentinel errors scoped to workflow operations:

```go
package workflow

import "errors"

var (
	ErrDocumentNotFound = errors.New("document not found")
	ErrRenderFailed     = errors.New("failed to render page images")
	ErrClassifyFailed   = errors.New("classification failed")
	ErrEnhanceFailed    = errors.New("enhancement failed")
)
```

### Step 3: `workflow/types.go`

New file — two core types plus `Confidence` constants and `WorkflowResult`. The schema is intentionally lean to minimize token consumption per LLM response across potentially thousands of pages. Prompt specifications (`internal/prompts/specs.go`) will be updated separately to align with this schema.

```go
package workflow

import (
	"slices"
	"time"

	"github.com/google/uuid"
)

type Confidence string

const (
	ConfidenceHigh   Confidence = "HIGH"
	ConfidenceMedium Confidence = "MEDIUM"
	ConfidenceLow    Confidence = "LOW"
)

type ClassificationPage struct {
	PageNumber    int      `json:"page_number"`
	ImagePath     string   `json:"image_path"`
	MarkingsFound []string `json:"markings_found"`
	Rationale     string   `json:"rationale"`
	Enhance       bool     `json:"enhance"`
	Enhancements  string   `json:"enhancements"`
}

type ClassificationState struct {
	Classification string               `json:"classification"`
	Confidence     Confidence           `json:"confidence"`
	Rationale      string               `json:"rationale"`
	Pages          []ClassificationPage `json:"pages"`
}

func (s *ClassificationState) NeedsEnhance() bool {
	return slices.ContainsFunc(s.Pages, func(p ClassificationPage) bool {
		return p.Enhance
	})
}

func (s *ClassificationState) EnhancePages() []int {
	var indices []int
	for i, p := range s.Pages {
		if p.Enhance {
			indices = append(indices, i)
		}
	}
	return indices
}

type WorkflowResult struct {
	DocumentID  uuid.UUID           `json:"document_id"`
	Filename    string              `json:"filename"`
	PageCount   int                 `json:"page_count"`
	State       ClassificationState `json:"state"`
	CompletedAt time.Time           `json:"completed_at"`
}
```

### Step 4: `workflow/runtime.go`

New file — dependency bundle for workflow nodes:

```go
package workflow

import (
	"log/slog"

	"github.com/JaimeStill/herald/internal/documents"
	"github.com/JaimeStill/herald/internal/prompts"
	"github.com/JaimeStill/herald/pkg/storage"

	gaconfig "github.com/JaimeStill/go-agents/pkg/config"
)

type Runtime struct {
	Agent     gaconfig.AgentConfig
	Storage   storage.System
	Documents documents.System
	Prompts   prompts.System
	Logger    *slog.Logger
}
```

### Step 5: `pkg/formatting/parse.go`

New file in existing `pkg/formatting/` package — generic JSON parser with markdown code fence fallback:

```go
package formatting

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var ErrParseFailed = errors.New("failed to parse response")

var jsonBlockRegex = regexp.MustCompile(`(?s)` + "```" + `(?:json)?\s*\n?(.*?)\n?` + "```")

func Parse[T any](content string) (T, error) {
	var result T
	content = strings.TrimSpace(content)

	if err := json.Unmarshal([]byte(content), &result); err == nil {
		return result, nil
	}

	matches := jsonBlockRegex.FindStringSubmatch(content)
	if len(matches) >= 2 {
		cleaned := strings.TrimSpace(matches[1])
		if err := json.Unmarshal([]byte(cleaned), &result); err == nil {
			return result, nil
		}
	}

	return result, fmt.Errorf("%w: %s", ErrParseFailed, content)
}
```

### Step 6: `workflow/prompts.go`

New file — prompt composition combining instructions, spec, and running classification state:

```go
package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/JaimeStill/herald/internal/prompts"
)

func ComposePrompt(
	ctx context.Context,
	ps prompts.System,
	stage prompts.Stage,
	state *ClassificationState,
) (string, error) {
	instructions, err := ps.Instructions(ctx, stage)
	if err != nil {
		return "", fmt.Errorf("load instructions for %s: %w", stage, err)
	}

	spec, err := ps.Spec(ctx, stage)
	if err != nil {
		return "", fmt.Errorf("load spec for %s: %w", stage, err)
	}

	var sb strings.Builder
	sb.WriteString(instructions)
	sb.WriteString("\n\n")
	sb.WriteString(spec)

	if state != nil {
		stateJSON, err := json.MarshalIndent(state, "", "  ")
		if err != nil {
			return "", fmt.Errorf("serialize classification state: %w", err)
		}

		sb.WriteString("\n\nCurrent classification state:\n\n")
		sb.WriteString(string(stateJSON))
	}

	return sb.String(), nil
}
```

## Validation Criteria

- [ ] go-agents-orchestration and document-context added to go.mod
- [ ] `rm workflow/doc.go` — placeholder removed
- [ ] `ClassificationPage` and `ClassificationState` defined with JSON tags
- [ ] `ClassificationPage.ImagePath` uses file path (not data URI)
- [ ] `NeedsEnhance()` and `EnhancePages()` methods on `ClassificationState`
- [ ] Runtime struct defined with all dependency fields
- [ ] `pkg/formatting/parse.go` handles both direct JSON and markdown-fenced JSON
- [ ] `ComposePrompt` calls `Instructions()` + `Spec()` from prompts system
- [ ] `go vet ./...` passes
- [ ] `mise run build` succeeds
- [ ] `go mod tidy` produces no changes
