package web_test

import (
	"testing"

	"github.com/JaimeStill/herald/pkg/web"
)

func TestViewDefFields(t *testing.T) {
	view := web.ViewDef{
		Route:    "/",
		Template: "index.html",
		Title:    "Home",
		Bundle:   "app",
	}

	if view.Route != "/" {
		t.Errorf("Route: got %q, want %q", view.Route, "/")
	}
	if view.Template != "index.html" {
		t.Errorf("Template: got %q, want %q", view.Template, "index.html")
	}
	if view.Title != "Home" {
		t.Errorf("Title: got %q, want %q", view.Title, "Home")
	}
	if view.Bundle != "app" {
		t.Errorf("Bundle: got %q, want %q", view.Bundle, "app")
	}
}

func TestViewDataFields(t *testing.T) {
	data := web.ViewData{
		Title:    "Test",
		Bundle:   "app",
		BasePath: "/app",
		Data:     map[string]string{"key": "value"},
	}

	if data.Title != "Test" {
		t.Errorf("Title: got %q, want %q", data.Title, "Test")
	}
	if data.BasePath != "/app" {
		t.Errorf("BasePath: got %q, want %q", data.BasePath, "/app")
	}
	if data.Bundle != "app" {
		t.Errorf("Bundle: got %q, want %q", data.Bundle, "app")
	}
	if data.Data == nil {
		t.Error("Data: got nil, want non-nil")
	}
}
