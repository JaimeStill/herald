# 29 - Agent Configuration and Infrastructure Wiring

## Problem Context

Herald's config system uses TOML format via the `pelletier/go-toml/v2` third-party dependency. Phase 2 requires agent configuration using go-agents' `config.AgentConfig` type, which uses JSON struct tags. Migrating the entire config system to JSON eliminates the third-party dependency (stdlib `encoding/json`), allows go-agents types to embed directly without a conversion layer, and aligns config file format with go-agents' native JSON format.

## Architecture Approach

- All struct tags change from `toml:"x"` to `json:"x"` across 7 files
- `toml.Unmarshal` replaced with `json.Unmarshal`, config file constants updated
- `Config.Agent` and `Infrastructure.Agent` are both `gaconfig.AgentConfig` (value type, consistent with other config fields)
- `internal/config/agent.go` provides standalone functions (`FinalizeAgent`, `MergeAgent`, etc.) that implement Herald's three-phase finalize pattern for the go-agents type
- `HERALD_AGENT_TOKEN` is the only agent-specific env var; injected into `Provider.Options["token"]` when set
- Startup validation delegates to `agent.New()` which validates provider-specific requirements (Azure requires token, Ollama does not)
- go-toml dependency removed via `go mod tidy` after all imports are updated

## Implementation

### Step 1: Add go-agents dependency

```bash
go get github.com/JaimeStill/go-agents
```

### Step 2: Migrate struct tags (7 files)

Replace all `toml:` struct tags with `json:` tags. Each file listed below requires a mechanical find-replace.

**`pkg/database/config.go`** — 10 fields on `Config`:

```go
type Config struct {
	Host            string `json:"host"`
	Port            int    `json:"port"`
	Name            string `json:"name"`
	User            string `json:"user"`
	Password        string `json:"password"`
	SSLMode         string `json:"ssl_mode"`
	MaxOpenConns    int    `json:"max_open_conns"`
	MaxIdleConns    int    `json:"max_idle_conns"`
	ConnMaxLifetime string `json:"conn_max_lifetime"`
	ConnTimeout     string `json:"conn_timeout"`
}
```

**`pkg/storage/config.go`** — 3 fields on `Config`:

```go
type Config struct {
	ContainerName    string `json:"container_name"`
	ConnectionString string `json:"connection_string"`
	MaxListSize      int32  `json:"max_list_size"`
}
```

**`pkg/middleware/config.go`** — 6 fields on `CORSConfig`:

```go
type CORSConfig struct {
	Enabled          bool     `json:"enabled"`
	Origins          []string `json:"origins"`
	AllowedMethods   []string `json:"allowed_methods"`
	AllowedHeaders   []string `json:"allowed_headers"`
	AllowCredentials bool     `json:"allow_credentials"`
	MaxAge           int      `json:"max_age"`
}
```

**`pkg/pagination/config.go`** — 2 fields on `Config`:

```go
type Config struct {
	DefaultPageSize int `json:"default_page_size"`
	MaxPageSize     int `json:"max_page_size"`
}
```

**`internal/config/server.go`** — 5 fields on `ServerConfig`:

```go
type ServerConfig struct {
	Host            string `json:"host"`
	Port            int    `json:"port"`
	ReadTimeout     string `json:"read_timeout"`
	WriteTimeout    string `json:"write_timeout"`
	ShutdownTimeout string `json:"shutdown_timeout"`
}
```

**`internal/config/api.go`** — 4 fields on `APIConfig`:

```go
type APIConfig struct {
	BasePath      string                `json:"base_path"`
	MaxUploadSize string                `json:"max_upload_size"`
	CORS          middleware.CORSConfig `json:"cors"`
	Pagination    pagination.Config     `json:"pagination"`
}
```

**`internal/config/config.go`** — 6 existing fields on `Config` (Agent field added in Step 4):

```go
type Config struct {
	Server          ServerConfig    `json:"server"`
	Database        database.Config `json:"database"`
	Storage         storage.Config  `json:"storage"`
	API             APIConfig       `json:"api"`
	ShutdownTimeout string          `json:"shutdown_timeout"`
	Version         string          `json:"version"`
}
```

### Step 3: Switch config loading from TOML to JSON

In `internal/config/config.go`:

Update the constants:

```go
const (
	BaseConfigFile       = "config.json"
	OverlayConfigPattern = "config.%s.json"

	EnvHeraldEnv             = "HERALD_ENV"
	EnvHeraldShutdownTimeout = "HERALD_SHUTDOWN_TIMEOUT"
	EnvHeraldVersion         = "HERALD_VERSION"
)
```

Replace the import — remove `github.com/pelletier/go-toml/v2`, add `encoding/json`:

```go
import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/JaimeStill/herald/pkg/database"
	"github.com/JaimeStill/herald/pkg/storage"
)
```

Update the `load()` function to use `json.Unmarshal`:

```go
func load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}
```

### Step 4: Create `internal/config/agent.go`

New file. Provides standalone functions that implement Herald's three-phase finalize pattern for go-agents' `AgentConfig` type.

```go
package config

import (
	"fmt"
	"os"

	gaconfig "github.com/JaimeStill/go-agents/pkg/config"
)

const (
	EnvAgentProviderName = "HERALD_AGENT_PROVIDER_NAME"
	EnvAgentBaseURL      = "HERALD_AGENT_BASE_URL"
	EnvAgentToken        = "HERALD_AGENT_TOKEN"
	EnvAgentDeployment   = "HERALD_AGENT_DEPLOYMENT"
	EnvAgentAPIVersion   = "HERALD_AGENT_API_VERSION"
	EnvAgentAuthType     = "HERALD_AGENT_AUTH_TYPE"
	EnvAgentModelName    = "HERALD_AGENT_MODEL_NAME"
)

func FinalizeAgent(c *gaconfig.AgentConfig) error {
	loadAgentDefaults(c)
	loadAgentEnv(c)
	return validateAgent(c)
}

func loadAgentDefaults(c *gaconfig.AgentConfig) {
	defaults := gaconfig.DefaultAgentConfig()
	defaults.Merge(c)
	*c = defaults
}

func loadAgentEnv(c *gaconfig.AgentConfig) {
	if v := os.Getenv(EnvAgentProviderName); v != "" {
		if c.Provider == nil {
			c.Provider = &gaconfig.ProviderConfig{}
		}
		c.Provider.Name = v
	}
	if v := os.Getenv(EnvAgentBaseURL); v != "" {
		if c.Provider == nil {
			c.Provider = &gaconfig.ProviderConfig{}
		}
		c.Provider.BaseURL = v
	}
	if v := os.Getenv(EnvAgentModelName); v != "" {
		if c.Model == nil {
			c.Model = &gaconfig.ModelConfig{}
		}
		c.Model.Name = v
	}

	setOption := func(envVar, key string) {
		if v := os.Getenv(envVar); v != "" {
			if c.Provider == nil {
				c.Provider = &gaconfig.ProviderConfig{}
			}
			if c.Provider.Options == nil {
				c.Provider.Options = make(map[string]any)
			}
			c.Provider.Options[key] = v
		}
	}

	setOption(EnvAgentToken, "token")
	setOption(EnvAgentDeployment, "deployment")
	setOption(EnvAgentAPIVersion, "api_version")
	setOption(EnvAgentAuthType, "auth_type")
}

func validateAgent(c *gaconfig.AgentConfig) error {
	if c.Name == "" {
		return fmt.Errorf("name required")
	}
	if c.Provider == nil {
		return fmt.Errorf("provider required")
	}
	if c.Provider.Name == "" {
		return fmt.Errorf("provider name required")
	}
	if c.Model == nil {
		return fmt.Errorf("model required")
	}
	return nil
}
```

The `loadAgentDefaults` function uses go-agents' own `DefaultAgentConfig()` as the base, then merges in the loaded config values on top — same semantic as Herald's other configs where defaults are set first, then overwritten by file/env values.

### Step 5: Add Agent field to Config

In `internal/config/config.go`:

Add the go-agents config import:

```go
import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	gaconfig "github.com/JaimeStill/go-agents/pkg/config"
	"github.com/JaimeStill/herald/pkg/database"
	"github.com/JaimeStill/herald/pkg/storage"
)
```

Add the Agent field to Config (value type, consistent with other fields):

```go
type Config struct {
	Server          ServerConfig       `json:"server"`
	Database        database.Config    `json:"database"`
	Storage         storage.Config     `json:"storage"`
	API             APIConfig          `json:"api"`
	Agent           gaconfig.AgentConfig `json:"agent"`
	ShutdownTimeout string             `json:"shutdown_timeout"`
	Version         string             `json:"version"`
}
```

Add agent merge to `Merge()` — append after `c.API.Merge(&overlay.API)`:

```go
c.Agent.Merge(&overlay.Agent)
```

Add agent finalization in `finalize()` — append after the `c.API.Finalize()` block (before the final `return nil`):

```go
if err := FinalizeAgent(&c.Agent); err != nil {
	return fmt.Errorf("agent: %w", err)
}
```

### Step 6: Add Agent field to Infrastructure

In `internal/infrastructure/infrastructure.go`:

Add go-agents imports:

```go
import (
	"fmt"
	"log/slog"
	"os"

	"github.com/JaimeStill/go-agents/pkg/agent"
	gaconfig "github.com/JaimeStill/go-agents/pkg/config"
	"github.com/JaimeStill/herald/internal/config"
	"github.com/JaimeStill/herald/pkg/database"
	"github.com/JaimeStill/herald/pkg/lifecycle"
	"github.com/JaimeStill/herald/pkg/storage"
)
```

Add the Agent field to Infrastructure (value type):

```go
type Infrastructure struct {
	Lifecycle *lifecycle.Coordinator
	Logger    *slog.Logger
	Database  database.System
	Storage   storage.System
	Agent     gaconfig.AgentConfig
}
```

In `New()`, add agent validation after storage init and before the return statement. Validate by calling `agent.New()` with a pointer to the config and discarding the result:

```go
if _, err := agent.New(&cfg.Agent); err != nil {
	return nil, fmt.Errorf("agent validation failed: %w", err)
}
```

Add `Agent: cfg.Agent` to the returned Infrastructure struct literal.

### Step 7: Create config.json and delete config.toml

Delete `config.toml` and create `config.json` with the following content:

```json
{
  "shutdown_timeout": "30s",
  "version": "0.2.0",
  "server": {
    "host": "0.0.0.0",
    "port": 8080,
    "read_timeout": "1m",
    "write_timeout": "15m",
    "shutdown_timeout": "30s"
  },
  "database": {
    "host": "localhost",
    "port": 5432,
    "name": "herald",
    "user": "herald",
    "password": "herald",
    "ssl_mode": "disable",
    "max_open_conns": 25,
    "max_idle_conns": 5,
    "conn_max_lifetime": "15m",
    "conn_timeout": "5s"
  },
  "storage": {
    "container_name": "documents",
    "connection_string": "DefaultEndpointsProtocol=http;AccountName=heraldstore;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;BlobEndpoint=http://127.0.0.1:10000/heraldstore;"
  },
  "api": {
    "base_path": "/api",
    "cors": {
      "enabled": false,
      "origins": [],
      "allowed_methods": ["GET", "POST", "PUT", "DELETE", "OPTIONS"],
      "allowed_headers": ["Content-Type", "Authorization"],
      "allow_credentials": false,
      "max_age": 3600
    },
    "pagination": {
      "default_page_size": 20,
      "max_page_size": 100
    }
  },
  "agent": {
    "name": "herald-classifier",
    "provider": {
      "name": "azure",
      "base_url": "https://placeholder.openai.azure.com",
      "options": {
        "deployment": "gpt-5-mini",
        "api_version": "2024-12-01-preview",
        "auth_type": "api_key"
      }
    },
    "model": {
      "name": "gpt-5-mini",
      "capabilities": {
        "vision": {
          "max_tokens": 4096,
          "temperature": 0.1
        }
      }
    }
  }
}
```

### Step 8: Remove go-toml and tidy

```bash
go mod tidy
```

This removes `pelletier/go-toml/v2` from go.mod (no longer imported).

## Validation Criteria

- [ ] All struct tags migrated from `toml` to `json`
- [ ] `pelletier/go-toml/v2` dependency removed, `encoding/json` used
- [ ] `config.json` replaces `config.toml` with overlay pattern `config.<env>.json`
- [ ] `Config.Agent` is `gaconfig.AgentConfig` (go-agents value type)
- [ ] `internal/config/agent.go` implements `FinalizeAgent`/`MergeAgent` with Herald's three-phase pattern
- [ ] `HERALD_AGENT_TOKEN` injected into `Provider.Options["token"]` when env var is set
- [ ] `Infrastructure.Agent` validated via `agent.New()` during `New()`
- [ ] `go vet ./...` passes
- [ ] `go mod tidy` produces no changes
