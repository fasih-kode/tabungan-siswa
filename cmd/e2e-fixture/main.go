// Command e2e-fixture creates and removes a deterministic browser-test
// fixture. It is restricted to the explicitly named disposable test database.
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/fasih/tabungan-siswa/internal/platform/database"
	"github.com/fasih/tabungan-siswa/internal/security"
)

const (
	databaseDSNEnv = "TABUNGAN_SISWA_DATABASE_DSN"
	testDatabase   = "tabungan_siswa_fase168_clean_test"
	testPassword   = "E2E-Only-Fase23-2026"
)

const (
	adminID   = "f2300000-0000-4000-8000-000000000001"
	waliID    = "f2300000-0000-4000-8000-000000000002"
	studentID = "f2300000-0000-4000-8000-000000000010"
	yearID    = "f2300000-0000-4000-8000-000000000020"
	classID   = "f2300000-0000-4000-8000-000000000030"
	accountID = "f2300000-0000-4000-8000-000000000040"
	depositID = "f2300000-0000-4000-8000-000000000050"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "e2e-fixture:", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) != 2 || (os.Args[1] != "setup" && os.Args[1] != "cleanup") {
		return errors.New("usage: go run ./cmd/e2e-fixture {setup|cleanup}")
	}

	dsn := os.Getenv(databaseDSNEnv)
	if err := database.ValidateTestDatabaseDSN(dsn); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := database.OpenAndPing(ctx, dsn)
	if err != nil {
		return fmt.Errorf("connect to disposable database: %w", err)
	}
	defer db.Close()

	if err := verifyTargetDatabase(ctx, db); err != nil {
		return err
	}
	if err := verifySchema(ctx, db); err != nil {
		return err
	}

	if os.Args[1] == "setup" {
		if err := setup(ctx, db); err != nil {
			return err
		}
		fmt.Printf("Fixture FASE 23 siap di %s.\n", testDatabase)
		return nil
	}

	if err := cleanup(ctx, db); err != nil {
		return err
	}
	fmt.Printf("Fixture FASE 23 dibersihkan dari %s.\n", testDatabase)
	return nil
}

func verifyTargetDatabase(ctx context.Context, db *sql.DB) error {
	var databaseName string
	if err := db.QueryRowContext(ctx, `SELECT current_database()`).Scan(&databaseName); err != nil {
		return fmt.Errorf("identify target database: %w", err)
	}
	if databaseName != testDatabase {
		return fmt.Errorf("refusing to touch database %q; expected %q", databaseName, testDatabase)
	}
	return nil
}

func verifySchema(ctx context.Context, db *sql.DB) error {
	var version int
	var dirty bool
	err := db.QueryRowContext(ctx, `SELECT version, dirty FROM schema_migrations LIMIT 1`).Scan(&version, &dirty)
	if err != nil {
		return fmt.Errorf("read migration state: %w", err)
	}
	if version != 13 || dirty {
		return fmt.Errorf("expected schema version 13 clean; got version %d dirty=%t", version, dirty)
	}
	return nil
}

func setup(ctx context.Context, db *sql.DB) error {
	var exists bool
	err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM users WHERE id IN ($1, $2) OR username IN ($3, $4)
			UNION ALL SELECT 1 FROM academic_years WHERE id = $5
			UNION ALL SELECT 1 FROM classes WHERE id = $6
			UNION ALL SELECT 1 FROM students WHERE id = $7
			UNION ALL SELECT 1 FROM savings_accounts WHERE id = $8
		)`, adminID, waliID, "e2e-fase23-admin", "e2e-fase23-wali", yearID, classID, studentID, accountID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check fixture collision: %w", err)
	}
	if exists {
		return errors.New("fixture IDs or usernames already exist; clean only the E2E fixture before retrying")
	}

	hasher, err := security.NewDefaultArgon2idHasher()
	if err != nil {
		return fmt.Errorf("initialize test password hasher: %w", err)
	}
	passwordHash, err := hasher.Hash(testPassword)
	if err != nil {
		return fmt.Errorf("hash test-only password: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin fixture setup: %w", err)
	}
	defer tx.Rollback()

	statements := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO users(id, username, password_hash, role) VALUES ($1,$2,$3,'ADMIN'),($4,$5,$3,'WALI_KELAS')`, []any{adminID, "e2e-fase23-admin", passwordHash, waliID, "e2e-fase23-wali"}},
		{`INSERT INTO academic_years(id,name,start_date,end_date,status) VALUES ($1,'E2E Tahun 2025/2026','2025-01-01','2026-01-01','OPEN')`, []any{yearID}},
		{`INSERT INTO classes(id,name,level) VALUES ($1,'E2E Kelas A',6)`, []any{classID}},
		{`INSERT INTO students(id,name,nis,nisn,status) VALUES ($1,'E2E Siswa','E2E-23-001','E2E-23-NISN','ACTIVE')`, []any{studentID}},
		{`INSERT INTO student_class_histories(student_id,academic_year_id,class_id,validity) VALUES ($1,$2,$3,'[2025-01-01,2026-01-01)')`, []any{studentID, yearID, classID}},
		{`INSERT INTO teacher_class_assignments(user_id,academic_year_id,class_id,validity) VALUES ($1,$2,$3,'[2025-01-01,2026-01-01)')`, []any{waliID, yearID, classID}},
		{`INSERT INTO savings_accounts(id,student_id,academic_year_id,status) VALUES ($1,$2,$3,'OPEN')`, []any{accountID, studentID, yearID}},
		{`INSERT INTO transactions(id,savings_account_id,type,amount,transaction_date,status,created_by) VALUES ($1,$2,'DEPOSIT',2500,'2025-12-15','ACTIVE',$3)`, []any{depositID, accountID, adminID}},
		{`UPDATE academic_years SET status='CLOSING' WHERE id=$1`, []any{yearID}},
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement.query, statement.args...); err != nil {
			return fmt.Errorf("create FASE 23 fixture: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit fixture setup: %w", err)
	}
	return nil
}

func cleanup(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin fixture cleanup: %w", err)
	}
	defer tx.Rollback()

	statements := []string{
		`DELETE FROM audit_logs WHERE actor_user_id IN ($1,$2) OR entity_id IN ($3,$4,$5,$6,$7) OR entity_id IN (SELECT id FROM savings_settlements WHERE savings_account_id=$5) OR entity_id IN (SELECT id FROM settlement_reopen_requests WHERE savings_account_id=$5)`,
		`DELETE FROM settlement_cash_movements WHERE savings_settlement_id IN (SELECT id FROM savings_settlements WHERE savings_account_id=$1)`,
		`DELETE FROM settlement_reopen_requests WHERE savings_account_id=$1`,
		`DELETE FROM savings_settlements WHERE savings_account_id=$1`,
		`DELETE FROM transactions WHERE savings_account_id=$1`,
		`DELETE FROM savings_accounts WHERE id=$1`,
		`DELETE FROM teacher_class_assignments WHERE user_id=$1 AND academic_year_id=$2`,
		`DELETE FROM student_class_histories WHERE student_id=$1 AND academic_year_id=$2`,
		`DELETE FROM students WHERE id=$1`,
		`DELETE FROM classes WHERE id=$1`,
		`DELETE FROM academic_years WHERE id=$1`,
		`DELETE FROM sessions WHERE user_id IN ($1,$2)`,
		`DELETE FROM users WHERE id IN ($1,$2)`,
	}
	args := [][]any{
		{adminID, waliID, yearID, classID, accountID, studentID, depositID},
		{accountID}, {accountID}, {accountID}, {accountID}, {accountID},
		{waliID, yearID}, {studentID, yearID}, {studentID}, {classID}, {yearID},
		{adminID, waliID}, {adminID, waliID},
	}
	for i, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement, args[i]...); err != nil {
			return fmt.Errorf("remove FASE 23 fixture: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit fixture cleanup: %w", err)
	}
	return nil
}
