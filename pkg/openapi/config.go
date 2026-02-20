package openapi

import "os"

// Config holds OpenAPI metadata for spec generation.
type Config struct {
	Title       string `toml:"title"`
	Description string `toml:"description"`
}

// ConfigEnv maps config fields to environment variable names for override injection.
type ConfigEnv struct {
	Title       string
	Description string
}

// Finalize applies defaults and environment variable overrides.
func (c *Config) Finalize(env *ConfigEnv) error {
	c.loadDefaults()
	if env != nil {
		c.loadEnv(env)
	}
	return nil
}

// Merge overwrites non-zero fields from overlay.
func (c *Config) Merge(overlay *Config) {
	if overlay.Title != "" {
		c.Title = overlay.Title
	}
	if overlay.Description != "" {
		c.Description = overlay.Description
	}
}

func (c *Config) loadDefaults() {
	if c.Title == "" {
		c.Title = "Herald API"
	}
	if c.Description == "" {
		c.Description = "Security marking classification service for DoD PDF documents."
	}
}

func (c *Config) loadEnv(env *ConfigEnv) {
	if env.Title != "" {
		if v := os.Getenv(env.Title); v != "" {
			c.Title = v
		}
	}
	if env.Description != "" {
		if v := os.Getenv(env.Description); v != "" {
			c.Description = v
		}
	}
}
