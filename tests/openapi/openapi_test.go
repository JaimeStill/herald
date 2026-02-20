package openapi_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/JaimeStill/herald/pkg/openapi"
)

func TestNewSpec(t *testing.T) {
	spec := openapi.NewSpec("Test API", "1.0.0")

	if spec.OpenAPI != "3.1.0" {
		t.Errorf("openapi version: got %s, want 3.1.0", spec.OpenAPI)
	}
	if spec.Info.Title != "Test API" {
		t.Errorf("title: got %s, want Test API", spec.Info.Title)
	}
	if spec.Info.Version != "1.0.0" {
		t.Errorf("version: got %s, want 1.0.0", spec.Info.Version)
	}
	if spec.Components == nil {
		t.Fatal("components should not be nil")
	}
	if spec.Paths == nil {
		t.Fatal("paths should not be nil")
	}
}

func TestAddServer(t *testing.T) {
	spec := openapi.NewSpec("Test", "1.0.0")
	spec.AddServer("http://localhost:8080")

	if len(spec.Servers) != 1 {
		t.Fatalf("servers: got %d, want 1", len(spec.Servers))
	}
	if spec.Servers[0].URL != "http://localhost:8080" {
		t.Errorf("server url: got %s", spec.Servers[0].URL)
	}
}

func TestSetDescription(t *testing.T) {
	spec := openapi.NewSpec("Test", "1.0.0")
	spec.SetDescription("A test API")

	if spec.Info.Description != "A test API" {
		t.Errorf("description: got %s", spec.Info.Description)
	}
}

func TestSchemaRef(t *testing.T) {
	ref := openapi.SchemaRef("Document")
	expected := "#/components/schemas/Document"

	if ref.Ref != expected {
		t.Errorf("ref: got %s, want %s", ref.Ref, expected)
	}
}

func TestResponseRef(t *testing.T) {
	ref := openapi.ResponseRef("NotFound")
	expected := "#/components/responses/NotFound"

	if ref.Ref != expected {
		t.Errorf("ref: got %s, want %s", ref.Ref, expected)
	}
}

func TestRequestBodyJSON(t *testing.T) {
	rb := openapi.RequestBodyJSON("CreateDoc", true)

	if !rb.Required {
		t.Error("required should be true")
	}
	ct, ok := rb.Content["application/json"]
	if !ok {
		t.Fatal("missing application/json content type")
	}
	if ct.Schema.Ref != "#/components/schemas/CreateDoc" {
		t.Errorf("schema ref: got %s", ct.Schema.Ref)
	}
}

func TestResponseJSON(t *testing.T) {
	resp := openapi.ResponseJSON("Success", "Document")

	if resp.Description != "Success" {
		t.Errorf("description: got %s", resp.Description)
	}
	ct, ok := resp.Content["application/json"]
	if !ok {
		t.Fatal("missing application/json content type")
	}
	if ct.Schema.Ref != "#/components/schemas/Document" {
		t.Errorf("schema ref: got %s", ct.Schema.Ref)
	}
}

func TestPathParam(t *testing.T) {
	p := openapi.PathParam("id", "Document ID")

	if p.Name != "id" {
		t.Errorf("name: got %s", p.Name)
	}
	if p.In != "path" {
		t.Errorf("in: got %s", p.In)
	}
	if !p.Required {
		t.Error("path params should be required")
	}
	if p.Schema.Type != "string" || p.Schema.Format != "uuid" {
		t.Errorf("schema: got type=%s format=%s", p.Schema.Type, p.Schema.Format)
	}
}

func TestQueryParam(t *testing.T) {
	p := openapi.QueryParam("search", "string", "Search query", false)

	if p.Name != "search" {
		t.Errorf("name: got %s", p.Name)
	}
	if p.In != "query" {
		t.Errorf("in: got %s", p.In)
	}
	if p.Required {
		t.Error("should not be required")
	}
	if p.Schema.Type != "string" {
		t.Errorf("schema type: got %s", p.Schema.Type)
	}
}

func TestNewComponentsDefaults(t *testing.T) {
	c := openapi.NewComponents()

	schemas := []string{"PageRequest"}
	for _, name := range schemas {
		if _, ok := c.Schemas[name]; !ok {
			t.Errorf("missing default schema: %s", name)
		}
	}

	responses := []string{"BadRequest", "NotFound", "Conflict"}
	for _, name := range responses {
		if _, ok := c.Responses[name]; !ok {
			t.Errorf("missing default response: %s", name)
		}
	}
}

func TestAddSchemas(t *testing.T) {
	c := openapi.NewComponents()
	c.AddSchemas(map[string]*openapi.Schema{
		"Document": {Type: "object"},
	})

	if _, ok := c.Schemas["Document"]; !ok {
		t.Error("Document schema not added")
	}
	if _, ok := c.Schemas["PageRequest"]; !ok {
		t.Error("default PageRequest schema should still exist")
	}
}

func TestAddResponses(t *testing.T) {
	c := openapi.NewComponents()
	c.AddResponses(map[string]*openapi.Response{
		"Unauthorized": {Description: "Not authenticated"},
	})

	if _, ok := c.Responses["Unauthorized"]; !ok {
		t.Error("Unauthorized response not added")
	}
	if _, ok := c.Responses["BadRequest"]; !ok {
		t.Error("default BadRequest response should still exist")
	}
}

func TestMarshalJSON(t *testing.T) {
	spec := openapi.NewSpec("Test", "1.0.0")
	data, err := openapi.MarshalJSON(spec)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if parsed["openapi"] != "3.1.0" {
		t.Errorf("openapi: got %v", parsed["openapi"])
	}
}

func TestWriteJSON(t *testing.T) {
	spec := openapi.NewSpec("Test", "1.0.0")
	path := filepath.Join(t.TempDir(), "spec.json")

	if err := openapi.WriteJSON(spec, path); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if parsed["openapi"] != "3.1.0" {
		t.Errorf("openapi: got %v", parsed["openapi"])
	}
}

func TestServeSpec(t *testing.T) {
	spec := openapi.NewSpec("Test", "1.0.0")
	data, _ := openapi.MarshalJSON(spec)

	handler := openapi.ServeSpec(data)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/openapi.json", nil)

	handler(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want 200", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("content-type: got %s", ct)
	}

	body, _ := io.ReadAll(res.Body)
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("body unmarshal failed: %v", err)
	}
}

func TestConfigFinalizeDefaults(t *testing.T) {
	cfg := openapi.Config{}
	if err := cfg.Finalize(nil); err != nil {
		t.Fatalf("finalize failed: %v", err)
	}

	if cfg.Title != "Herald API" {
		t.Errorf("title: got %s, want Herald API", cfg.Title)
	}
	if cfg.Description != "Security marking classification service for DoD PDF documents." {
		t.Errorf("description: got %s", cfg.Description)
	}
}

func TestConfigFinalizeEnv(t *testing.T) {
	t.Setenv("TEST_TITLE", "Custom API")
	t.Setenv("TEST_DESC", "Custom desc")

	env := &openapi.ConfigEnv{
		Title:       "TEST_TITLE",
		Description: "TEST_DESC",
	}

	cfg := openapi.Config{}
	if err := cfg.Finalize(env); err != nil {
		t.Fatalf("finalize failed: %v", err)
	}

	if cfg.Title != "Custom API" {
		t.Errorf("title: got %s, want Custom API", cfg.Title)
	}
	if cfg.Description != "Custom desc" {
		t.Errorf("description: got %s, want Custom desc", cfg.Description)
	}
}

func TestConfigMerge(t *testing.T) {
	base := openapi.Config{Title: "Base"}
	overlay := openapi.Config{Title: "Overlay"}
	base.Merge(&overlay)

	if base.Title != "Overlay" {
		t.Errorf("title: got %s, want Overlay", base.Title)
	}
}
