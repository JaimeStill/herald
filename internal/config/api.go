package config

import (
	"fmt"
	"os"

	"github.com/JaimeStill/herald/pkg/formatting"
	"github.com/JaimeStill/herald/pkg/middleware"
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

var paginationEnv = &pagination.ConfigEnv{
	DefaultPageSize: "HERALD_PAGINATION_DEFAULT_PAGE_SIZE",
	MaxPageSize:     "HERALD_PAGINATION_MAX_PAGE_SIZE",
}

// APIConfig holds API routing, CORS, and pagination settings.
type APIConfig struct {
	BasePath      string                `toml:"base_path"`
	MaxUploadSize string                `toml:"max_upload_size"`
	CORS          middleware.CORSConfig `toml:"cors"`
	Pagination    pagination.Config     `toml:"pagination"`
}

func (c *APIConfig) MaxUploadSizeBytes() int64 {
	size, err := formatting.ParseBytes(c.MaxUploadSize)
	if err != nil {
		return 50 * 1024 * 1024 // 50MB fallback
	}
	return size
}

// Finalize applies defaults, environment variable overrides, and validation
// for the API config and its nested CORS and pagination configs.
func (c *APIConfig) Finalize() error {
	c.loadDefaults()
	c.loadEnv()

	if err := c.CORS.Finalize(corsEnv); err != nil {
		return fmt.Errorf("cors: %w", err)
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
	if overlay.MaxUploadSize != "" {
		c.MaxUploadSize = overlay.MaxUploadSize
	}

	c.CORS.Merge(&overlay.CORS)
	c.Pagination.Merge(&overlay.Pagination)
}

func (c *APIConfig) loadDefaults() {
	if c.BasePath == "" {
		c.BasePath = "/api"
	}
	if c.MaxUploadSize == "" {
		c.MaxUploadSize = "50MB"
	}
}

func (c *APIConfig) loadEnv() {
	if v := os.Getenv("HERALD_API_BASE_PATH"); v != "" {
		c.BasePath = v
	}
	if v := os.Getenv("HERALD_API_MAX_UPLOAD_SIZE"); v != "" {
		c.MaxUploadSize = v
	}
}
