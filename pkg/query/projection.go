// Package query provides SQL query building utilities with projection mapping.
package query

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// JoinClause describes a SQL JOIN between two tables.
// Index preserves insertion order for deterministic SQL generation.
type JoinClause struct {
	Index    int
	JoinType string
	Schema   string
	Table    string
	Alias    string
	On       string
}

// ProjectionMap maps view property names to qualified column references (alias.column).
// It defines the table, alias, and column mappings for SQL query construction.
type ProjectionMap struct {
	schema       string
	table        string
	alias        string
	currentAlias string
	columns      map[string]string
	columnList   []string
	joins        map[string]JoinClause
}

// NewProjectionMap creates a ProjectionMap for the given schema, table, and alias.
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

// Project adds a column mapping from database column to view property name.
func (p *ProjectionMap) Project(column, viewName string) *ProjectionMap {
	qualified := fmt.Sprintf("%s.%s", p.currentAlias, column)
	p.columns[viewName] = qualified
	p.columnList = append(p.columnList, qualified)
	return p
}

// Join adds a JOIN clause and switches the projection context to the joined table.
// Subsequent Project calls map columns to the joined table's alias.
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

// Alias returns the table alias.
func (p *ProjectionMap) Alias() string {
	return p.alias
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
		fmt.Fprintf(
			&b, " %s %s.%s %s ON %s",
			j.JoinType,
			j.Schema,
			j.Table,
			j.Alias,
			j.On,
		)
	}
	return b.String()
}

// Joins returns all join clauses sorted by insertion order.
func (p *ProjectionMap) Joins() []JoinClause {
	return slices.SortedFunc(maps.Values(p.joins), func(a, b JoinClause) int {
		return a.Index - b.Index
	})
}

// Table returns the fully qualified table reference with alias (schema.table alias).
func (p *ProjectionMap) Table() string {
	return fmt.Sprintf("%s.%s %s", p.schema, p.table, p.alias)
}

// Column returns the qualified column for a view property name, or the input if not mapped.
func (p *ProjectionMap) Column(viewName string) string {
	if col, ok := p.columns[viewName]; ok {
		return col
	}
	return viewName
}

// Columns returns all mapped columns as a comma-separated string.
func (p *ProjectionMap) Columns() string {
	return strings.Join(p.columnList, ", ")
}

// ColumnList returns all mapped columns as a slice.
func (p *ProjectionMap) ColumnList() []string {
	return p.columnList
}
