// Package database provides PostgreSQL connection management with lifecycle coordination.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/JaimeStill/herald/pkg/lifecycle"
)

// System manages database connections and lifecycle coordination.
type System interface {
	// Connection returns the underlying database connection pool.
	Connection() *sql.DB
	// Start registers startup and shutdown hooks with the lifecycle coordinator.
	Start(lc *lifecycle.Coordinator) error
}

type database struct {
	conn        *sql.DB
	logger      *slog.Logger
	connTimeout time.Duration
}

// New creates a database system with the given configuration.
// It calls sql.Open to validate the DSN and configure pool parameters,
// but does not establish a connection until Start is called.
func New(cfg *Config, logger *slog.Logger) (System, error) {
	db, err := sql.Open("pgx", cfg.Dsn())
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetimeDuration())

	return &database{
		conn:        db,
		logger:      logger.With("system", "database"),
		connTimeout: cfg.ConnTimeoutDuration(),
	}, nil
}

// NewWithCredential creates a database system using an Azure token credential.
// It configures a BeforeConnect hook that acquires a fresh Entra token on each
// new connection. ConnMaxLifetime is set to TokenLifetime (default 45m) to force
// connection recycling before token expiry.
func NewWithCredential(
	cfg *Config,
	cred azcore.TokenCredential,
	logger *slog.Logger,
) (System, error) {
	connConfig, err := pgx.ParseConfig(cfg.Dsn())
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}

	beforeConnect := stdlib.OptionBeforeConnect(
		func(ctx context.Context, cc *pgx.ConnConfig) error {
			token, err := cred.GetToken(ctx, policy.TokenRequestOptions{
				Scopes: []string{cfg.TokenScope},
			})
			if err != nil {
				return fmt.Errorf("acquire database token: %w", err)
			}

			cc.Password = token.Token
			return nil
		},
	)

	db := stdlib.OpenDB(*connConfig, beforeConnect)

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.TokenLifetimeDuration())

	return &database{
		conn:        db,
		logger:      logger.With("system", "database"),
		connTimeout: cfg.ConnTimeoutDuration(),
	}, nil
}

func (d *database) Connection() *sql.DB {
	return d.conn
}

func (d *database) Start(lc *lifecycle.Coordinator) error {
	d.logger.Info("starting database connection")

	lc.OnStartup(func() {
		pingCtx, cancel := context.WithTimeout(lc.Context(), d.connTimeout)
		defer cancel()

		if err := d.conn.PingContext(pingCtx); err != nil {
			d.logger.Error("database ping failed", "error", err)
			return
		}

		d.logger.Info("database connection established")
	})

	lc.OnShutdown(func() {
		<-lc.Context().Done()
		d.logger.Info("closing database connection")

		if err := d.conn.Close(); err != nil {
			d.logger.Error("database close failed", "error", err)
			return
		}

		d.logger.Info("database connection closed")
	})

	return nil
}
