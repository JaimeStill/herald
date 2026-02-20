// Package query provides SQL query building utilities with projection mapping.
package query

import (
	"fmt"
	"strings"
)

// ProjectionMap maps view property names to qualified column references (alias.column).
// It defines the table, alias, and column mappings for SQL query construction.
type ProjectionMap struct {
	schema     string
	table      string
	alias      string
	columns    map[string]string
	columnList []string
}

// NewProjectionMap creates a ProjectionMap for the given schema, table, and alias.
func NewProjectionMap(schema, table, alias string) *ProjectionMap {
	return &ProjectionMap{
		schema:     schema,
		table:      table,
		alias:      alias,
		columns:    make(map[string]string),
		columnList: make([]string, 0),
	}
}

// Project adds a column mapping from database column to view property name.
func (p *ProjectionMap) Project(column, viewName string) *ProjectionMap {
	qualified := fmt.Sprintf("%s.%s", p.alias, column)
	p.columns[viewName] = qualified
	p.columnList = append(p.columnList, qualified)
	return p
}

// Alias returns the table alias.
func (p *ProjectionMap) Alias() string {
	return p.alias
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
