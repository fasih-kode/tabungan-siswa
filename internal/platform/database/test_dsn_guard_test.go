package database_test

import (
	"testing"

	"github.com/fasih/tabungan-siswa/internal/platform/database"
)

func TestValidateTestDatabaseDSN(t *testing.T) {
	tests := []struct {
		name    string
		dsn     string
		wantErr bool
	}{
		{
			name: "test database URL is accepted",
			dsn:  "postgres://tabungan_siswa:test-password@127.0.0.1:5432/tabungan_siswa_fase168_test?sslmode=disable",
		},
		{
			name: "clean migration test database URL is accepted",
			dsn:  "postgres://tabungan_siswa:test-password@127.0.0.1:5432/tabungan_siswa_fase168_clean_test?sslmode=disable",
		},
		{
			name:    "development database URL is rejected",
			dsn:     "postgres://tabungan_siswa:test-password@127.0.0.1:5432/tabungan_siswa_dev?sslmode=disable",
			wantErr: true,
		},
		{
			name:    "development database name comparison is case insensitive",
			dsn:     "postgres://tabungan_siswa:test-password@127.0.0.1:5432/TABUNGAN_SISWA_DEV?sslmode=disable",
			wantErr: true,
		},
		{
			name:    "development database selected through query parameter is rejected",
			dsn:     "postgres://tabungan_siswa:test-password@127.0.0.1:5432/tabungan_siswa_fase168_test?dbname=tabungan_siswa_dev&sslmode=disable",
			wantErr: true,
		},
		{
			name:    "development database keyword DSN is rejected",
			dsn:     "host=127.0.0.1 port=5432 user=tabungan_siswa dbname=tabungan_siswa_dev sslmode=disable",
			wantErr: true,
		},
		{
			name: "test database keyword DSN is accepted",
			dsn:  "host=127.0.0.1 port=5432 user=tabungan_siswa dbname=tabungan_siswa_fase168_test sslmode=disable",
		},
		{
			name:    "empty DSN is rejected",
			dsn:     "",
			wantErr: true,
		},
		{
			name:    "invalid DSN is rejected",
			dsn:     "://not-a-url",
			wantErr: true,
		},
		{
			name:    "unsupported scheme is rejected",
			dsn:     "mysql://user:password@127.0.0.1:3306/test_db",
			wantErr: true,
		},
		{
			name:    "unknown database is rejected",
			dsn:     "postgres://user:password@127.0.0.1:5432/some_other_database?sslmode=disable",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := database.ValidateTestDatabaseDSN(tt.dsn)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateTestDatabaseDSN() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
