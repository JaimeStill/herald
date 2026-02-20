package config

import (
	"fmt"
	"os"
	"time"

	"github.com/JaimeStill/herald/pkg/database"
	"github.com/JaimeStill/herald/pkg/storage"
	"github.com/pelletier/go-toml/v2"
)

const (
	BaseConfigFile       = "config.toml"
	OverlayConfigPattern = "config.%s.toml"

	EnvHeraldEnv             = "HERALD_ENV"
	EnvHeraldShutdownTimeout = "HERALD_SHUTDOWN_TIMEOUT"
	EnvHeraldVersion         = "HERALD_VERSION"
)

var databaseEnv = &database.Env{
	Host:            "HERALD_DB_HOST",
	Port:            "HERALD_DB_PORT",
	Name:            "HERALD_DB_NAME",
	User:            "HERALD_DB_USER",
	Password:        "HERALD_DB_PASSWORD",
	SSLMode:         "HERALD_DB_SSL_MODE",
	MaxOpenConns:    "HERALD_DB_MAX_OPEN_CONNS",
	MaxIdleConns:    "HERALD_DB_MAX_IDLE_CONNS",
	ConnMaxLifetime: "HERALD_DB_CONN_MAX_LIFETIME",
	ConnTimeout:     "HERALD_DB_CONN_TIMEOUT",
}

var storageEnv = &storage.Env{
	ContainerName:    "HERALD_STORAGE_CONTAINER_NAME",
	ConnectionString: "HERALD_STORAGE_CONNECTION_STRING",
}

// Config is the root configuration for the Herald service.
type Config struct {
	Server          ServerConfig    `toml:"server"`
	Database        database.Config `toml:"database"`
	Storage         storage.Config  `toml:"storage"`
	API             APIConfig       `toml:"api"`
	ShutdownTimeout string          `toml:"shutdown_timeout"`
	Version         string          `toml:"version"`
}

// Env returns the HERALD_ENV value, defaulting to "local".
func (c *Config) Env() string {
	if env := os.Getenv(EnvHeraldEnv); env != "" {
		return env
	}
	return "local"
}

// ShutdownTimeoutDuration returns ShutdownTimeout as a time.Duration.
func (c *Config) ShutdownTimeoutDuration() time.Duration {
	d, _ := time.ParseDuration(c.ShutdownTimeout)
	return d
}

// Load reads the base config (if present), applies any environment overlay,
// and finalizes all values. If no config.toml exists, defaults and environment
// variables provide all configuration.
func Load() (*Config, error) {
	cfg := &Config{}

	if _, err := os.Stat(BaseConfigFile); err == nil {
		loaded, err := load(BaseConfigFile)
		if err != nil {
			return nil, err
		}
		cfg = loaded
	}

	if path := overlayPath(); path != "" {
		overlay, err := load(path)
		if err != nil {
			return nil, fmt.Errorf("load overlay %s: %w", path, err)
		}
		cfg.Merge(overlay)
	}

	if err := cfg.finalize(); err != nil {
		return nil, fmt.Errorf("finalize config: %w", err)
	}

	return cfg, nil
}

// Merge overwrites non-zero fields from overlay across all sub-configs.
func (c *Config) Merge(overlay *Config) {
	if overlay.ShutdownTimeout != "" {
		c.ShutdownTimeout = overlay.ShutdownTimeout
	}
	if overlay.Version != "" {
		c.Version = overlay.Version
	}
	c.Server.Merge(&overlay.Server)
	c.Database.Merge(&overlay.Database)
	c.Storage.Merge(&overlay.Storage)
	c.API.Merge(&overlay.API)
}

func (c *Config) finalize() error {
	c.loadDefaults()
	c.loadEnv()

	if err := c.validate(); err != nil {
		return err
	}
	if err := c.Server.Finalize(); err != nil {
		return fmt.Errorf("server: %w", err)
	}
	if err := c.Database.Finalize(databaseEnv); err != nil {
		return fmt.Errorf("database: %w", err)
	}
	if err := c.Storage.Finalize(storageEnv); err != nil {
		return fmt.Errorf("storage: %w", err)
	}
	if err := c.API.Finalize(); err != nil {
		return fmt.Errorf("api: %w", err)
	}
	return nil
}
func (c *Config) loadDefaults() {
	if c.ShutdownTimeout == "" {
		c.ShutdownTimeout = "30s"
	}
	if c.Version == "" {
		c.Version = "0.1.0"
	}
}

func (c *Config) loadEnv() {
	if v := os.Getenv(EnvHeraldShutdownTimeout); v != "" {
		c.ShutdownTimeout = v
	}
	if v := os.Getenv(EnvHeraldVersion); v != "" {
		c.Version = v
	}
}

func (c *Config) validate() error {
	if _, err := time.ParseDuration(c.ShutdownTimeout); err != nil {
		return fmt.Errorf("invalid shutdown_timeout: %w", err)
	}
	return nil
}

func load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}

func overlayPath() string {
	if env := os.Getenv(EnvHeraldEnv); env != "" {
		path := fmt.Sprintf(OverlayConfigPattern, env)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}
