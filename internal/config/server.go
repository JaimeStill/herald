package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	EnvServerHost            = "HERALD_SERVER_HOST"
	EnvServerPort            = "HERALD_SERVER_PORT"
	EnvServerReadTimeout     = "HERALD_SERVER_READ_TIMEOUT"
	EnvServerWriteTimeout    = "HERALD_SERVER_WRITE_TIMEOUT"
	EnvServerShutdownTimeout = "HERALD_SERVER_SHUTDOWN_TIMEOUT"
)

// ServerConfig holds HTTP server parameters.
type ServerConfig struct {
	Host            string `toml:"host"`
	Port            int    `toml:"port"`
	ReadTimeout     string `toml:"read_timeout"`
	WriteTimeout    string `toml:"write_timeout"`
	ShutdownTimeout string `toml:"shutdown_timeout"`
}

// Addr returns the host:port listen address.
func (c *ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// ReadTimeoutDuration returns ReadTimeout as a time.Duration.
func (c *ServerConfig) ReadTimeoutDuration() time.Duration {
	d, _ := time.ParseDuration(c.ReadTimeout)
	return d
}

// WriteTimeoutDuration returns WriteTimeout as a time.Duration.
func (c *ServerConfig) WriteTimeoutDuration() time.Duration {
	d, _ := time.ParseDuration(c.WriteTimeout)
	return d
}

// ShutdownTimeoutDuration returns ShutdownTimeout as a time.Duration.
func (c *ServerConfig) ShutdownTimeoutDuration() time.Duration {
	d, _ := time.ParseDuration(c.ShutdownTimeout)
	return d
}

// Finalize applies defaults, environment variable overrides, and validation.
func (c *ServerConfig) Finalize() error {
	c.loadDefaults()
	c.loadEnv()
	return c.validate()
}

// Merge overwrites non-zero fields from overlay.
func (c *ServerConfig) Merge(overlay *ServerConfig) {
	if overlay.Host != "" {
		c.Host = overlay.Host
	}
	if overlay.Port != 0 {
		c.Port = overlay.Port
	}
	if overlay.ReadTimeout != "" {
		c.ReadTimeout = overlay.ReadTimeout
	}
	if overlay.WriteTimeout != "" {
		c.WriteTimeout = overlay.WriteTimeout
	}
	if overlay.ShutdownTimeout != "" {
		c.ShutdownTimeout = overlay.ShutdownTimeout
	}
}

func (c *ServerConfig) loadDefaults() {
	if c.Host == "" {
		c.Host = "0.0.0.0"
	}
	if c.Port == 0 {
		c.Port = 8080
	}
	if c.ReadTimeout == "" {
		c.ReadTimeout = "1m"
	}
	if c.WriteTimeout == "" {
		c.WriteTimeout = "15m"
	}
	if c.ShutdownTimeout == "" {
		c.ShutdownTimeout = "30s"
	}
}

func (c *ServerConfig) loadEnv() {
	if v := os.Getenv(EnvServerHost); v != "" {
		c.Host = v
	}
	if v := os.Getenv(EnvServerPort); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.Port = port
		}
	}
	if v := os.Getenv(EnvServerReadTimeout); v != "" {
		c.ReadTimeout = v
	}
	if v := os.Getenv(EnvServerWriteTimeout); v != "" {
		c.WriteTimeout = v
	}
	if v := os.Getenv(EnvServerShutdownTimeout); v != "" {
		c.ShutdownTimeout = v
	}
}

func (c *ServerConfig) validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}
	if _, err := time.ParseDuration(c.ReadTimeout); err != nil {
		return fmt.Errorf("invalid read_timeout: %w", err)
	}
	if _, err := time.ParseDuration(c.WriteTimeout); err != nil {
		return fmt.Errorf("invalid write_timeout: %w", err)
	}
	if _, err := time.ParseDuration(c.ShutdownTimeout); err != nil {
		return fmt.Errorf("invalid shutdown_timeout: %w", err)
	}
	return nil
}
