// Package pagination provides types and utilities for paginated data queries.
package pagination

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds pagination settings including page size limits.
type Config struct {
	DefaultPageSize int `json:"default_page_size"`
	MaxPageSize     int `json:"max_page_size"`
}

// ConfigEnv maps environment variable names for pagination configuration.
type ConfigEnv struct {
	DefaultPageSize string
	MaxPageSize     string
}

// Finalize applies defaults, environment variable overrides, and validation.
func (c *Config) Finalize(env *ConfigEnv) error {
	c.loadDefaults()
	if env != nil {
		c.loadEnv(env)
	}
	return c.validate()
}

// Merge applies non-zero values from the overlay configuration.
func (c *Config) Merge(overlay *Config) {
	if overlay.DefaultPageSize != 0 {
		c.DefaultPageSize = overlay.DefaultPageSize
	}
	if overlay.MaxPageSize != 0 {
		c.MaxPageSize = overlay.MaxPageSize
	}
}

func (c *Config) loadDefaults() {
	if c.DefaultPageSize <= 0 {
		c.DefaultPageSize = 20
	}
	if c.MaxPageSize <= 0 {
		c.MaxPageSize = 100
	}
}

func (c *Config) loadEnv(env *ConfigEnv) {
	if env.DefaultPageSize != "" {
		if v := os.Getenv(env.DefaultPageSize); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				c.DefaultPageSize = n
			}
		}
	}
	if env.MaxPageSize != "" {
		if v := os.Getenv(env.MaxPageSize); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				c.MaxPageSize = n
			}
		}
	}
}

func (c *Config) validate() error {
	if c.DefaultPageSize < 1 {
		return fmt.Errorf("default_page_size must be positive")
	}
	if c.MaxPageSize < 1 {
		return fmt.Errorf("max_page_size must be positive")
	}
	if c.DefaultPageSize > c.MaxPageSize {
		return fmt.Errorf("default_page_size cannot exceed max_page_size")
	}
	return nil
}
