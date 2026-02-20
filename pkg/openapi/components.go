package openapi

import "maps"

// NewComponents creates Components with shared schemas and error responses.
func NewComponents() *Components {
	return &Components{
		Schemas: map[string]*Schema{
			"PageRequest": {
				Type: "object",
				Properties: map[string]*Schema{
					"page":      {Type: "integer", Description: "Page number (1-indexed)", Example: 1},
					"page_size": {Type: "integer", Description: "Results per page", Example: 20},
					"search":    {Type: "string", Description: "Search query"},
					"sort":      {Type: "string", Description: "Comma-separated sort fields. Prefix with - for descending. Example: name,-created_at"},
				},
			},
		},
		Responses: map[string]*Response{
			"BadRequest": {
				Description: "Invalid request",
				Content: map[string]*MediaType{
					"application/json": {
						Schema: &Schema{
							Type: "object",
							Properties: map[string]*Schema{
								"error": {Type: "string", Description: "Error message"},
							},
						},
					},
				},
			},
			"NotFound": {
				Description: "Resource not found",
				Content: map[string]*MediaType{
					"application/json": {
						Schema: &Schema{
							Type: "object",
							Properties: map[string]*Schema{
								"error": {Type: "string", Description: "Error message"},
							},
						},
					},
				},
			},
			"Conflict": {
				Description: "Resource conflict (duplicate name)",
				Content: map[string]*MediaType{
					"application/json": {
						Schema: &Schema{
							Type: "object",
							Properties: map[string]*Schema{
								"error": {Type: "string", Description: "Error message"},
							},
						},
					},
				},
			},
		},
	}
}

// AddSchemas merges the given schemas into the component schemas.
func (c *Components) AddSchemas(schemas map[string]*Schema) {
	maps.Copy(c.Schemas, schemas)
}

// AddResponses merges the given responses into the component responses.
func (c *Components) AddResponses(responses map[string]*Response) {
	maps.Copy(c.Responses, responses)
}
