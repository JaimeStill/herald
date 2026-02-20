package config

import (
	"fmt"
	"os"

	"github.com/JaimeStill/herald/pkg/middleware"
	"github.com/JaimeStill/herald/pkg/openapi"
	"github.com/JaimeStill/herald/pkg/pagination"
)

var corsEnv = &middleware.CORSEnv{
	Enabled:          "HERALD_CORS_ENABLED",
	Origins:          "HERALD_CORS_ORIGINS",
	AllowedMethods:   "HERALD_CORS_ALLOWED_METHODS",
	AllowedHeaders:   "HERALD_CORS_ALLOWED_HEADERS",
	AllowCredentials: "HERALD_CORS_ALLOW_CREDENTIALS",
	MaxAge:           "HERALD_CORS_MAX_AGE",
}

var openAPIEnv = &openapi.ConfigEnv{
	Title:       "HERALD_OPENAPI_TITLE",
	Description: "HERALD_OPENAPI_DESCRIPTION",
}

var paginationEnv = &pagination.ConfigEnv{
	DefaultPageSize: "HERALD_PAGINATION_DEFAULT_PAGE_SIZE",
	MaxPageSize:     "HERALD_PAGINATION_MAX_PAGE_SIZE",
}

// APIConfig holds API routing, CORS, and OpenAPI settings.
type APIConfig struct {
	BasePath   string                `toml:"base_path"`
	CORS       middleware.CORSConfig `toml:"cors"`
	OpenAPI    openapi.Config        `toml:"openapi"`
	Pagination pagination.Config     `toml:"pagination"`
}

// Finalize applies defaults, environment variable overrides, and validation
// for the API config and its nested CORS and OpenAPI configs.
func (c *APIConfig) Finalize() error {
	c.loadDefaults()
	c.loadEnv()

	if err := c.CORS.Finalize(corsEnv); err != nil {
		return fmt.Errorf("cors: %w", err)
	}
	if err := c.OpenAPI.Finalize(openAPIEnv); err != nil {
		return fmt.Errorf("openapi: %w", err)
	}
	if err := c.Pagination.Finalize(paginationEnv); err != nil {
		return fmt.Errorf("pagination: %w", err)
	}
	return nil
}

// Merge overwrites non-zero fields from overlay across nested configs.
func (c *APIConfig) Merge(overlay *APIConfig) {
	if overlay.BasePath != "" {
		c.BasePath = overlay.BasePath
	}
	c.CORS.Merge(&overlay.CORS)
	c.OpenAPI.Merge(&overlay.OpenAPI)
	c.Pagination.Merge(&overlay.Pagination)
}

func (c *APIConfig) loadDefaults() {
	if c.BasePath == "" {
		c.BasePath = "/api"
	}
}

func (c *APIConfig) loadEnv() {
	if v := os.Getenv("HERALD_API_BASE_PATH"); v != "" {
		c.BasePath = v
	}
}
