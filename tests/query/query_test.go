package query_test

import (
	"testing"

	"github.com/JaimeStill/herald/pkg/query"
)

func testProjection() *query.ProjectionMap {
	return query.NewProjectionMap("public", "documents", "d").
		Project("id", "id").
		Project("filename", "filename").
		Project("created_at", "createdAt")
}

func ptr(s string) *string { return &s }

func TestProjectionMapTable(t *testing.T) {
	p := testProjection()
	got := p.Table()
	want := "public.documents d"
	if got != want {
		t.Errorf("Table() = %q, want %q", got, want)
	}
}

func TestProjectionMapAlias(t *testing.T) {
	p := testProjection()
	if got := p.Alias(); got != "d" {
		t.Errorf("Alias() = %q, want %q", got, "d")
	}
}

func TestProjectionMapColumns(t *testing.T) {
	p := testProjection()
	got := p.Columns()
	want := "d.id, d.filename, d.created_at"
	if got != want {
		t.Errorf("Columns() = %q, want %q", got, want)
	}
}

func TestProjectionMapColumnList(t *testing.T) {
	p := testProjection()
	got := p.ColumnList()
	if len(got) != 3 {
		t.Fatalf("ColumnList() length = %d, want 3", len(got))
	}
	want := []string{"d.id", "d.filename", "d.created_at"}
	for i, col := range got {
		if col != want[i] {
			t.Errorf("ColumnList()[%d] = %q, want %q", i, col, want[i])
		}
	}
}

func TestProjectionMapColumnLookup(t *testing.T) {
	p := testProjection()

	tests := []struct {
		name     string
		viewName string
		want     string
	}{
		{"mapped field", "filename", "d.filename"},
		{"mapped camel", "createdAt", "d.created_at"},
		{"unmapped passthrough", "unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := p.Column(tt.viewName); got != tt.want {
				t.Errorf("Column(%q) = %q, want %q", tt.viewName, got, tt.want)
			}
		})
	}
}

func TestParseSortFields(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []query.SortField
	}{
		{
			name:  "empty string",
			input: "",
			want:  nil,
		},
		{
			name:  "single ascending",
			input: "name",
			want:  []query.SortField{{Field: "name", Descending: false}},
		},
		{
			name:  "single descending",
			input: "-createdAt",
			want:  []query.SortField{{Field: "createdAt", Descending: true}},
		},
		{
			name:  "multiple mixed",
			input: "name,-createdAt",
			want: []query.SortField{
				{Field: "name", Descending: false},
				{Field: "createdAt", Descending: true},
			},
		},
		{
			name:  "with spaces",
			input: " name , -createdAt ",
			want: []query.SortField{
				{Field: "name", Descending: false},
				{Field: "createdAt", Descending: true},
			},
		},
		{
			name:  "empty parts skipped",
			input: "name,,createdAt",
			want: []query.SortField{
				{Field: "name", Descending: false},
				{Field: "createdAt", Descending: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := query.ParseSortFields(tt.input)
			if tt.want == nil {
				if got != nil {
					t.Errorf("ParseSortFields(%q) = %v, want nil", tt.input, got)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("ParseSortFields(%q) length = %d, want %d", tt.input, len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ParseSortFields(%q)[%d] = %v, want %v", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestBuilderBuild(t *testing.T) {
	p := testProjection()
	b := query.NewBuilder(p)
	sql, args := b.Build()

	wantSQL := "SELECT d.id, d.filename, d.created_at FROM public.documents d"
	if sql != wantSQL {
		t.Errorf("Build() sql = %q, want %q", sql, wantSQL)
	}
	if len(args) != 0 {
		t.Errorf("Build() args = %v, want empty", args)
	}
}

func TestBuilderBuildCount(t *testing.T) {
	p := testProjection()
	b := query.NewBuilder(p)
	sql, args := b.BuildCount()

	wantSQL := "SELECT COUNT(*) FROM public.documents d"
	if sql != wantSQL {
		t.Errorf("BuildCount() sql = %q, want %q", sql, wantSQL)
	}
	if len(args) != 0 {
		t.Errorf("BuildCount() args = %v, want empty", args)
	}
}

func TestBuilderBuildPage(t *testing.T) {
	p := testProjection()
	b := query.NewBuilder(p, query.SortField{Field: "createdAt", Descending: true})
	sql, args := b.BuildPage(2, 10)

	wantSQL := "SELECT d.id, d.filename, d.created_at FROM public.documents d ORDER BY d.created_at DESC LIMIT 10 OFFSET 10"
	if sql != wantSQL {
		t.Errorf("BuildPage() sql = %q, want %q", sql, wantSQL)
	}
	if len(args) != 0 {
		t.Errorf("BuildPage() args = %v, want empty", args)
	}
}

func TestBuilderBuildSingle(t *testing.T) {
	p := testProjection()
	b := query.NewBuilder(p)
	sql, args := b.BuildSingle("id", "abc-123")

	wantSQL := "SELECT d.id, d.filename, d.created_at FROM public.documents d WHERE d.id = $1"
	if sql != wantSQL {
		t.Errorf("BuildSingle() sql = %q, want %q", sql, wantSQL)
	}
	if len(args) != 1 || args[0] != "abc-123" {
		t.Errorf("BuildSingle() args = %v, want [abc-123]", args)
	}
}

func TestBuilderBuildSingleOrNull(t *testing.T) {
	p := testProjection()
	b := query.NewBuilder(p)
	b.WhereEquals("filename", "test.pdf")
	sql, args := b.BuildSingleOrNull()

	wantSQL := "SELECT d.id, d.filename, d.created_at FROM public.documents d WHERE d.filename = $1 LIMIT 1"
	if sql != wantSQL {
		t.Errorf("BuildSingleOrNull() sql = %q, want %q", sql, wantSQL)
	}
	if len(args) != 1 || args[0] != "test.pdf" {
		t.Errorf("BuildSingleOrNull() args = %v, want [test.pdf]", args)
	}
}

func TestBuilderWhereEquals(t *testing.T) {
	p := testProjection()
	b := query.NewBuilder(p)
	b.WhereEquals("filename", "test.pdf")
	sql, args := b.Build()

	wantSQL := "SELECT d.id, d.filename, d.created_at FROM public.documents d WHERE d.filename = $1"
	if sql != wantSQL {
		t.Errorf("sql = %q, want %q", sql, wantSQL)
	}
	if len(args) != 1 || args[0] != "test.pdf" {
		t.Errorf("args = %v, want [test.pdf]", args)
	}
}

func TestBuilderWhereEqualsNilSkipped(t *testing.T) {
	p := testProjection()
	b := query.NewBuilder(p)
	b.WhereEquals("filename", nil)
	sql, args := b.Build()

	wantSQL := "SELECT d.id, d.filename, d.created_at FROM public.documents d"
	if sql != wantSQL {
		t.Errorf("sql = %q, want %q", sql, wantSQL)
	}
	if len(args) != 0 {
		t.Errorf("args = %v, want empty", args)
	}
}

func TestBuilderWhereContains(t *testing.T) {
	p := testProjection()
	b := query.NewBuilder(p)
	b.WhereContains("filename", ptr("test"))
	sql, args := b.Build()

	wantSQL := "SELECT d.id, d.filename, d.created_at FROM public.documents d WHERE d.filename ILIKE $1"
	if sql != wantSQL {
		t.Errorf("sql = %q, want %q", sql, wantSQL)
	}
	if len(args) != 1 || args[0] != "%test%" {
		t.Errorf("args = %v, want [%%test%%]", args)
	}
}

func TestBuilderWhereContainsNilSkipped(t *testing.T) {
	p := testProjection()
	b := query.NewBuilder(p)
	b.WhereContains("filename", nil)
	_, args := b.Build()

	if len(args) != 0 {
		t.Errorf("args = %v, want empty", args)
	}
}

func TestBuilderWhereContainsEmptySkipped(t *testing.T) {
	p := testProjection()
	b := query.NewBuilder(p)
	b.WhereContains("filename", ptr(""))
	_, args := b.Build()

	if len(args) != 0 {
		t.Errorf("args = %v, want empty", args)
	}
}

func TestBuilderWhereIn(t *testing.T) {
	p := testProjection()
	b := query.NewBuilder(p)
	b.WhereIn("id", []any{"a", "b", "c"})
	sql, args := b.Build()

	wantSQL := "SELECT d.id, d.filename, d.created_at FROM public.documents d WHERE d.id IN ($1, $2, $3)"
	if sql != wantSQL {
		t.Errorf("sql = %q, want %q", sql, wantSQL)
	}
	if len(args) != 3 {
		t.Errorf("args length = %d, want 3", len(args))
	}
}

func TestBuilderWhereInEmptySkipped(t *testing.T) {
	p := testProjection()
	b := query.NewBuilder(p)
	b.WhereIn("id", []any{})
	_, args := b.Build()

	if len(args) != 0 {
		t.Errorf("args = %v, want empty", args)
	}
}

func TestBuilderWhereNullable(t *testing.T) {
	t.Run("nil value generates IS NULL", func(t *testing.T) {
		p := testProjection()
		b := query.NewBuilder(p)
		b.WhereNullable("filename", nil)
		sql, args := b.Build()

		wantSQL := "SELECT d.id, d.filename, d.created_at FROM public.documents d WHERE d.filename IS NULL"
		if sql != wantSQL {
			t.Errorf("sql = %q, want %q", sql, wantSQL)
		}
		if len(args) != 0 {
			t.Errorf("args = %v, want empty", args)
		}
	})

	t.Run("non-nil value generates equals", func(t *testing.T) {
		p := testProjection()
		b := query.NewBuilder(p)
		b.WhereNullable("filename", "test.pdf")
		sql, args := b.Build()

		wantSQL := "SELECT d.id, d.filename, d.created_at FROM public.documents d WHERE d.filename = $1"
		if sql != wantSQL {
			t.Errorf("sql = %q, want %q", sql, wantSQL)
		}
		if len(args) != 1 || args[0] != "test.pdf" {
			t.Errorf("args = %v, want [test.pdf]", args)
		}
	})
}

func TestBuilderWhereSearch(t *testing.T) {
	p := testProjection()
	b := query.NewBuilder(p)
	b.WhereSearch(ptr("test"), "filename", "id")
	sql, args := b.Build()

	wantSQL := "SELECT d.id, d.filename, d.created_at FROM public.documents d WHERE (d.filename ILIKE $1 OR d.id ILIKE $2)"
	if sql != wantSQL {
		t.Errorf("sql = %q, want %q", sql, wantSQL)
	}
	if len(args) != 2 || args[0] != "%test%" || args[1] != "%test%" {
		t.Errorf("args = %v, want [%%test%% %%test%%]", args)
	}
}

func TestBuilderWhereSearchNilSkipped(t *testing.T) {
	p := testProjection()
	b := query.NewBuilder(p)
	b.WhereSearch(nil, "filename")
	_, args := b.Build()

	if len(args) != 0 {
		t.Errorf("args = %v, want empty", args)
	}
}

func TestBuilderMultipleConditions(t *testing.T) {
	p := testProjection()
	b := query.NewBuilder(p)
	b.WhereEquals("filename", "test.pdf")
	b.WhereContains("id", ptr("abc"))
	sql, args := b.Build()

	wantSQL := "SELECT d.id, d.filename, d.created_at FROM public.documents d WHERE d.filename = $1 AND d.id ILIKE $2"
	if sql != wantSQL {
		t.Errorf("sql = %q, want %q", sql, wantSQL)
	}
	if len(args) != 2 {
		t.Errorf("args length = %d, want 2", len(args))
	}
	if args[0] != "test.pdf" {
		t.Errorf("args[0] = %v, want test.pdf", args[0])
	}
	if args[1] != "%abc%" {
		t.Errorf("args[1] = %v, want %%abc%%", args[1])
	}
}

func TestBuilderOrderByFields(t *testing.T) {
	p := testProjection()
	b := query.NewBuilder(p, query.SortField{Field: "id", Descending: false})
	b.OrderByFields([]query.SortField{
		{Field: "createdAt", Descending: true},
		{Field: "filename", Descending: false},
	})
	sql, _ := b.Build()

	wantSQL := "SELECT d.id, d.filename, d.created_at FROM public.documents d ORDER BY d.created_at DESC, d.filename ASC"
	if sql != wantSQL {
		t.Errorf("sql = %q, want %q", sql, wantSQL)
	}
}

func TestBuilderDefaultSort(t *testing.T) {
	p := testProjection()
	b := query.NewBuilder(p, query.SortField{Field: "createdAt", Descending: true})
	sql, _ := b.Build()

	wantSQL := "SELECT d.id, d.filename, d.created_at FROM public.documents d ORDER BY d.created_at DESC"
	if sql != wantSQL {
		t.Errorf("sql = %q, want %q", sql, wantSQL)
	}
}

func TestBuilderBuildCountWithConditions(t *testing.T) {
	p := testProjection()
	b := query.NewBuilder(p)
	b.WhereEquals("filename", "test.pdf")
	sql, args := b.BuildCount()

	wantSQL := "SELECT COUNT(*) FROM public.documents d WHERE d.filename = $1"
	if sql != wantSQL {
		t.Errorf("sql = %q, want %q", sql, wantSQL)
	}
	if len(args) != 1 || args[0] != "test.pdf" {
		t.Errorf("args = %v, want [test.pdf]", args)
	}
}

func TestBuilderBuildPageWithConditions(t *testing.T) {
	p := testProjection()
	b := query.NewBuilder(p, query.SortField{Field: "id"})
	b.WhereContains("filename", ptr("report"))
	sql, args := b.BuildPage(3, 25)

	wantSQL := "SELECT d.id, d.filename, d.created_at FROM public.documents d WHERE d.filename ILIKE $1 ORDER BY d.id ASC LIMIT 25 OFFSET 50"
	if sql != wantSQL {
		t.Errorf("sql = %q, want %q", sql, wantSQL)
	}
	if len(args) != 1 || args[0] != "%report%" {
		t.Errorf("args = %v, want [%%report%%]", args)
	}
}

// --- JOIN tests ---

func testJoinedProjection() *query.ProjectionMap {
	return query.NewProjectionMap("public", "documents", "d").
		Project("id", "ID").
		Project("filename", "Filename").
		Join("public", "classifications", "c", "LEFT JOIN", "d.id = c.document_id").
		Project("classification", "Classification").
		Project("confidence", "Confidence")
}

func TestProjectionMapFromNoJoins(t *testing.T) {
	p := testProjection()
	got := p.From()
	want := "public.documents d"
	if got != want {
		t.Errorf("From() = %q, want %q", got, want)
	}
}

func TestProjectionMapFromWithJoin(t *testing.T) {
	p := testJoinedProjection()
	got := p.From()
	want := "public.documents d LEFT JOIN public.classifications c ON d.id = c.document_id"
	if got != want {
		t.Errorf("From() = %q, want %q", got, want)
	}
}

func TestProjectionMapJoinContextSwitching(t *testing.T) {
	p := testJoinedProjection()

	tests := []struct {
		name     string
		viewName string
		want     string
	}{
		{"base table column", "Filename", "d.filename"},
		{"joined table column", "Classification", "c.classification"},
		{"joined table column 2", "Confidence", "c.confidence"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := p.Column(tt.viewName); got != tt.want {
				t.Errorf("Column(%q) = %q, want %q", tt.viewName, got, tt.want)
			}
		})
	}
}

func TestProjectionMapJoinedColumns(t *testing.T) {
	p := testJoinedProjection()
	got := p.Columns()
	want := "d.id, d.filename, c.classification, c.confidence"
	if got != want {
		t.Errorf("Columns() = %q, want %q", got, want)
	}
}

func TestProjectionMapJoins(t *testing.T) {
	p := testJoinedProjection()
	joins := p.Joins()
	if len(joins) != 1 {
		t.Fatalf("Joins() length = %d, want 1", len(joins))
	}
	j := joins[0]
	if j.Alias != "c" {
		t.Errorf("Joins()[0].Alias = %q, want %q", j.Alias, "c")
	}
	if j.JoinType != "LEFT JOIN" {
		t.Errorf("Joins()[0].JoinType = %q, want %q", j.JoinType, "LEFT JOIN")
	}
}

func TestProjectionMapJoinsOrder(t *testing.T) {
	p := query.NewProjectionMap("public", "documents", "d").
		Project("id", "ID").
		Join("public", "classifications", "c", "LEFT JOIN", "d.id = c.document_id").
		Project("classification", "Classification").
		Join("public", "users", "u", "LEFT JOIN", "c.validated_by = u.id").
		Project("name", "ValidatedByName")

	joins := p.Joins()
	if len(joins) != 2 {
		t.Fatalf("Joins() length = %d, want 2", len(joins))
	}
	if joins[0].Alias != "c" {
		t.Errorf("Joins()[0].Alias = %q, want %q", joins[0].Alias, "c")
	}
	if joins[1].Alias != "u" {
		t.Errorf("Joins()[1].Alias = %q, want %q", joins[1].Alias, "u")
	}
}

func TestBuilderBuildWithJoin(t *testing.T) {
	p := testJoinedProjection()
	b := query.NewBuilder(p)
	sql, args := b.Build()

	wantSQL := "SELECT d.id, d.filename, c.classification, c.confidence FROM public.documents d LEFT JOIN public.classifications c ON d.id = c.document_id"
	if sql != wantSQL {
		t.Errorf("Build() sql = %q, want %q", sql, wantSQL)
	}
	if len(args) != 0 {
		t.Errorf("Build() args = %v, want empty", args)
	}
}

func TestBuilderBuildCountWithJoin(t *testing.T) {
	p := testJoinedProjection()
	b := query.NewBuilder(p)
	sql, _ := b.BuildCount()

	wantSQL := "SELECT COUNT(*) FROM public.documents d LEFT JOIN public.classifications c ON d.id = c.document_id"
	if sql != wantSQL {
		t.Errorf("BuildCount() sql = %q, want %q", sql, wantSQL)
	}
}

func TestBuilderBuildPageWithJoin(t *testing.T) {
	p := testJoinedProjection()
	b := query.NewBuilder(p, query.SortField{Field: "Filename"})
	sql, _ := b.BuildPage(1, 20)

	wantSQL := "SELECT d.id, d.filename, c.classification, c.confidence FROM public.documents d LEFT JOIN public.classifications c ON d.id = c.document_id ORDER BY d.filename ASC LIMIT 20 OFFSET 0"
	if sql != wantSQL {
		t.Errorf("BuildPage() sql = %q, want %q", sql, wantSQL)
	}
}

func TestBuilderBuildSingleWithJoin(t *testing.T) {
	p := testJoinedProjection()
	b := query.NewBuilder(p)
	sql, args := b.BuildSingle("ID", "abc-123")

	wantSQL := "SELECT d.id, d.filename, c.classification, c.confidence FROM public.documents d LEFT JOIN public.classifications c ON d.id = c.document_id WHERE d.id = $1"
	if sql != wantSQL {
		t.Errorf("BuildSingle() sql = %q, want %q", sql, wantSQL)
	}
	if len(args) != 1 || args[0] != "abc-123" {
		t.Errorf("BuildSingle() args = %v, want [abc-123]", args)
	}
}

func TestBuilderWhereOnJoinedColumn(t *testing.T) {
	p := testJoinedProjection()
	b := query.NewBuilder(p)
	b.WhereEquals("Classification", "SECRET")
	sql, args := b.Build()

	wantSQL := "SELECT d.id, d.filename, c.classification, c.confidence FROM public.documents d LEFT JOIN public.classifications c ON d.id = c.document_id WHERE c.classification = $1"
	if sql != wantSQL {
		t.Errorf("sql = %q, want %q", sql, wantSQL)
	}
	if len(args) != 1 || args[0] != "SECRET" {
		t.Errorf("args = %v, want [SECRET]", args)
	}
}
