package database_test

import (
	"errors"
	"log/slog"
	"testing"

	"github.com/JaimeStill/herald/pkg/database"
)

func TestNewReturnsSystem(t *testing.T) {
	cfg := database.Config{
		Host:            "localhost",
		Port:            5432,
		Name:            "testdb",
		User:            "testuser",
		Password:        "testpass",
		SSLMode:         "disable",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: "15m",
		ConnTimeout:     "5s",
	}

	logger := slog.Default()
	sys, err := database.New(&cfg, logger)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if sys == nil {
		t.Fatal("New() returned nil system")
	}

	conn := sys.Connection()
	if conn == nil {
		t.Fatal("Connection() returned nil")
	}

	// sql.Open is lazy — Close should succeed even without a real database
	conn.Close()
}

func TestNewSetsPoolParams(t *testing.T) {
	cfg := database.Config{
		Host:            "localhost",
		Port:            5432,
		Name:            "testdb",
		User:            "testuser",
		SSLMode:         "disable",
		MaxOpenConns:    42,
		MaxIdleConns:    7,
		ConnMaxLifetime: "10m",
		ConnTimeout:     "3s",
	}

	sys, err := database.New(&cfg, slog.Default())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	conn := sys.Connection()
	defer conn.Close()

	stats := conn.Stats()
	if stats.MaxOpenConnections != 42 {
		t.Errorf("MaxOpenConnections = %d, want 42", stats.MaxOpenConnections)
	}
}

func TestErrNotReady(t *testing.T) {
	if !errors.Is(database.ErrNotReady, database.ErrNotReady) {
		t.Error("ErrNotReady should match itself")
	}

	if database.ErrNotReady.Error() != "database not ready" {
		t.Errorf("ErrNotReady.Error() = %q, want %q", database.ErrNotReady.Error(), "database not ready")
	}
}
