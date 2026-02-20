package database

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds PostgreSQL connection parameters.
type Config struct {
	Host            string `toml:"host"`
	Port            int    `toml:"port"`
	Name            string `toml:"name"`
	User            string `toml:"user"`
	Password        string `toml:"password"`
	SSLMode         string `toml:"ssl_mode"`
	MaxOpenConns    int    `toml:"max_open_conns"`
	MaxIdleConns    int    `toml:"max_idle_conns"`
	ConnMaxLifetime string `toml:"conn_max_lifetime"`
	ConnTimeout     string `toml:"conn_timeout"`
}

// Env maps config fields to environment variable names for override injection.
type Env struct {
	Host            string
	Port            string
	Name            string
	User            string
	Password        string
	SSLMode         string
	MaxOpenConns    string
	MaxIdleConns    string
	ConnMaxLifetime string
	ConnTimeout     string
}

// ConnMaxLifetimeDuration returns ConnMaxLifetime as a time.Duration.
func (c *Config) ConnMaxLifetimeDuration() time.Duration {
	d, _ := time.ParseDuration(c.ConnMaxLifetime)
	return d
}

// ConnTimeoutDuration returns ConnTimeout as a time.Duration.
func (c *Config) ConnTimeoutDuration() time.Duration {
	d, _ := time.ParseDuration(c.ConnTimeout)
	return d
}

// Dsn returns a PostgreSQL connection string.
func (c *Config) Dsn() string {
	return fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		c.Host, c.Port, c.Name, c.User, c.Password, c.SSLMode,
	)
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
	if overlay.Host != "" {
		c.Host = overlay.Host
	}
	if overlay.Port != 0 {
		c.Port = overlay.Port
	}
	if overlay.Name != "" {
		c.Name = overlay.Name
	}
	if overlay.User != "" {
		c.User = overlay.User
	}
	if overlay.Password != "" {
		c.Password = overlay.Password
	}
	if overlay.SSLMode != "" {
		c.SSLMode = overlay.SSLMode
	}
	if overlay.MaxOpenConns != 0 {
		c.MaxOpenConns = overlay.MaxOpenConns
	}
	if overlay.MaxIdleConns != 0 {
		c.MaxIdleConns = overlay.MaxIdleConns
	}
	if overlay.ConnMaxLifetime != "" {
		c.ConnMaxLifetime = overlay.ConnMaxLifetime
	}
	if overlay.ConnTimeout != "" {
		c.ConnTimeout = overlay.ConnTimeout
	}
}

func (c *Config) loadDefaults() {
	if c.Host == "" {
		c.Host = "localhost"
	}
	if c.Port == 0 {
		c.Port = 5432
	}
	if c.SSLMode == "" {
		c.SSLMode = "disable"
	}
	if c.MaxOpenConns == 0 {
		c.MaxOpenConns = 25
	}
	if c.MaxIdleConns == 0 {
		c.MaxIdleConns = 5
	}
	if c.ConnMaxLifetime == "" {
		c.ConnMaxLifetime = "15m"
	}
	if c.ConnTimeout == "" {
		c.ConnTimeout = "5s"
	}
}

func (c *Config) loadEnv(env *Env) {
	if env.Host != "" {
		if v := os.Getenv(env.Host); v != "" {
			c.Host = v
		}
	}
	if env.Port != "" {
		if v := os.Getenv(env.Port); v != "" {
			if port, err := strconv.Atoi(v); err == nil {
				c.Port = port
			}
		}
	}
	if env.Name != "" {
		if v := os.Getenv(env.Name); v != "" {
			c.Name = v
		}
	}
	if env.User != "" {
		if v := os.Getenv(env.User); v != "" {
			c.User = v
		}
	}
	if env.Password != "" {
		if v := os.Getenv(env.Password); v != "" {
			c.Password = v
		}
	}
	if env.SSLMode != "" {
		if v := os.Getenv(env.SSLMode); v != "" {
			c.SSLMode = v
		}
	}
	if env.MaxOpenConns != "" {
		if v := os.Getenv(env.MaxOpenConns); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				c.MaxOpenConns = n
			}
		}
	}
	if env.MaxIdleConns != "" {
		if v := os.Getenv(env.MaxIdleConns); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				c.MaxIdleConns = n
			}
		}
	}
	if env.ConnMaxLifetime != "" {
		if v := os.Getenv(env.ConnMaxLifetime); v != "" {
			c.ConnMaxLifetime = v
		}
	}
	if env.ConnTimeout != "" {
		if v := os.Getenv(env.ConnTimeout); v != "" {
			c.ConnTimeout = v
		}
	}
}

func (c *Config) validate() error {
	if c.Name == "" {
		return fmt.Errorf("name required")
	}
	if c.User == "" {
		return fmt.Errorf("user required")
	}
	if _, err := time.ParseDuration(c.ConnMaxLifetime); err != nil {
		return fmt.Errorf("invalid conn_max_lifetime: %w", err)
	}
	if _, err := time.ParseDuration(c.ConnTimeout); err != nil {
		return fmt.Errorf("invalid conn_timeout: %w", err)
	}
	return nil
}
