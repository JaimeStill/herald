# 4 - Project Scaffolding, Configuration, Lifecycle, and HTTP Infrastructure

## Problem Context

Herald has zero Go source code. This task establishes the Go module, project structure, build tooling, local development environment, and implements the foundational packages: configuration, lifecycle coordination, OpenAPI spec infrastructure, and HTTP routing/middleware. All patterns adapted from agent-lab.

## Architecture Approach

Layered Composition Architecture (LCA) adapted from agent-lab. Each config struct follows the three-phase finalize pattern (loadDefaults → loadEnv → validate) with Env struct injection for env var names. HTTP routing uses a two-tier module system: top-level Router dispatches by path prefix, Module strips prefix and delegates to inner ServeMux. OpenAPI specs are built programmatically at startup and served as pre-serialized JSON.

## Implementation

### Step 1: Project Scaffolding

Terminal commands:

```bash
cd /home/jaime/code/herald
go mod init github.com/JaimeStill/herald
go get github.com/pelletier/go-toml/v2
```

**`.gitignore`**

```gitignore
# Binaries
/bin/
*.exe
*.dll
*.so
*.dylib

# Test
*.test
*.out
coverage*

# Go workspace
go.work
go.work.sum

# Vendor
vendor/

# IDE
.vscode/
.idea/
*.swp
*.swo
*~

# OS
.DS_Store
Thumbs.db

# Temporary
tmp/
temp/
*.log
```

**doc.go stubs** — create one file per directory, each containing only `package <name>`:

| File | Content |
|------|---------|
| `internal/api/doc.go` | `package api` |
| `internal/infrastructure/doc.go` | `package infrastructure` |
| `internal/documents/doc.go` | `package documents` |
| `internal/classifications/doc.go` | `package classifications` |
| `internal/prompts/doc.go` | `package prompts` |
| `workflow/doc.go` | `package workflow` |
| `pkg/pagination/doc.go` | `package pagination` |
| `pkg/query/doc.go` | `package query` |
| `pkg/repository/doc.go` | `package repository` |
| `pkg/web/doc.go` | `package web` |

**`cmd/migrate/main.go`** — stub entry point:

```go
package main

func main() {}
```

### Step 2: Build Tooling

**`.mise.toml`**

```toml
[tasks.web]
description = "Build web assets"
run = "cd web && bun install && bun run build"

[tasks.dev]
description = "Build web assets and run the server"
depends = ["web"]
run = "go run ./cmd/server/"

[tasks.build]
description = "Build web assets and compile server binary"
depends = ["web"]
run = "go build -o bin/server ./cmd/server"

[tasks.test]
description = "Run all tests"
run = "go test ./tests/..."

[tasks.vet]
description = "Run go vet"
run = "go vet ./..."
```

**`docker-compose.yml`**

```yaml
include:
  - compose/postgres.yml
  - compose/azurite.yml
```

**`compose/postgres.yml`**

```yaml
services:
  postgres:
    image: postgres:17
    container_name: herald-postgres
    environment:
      POSTGRES_DB: ${POSTGRES_DB:-herald}
      POSTGRES_USER: ${POSTGRES_USER:-herald}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-herald}
    volumes:
      - herald-postgres:/var/lib/postgresql/data
    ports:
      - "${POSTGRES_PORT:-5432}:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER:-herald}"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - herald

volumes:
  herald-postgres:
    driver: local
    driver_opts:
      type: none
      o: bind
      device: ${HOME}/storage/herald/postgres

networks:
  herald:
    name: herald
    driver: bridge
```

**`compose/azurite.yml`**

```yaml
services:
  azurite:
    image: mcr.microsoft.com/azure-storage/azurite
    container_name: herald-azurite
    environment:
      AZURITE_ACCOUNTS: "heraldstore:Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw=="
    ports:
      - "${AZURITE_BLOB_PORT:-10000}:10000"
      - "${AZURITE_QUEUE_PORT:-10001}:10001"
      - "${AZURITE_TABLE_PORT:-10002}:10002"
    volumes:
      - herald-azurite:/data
    command: "azurite --blobHost 0.0.0.0 --queueHost 0.0.0.0 --tableHost 0.0.0.0"
    healthcheck:
      test: ["CMD", "nc", "-z", "127.0.0.1", "10000"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - herald

volumes:
  herald-azurite:
    driver: local
    driver_opts:
      type: none
      o: bind
      device: ${HOME}/storage/herald/azurite

networks:
  herald:
    name: herald
    external: true
```

**`config.toml`**

```toml
shutdown_timeout = "30s"
version = "0.1.0"

[server]
host = "0.0.0.0"
port = 8080
read_timeout = "1m"
write_timeout = "15m"
shutdown_timeout = "30s"

[database]
host = "localhost"
port = 5432
name = "herald"
user = "herald"
password = "herald"
ssl_mode = "disable"
max_open_conns = 25
max_idle_conns = 5
conn_max_lifetime = "15m"
conn_timeout = "5s"

[storage]
container_name = "documents"
connection_string = "DefaultEndpointsProtocol=http;AccountName=heraldstore;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;BlobEndpoint=http://127.0.0.1:10000/heraldstore;"

[api]
base_path = "/api"

[api.cors]
enabled = false
origins = []
allowed_methods = ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
allowed_headers = ["Content-Type", "Authorization"]
allow_credentials = false
max_age = 3600

[api.openapi]
title = "Herald API"
description = "Security marking classification service for DoD PDF documents."
```

### Step 3: Package Config Types

**`pkg/database/config.go`**

```go
package database

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

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

func (c *Config) ConnMaxLifetimeDuration() time.Duration {
	d, _ := time.ParseDuration(c.ConnMaxLifetime)
	return d
}

func (c *Config) ConnTimeoutDuration() time.Duration {
	d, _ := time.ParseDuration(c.ConnTimeout)
	return d
}

func (c *Config) Dsn() string {
	return fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		c.Host, c.Port, c.Name, c.User, c.Password, c.SSLMode,
	)
}

func (c *Config) Finalize(env *Env) error {
	c.loadDefaults()
	if env != nil {
		c.loadEnv(env)
	}
	return c.validate()
}

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
```

**`pkg/storage/config.go`**

```go
package storage

import (
	"fmt"
	"os"
)

type Config struct {
	ContainerName    string `toml:"container_name"`
	ConnectionString string `toml:"connection_string"`
}

type Env struct {
	ContainerName    string
	ConnectionString string
}

func (c *Config) Finalize(env *Env) error {
	c.loadDefaults()
	if env != nil {
		c.loadEnv(env)
	}
	return c.validate()
}

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
```

**`pkg/openapi/config.go`**

```go
package openapi

import "os"

type Config struct {
	Title       string `toml:"title"`
	Description string `toml:"description"`
}

type ConfigEnv struct {
	Title       string
	Description string
}

func (c *Config) Finalize(env *ConfigEnv) error {
	c.loadDefaults()
	if env != nil {
		c.loadEnv(env)
	}
	return nil
}

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
```

**`pkg/middleware/config.go`**

```go
package middleware

import (
	"os"
	"strconv"
	"strings"
)

type CORSConfig struct {
	Enabled          bool     `toml:"enabled"`
	Origins          []string `toml:"origins"`
	AllowedMethods   []string `toml:"allowed_methods"`
	AllowedHeaders   []string `toml:"allowed_headers"`
	AllowCredentials bool     `toml:"allow_credentials"`
	MaxAge           int      `toml:"max_age"`
}

type CORSEnv struct {
	Enabled          string
	Origins          string
	AllowedMethods   string
	AllowedHeaders   string
	AllowCredentials string
	MaxAge           string
}

func (c *CORSConfig) Finalize(env *CORSEnv) error {
	c.loadDefaults()
	if env != nil {
		c.loadEnv(env)
	}
	return nil
}

func (c *CORSConfig) Merge(overlay *CORSConfig) {
	c.Enabled = overlay.Enabled
	c.AllowCredentials = overlay.AllowCredentials

	if overlay.Origins != nil {
		c.Origins = overlay.Origins
	}
	if overlay.AllowedMethods != nil {
		c.AllowedMethods = overlay.AllowedMethods
	}
	if overlay.AllowedHeaders != nil {
		c.AllowedHeaders = overlay.AllowedHeaders
	}
	if overlay.MaxAge >= 0 {
		c.MaxAge = overlay.MaxAge
	}
}

func (c *CORSConfig) loadDefaults() {
	if len(c.AllowedMethods) == 0 {
		c.AllowedMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	}
	if len(c.AllowedHeaders) == 0 {
		c.AllowedHeaders = []string{"Content-Type", "Authorization"}
	}
	if c.MaxAge <= 0 {
		c.MaxAge = 3600
	}
}

func (c *CORSConfig) loadEnv(env *CORSEnv) {
	if env.Enabled != "" {
		if v := os.Getenv(env.Enabled); v != "" {
			if enabled, err := strconv.ParseBool(v); err == nil {
				c.Enabled = enabled
			}
		}
	}
	if env.Origins != "" {
		if v := os.Getenv(env.Origins); v != "" {
			origins := strings.Split(v, ",")
			c.Origins = make([]string, 0, len(origins))
			for _, origin := range origins {
				if trimmed := strings.TrimSpace(origin); trimmed != "" {
					c.Origins = append(c.Origins, trimmed)
				}
			}
		}
	}
	if env.AllowedMethods != "" {
		if v := os.Getenv(env.AllowedMethods); v != "" {
			methods := strings.Split(v, ",")
			c.AllowedMethods = make([]string, 0, len(methods))
			for _, method := range methods {
				if trimmed := strings.TrimSpace(method); trimmed != "" {
					c.AllowedMethods = append(c.AllowedMethods, trimmed)
				}
			}
		}
	}
	if env.AllowedHeaders != "" {
		if v := os.Getenv(env.AllowedHeaders); v != "" {
			headers := strings.Split(v, ",")
			c.AllowedHeaders = make([]string, 0, len(headers))
			for _, header := range headers {
				if trimmed := strings.TrimSpace(header); trimmed != "" {
					c.AllowedHeaders = append(c.AllowedHeaders, trimmed)
				}
			}
		}
	}
	if env.AllowCredentials != "" {
		if v := os.Getenv(env.AllowCredentials); v != "" {
			if creds, err := strconv.ParseBool(v); err == nil {
				c.AllowCredentials = creds
			}
		}
	}
	if env.MaxAge != "" {
		if v := os.Getenv(env.MaxAge); v != "" {
			if maxAge, err := strconv.Atoi(v); err == nil {
				c.MaxAge = maxAge
			}
		}
	}
}
```

### Step 4: Configuration System

**`internal/config/config.go`**

```go
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

type Config struct {
	Server          ServerConfig    `toml:"server"`
	Database        database.Config `toml:"database"`
	Storage         storage.Config  `toml:"storage"`
	API             APIConfig       `toml:"api"`
	ShutdownTimeout string          `toml:"shutdown_timeout"`
	Version         string          `toml:"version"`
}

func (c *Config) Env() string {
	if env := os.Getenv(EnvHeraldEnv); env != "" {
		return env
	}
	return "local"
}

func (c *Config) ShutdownTimeoutDuration() time.Duration {
	d, _ := time.ParseDuration(c.ShutdownTimeout)
	return d
}

func Load() (*Config, error) {
	cfg, err := load(BaseConfigFile)
	if err != nil {
		return nil, err
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
```

**`internal/config/server.go`**

```go
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

type ServerConfig struct {
	Host            string `toml:"host"`
	Port            int    `toml:"port"`
	ReadTimeout     string `toml:"read_timeout"`
	WriteTimeout    string `toml:"write_timeout"`
	ShutdownTimeout string `toml:"shutdown_timeout"`
}

func (c *ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func (c *ServerConfig) ReadTimeoutDuration() time.Duration {
	d, _ := time.ParseDuration(c.ReadTimeout)
	return d
}

func (c *ServerConfig) WriteTimeoutDuration() time.Duration {
	d, _ := time.ParseDuration(c.WriteTimeout)
	return d
}

func (c *ServerConfig) ShutdownTimeoutDuration() time.Duration {
	d, _ := time.ParseDuration(c.ShutdownTimeout)
	return d
}

func (c *ServerConfig) Finalize() error {
	c.loadDefaults()
	c.loadEnv()
	return c.validate()
}

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
```

**`internal/config/api.go`**

```go
package config

import (
	"fmt"
	"os"

	"github.com/JaimeStill/herald/pkg/middleware"
	"github.com/JaimeStill/herald/pkg/openapi"
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

type APIConfig struct {
	BasePath string                `toml:"base_path"`
	CORS     middleware.CORSConfig `toml:"cors"`
	OpenAPI  openapi.Config        `toml:"openapi"`
}

func (c *APIConfig) Finalize() error {
	c.loadDefaults()
	c.loadEnv()

	if err := c.CORS.Finalize(corsEnv); err != nil {
		return fmt.Errorf("cors: %w", err)
	}
	if err := c.OpenAPI.Finalize(openAPIEnv); err != nil {
		return fmt.Errorf("openapi: %w", err)
	}
	return nil
}

func (c *APIConfig) Merge(overlay *APIConfig) {
	if overlay.BasePath != "" {
		c.BasePath = overlay.BasePath
	}
	c.CORS.Merge(&overlay.CORS)
	c.OpenAPI.Merge(&overlay.OpenAPI)
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
```

### Step 5: Lifecycle Coordinator

**`pkg/lifecycle/lifecycle.go`**

```go
package lifecycle

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type ReadinessChecker interface {
	Ready() bool
}

type Coordinator struct {
	ctx        context.Context
	cancel     context.CancelFunc
	startupWg  sync.WaitGroup
	shutdownWg sync.WaitGroup
	ready      bool
	readyMu    sync.RWMutex
}

func New() *Coordinator {
	ctx, cancel := context.WithCancel(context.Background())
	return &Coordinator{
		ctx:    ctx,
		cancel: cancel,
	}
}

func (c *Coordinator) Context() context.Context {
	return c.ctx
}

func (c *Coordinator) OnStartup(fn func()) {
	c.startupWg.Go(fn)
}

func (c *Coordinator) OnShutdown(fn func()) {
	c.shutdownWg.Go(fn)
}

func (c *Coordinator) Ready() bool {
	c.readyMu.RLock()
	defer c.readyMu.RUnlock()
	return c.ready
}

func (c *Coordinator) WaitForStartup() {
	c.startupWg.Wait()
	c.readyMu.Lock()
	c.ready = true
	c.readyMu.Unlock()
}

func (c *Coordinator) Shutdown(timeout time.Duration) error {
	c.cancel()

	done := make(chan struct{})
	go func() {
		c.shutdownWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("shutdown timeout after %v", timeout)
	}
}
```

### Step 6: OpenAPI Infrastructure

**`pkg/openapi/types.go`**

```go
package openapi

type Info struct {
	Title       string `json:"title"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
}

type Server struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

type PathItem struct {
	Get    *Operation `json:"get,omitempty"`
	Post   *Operation `json:"post,omitempty"`
	Put    *Operation `json:"put,omitempty"`
	Delete *Operation `json:"delete,omitempty"`
}

type Operation struct {
	Summary     string            `json:"summary,omitempty"`
	Description string            `json:"description,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Parameters  []*Parameter      `json:"parameters,omitempty"`
	RequestBody *RequestBody      `json:"requestBody,omitempty"`
	Responses   map[int]*Response `json:"responses"`
}

type Parameter struct {
	Name        string  `json:"name"`
	In          string  `json:"in"`
	Required    bool    `json:"required,omitempty"`
	Description string  `json:"description,omitempty"`
	Schema      *Schema `json:"schema"`
}

type RequestBody struct {
	Description string                `json:"description,omitempty"`
	Required    bool                  `json:"required,omitempty"`
	Content     map[string]*MediaType `json:"content"`
}

type Response struct {
	Description string                `json:"description"`
	Content     map[string]*MediaType `json:"content,omitempty"`
	Ref         string                `json:"$ref,omitempty"`
}

type MediaType struct {
	Schema *Schema `json:"schema,omitempty"`
}

type Schema struct {
	Type        string             `json:"type,omitempty"`
	Format      string             `json:"format,omitempty"`
	Description string             `json:"description,omitempty"`
	Properties  map[string]*Schema `json:"properties,omitempty"`
	Required    []string           `json:"required,omitempty"`
	Items       *Schema            `json:"items,omitempty"`
	Ref         string             `json:"$ref,omitempty"`

	Example any   `json:"example,omitempty"`
	Default any   `json:"default,omitempty"`
	Enum    []any `json:"enum,omitempty"`

	Minimum   *float64 `json:"minimum,omitempty"`
	Maximum   *float64 `json:"maximum,omitempty"`
	MinLength *int     `json:"minLength,omitempty"`
	MaxLength *int     `json:"maxLength,omitempty"`
	Pattern   string   `json:"pattern,omitempty"`
}

type Components struct {
	Schemas   map[string]*Schema   `json:"schemas,omitempty"`
	Responses map[string]*Response `json:"responses,omitempty"`
}

func SchemaRef(name string) *Schema {
	return &Schema{Ref: "#/components/schemas/" + name}
}

func ResponseRef(name string) *Response {
	return &Response{Ref: "#/components/responses/" + name}
}

func RequestBodyJSON(schemaName string, required bool) *RequestBody {
	return &RequestBody{
		Required: required,
		Content: map[string]*MediaType{
			"application/json": {Schema: SchemaRef(schemaName)},
		},
	}
}

func ResponseJSON(description, schemaName string) *Response {
	return &Response{
		Description: description,
		Content: map[string]*MediaType{
			"application/json": {Schema: SchemaRef(schemaName)},
		},
	}
}

func PathParam(name, description string) *Parameter {
	return &Parameter{
		Name:        name,
		In:          "path",
		Required:    true,
		Description: description,
		Schema:      &Schema{Type: "string", Format: "uuid"},
	}
}

func QueryParam(name, typ, description string, required bool) *Parameter {
	return &Parameter{
		Name:        name,
		In:          "query",
		Required:    required,
		Description: description,
		Schema:      &Schema{Type: typ},
	}
}
```

**`pkg/openapi/spec.go`**

```go
package openapi

import "net/http"

type Spec struct {
	OpenAPI    string               `json:"openapi"`
	Info       *Info                `json:"info"`
	Servers    []*Server            `json:"servers,omitempty"`
	Paths      map[string]*PathItem `json:"paths"`
	Components *Components          `json:"components,omitempty"`
}

func NewSpec(title, version string) *Spec {
	return &Spec{
		OpenAPI: "3.1.0",
		Info: &Info{
			Title:   title,
			Version: version,
		},
		Components: NewComponents(),
		Paths:      make(map[string]*PathItem),
	}
}

func (s *Spec) AddServer(url string) {
	s.Servers = append(s.Servers, &Server{URL: url})
}

func (s *Spec) SetDescription(desc string) {
	s.Info.Description = desc
}

func ServeSpec(specBytes []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(specBytes)
	}
}
```

**`pkg/openapi/components.go`**

```go
package openapi

import "maps"

func NewComponents() *Components {
	return &Components{
		Schemas: map[string]*Schema{
			"PageRequest": {
				Type: "object",
				Properties: map[string]*Schema{
					"page":      {Type: "integer", Description: "Page number (1-indexed)", Example: 1},
					"page_size": {Type: "integer", Description: "Results per page", Example: 20},
					"search":    {Type: "string", Description: "Search query"},
					"sort":      {Type: "string", Description: "Comma-separated sort fields. Prefix with - for descending. Example: name,-created_at"},
				},
			},
		},
		Responses: map[string]*Response{
			"BadRequest": {
				Description: "Invalid request",
				Content: map[string]*MediaType{
					"application/json": {
						Schema: &Schema{
							Type: "object",
							Properties: map[string]*Schema{
								"error": {Type: "string", Description: "Error message"},
							},
						},
					},
				},
			},
			"NotFound": {
				Description: "Resource not found",
				Content: map[string]*MediaType{
					"application/json": {
						Schema: &Schema{
							Type: "object",
							Properties: map[string]*Schema{
								"error": {Type: "string", Description: "Error message"},
							},
						},
					},
				},
			},
			"Conflict": {
				Description: "Resource conflict (duplicate name)",
				Content: map[string]*MediaType{
					"application/json": {
						Schema: &Schema{
							Type: "object",
							Properties: map[string]*Schema{
								"error": {Type: "string", Description: "Error message"},
							},
						},
					},
				},
			},
		},
	}
}

func (c *Components) AddSchemas(schemas map[string]*Schema) {
	maps.Copy(c.Schemas, schemas)
}

func (c *Components) AddResponses(responses map[string]*Response) {
	maps.Copy(c.Responses, responses)
}
```

**`pkg/openapi/json.go`**

```go
package openapi

import (
	"encoding/json"
	"os"
)

func MarshalJSON(spec *Spec) ([]byte, error) {
	return json.MarshalIndent(spec, "", "  ")
}

func WriteJSON(spec *Spec, filename string) error {
	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}
```

### Step 7: HTTP Infrastructure

**`pkg/handlers/handlers.go`**

```go
package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func RespondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func RespondError(w http.ResponseWriter, logger *slog.Logger, status int, err error) {
	logger.Error("handler error", "error", err, "status", status)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
```

**`pkg/routes/route.go`**

```go
package routes

import (
	"net/http"

	"github.com/JaimeStill/herald/pkg/openapi"
)

type Route struct {
	Method  string
	Pattern string
	Handler http.HandlerFunc
	OpenAPI *openapi.Operation
}
```

**`pkg/routes/group.go`**

```go
package routes

import (
	"maps"
	"net/http"

	"github.com/JaimeStill/herald/pkg/openapi"
)

type Group struct {
	Prefix      string
	Tags        []string
	Description string
	Routes      []Route
	Children    []Group
	Schemas     map[string]*openapi.Schema
}

func (g *Group) AddToSpec(basePath string, spec *openapi.Spec) {
	g.addOperations(basePath, spec)
}

func (g *Group) addOperations(parentPrefix string, spec *openapi.Spec) {
	fullPrefix := parentPrefix + g.Prefix

	maps.Copy(spec.Components.Schemas, g.Schemas)

	for _, route := range g.Routes {
		if route.OpenAPI == nil {
			continue
		}

		path := fullPrefix + route.Pattern
		op := route.OpenAPI

		if len(op.Tags) == 0 {
			op.Tags = g.Tags
		}

		if spec.Paths[path] == nil {
			spec.Paths[path] = &openapi.PathItem{}
		}

		switch route.Method {
		case "GET":
			spec.Paths[path].Get = op
		case "POST":
			spec.Paths[path].Post = op
		case "PUT":
			spec.Paths[path].Put = op
		case "DELETE":
			spec.Paths[path].Delete = op
		}
	}

	for _, child := range g.Children {
		child.addOperations(fullPrefix, spec)
	}
}

func Register(mux *http.ServeMux, basePath string, spec *openapi.Spec, groups ...Group) {
	for _, group := range groups {
		group.AddToSpec(basePath, spec)
		registerGroup(mux, "", group)
	}
}

func registerGroup(mux *http.ServeMux, parentPrefix string, group Group) {
	fullPrefix := parentPrefix + group.Prefix
	for _, route := range group.Routes {
		pattern := route.Method + " " + fullPrefix + route.Pattern
		mux.HandleFunc(pattern, route.Handler)
	}
	for _, child := range group.Children {
		registerGroup(mux, fullPrefix, child)
	}
}
```

**`pkg/middleware/middleware.go`**

```go
package middleware

import "net/http"

type System interface {
	Use(mw func(http.Handler) http.Handler)
	Apply(handler http.Handler) http.Handler
}

type mw struct {
	stack []func(http.Handler) http.Handler
}

func New() System {
	return &mw{
		stack: []func(http.Handler) http.Handler{},
	}
}

func (m *mw) Use(fn func(http.Handler) http.Handler) {
	m.stack = append(m.stack, fn)
}

func (m *mw) Apply(handler http.Handler) http.Handler {
	for i := len(m.stack) - 1; i >= 0; i-- {
		handler = m.stack[i](handler)
	}
	return handler
}
```

**`pkg/middleware/cors.go`**

```go
package middleware

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
)

func CORS(cfg *CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !cfg.Enabled || len(cfg.Origins) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			origin := r.Header.Get("Origin")
			allowed := slices.Contains(cfg.Origins, origin)

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(cfg.AllowedMethods, ", "))
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(cfg.AllowedHeaders, ", "))

				if cfg.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}

				if cfg.MaxAge > 0 {
					w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", cfg.MaxAge))
				}
			}

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
```

**`pkg/middleware/logger.go`**

```go
package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

func Logger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			logger.Info(
				"request",
				"method", r.Method,
				"uri", r.URL.RequestURI(),
				"addr", r.RemoteAddr,
				"duration", time.Since(start),
			)
		})
	}
}
```

**`pkg/module/module.go`**

```go
package module

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/JaimeStill/herald/pkg/middleware"
)

type Module struct {
	prefix     string
	router     http.Handler
	middleware middleware.System
}

func New(prefix string, router http.Handler) *Module {
	if err := validatePrefix(prefix); err != nil {
		panic(err)
	}
	return &Module{
		prefix:     prefix,
		router:     router,
		middleware: middleware.New(),
	}
}

func (m *Module) Handler() http.Handler {
	return m.middleware.Apply(m.router)
}

func (m *Module) Prefix() string {
	return m.prefix
}

func (m *Module) Serve(w http.ResponseWriter, req *http.Request) {
	path := extractPath(req.URL.Path, m.prefix)
	request := cloneRequest(req, path)
	m.Handler().ServeHTTP(w, request)
}

func (m *Module) Use(mw func(http.Handler) http.Handler) {
	m.middleware.Use(mw)
}

func cloneRequest(req *http.Request, path string) *http.Request {
	request := new(http.Request)
	*request = *req
	request.URL = new(url.URL)
	*request.URL = *req.URL
	request.URL.Path = path
	request.URL.RawPath = ""
	return request
}

func extractPath(fullPath, prefix string) string {
	path := fullPath[len(prefix):]
	if path == "" {
		return "/"
	}
	return path
}

func validatePrefix(prefix string) error {
	if prefix == "" {
		return fmt.Errorf("module prefix cannot be empty")
	}
	if !strings.HasPrefix(prefix, "/") {
		return fmt.Errorf("module prefix must start with /: %s", prefix)
	}
	if strings.Count(prefix, "/") != 1 {
		return fmt.Errorf("module prefix must be single-level sub-path: %s", prefix)
	}
	return nil
}
```

**`pkg/module/router.go`**

```go
package module

import (
	"net/http"
	"strings"
)

type Router struct {
	modules map[string]*Module
	native  *http.ServeMux
}

func NewRouter() *Router {
	return &Router{
		modules: make(map[string]*Module),
		native:  http.NewServeMux(),
	}
}

func (r *Router) HandleNative(pattern string, handler http.HandlerFunc) {
	r.native.HandleFunc(pattern, handler)
}

func (r *Router) Mount(m *Module) {
	r.modules[m.prefix] = m
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	path := normalizePath(req)
	prefix := extractPrefix(path)

	if m, ok := r.modules[prefix]; ok {
		m.Serve(w, req)
		return
	}

	r.native.ServeHTTP(w, req)
}

func extractPrefix(path string) string {
	parts := strings.SplitN(path, "/", 3)
	if len(parts) >= 2 {
		return "/" + parts[1]
	}
	return path
}

func normalizePath(req *http.Request) string {
	path := req.URL.Path
	if len(path) > 1 && strings.HasSuffix(path, "/") {
		path = strings.TrimSuffix(path, "/")
		req.URL.Path = path
	}
	return path
}
```

### Step 8: Web Build Pipeline and Scalar Module

The web build uses Bun + Vite, following agent-lab's multi-entry architecture. Built assets (scalar.js, scalar.css) are committed so `go build` works without Bun installed. The `mise run web` task rebuilds them.

**`web/.gitignore`**

```gitignore
node_modules
```

**`web/package.json`**

```json
{
  "name": "herald-web",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "vite build --watch",
    "build": "vite build"
  },
  "devDependencies": {
    "typescript": "^5.9.3",
    "vite": "^7.3.1"
  },
  "dependencies": {
    "@scalar/api-reference": "1.43.2"
  }
}
```

**`web/tsconfig.json`**

```json
{
  "compilerOptions": {
    "target": "ES2024",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "strict": true,
    "noEmit": true,
    "skipLibCheck": true,
    "esModuleInterop": true,
    "isolatedModules": true
  },
  "include": [
    "scalar/**/*"
  ],
  "exclude": [
    "node_modules"
  ]
}
```

**`web/vite.client.ts`**

```typescript
import { resolve } from 'path'
import type { PreRenderedAsset, PreRenderedChunk, RollupOptions } from 'rollup'
import type { UserConfig } from 'vite'

export interface ClientConfig {
  name: string
  input?: string
  output?: {
    entryFileNames?: string | ((chunk: PreRenderedChunk) => string)
    assetFileNames?: string | ((asset: PreRenderedAsset) => string)
  }
  aliases?: Record<string, string>
}

const root = __dirname

export function merge(clients: ClientConfig[]): UserConfig {
  return {
    build: {
      outDir: '.',
      emptyOutDir: false,
      rollupOptions: mergeRollup(clients),
    },
    resolve: mergeResolve(clients),
  }
}

function defaultInput(name: string) {
  return resolve(root, `${name}/client/app.ts`)
}

function defaultEntry(name: string) {
  return `${name}/dist/app.js`
}

function defaultAssets(name: string) {
  return `${name}/dist/[name][extname]`
}

function mergeRollup(clients: ClientConfig[]): RollupOptions {
  return {
    input: Object.fromEntries(
      clients.map(c => [c.name, c.input ?? defaultInput(c.name)])
    ),
    output: {
      entryFileNames: (chunk: PreRenderedChunk): string => {
        const client = clients.find(c => c.name === chunk.name)
        const custom = client?.output?.entryFileNames
        if (custom) return typeof custom === 'function' ? custom(chunk) : custom
        return defaultEntry(chunk.name)
      },
      assetFileNames: (asset: PreRenderedAsset): string => {
        const originalPath = asset.originalFileNames?.[0] ?? ''
        const client = clients.find(c => originalPath.startsWith(`${c.name}/`))
        if (client?.output?.assetFileNames) {
          const custom = client.output.assetFileNames
          return typeof custom === 'function' ? custom(asset) : custom
        }
        return client ? defaultAssets(client.name) : 'app/dist/[name][extname]'
      },
    },
  }
}

function mergeResolve(clients: ClientConfig[]): UserConfig['resolve'] {
  return {
    alias: Object.assign({}, ...clients.map(c => c.aliases ?? {})),
  }
}
```

**`web/vite.config.ts`**

```typescript
import { defineConfig } from 'vite';
import { merge } from './vite.client';
import scalarConfig from './scalar/client.config.ts';

export default defineConfig(merge([
  scalarConfig
]))
```

**`web/scalar/app.ts`**

```typescript
import { createApiReference } from '@scalar/api-reference'
import '@scalar/api-reference/style.css'

createApiReference('#api-reference', {
  url: '/api/openapi.json',
  withDefaultFonts: false,
})
```

**`web/scalar/client.config.ts`**

```typescript
import { resolve } from 'path'
import type { ClientConfig } from '../vite.client';

const config: ClientConfig = {
  name: 'scalar',
  input: resolve(__dirname, 'app.ts'),
  output: {
    entryFileNames: 'scalar/scalar.js',
    assetFileNames: 'scalar/scalar.css',
  },
}

export default config
```

**`web/scalar/index.html`**

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <base href="{{ .BasePath }}/">
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Herald - API Documentation</title>
  <link rel="stylesheet" href="scalar.css">
  <style>
    :root {
      --scalar-font: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
      --scalar-font-code: ui-monospace, 'Cascadia Code', 'SF Mono', Menlo, Monaco, Consolas, monospace;
    }
  </style>
</head>
<body>
  <div id="api-reference"></div>
  <script type="module" src="scalar.js"></script>
</body>
</html>
```

**`web/scalar/scalar.go`**

```go
package scalar

import (
	"embed"
	"html/template"
	"net/http"

	"github.com/JaimeStill/herald/pkg/module"
)

//go:embed index.html scalar.css scalar.js
var staticFS embed.FS

func NewModule(basePath string) *module.Module {
	router := buildRouter(basePath)
	return module.New(basePath, router)
}

func buildRouter(basePath string) http.Handler {
	mux := http.NewServeMux()

	tmpl := template.Must(template.ParseFS(staticFS, "index.html"))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tmpl.Execute(w, map[string]string{"BasePath": basePath})
	})

	mux.Handle("GET /", http.FileServer(http.FS(staticFS)))

	return mux
}
```

After creating all web files, build the assets:

```bash
cd web && bun install && bun run build
```

This produces `web/scalar/scalar.js` and `web/scalar/scalar.css` which are embedded by `scalar.go` at compile time. Commit these built assets so `go build` works without Bun.

### Step 9: Walking Skeleton

**`cmd/server/main.go`**

```go
package main

import (
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/JaimeStill/herald/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("config load failed:", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	logger.Info("herald starting",
		"version", cfg.Version,
		"addr", cfg.Server.Addr(),
		"env", cfg.Env(),
	)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info("herald stopped")
}
```

After creating all files, run:

```bash
go mod tidy
go build ./...
go vet ./...
```

## Validation Criteria

- [ ] `go build ./...` passes
- [ ] `go vet ./...` passes
- [ ] `mise run build` produces `bin/server`
- [ ] `docker compose up -d` brings up PostgreSQL (5432) and Azurite (10000)
- [ ] `mise run dev` starts the server and logs config info
- [ ] All packages exist with at least one `.go` file
