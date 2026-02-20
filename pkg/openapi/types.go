package openapi

// Info represents the OpenAPI info object.
type Info struct {
	Title       string `json:"title"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
}

// Server represents an OpenAPI server object.
type Server struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// PathItem groups operations available on a single path.
type PathItem struct {
	Get    *Operation `json:"get,omitempty"`
	Post   *Operation `json:"post,omitempty"`
	Put    *Operation `json:"put,omitempty"`
	Delete *Operation `json:"delete,omitempty"`
}

// Operation describes a single API operation on a path.
type Operation struct {
	Summary     string            `json:"summary,omitempty"`
	Description string            `json:"description,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Parameters  []*Parameter      `json:"parameters,omitempty"`
	RequestBody *RequestBody      `json:"requestBody,omitempty"`
	Responses   map[int]*Response `json:"responses"`
}

// Parameter describes a single operation parameter.
type Parameter struct {
	Name        string  `json:"name"`
	In          string  `json:"in"`
	Required    bool    `json:"required,omitempty"`
	Description string  `json:"description,omitempty"`
	Schema      *Schema `json:"schema"`
}

// RequestBody describes a single request body.
type RequestBody struct {
	Description string                `json:"description,omitempty"`
	Required    bool                  `json:"required,omitempty"`
	Content     map[string]*MediaType `json:"content"`
}

// Response describes a single response from an API operation.
type Response struct {
	Description string                `json:"description"`
	Content     map[string]*MediaType `json:"content,omitempty"`
	Ref         string                `json:"$ref,omitempty"`
}

// MediaType describes a media type with a schema.
type MediaType struct {
	Schema *Schema `json:"schema,omitempty"`
}

// Schema represents a JSON Schema object used in OpenAPI.
type Schema struct {
	Type        string             `json:"type,omitempty"`
	Format      string             `json:"format,omitempty"`
	Description string             `json:"description,omitempty"`
	Properties  map[string]*Schema `json:"properties,omitempty"`
	Required    []string           `json:"required,omitempty"`
	Items       *Schema            `json:"items,omitempty"`
	Ref         string             `json:"$ref,omitempty"`

	Example any   `json:"example,omitempty"`
	Default any   `json:"default,omitempty"`
	Enum    []any `json:"enum,omitempty"`

	Minimum   *float64 `json:"minimum,omitempty"`
	Maximum   *float64 `json:"maximum,omitempty"`
	MinLength *int     `json:"minLength,omitempty"`
	MaxLength *int     `json:"maxLength,omitempty"`
	Pattern   string   `json:"pattern,omitempty"`
}

// Components holds reusable schemas and responses.
type Components struct {
	Schemas   map[string]*Schema   `json:"schemas,omitempty"`
	Responses map[string]*Response `json:"responses,omitempty"`
}

// SchemaRef returns a Schema with a $ref to the named component schema.
func SchemaRef(name string) *Schema {
	return &Schema{Ref: "#/components/schemas/" + name}
}

// ResponseRef returns a Response with a $ref to the named component response.
func ResponseRef(name string) *Response {
	return &Response{Ref: "#/components/responses/" + name}
}

// RequestBodyJSON creates a JSON request body referencing the named schema.
func RequestBodyJSON(schemaName string, required bool) *RequestBody {
	return &RequestBody{
		Required: required,
		Content: map[string]*MediaType{
			"application/json": {Schema: SchemaRef(schemaName)},
		},
	}
}

// ResponseJSON creates a JSON response referencing the named schema.
func ResponseJSON(description, schemaName string) *Response {
	return &Response{
		Description: description,
		Content: map[string]*MediaType{
			"application/json": {Schema: SchemaRef(schemaName)},
		},
	}
}

// PathParam creates a required UUID path parameter.
func PathParam(name, description string) *Parameter {
	return &Parameter{
		Name:        name,
		In:          "path",
		Required:    true,
		Description: description,
		Schema:      &Schema{Type: "string", Format: "uuid"},
	}
}

// QueryParam creates a query parameter with the given type.
func QueryParam(name, typ, description string, required bool) *Parameter {
	return &Parameter{
		Name:        name,
		In:          "query",
		Required:    required,
		Description: description,
		Schema:      &Schema{Type: typ},
	}
}
