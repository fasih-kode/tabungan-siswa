package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// DBTX adalah execution interface yang dapat dipenuhi oleh
// *sql.DB maupun *sql.Tx.
//
// Interface ini sengaja berada di infrastructure layer.
// Repository contract tidak bergantung pada database/sql.
type DBTX interface {
	ExecContext(
		ctx context.Context,
		query string,
		args ...any,
	) (sql.Result, error)

	QueryContext(
		ctx context.Context,
		query string,
		args ...any,
	) (*sql.Rows, error)

	QueryRowContext(
		ctx context.Context,
		query string,
		args ...any,
	) *sql.Row
}

const defaultPingTimeout = 5 * time.Second

// Open membuka connection pool PostgreSQL menggunakan database/sql.
//
// Fungsi ini tidak melakukan ping.
// Caller harus memanggil Ping secara eksplisit.
func Open(dsn string) (*sql.DB, error) {
	if dsn == "" {
		return nil, errors.New("database DSN is required")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	return db, nil
}

// Ping memastikan database dapat diakses.
func Ping(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return errors.New("database connection is nil")
	}

	pingCtx := ctx

	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		pingCtx, cancel = context.WithTimeout(ctx, defaultPingTimeout)
		defer cancel()
	}

	if err := db.PingContext(pingCtx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	return nil
}

// Close menutup database pool.
func Close(db *sql.DB) error {
	if db == nil {
		return nil
	}

	if err := db.Close(); err != nil {
		return fmt.Errorf("close database: %w", err)
	}

	return nil
}
