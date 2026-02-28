# 53 - Query Builder JOIN Support and Document Classification View

## Problem Context

The Phase 3 web client needs to display documents alongside their classification metadata and filter by classification fields. Currently, `Document` and `Classification` are separate types in separate packages with no joined view. A SQL LEFT JOIN populates nullable classification fields on `Document` without introducing circular dependencies.

## Architecture Approach

Adapt the S2VA `ProjectionMap.Join()` context-switching pattern: `Join()` switches `currentAlias` so subsequent `Project()` calls map to the joined table. A new `From()` method returns the base table reference plus all accumulated JOIN clauses — backward compatible (returns `Table()` when no joins exist). The `Builder` switches from `Table()` to `From()` in all build methods.

## Implementation

### Step 1: Extend `ProjectionMap` with JOIN support

**File:** `pkg/query/projection.go`

Add the `joinClause` struct and new fields to `ProjectionMap`:

```go
type JoinClause struct {
	Index    int
	JoinType string
	Schema   string
	Table    string
	Alias    string
	On       string
}

type ProjectionMap struct {
	schema       string
	table        string
	alias        string
	currentAlias string
	columns      map[string]string
	columnList   []string
	joins map[string]JoinClause
}
```

The `joins` map is keyed by alias for easy lookup.

Update `NewProjectionMap` to initialize `currentAlias` and the joins map:

```go
func NewProjectionMap(schema, table, alias string) *ProjectionMap {
	return &ProjectionMap{
		schema:       schema,
		table:        table,
		alias:        alias,
		currentAlias: alias,
		columns:      make(map[string]string),
		columnList:   make([]string, 0),
		joins:        make(map[string]JoinClause),
	}
}
```

Update `Project` to use `currentAlias`:

```go
func (p *ProjectionMap) Project(column, viewName string) *ProjectionMap {
	qualified := fmt.Sprintf("%s.%s", p.currentAlias, column)
	p.columns[viewName] = qualified
	p.columnList = append(p.columnList, qualified)
	return p
}
```

Add `Join` and `From` methods:

```go
// Join adds a JOIN clause and switches the projection context to the joined table.
// Subsequent Project calls will use the joined table's alias.
func (p *ProjectionMap) Join(schema, table, alias, joinType, on string) *ProjectionMap {
	p.joins[alias] = JoinClause{
		Index:    len(p.joins),
		JoinType: joinType,
		Schema:   schema,
		Table:    table,
		Alias:    alias,
		On:       on,
	}
	p.currentAlias = alias
	return p
}

// Joins returns all join clauses sorted by insertion order.
func (p *ProjectionMap) Joins() []JoinClause {
	return slices.SortedFunc(maps.Values(p.joins), func(a, b JoinClause) int {
		return a.Index - b.Index
	})
}

// From returns the full FROM clause including the base table and all JOIN clauses.
// When no joins exist, this returns the same value as Table().
func (p *ProjectionMap) From() string {
	joins := p.Joins()
	if len(joins) == 0 {
		return p.Table()
	}

	var b strings.Builder
	b.WriteString(p.Table())
	for _, j := range joins {
		fmt.Fprintf(&b, " %s %s.%s %s ON %s", j.JoinType, j.Schema, j.Table, j.Alias, j.On)
	}
	return b.String()
}
```

### Step 2: Update `Builder` to use `From()`

**File:** `pkg/query/builder.go`

Replace `b.projection.Table()` with `b.projection.From()` in all five build methods:

In `Build()`:
```go
// change: b.projection.Table()  →  b.projection.From()
sql := fmt.Sprintf(
    "SELECT %s FROM %s%s%s",
    b.projection.Columns(),
    b.projection.From(),
    where,
    orderBy,
)
```

In `BuildCount()`:
```go
sql := fmt.Sprintf("SELECT COUNT(*) FROM %s%s", b.projection.From(), where)
```

In `BuildPage()`:
```go
sql := fmt.Sprintf(
    "SELECT %s FROM %s%s%s LIMIT %d OFFSET %d",
    b.projection.Columns(),
    b.projection.From(),
    where,
    orderBy,
    pageSize,
    offset,
)
```

In `BuildSingle()`:
```go
sql := fmt.Sprintf(
    "SELECT %s FROM %s WHERE %s = $1",
    b.projection.Columns(),
    b.projection.From(),
    col,
)
```

In `BuildSingleOrNull()`:
```go
sql := fmt.Sprintf(
    "SELECT %s FROM %s%s LIMIT 1",
    b.projection.Columns(),
    b.projection.From(),
    where,
)
```

### Step 3: Extend `Document` struct

**File:** `internal/documents/document.go`

Add three nullable fields after `UpdatedAt`:

```go
type Document struct {
	ID               uuid.UUID  `json:"id"`
	ExternalID       int        `json:"external_id"`
	ExternalPlatform string     `json:"external_platform"`
	Filename         string     `json:"filename"`
	ContentType      string     `json:"content_type"`
	SizeBytes        int64      `json:"size_bytes"`
	PageCount        *int       `json:"page_count"`
	StorageKey       string     `json:"storage_key"`
	Status           string     `json:"status"`
	UploadedAt       time.Time  `json:"uploaded_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	Classification   *string    `json:"classification,omitempty"`
	Confidence       *string    `json:"confidence,omitempty"`
	ClassifiedAt     *time.Time `json:"classified_at,omitempty"`
}
```

### Step 4: Extend projection, scan, and filters

**File:** `internal/documents/mapping.go`

Extend the `projection` variable with the LEFT JOIN and three additional columns:

```go
var projection = query.
	NewProjectionMap("public", "documents", "d").
	Project("id", "ID").
	Project("external_id", "ExternalID").
	Project("external_platform", "ExternalPlatform").
	Project("filename", "Filename").
	Project("content_type", "ContentType").
	Project("size_bytes", "SizeBytes").
	Project("page_count", "PageCount").
	Project("storage_key", "StorageKey").
	Project("status", "Status").
	Project("uploaded_at", "UploadedAt").
	Project("updated_at", "UpdatedAt").
	Join("public", "classifications", "c", "LEFT JOIN", "d.id = c.document_id").
	Project("classification", "Classification").
	Project("confidence", "Confidence").
	Project("classified_at", "ClassifiedAt")
```

Extend `scanDocument` to scan 14 columns:

```go
func scanDocument(s repository.Scanner) (Document, error) {
	var d Document
	err := s.Scan(
		&d.ID,
		&d.ExternalID,
		&d.ExternalPlatform,
		&d.Filename,
		&d.ContentType,
		&d.SizeBytes,
		&d.PageCount,
		&d.StorageKey,
		&d.Status,
		&d.UploadedAt,
		&d.UpdatedAt,
		&d.Classification,
		&d.Confidence,
		&d.ClassifiedAt,
	)
	return d, err
}
```

Add `Classification` and `Confidence` to the `Filters` struct:

```go
type Filters struct {
	Status           *string `json:"status,omitempty"`
	Filename         *string `json:"filename,omitempty"`
	ExternalID       *int    `json:"external_id,omitempty"`
	ExternalPlatform *string `json:"external_platform,omitempty"`
	ContentType      *string `json:"content_type,omitempty"`
	StorageKey       *string `json:"storage_key,omitempty"`
	Classification   *string `json:"classification,omitempty"`
	Confidence       *string `json:"confidence,omitempty"`
}
```

Add the two new filters to `Apply`:

```go
func (f Filters) Apply(b *query.Builder) *query.Builder {
	return b.
		WhereEquals("Status", f.Status).
		WhereContains("Filename", f.Filename).
		WhereEquals("ExternalID", f.ExternalID).
		WhereEquals("ExternalPlatform", f.ExternalPlatform).
		WhereEquals("ContentType", f.ContentType).
		WhereContains("StorageKey", f.StorageKey).
		WhereEquals("Classification", f.Classification).
		WhereEquals("Confidence", f.Confidence)
}
```

Add query parameter parsing in `FiltersFromQuery`:

```go
// add at the end, before the return
if cl := values.Get("classification"); cl != "" {
    f.Classification = &cl
}

if co := values.Get("confidence"); co != "" {
    f.Confidence = &co
}
```

## Validation Criteria

- [ ] `go vet ./...` passes
- [ ] `go test ./tests/...` passes
- [ ] Existing builder tests still pass (backward compatible — `From()` returns `Table()` when no joins)
- [ ] `GET /api/documents` returns documents with `classification`, `confidence`, `classified_at` fields
- [ ] Unclassified documents return `null`/omitted for classification fields
- [ ] `GET /api/documents?classification=SECRET` filters by classification level
- [ ] `GET /api/documents?confidence=HIGH` filters by confidence
- [ ] `GET /api/documents/{id}` returns document with classification fields
