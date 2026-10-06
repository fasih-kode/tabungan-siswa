package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/platform/database"
)

func TestOpenRejectsEmptyDSN(t *testing.T) {
	db, err := database.Open("")
	if err == nil {
		t.Fatal("expected error for empty DSN")
	}

	if db != nil {
		t.Fatal("expected nil database on open failure")
	}
}

func TestPingRejectsNilDatabase(t *testing.T) {
	if err := database.Ping(context.Background(), nil); err == nil {
		t.Fatal("expected error for nil database")
	}
}

func TestOpenAndPingClosesDatabaseWhenPingFails(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	db, err := database.OpenAndPing(
		ctx,
		"postgres://invalid:invalid@127.0.0.1:1/tabungan_siswa?sslmode=disable",
	)
	if err == nil {
		t.Fatal("expected ping error")
	}

	if db != nil {
		t.Fatal("expected nil database when ping fails")
	}
}

func TestCloseAcceptsNilDatabase(t *testing.T) {
	if err := database.Close(nil); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}
