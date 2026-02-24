package storage

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds Azure Blob Storage connection parameters.
type Config struct {
	ContainerName    string `json:"container_name"`
	ConnectionString string `json:"connection_string"`
	MaxListSize      int32  `json:"max_list_size"`
}

// Env maps config fields to environment variable names for override injection.
type Env struct {
	ContainerName    string
	ConnectionString string
	MaxListSize      string
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
	if overlay.MaxListSize != 0 {
		c.MaxListSize = overlay.MaxListSize
	}
}

func (c *Config) loadDefaults() {
	if c.ContainerName == "" {
		c.ContainerName = "documents"
	}
	if c.MaxListSize == 0 {
		c.MaxListSize = 50
	}
	if c.MaxListSize > MaxListCap {
		c.MaxListSize = MaxListCap
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
	if env.MaxListSize != "" {
		if v := os.Getenv(env.MaxListSize); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				c.MaxListSize = min(int32(n), MaxListCap)
			}
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
