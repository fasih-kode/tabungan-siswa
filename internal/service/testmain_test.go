package service_test

import (
	"context"
	"os"
	"testing"

	"github.com/fasih/tabungan-siswa/internal/platform/database"
)

const (
	testAdvisoryLockKey1 int64 = 20260930
	testAdvisoryLockKey2 int64 = 12
)

func TestMain(m *testing.M) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		panic("DATABASE_URL is not set")
	}

	db, err := database.Open(dsn)
	if err != nil {
		panic("open test database: " + err.Error())
	}

	ctx := context.Background()

	if err := database.Ping(ctx, db); err != nil {
		_ = db.Close()
		panic("ping test database: " + err.Error())
	}

	conn, err := db.Conn(ctx)
	if err != nil {
		_ = db.Close()
		panic("get test database connection: " + err.Error())
	}

	if _, err := conn.ExecContext(
		ctx,
		"SELECT pg_advisory_lock($1, $2)",
		testAdvisoryLockKey1,
		testAdvisoryLockKey2,
	); err != nil {
		_ = conn.Close()
		_ = db.Close()
		panic("acquire test database advisory lock: " + err.Error())
	}

	code := m.Run()

	if _, err := conn.ExecContext(
		ctx,
		"SELECT pg_advisory_unlock($1, $2)",
		testAdvisoryLockKey1,
		testAdvisoryLockKey2,
	); err != nil {
		code = 1
	}

	_ = conn.Close()

	if err := database.Close(db); err != nil {
		code = 1
	}

	os.Exit(code)
}
