package api_test

import (
	"testing"

	gaconfig "github.com/JaimeStill/go-agents/pkg/config"
	"github.com/JaimeStill/herald/internal/api"
	"github.com/JaimeStill/herald/internal/config"
	"github.com/JaimeStill/herald/internal/infrastructure"
	"github.com/JaimeStill/herald/pkg/database"
	"github.com/JaimeStill/herald/pkg/middleware"
	"github.com/JaimeStill/herald/pkg/pagination"
	"github.com/JaimeStill/herald/pkg/storage"
)

const azuriteConnString = "DefaultEndpointsProtocol=http;AccountName=heraldstore;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;BlobEndpoint=http://127.0.0.1:10000/heraldstore;"

func validConfig() *config.Config {
	return &config.Config{
		Agent: gaconfig.AgentConfig{
			Name: "test-agent",
			Provider: &gaconfig.ProviderConfig{
				Name:    "ollama",
				BaseURL: "http://localhost:11434",
				Options: make(map[string]any),
			},
			Model: &gaconfig.ModelConfig{
				Name: "llama3.1:8b",
			},
		},
		Server: config.ServerConfig{
			Host:            "0.0.0.0",
			Port:            8080,
			ReadTimeout:     "1m",
			WriteTimeout:    "15m",
			ShutdownTimeout: "30s",
		},
		Database: database.Config{
			Host:            "localhost",
			Port:            5432,
			Name:            "herald",
			User:            "herald",
			Password:        "herald",
			SSLMode:         "disable",
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: "15m",
			ConnTimeout:     "5s",
		},
		Storage: storage.Config{
			ContainerName:    "documents",
			ConnectionString: azuriteConnString,
		},
		API: config.APIConfig{
			BasePath: "/api",
			CORS: middleware.CORSConfig{
				Enabled: false,
			},
			Pagination: pagination.Config{
				DefaultPageSize: 20,
				MaxPageSize:     100,
			},
		},
		ShutdownTimeout: "30s",
		Version:         "0.1.0",
	}
}

func setupInfra(t *testing.T) *infrastructure.Infrastructure {
	t.Helper()
	infra, err := infrastructure.New(validConfig())
	if err != nil {
		t.Fatalf("infrastructure.New() error = %v", err)
	}
	return infra
}

func TestNewModule(t *testing.T) {
	cfg := validConfig()
	infra := setupInfra(t)

	m, err := api.NewModule(cfg, infra)
	if err != nil {
		t.Fatalf("NewModule() error = %v", err)
	}

	if m.Prefix() != "/api" {
		t.Errorf("prefix: got %s, want /api", m.Prefix())
	}
}

func TestNewRuntime(t *testing.T) {
	cfg := validConfig()
	infra := setupInfra(t)

	runtime := api.NewRuntime(cfg, infra)

	if runtime.Pagination.DefaultPageSize != 20 {
		t.Errorf("pagination default page size: got %d, want 20", runtime.Pagination.DefaultPageSize)
	}
	if runtime.Pagination.MaxPageSize != 100 {
		t.Errorf("pagination max page size: got %d, want 100", runtime.Pagination.MaxPageSize)
	}
	if runtime.Logger == nil {
		t.Error("runtime logger is nil")
	}
	if runtime.Database == nil {
		t.Error("runtime database is nil")
	}
	if runtime.Storage == nil {
		t.Error("runtime storage is nil")
	}
	if runtime.Lifecycle == nil {
		t.Error("runtime lifecycle is nil")
	}
}

func TestNewDomain(t *testing.T) {
	cfg := validConfig()
	infra := setupInfra(t)
	runtime := api.NewRuntime(cfg, infra)

	domain := api.NewDomain(runtime)
	if domain == nil {
		t.Fatal("NewDomain() returned nil")
	}
}
