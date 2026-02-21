# 13 - Migration CLI and Initial Schema

## Problem Context

Herald needs a database migration workflow before the document domain (Objective #3) can be built. The `cmd/migrate/main.go` exists as a stub. This task implements the standalone golang-migrate CLI with embedded SQL files and the initial PostgreSQL schema (UUID extension + documents table).

## Architecture Approach

The migration CLI is standalone — it does not import `internal/config/` or `pkg/database/`. Database connection is via `-dsn` flag or `DATABASE_DSN` env var, with a default DSN matching `config.toml` and Docker Compose defaults. Migrations are embedded in the binary via `//go:embed` and loaded with `iofs.New()`. This follows agent-lab's proven `cmd/migrate/` pattern exactly.

## Implementation

### Step 1: Add golang-migrate dependency

```bash
go get github.com/golang-migrate/migrate/v4
go get github.com/golang-migrate/migrate/v4/database/postgres
go get github.com/golang-migrate/migrate/v4/source/iofs
```

Then run `go mod tidy` to clean up.

### Step 2: Implement `cmd/migrate/main.go`

Replace the stub with the full migration CLI:

```go
package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrations embed.FS

const (
	envDatabaseDSN = "HERALD_DB_DSN"
	defaultDSN     = "postgres://herald:herald@localhost:5432/herald?sslmode=disable"
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

	if *dsn == "" {
		*dsn = os.Getenv(envDatabaseDSN)
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
	case *force >= 0:
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
		fmt.Println("usage: migrate -dsn <connection-string> [-up|-down|-steps N|-version|-force N]")
		flag.PrintDefaults()
	}
}
```

### Step 3: Create SQL migration files

Create the `cmd/migrate/migrations/` directory, then add the initial schema migration.

**`cmd/migrate/migrations/000001_initial_schema.up.sql`**

```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_id INTEGER NOT NULL,
    external_platform TEXT NOT NULL,
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    page_count INTEGER,
    storage_key TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'review', 'complete')),
    uploaded_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_documents_status ON documents(status);
CREATE INDEX idx_documents_filename ON documents(filename);
CREATE INDEX idx_documents_uploaded_at ON documents(uploaded_at DESC);
CREATE INDEX idx_documents_external ON documents(external_platform, external_id);
```

**`cmd/migrate/migrations/000001_initial_schema.down.sql`**

```sql
DROP TABLE IF EXISTS documents;
DROP EXTENSION IF EXISTS "uuid-ossp";
```

### Step 4: Add mise tasks

Add the following to `.mise.toml`:

```toml
[tasks."migrate:up"]
description = "Run all up migrations"
run = "go run ./cmd/migrate -up"

[tasks."migrate:down"]
description = "Run all down migrations"
run = "go run ./cmd/migrate -down"

[tasks."migrate:version"]
description = "Print current migration version"
run = "go run ./cmd/migrate -version"
```

## Remediation

### R1: Force flag sentinel value conflict

The `-force` flag used `-1` as a default sentinel, but `-1` is also a valid argument (force to no version). The switch condition `*force >= 0` silently ignored `-force -1`. Fix: use `flag.Visit` after `flag.Parse()` to detect whether `-force` was explicitly provided, then switch on `forceSet` instead of checking the value.

## Validation Criteria

- [ ] `go build ./cmd/migrate/` succeeds
- [ ] `go vet ./...` passes
- [ ] `mise run migrate:up` applies the migration against local PostgreSQL
- [ ] `mise run migrate:down` cleanly reverts the migration
- [ ] `psql` confirms table structure, column types, and indexes
- [ ] `go mod tidy` produces no changes
