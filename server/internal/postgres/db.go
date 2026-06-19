// Package postgres holds the Postgres adapters for the ergonomos ports
// (ADR-0008). It implements auth.Store, task.Store, authz.Authorizer, and
// events.Outbox over a pgx connection pool, and provides the transaction
// boundary the transactional outbox relies on (ADR-0003).
//
// The driver lives only here and in cmd/ergonomos; the domain and application
// services never import it.
package postgres

import (
	"context"
	"embed"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// querier is the subset of pgx shared by the pool and a transaction, so an
// adapter can run the same statements whether or not a transaction is active.
type querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// DB owns the connection pool and is the unit-of-work boundary (ADR-0008).
type DB struct {
	pool *pgxpool.Pool
	url  string
}

// Open connects to url and verifies the connection.
func Open(ctx context.Context, url string) (*DB, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("postgres: connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: ping: %w", err)
	}
	return &DB{pool: pool, url: url}, nil
}

// Close releases the pool.
func (db *DB) Close() { db.pool.Close() }

// Migrate applies all pending migrations. It uses a short-lived database/sql
// handle because goose speaks database/sql; the application itself uses pgx.
func (db *DB) Migrate(ctx context.Context) error {
	sqlDB := stdlib.OpenDBFromPool(db.pool)
	defer sqlDB.Close()
	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("postgres: goose dialect: %w", err)
	}
	if err := goose.UpContext(ctx, sqlDB, "migrations"); err != nil {
		return fmt.Errorf("postgres: migrate: %w", err)
	}
	return nil
}

// txKey is the context key under which an active transaction is stored.
type txKey struct{}

// WithinTx runs fn inside a single transaction (ADR-0008). The transaction is
// carried in the context so adapter methods pick it up via q(); fn's
// operations commit together or roll back together. A nested WithinTx joins the
// existing transaction rather than starting a new one.
func (db *DB) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres: begin: %w", err)
	}
	if err := fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	return tx.Commit(ctx)
}

// q returns the ambient transaction if one is active, otherwise the pool.
func (db *DB) q(ctx context.Context) querier {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return db.pool
}

// isUniqueViolation reports whether err is a Postgres unique-constraint error.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
