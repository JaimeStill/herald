package prompts

import (
	"net/url"
	"strconv"

	"github.com/JaimeStill/herald/pkg/query"
	"github.com/JaimeStill/herald/pkg/repository"
)

var projection = query.
	NewProjectionMap("public", "prompts", "p").
	Project("id", "ID").
	Project("name", "Name").
	Project("stage", "Stage").
	Project("instructions", "Instructions").
	Project("description", "Description").
	Project("active", "Active")

var defaultSort = query.SortField{
	Field: "name",
}

// Filters contains optional filtering criteria for prompt queries.
// Nil fields are ignored. Stage and Active use exact matching.
// Name uses case-insensitive contains matching.
type Filters struct {
	Stage  *Stage  `json:"stage,omitempty"`
	Name   *string `json:"name,omitempty"`
	Active *bool   `json:"active,omitempty"`
}

// Apply adds filter conditions to a query builder.
func (f Filters) Apply(b *query.Builder) *query.Builder {
	return b.
		WhereEquals("Stage", f.Stage).
		WhereContains("Name", f.Name).
		WhereEquals("Active", f.Active)
}

// FiltersFromQuery extracts filter values from URL query parameters.
func FiltersFromQuery(values url.Values) Filters {
	var f Filters

	if s := values.Get("stage"); s != "" {
		stage := Stage(s)
		f.Stage = &stage
	}

	if n := values.Get("name"); n != "" {
		f.Name = &n
	}

	if a := values.Get("active"); a != "" {
		if v, err := strconv.ParseBool(a); err == nil {
			f.Active = &v
		}
	}

	return f
}

func scanPrompt(s repository.Scanner) (Prompt, error) {
	var p Prompt
	err := s.Scan(
		&p.ID,
		&p.Name,
		&p.Stage,
		&p.Instructions,
		&p.Description,
		&p.Active,
	)
	return p, err
}
