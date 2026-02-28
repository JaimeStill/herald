package query

import (
	"fmt"
	"reflect"
	"strings"
)

type condition struct {
	clause string
	args   []any
}

// SortField represents a single column in an ORDER BY clause.
// Field is the logical field name (mapped via ProjectionMap).
// Descending controls sort direction (false = ASC, true = DESC).
type SortField struct {
	Field      string
	Descending bool
}

// Builder constructs SQL queries using a fluent API with automatic parameter numbering.
type Builder struct {
	projection        *ProjectionMap
	conditions        []condition
	orderByFields     []SortField
	defaultSortFields []SortField
}

// NewBuilder creates a Builder for the given projection with optional default sort fields.
func NewBuilder(projection *ProjectionMap, defaultSort ...SortField) *Builder {
	return &Builder{
		projection:        projection,
		conditions:        make([]condition, 0),
		defaultSortFields: defaultSort,
	}
}

// ParseSortFields parses a comma-separated sort string into a SortField slice.
// Fields prefixed with "-" are descending. Example: "name,-createdAt".
// Returns nil for empty input.
func ParseSortFields(s string) []SortField {
	if s == "" {
		return nil
	}

	parts := strings.Split(s, ",")
	fields := make([]SortField, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if after, ok := strings.CutPrefix(part, "-"); ok {
			fields = append(fields, SortField{
				Field:      after,
				Descending: true,
			})
		} else {
			fields = append(fields, SortField{
				Field:      part,
				Descending: false,
			})
		}
	}

	return fields
}

// Build returns a SELECT query with the current conditions and ordering.
func (b *Builder) Build() (string, []any) {
	where, args, _ := b.buildWhere(1)
	orderBy := b.buildOrderBy()

	sql := fmt.Sprintf(
		"SELECT %s FROM %s%s%s",
		b.projection.Columns(),
		b.projection.From(),
		where,
		orderBy,
	)

	return sql, args
}

// BuildCount returns a COUNT(*) query with the current conditions.
func (b *Builder) BuildCount() (string, []any) {
	where, args, _ := b.buildWhere(1)
	sql := fmt.Sprintf("SELECT COUNT(*) FROM %s%s", b.projection.From(), where)
	return sql, args
}

// BuildPage returns a paginated SELECT query with ordering, limit, and offset.
func (b *Builder) BuildPage(page, pageSize int) (string, []any) {
	where, args, _ := b.buildWhere(1)
	orderBy := b.buildOrderBy()
	offset := (page - 1) * pageSize

	sql := fmt.Sprintf(
		"SELECT %s FROM %s%s%s LIMIT %d OFFSET %d",
		b.projection.Columns(),
		b.projection.From(),
		where,
		orderBy,
		pageSize,
		offset,
	)

	return sql, args
}

// BuildSingle returns a SELECT query for a single record by ID.
func (b *Builder) BuildSingle(idField string, id any) (string, []any) {
	col := b.projection.Column(idField)
	sql := fmt.Sprintf(
		"SELECT %s FROM %s WHERE %s = $1",
		b.projection.Columns(),
		b.projection.From(),
		col,
	)
	return sql, []any{id}
}

// BuildSingleOrNull returns a SELECT query limited to one row with the current conditions.
func (b *Builder) BuildSingleOrNull() (string, []any) {
	where, args, _ := b.buildWhere(1)
	sql := fmt.Sprintf(
		"SELECT %s FROM %s%s LIMIT 1",
		b.projection.Columns(),
		b.projection.From(),
		where,
	)
	return sql, args
}

// OrderByFields sets the sort order, overriding default sort fields.
func (b *Builder) OrderByFields(fields []SortField) *Builder {
	b.orderByFields = fields
	return b
}

// WhereContains adds a case-insensitive ILIKE condition. No-op for nil or empty values.
func (b *Builder) WhereContains(field string, value *string) *Builder {
	if value == nil || *value == "" {
		return b
	}
	col := b.projection.Column(field)
	b.conditions = append(b.conditions, condition{
		clause: fmt.Sprintf("%s ILIKE $%%d", col),
		args:   []any{"%" + *value + "%"},
	})
	return b
}

// WhereEquals adds an equality condition. No-op for nil values.
func (b *Builder) WhereEquals(field string, value any) *Builder {
	if isNil(value) {
		return b
	}
	col := b.projection.Column(field)
	b.conditions = append(b.conditions, condition{
		clause: fmt.Sprintf("%s = $%%d", col),
		args:   []any{value},
	})
	return b
}

// WhereIn adds an IN condition for multiple values. No-op for empty slices.
func (b *Builder) WhereIn(field string, values []any) *Builder {
	if len(values) == 0 {
		return b
	}
	col := b.projection.Column(field)
	placeholders := make([]string, len(values))
	for i := range values {
		placeholders[i] = "$%d"
	}
	b.conditions = append(b.conditions, condition{
		clause: fmt.Sprintf("%s IN (%s)", col, strings.Join(placeholders, ", ")),
		args:   values,
	})
	return b
}

// WhereNullable adds an equality or IS NULL condition depending on whether value is nil.
func (b *Builder) WhereNullable(column string, val any) *Builder {
	col := b.projection.Column(column)
	if isNil(val) {
		b.conditions = append(b.conditions, condition{
			clause: col + " IS NULL",
			args:   nil,
		})
	} else {
		b.conditions = append(b.conditions, condition{
			clause: fmt.Sprintf("%s = $%%d", col),
			args:   []any{val},
		})
	}

	return b
}

// WhereSearch adds an OR condition across multiple fields with ILIKE. No-op for nil or empty search.
func (b *Builder) WhereSearch(search *string, fields ...string) *Builder {
	if search == nil || *search == "" || len(fields) == 0 {
		return b
	}

	clauses := make([]string, len(fields))
	args := make([]any, len(fields))
	searchPattern := "%" + *search + "%"

	for i, field := range fields {
		col := b.projection.Column(field)
		clauses[i] = fmt.Sprintf("%s ILIKE $%%d", col)
		args[i] = searchPattern
	}

	b.conditions = append(b.conditions, condition{
		clause: "(" + strings.Join(clauses, " OR ") + ")",
		args:   args,
	})
	return b
}

func (b *Builder) buildOrderBy() string {
	fields := b.orderByFields
	if len(fields) == 0 {
		fields = b.defaultSortFields
	}

	if len(fields) == 0 {
		return ""
	}

	parts := make([]string, len(fields))
	for i, f := range fields {
		col := b.projection.Column(f.Field)
		dir := "ASC"
		if f.Descending {
			dir = "DESC"
		}
		parts[i] = fmt.Sprintf("%s %s", col, dir)
	}

	return " ORDER BY " + strings.Join(parts, ", ")
}

func (b *Builder) buildWhere(startParam int) (string, []any, int) {
	if len(b.conditions) == 0 {
		return "", nil, startParam
	}

	clauses := make([]string, 0, len(b.conditions))
	args := make([]any, 0)
	paramIdx := startParam

	for _, cond := range b.conditions {
		clause := cond.clause
		for _, arg := range cond.args {
			clause = strings.Replace(clause, "$%d", fmt.Sprintf("$%d", paramIdx), 1)
			args = append(args, arg)
			paramIdx++
		}
		clauses = append(clauses, clause)
	}

	return " WHERE " + strings.Join(clauses, " AND "), args, paramIdx
}

func isNil(value any) bool {
	if value == nil {
		return true
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func, reflect.Interface:
		return v.IsNil()
	}

	return false
}
