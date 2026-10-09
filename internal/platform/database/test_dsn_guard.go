package database

import (
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

const developmentDatabaseName = "tabungan_siswa_dev"

var allowedTestDatabaseNames = map[string]struct{}{
	"tabungan_siswa_fase168_test":       {},
	"tabungan_siswa_fase168_clean_test": {},
}

// ValidateTestDatabaseDSN rejects test connection strings that cannot be
// parsed by the PostgreSQL driver or that target the development database.
// It must be called before opening a database connection in integration tests.
func ValidateTestDatabaseDSN(dsn string) error {
	if strings.TrimSpace(dsn) == "" {
		return errors.New("DATABASE_URL is required")
	}

	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		return errors.New("DATABASE_URL must be a valid PostgreSQL connection string")
	}

	if strings.TrimSpace(config.Database) == "" {
		return errors.New("DATABASE_URL must identify a database")
	}

	databaseName := strings.ToLower(config.Database)
	if databaseName == developmentDatabaseName {
		return errors.New("integration tests are not allowed to use the development database")
	}
	if _, allowed := allowedTestDatabaseNames[databaseName]; !allowed {
		return errors.New("DATABASE_URL must target an explicitly approved test database")
	}

	return nil
}
