package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
)

//go:embed migrations/*.sql
var migrations embed.FS

const (
	envDSN     = "HERALD_DB_DSN"
	defaultDSN = "postgres://herald:herald@localhost:5432/herald?sslmode=disable"
)

func main() {
	var (
		dsn     = flag.String("dsn", "", "Database connection string")
		up      = flag.Bool("up", false, "Run all up migrations")
		down    = flag.Bool("down", false, "Run all down migrations")
		steps   = flag.Int("steps", 0, "Number of migrations (positive=up, negative=down)")
		version = flag.Bool("version", false, "Print current migration version")
		force   = flag.Int("force", -1, "Force set version (use with caution)")
	)
	flag.Parse()

	forceSet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "force" {
			forceSet = true
		}
	})

	if *dsn == "" {
		*dsn = os.Getenv(envDSN)
	}
	if *dsn == "" {
		*dsn = defaultDSN
	}

	source, err := iofs.New(migrations, "migrations")
	if err != nil {
		log.Fatalf("failed to create migration source: %v", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, *dsn)
	if err != nil {
		log.Fatalf("failed to create migrator: %v", err)
	}
	defer m.Close()

	switch {
	case *version:
		v, dirty, err := m.Version()
		if err != nil {
			log.Fatalf("failed to get version: %v", err)
		}
		fmt.Printf("version: %d, dirty: %v\n", v, dirty)
	case forceSet:
		if err := m.Force(*force); err != nil {
			log.Fatalf("failed to force version: %v", err)
		}
		fmt.Printf("forced to version %d\n", *force)
	case *up:
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("failed to run up migrations: %v", err)
		}
		fmt.Println("migrations applied successfully")
	case *down:
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("failed to run down migrations: %v", err)
		}
		fmt.Println("migrations reverted successfully")
	case *steps != 0:
		if err := m.Steps(*steps); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("failed to run migrations: %v", err)
		}
		fmt.Printf("applied %d migration steps\n", *steps)
	default:
		fmt.Println("usage: migrate -dsn <connection-string> [-up|-down|-steps N|-version|-force N}]")
		flag.PrintDefaults()
	}
}
