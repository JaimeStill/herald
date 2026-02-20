package storage

import (
	"fmt"
	"os"
)

// Config holds Azure Blob Storage connection parameters.
type Config struct {
	ContainerName    string `toml:"container_name"`
	ConnectionString string `toml:"connection_string"`
}

// Env maps config fields to environment variable names for override injection.
type Env struct {
	ContainerName    string
	ConnectionString string
}

// Finalize applies defaults, environment variable overrides, and validation.
func (c *Config) Finalize(env *Env) error {
	c.loadDefaults()
	if env != nil {
		c.loadEnv(env)
	}
	return c.validate()
}

// Merge overwrites non-zero fields from overlay.
func (c *Config) Merge(overlay *Config) {
	if overlay.ContainerName != "" {
		c.ContainerName = overlay.ContainerName
	}
	if overlay.ConnectionString != "" {
		c.ConnectionString = overlay.ConnectionString
	}
}

func (c *Config) loadDefaults() {
	if c.ContainerName == "" {
		c.ContainerName = "documents"
	}
}

func (c *Config) loadEnv(env *Env) {
	if env.ContainerName != "" {
		if v := os.Getenv(env.ContainerName); v != "" {
			c.ContainerName = v
		}
	}
	if env.ConnectionString != "" {
		if v := os.Getenv(env.ConnectionString); v != "" {
			c.ConnectionString = v
		}
	}
}

func (c *Config) validate() error {
	if c.ContainerName == "" {
		return fmt.Errorf("container_name required")
	}
	if c.ConnectionString == "" {
		return fmt.Errorf("connection_string required")
	}
	return nil
}
