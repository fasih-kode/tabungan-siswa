package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/fasih/tabungan-siswa/internal/repository"
)

var (
	_ repository.UnitOfWork        = (*PostgresUnitOfWork)(nil)
	_ repository.UnitOfWorkManager = (*PostgresUnitOfWorkManager)(nil)
)

// PostgresUnitOfWork mengikat seluruh repository pada satu
// database transaction PostgreSQL yang sama.
type PostgresUnitOfWork struct {
	tx    *sql.Tx
	repos repository.RepositorySet
}

// PostgresUnitOfWorkManager membuat UnitOfWork menggunakan
// database connection pool PostgreSQL.
type PostgresUnitOfWorkManager struct {
	db *sql.DB
}

// NewUnitOfWorkManager membuat UnitOfWorkManager baru.
func NewUnitOfWorkManager(db *sql.DB) *PostgresUnitOfWorkManager {
	return &PostgresUnitOfWorkManager{
		db: db,
	}
}

// Begin memulai database transaction baru dan membuat seluruh
// repository menggunakan transaction tersebut.
func (m *PostgresUnitOfWorkManager) Begin(
	ctx context.Context,
) (repository.UnitOfWork, error) {
	if m == nil || m.db == nil {
		return nil, errors.New("database connection is nil")
	}

	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}

	repos := NewRepositorySet(tx)

	return &PostgresUnitOfWork{
		tx:    tx,
		repos: repos,
	}, nil
}

// Repositories mengembalikan seluruh repository yang menggunakan
// database transaction yang sama.
func (u *PostgresUnitOfWork) Repositories() repository.RepositorySet {
	return u.repos
}

// Commit menyelesaikan database transaction.
func (u *PostgresUnitOfWork) Commit() error {
	if u == nil || u.tx == nil {
		return errors.New("transaction is nil")
	}

	if err := u.tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// Rollback membatalkan database transaction.
func (u *PostgresUnitOfWork) Rollback() error {
	if u == nil || u.tx == nil {
		return errors.New("transaction is nil")
	}

	if err := u.tx.Rollback(); err != nil {
		return fmt.Errorf("rollback transaction: %w", err)
	}

	return nil
}
