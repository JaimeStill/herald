package infrastructure_test

import (
	"testing"

	"github.com/JaimeStill/herald/internal/config"
	"github.com/JaimeStill/herald/internal/infrastructure"
	"github.com/JaimeStill/herald/pkg/database"
	"github.com/JaimeStill/herald/pkg/storage"
)

const azuriteConnString = "DefaultEndpointsProtocol=http;AccountName=heraldstore;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;BlobEndpoint=http://127.0.0.1:10000/heraldstore;"

func validConfig() *config.Config {
	return &config.Config{
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
		Version: "0.1.0",
	}
}

func TestNew(t *testing.T) {
	infra, err := infrastructure.New(validConfig())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if infra.Lifecycle == nil {
		t.Error("Lifecycle is nil")
	}
	if infra.Logger == nil {
		t.Error("Logger is nil")
	}
	if infra.Database == nil {
		t.Error("Database is nil")
	}
	if infra.Storage == nil {
		t.Error("Storage is nil")
	}
}

func TestNewDatabaseConnection(t *testing.T) {
	infra, err := infrastructure.New(validConfig())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	conn := infra.Database.Connection()
	if conn == nil {
		t.Fatal("Database.Connection() returned nil")
	}
	conn.Close()
}

func TestNewInvalidStorageConfig(t *testing.T) {
	cfg := validConfig()
	cfg.Storage.ConnectionString = "not-a-connection-string"

	_, err := infrastructure.New(cfg)
	if err == nil {
		t.Fatal("expected error for invalid storage connection string")
	}
}
