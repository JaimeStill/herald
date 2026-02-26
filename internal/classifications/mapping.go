package classifications

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/google/uuid"

	"github.com/JaimeStill/herald/pkg/query"
	"github.com/JaimeStill/herald/pkg/repository"
)

var projection = query.
	NewProjectionMap("public", "classifications", "c").
	Project("id", "ID").
	Project("document_id", "DocumentID").
	Project("classification", "Classification").
	Project("confidence", "Confidence").
	Project("markings_found", "MarkingsFound").
	Project("rationale", "Rationale").
	Project("classified_at", "ClassifiedAt").
	Project("model_name", "ModelName").
	Project("provider_name", "ProviderName").
	Project("validated_by", "ValidatedBy").
	Project("validated_at", "ValidatedAt")

var defaultSort = query.SortField{
	Field:      "ClassifiedAt",
	Descending: true,
}

// Filters contains optional filtering criteria for classification queries.
// Nil fields are ignored. All fields use exact matching.
type Filters struct {
	Classification *string    `json:"classification,omitempty"`
	Confidence     *string    `json:"confidence,omitempty"`
	DocumentID     *uuid.UUID `json:"document_id,omitempty"`
	ValidatedBy    *string    `json:"validated_by,omitempty"`
}

// Apply adds filter conditions to a query builder.
func (f Filters) Apply(b *query.Builder) *query.Builder {
	return b.
		WhereEquals("Classification", f.Classification).
		WhereEquals("Confidence", f.Confidence).
		WhereEquals("DocumentID", f.DocumentID).
		WhereEquals("ValidatedBy", f.ValidatedBy)
}

// FiltersFromQuery extracts filter values from URL query parameters.
func FiltersFromQuery(values url.Values) Filters {
	var f Filters

	if c := values.Get("classification"); c != "" {
		f.Classification = &c
	}

	if c := values.Get("confidence"); c != "" {
		f.Confidence = &c
	}

	if d := values.Get("document_id"); d != "" {
		if id, err := uuid.Parse(d); err == nil {
			f.DocumentID = &id
		}
	}

	if v := values.Get("validated_by"); v != "" {
		f.ValidatedBy = &v
	}

	return f
}

func scanClassification(s repository.Scanner) (Classification, error) {
	var c Classification
	var markingsRaw []byte

	err := s.Scan(
		&c.ID,
		&c.DocumentID,
		&c.Classification,
		&c.Confidence,
		&markingsRaw,
		&c.Rationale,
		&c.ClassifiedAt,
		&c.ModelName,
		&c.ProviderName,
		&c.ValidatedBy,
		&c.ValidatedAt,
	)

	if err != nil {
		return c, err
	}

	if len(markingsRaw) > 0 {
		if err := json.Unmarshal(markingsRaw, &c.MarkingsFound); err != nil {
			return c, fmt.Errorf("unmarshal markings_found: %w", err)
		}
	}

	if c.MarkingsFound == nil {
		c.MarkingsFound = []string{}
	}

	return c, nil
}
