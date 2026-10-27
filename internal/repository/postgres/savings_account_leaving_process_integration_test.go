package postgres_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/platform/database"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/repository/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestSavingsAccountLeavingProcessV17(t *testing.T) {
	db := openTestDatabase(t)
	ctx := context.Background()
	requireSchemaVersion17(t, db)

	t.Run("active blocks account in same and different year", func(t *testing.T) {
		for _, differentYear := range []bool{false, true} {
			t.Run(fmt.Sprintf("different_year_%t", differentYear), func(t *testing.T) {
				f := newSavingsAccountFixture(t, db)
				yearID := f.academicYear.ID
				if !differentYear {
					existing, err := domain.NewSavingsAccount(f.student.ID, f.academicYear.ID)
					if err != nil {
						t.Fatal(err)
					}
					if err := postgres.NewSavingsAccountRepository(db).Create(ctx, existing); err != nil {
						t.Fatal(err)
					}
					t.Cleanup(func() { deleteTestSavingsAccount(t, db, existing.ID) })
					for _, transaction := range []domain.Transaction{
						mustV17Transaction(t, existing.ID, f.user.ID, domain.TransactionDeposit, 9_000),
						mustV17Transaction(t, existing.ID, f.user.ID, domain.TransactionWithdrawal, 2_000),
					} {
						if err := postgres.NewTransactionRepository(db).Create(ctx, transaction); err != nil {
							t.Fatal(err)
						}
						t.Cleanup(func() {
							if _, err := db.ExecContext(ctx, `DELETE FROM transactions WHERE id=$1`, transaction.ID); err != nil {
								t.Errorf("delete transaction fixture: %v", err)
							}
						})
					}
				}
				request := newV17LeavingRequest(t, db, db, f, false)
				if differentYear {
					otherYear := v17CreateSecondOpenYear(t, db, f)
					yearID = otherYear.ID
				}
				before := v17FinancialCounts(t, db, f.student.ID)
				account, err := domain.NewSavingsAccount(f.student.ID, yearID)
				if err != nil {
					t.Fatal(err)
				}
				err = postgres.NewSavingsAccountRepository(db).Create(ctx, account)
				if !errors.Is(err, domain.ErrInvalidTransition) {
					t.Fatalf("repository Create error=%v, want ErrInvalidTransition", err)
				}
				var pgErr *pgconn.PgError
				_, directErr := db.ExecContext(ctx, `INSERT INTO savings_accounts (id, student_id, academic_year_id, status) VALUES ($1,$2,$3,'OPEN')`, uuid.New(), f.student.ID, yearID)
				if !errors.As(directErr, &pgErr) || pgErr.Code != "23514" || pgErr.ConstraintName != "savings_accounts_student_leaving_in_progress" {
					t.Fatalf("direct INSERT error=%v, want 23514 savings_accounts_student_leaving_in_progress", directErr)
				}
				var accountCount, activeCount int
				if err := db.QueryRowContext(ctx, `SELECT count(*) FROM savings_accounts WHERE student_id=$1`, f.student.ID).Scan(&accountCount); err != nil {
					t.Fatal(err)
				}
				if err := db.QueryRowContext(ctx, `SELECT count(*) FROM student_leaving_requests WHERE student_id=$1 AND status='IN_PROGRESS'`, f.student.ID).Scan(&activeCount); err != nil {
					t.Fatal(err)
				}
				if accountCount != before.accounts || activeCount != 1 || !before.equal(v17FinancialCounts(t, db, f.student.ID)) {
					t.Fatalf("rejection changed account/process/financial state: accounts=%d active=%d request=%s", accountCount, activeCount, request.ID)
				}
			})
		}
	})

	t.Run("cancelled process permits account and existing account transactions", func(t *testing.T) {
		f := newSavingsAccountFixture(t, db)
		request := newV17LeavingRequest(t, db, db, f, false)
		if err := request.Cancel(f.user.ID, time.Now(), "pindah dibatalkan"); err != nil {
			t.Fatal(err)
		}
		if err := postgres.NewStudentLeavingRequestRepository(db).Update(ctx, request); err != nil {
			t.Fatal(err)
		}
		account, err := domain.NewSavingsAccount(f.student.ID, f.academicYear.ID)
		if err != nil {
			t.Fatal(err)
		}
		if err := postgres.NewSavingsAccountRepository(db).Create(ctx, account); err != nil {
			t.Fatalf("cancelled process should not block account: %v", err)
		}
		t.Cleanup(func() { deleteTestSavingsAccount(t, db, account.ID) })
		for _, transaction := range []domain.Transaction{
			mustV17Transaction(t, account.ID, f.user.ID, domain.TransactionDeposit, 5_000),
			mustV17Transaction(t, account.ID, f.user.ID, domain.TransactionWithdrawal, 1_000),
		} {
			if err := postgres.NewTransactionRepository(db).Create(ctx, transaction); err != nil {
				t.Fatalf("transaction on existing OPEN account should remain allowed: %v", err)
			}
			t.Cleanup(func() {
				if _, err := db.ExecContext(ctx, `DELETE FROM transactions WHERE id=$1`, transaction.ID); err != nil {
					t.Errorf("delete transaction fixture: %v", err)
				}
			})
		}
	})

	t.Run("legacy eligibility and duplicate errors remain unchanged", func(t *testing.T) {
		t.Run("left student", func(t *testing.T) {
			f := newSavingsAccountFixture(t, db)
			if _, err := db.ExecContext(ctx, `UPDATE students SET status='LEFT' WHERE id=$1`, f.student.ID); err != nil {
				t.Fatal(err)
			}
			account, _ := domain.NewSavingsAccount(f.student.ID, f.academicYear.ID)
			if err := postgres.NewSavingsAccountRepository(db).Create(ctx, account); !errors.Is(err, domain.ErrInvalidTransition) {
				t.Fatalf("Create error=%v", err)
			}
		})
		for _, status := range []domain.AcademicYearStatus{domain.AcademicYearClosing, domain.AcademicYearClosed} {
			t.Run(string(status)+" year", func(t *testing.T) {
				f := newSavingsAccountFixture(t, db)
				if _, err := db.ExecContext(ctx, `UPDATE academic_years SET status=$1 WHERE id=$2`, status, f.academicYear.ID); err != nil {
					t.Fatal(err)
				}
				account, _ := domain.NewSavingsAccount(f.student.ID, f.academicYear.ID)
				if err := postgres.NewSavingsAccountRepository(db).Create(ctx, account); !errors.Is(err, domain.ErrInvalidTransition) {
					t.Fatalf("Create(%s year) error=%v", status, err)
				}
			})
		}
		t.Run("duplicate account", func(t *testing.T) {
			f := newSavingsAccountFixture(t, db)
			first, _ := domain.NewSavingsAccount(f.student.ID, f.academicYear.ID)
			second, _ := domain.NewSavingsAccount(f.student.ID, f.academicYear.ID)
			if err := postgres.NewSavingsAccountRepository(db).Create(ctx, first); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { deleteTestSavingsAccount(t, db, first.ID) })
			if err := postgres.NewSavingsAccountRepository(db).Create(ctx, second); !errors.Is(err, repository.ErrConflict) {
				t.Fatalf("duplicate Create error=%v, want ErrConflict", err)
			}
		})
	})
}

func TestSavingsAccountCreationDuringLeavingV16(t *testing.T) {
	db := openTestDatabase(t)
	ctx := context.Background()
	var version int
	var dirty bool
	if err := db.QueryRowContext(ctx, `SELECT version, dirty FROM schema_migrations LIMIT 1`).Scan(&version, &dirty); err != nil {
		t.Fatal(err)
	}
	if version != 16 || dirty {
		t.Skipf("requires clean schema v16 after rollback; found %d|%t", version, dirty)
	}
	f := newSavingsAccountFixture(t, db)
	newV17LeavingRequest(t, db, db, f, false)
	account, err := domain.NewSavingsAccount(f.student.ID, f.academicYear.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := postgres.NewSavingsAccountRepository(db).Create(ctx, account); err != nil {
		t.Fatalf("v16 trigger should retain its prior behavior after rollback: %v", err)
	}
	t.Cleanup(func() { deleteTestSavingsAccount(t, db, account.ID) })
}

func TestSavingsAccountLeavingProcessConcurrencyV17(t *testing.T) {
	db := openTestDatabase(t)
	requireSchemaVersion17(t, db)
	for _, scenario := range []string{"start_first_same_year", "account_first", "start_first_different_year", "cancel_first", "year_closes", "student_leaves"} {
		t.Run(scenario, func(t *testing.T) { runSavingsAccountLeavingConcurrencyV17(t, db, scenario) })
	}
}

func runSavingsAccountLeavingConcurrencyV17(t *testing.T, db *sql.DB, scenario string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	f := newSavingsAccountFixture(t, db)
	yearID := f.academicYear.ID
	if scenario == "start_first_different_year" {
		_ = newV17LeavingRequest(t, db, db, f, false)
		otherYear := v17CreateSecondOpenYear(t, db, f)
		yearID = otherYear.ID
	}
	account, err := domain.NewSavingsAccount(f.student.ID, yearID)
	if err != nil {
		t.Fatal(err)
	}
	startTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer startTx.Rollback()
	var blockerPID int
	if err := startTx.QueryRowContext(ctx, `SELECT pg_backend_pid()`).Scan(&blockerPID); err != nil {
		t.Fatal(err)
	}
	var request domain.StudentLeavingRequest
	var accountTx *sql.Tx
	if scenario == "account_first" {
		accountTx, err = db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer accountTx.Rollback()
		if err := postgres.NewSavingsAccountRepository(accountTx).Create(ctx, account); err != nil {
			t.Fatal(err)
		}
		var accountPID int
		if err := accountTx.QueryRowContext(ctx, `SELECT pg_backend_pid()`).Scan(&accountPID); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { deleteTestSavingsAccount(t, db, account.ID) })
		var startPID int
		started := make(chan int, 1)
		startResult := make(chan error, 1)
		go func() { startResult <- lockStartAndCreateV17(ctx, db, f, &request, started) }()
		select {
		case startPID = <-started:
		case <-ctx.Done():
			t.Fatalf("Start did not begin: %v", ctx.Err())
		}
		waitV17BlockedBy(t, db, startPID, accountPID, startResult)
		if err := accountTx.Commit(); err != nil {
			t.Fatal(err)
		}
		if err := <-startResult; err != nil {
			t.Fatalf("Start after account commit: %v", err)
		}
		t.Cleanup(func() {
			if _, err := db.ExecContext(context.Background(), `DELETE FROM student_leaving_requests WHERE id=$1`, request.ID); err != nil {
				t.Errorf("delete Start fixture: %v", err)
			}
		})
		return
	}

	if scenario == "cancel_first" {
		request = newV17LeavingRequest(t, db, db, f, false)
		if err := startTx.QueryRowContext(ctx, `SELECT id FROM students WHERE id=$1 FOR UPDATE`, f.student.ID).Scan(new(uuid.UUID)); err != nil {
			t.Fatal(err)
		}
		locked, err := postgres.NewStudentLeavingRequestRepository(startTx).GetByIDForUpdate(ctx, request.ID)
		if err != nil {
			t.Fatal(err)
		}
		if err := locked.Cancel(f.user.ID, time.Now(), "dibatalkan untuk tes"); err != nil {
			t.Fatal(err)
		}
		if err := postgres.NewStudentLeavingRequestRepository(startTx).Update(ctx, locked); err != nil {
			t.Fatal(err)
		}
	} else if scenario == "year_closes" {
		if _, err := startTx.ExecContext(ctx, `SELECT id FROM academic_years WHERE id=$1 FOR UPDATE`, f.academicYear.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := startTx.ExecContext(ctx, `UPDATE academic_years SET status='CLOSING' WHERE id=$1`, f.academicYear.ID); err != nil {
			t.Fatal(err)
		}
	} else if scenario == "student_leaves" {
		if _, err := startTx.ExecContext(ctx, `SELECT id FROM students WHERE id=$1 FOR UPDATE`, f.student.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := startTx.ExecContext(ctx, `UPDATE students SET status='LEFT' WHERE id=$1`, f.student.ID); err != nil {
			t.Fatal(err)
		}
	} else {
		if _, err := startTx.ExecContext(ctx, `SELECT id FROM academic_years WHERE id=$1 FOR UPDATE`, f.academicYear.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := startTx.ExecContext(ctx, `SELECT id FROM students WHERE id=$1 FOR UPDATE`, f.student.ID); err != nil {
			t.Fatal(err)
		}
		if scenario != "start_first_different_year" {
			request = newV17LeavingRequest(t, db, startTx, f, false)
		}
	}

	accountTx, err = db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer accountTx.Rollback()
	var waitingPID int
	if err := accountTx.QueryRowContext(ctx, `SELECT pg_backend_pid()`).Scan(&waitingPID); err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() { result <- postgres.NewSavingsAccountRepository(accountTx).Create(ctx, account) }()
	waitV17BlockedBy(t, db, waitingPID, blockerPID, result)
	if err := startTx.Commit(); err != nil {
		t.Fatal(err)
	}
	err = <-result
	if scenario == "cancel_first" {
		if err != nil {
			t.Fatalf("account after CANCELLED commit: %v", err)
		}
		if err := accountTx.Commit(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { deleteTestSavingsAccount(t, db, account.ID) })
		return
	}
	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("account after lifecycle change error=%v, want ErrInvalidTransition", err)
	}
	if err := accountTx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		t.Fatal(err)
	}
}

func lockStartAndCreateV17(ctx context.Context, db *sql.DB, f savingsAccountFixture, out *domain.StudentLeavingRequest, started chan<- int) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var pid int
	if err := tx.QueryRowContext(ctx, `SELECT pg_backend_pid()`).Scan(&pid); err != nil {
		return err
	}
	started <- pid
	if _, err := tx.ExecContext(ctx, `SELECT id FROM academic_years WHERE id=$1 FOR UPDATE`, f.academicYear.ID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `SELECT id FROM students WHERE id=$1 FOR UPDATE`, f.student.ID); err != nil {
		return err
	}
	request, err := domain.NewStudentLeavingRequest(f.student.ID, f.academicYear.ID, f.academicYear.StartDate, f.user.ID)
	if err != nil {
		return err
	}
	if err := postgres.NewStudentLeavingRequestRepository(tx).Create(ctx, request); err != nil {
		return err
	}
	*out = request
	return tx.Commit()
}

func waitV17BlockedBy(t *testing.T, db *sql.DB, waitingPID, blockingPID int, result <-chan error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var blocked bool
		err := db.QueryRowContext(ctx, `SELECT pg_blocking_pids($1) @> ARRAY[$2]::integer[] FROM pg_stat_activity WHERE pid=$1`, waitingPID, blockingPID).Scan(&blocked)
		if err != nil {
			t.Fatalf("observe PostgreSQL blocking: %v", err)
		}
		if blocked {
			return
		}
		select {
		case err := <-result:
			t.Fatalf("operation completed before expected lock wait: %v", err)
		case <-ticker.C:
		case <-ctx.Done():
			t.Fatalf("backend %d was not blocked by %d", waitingPID, blockingPID)
		}
	}
}

func requireSchemaVersion17(t *testing.T, db *sql.DB) {
	t.Helper()
	var version int
	var dirty bool
	if err := db.QueryRowContext(context.Background(), `SELECT version, dirty FROM schema_migrations LIMIT 1`).Scan(&version, &dirty); err != nil {
		t.Fatal(err)
	}
	if version != 17 || dirty {
		t.Skipf("requires clean schema v17; found %d|%t", version, dirty)
	}
}

func v17CreateSecondOpenYear(t *testing.T, db *sql.DB, f savingsAccountFixture) domain.AcademicYear {
	t.Helper()
	ctx := context.Background()
	repo := postgres.NewAcademicYearRepository(db)
	previous, err := repo.GetByID(ctx, f.academicYear.ID)
	if err != nil {
		t.Fatal(err)
	}
	previous.Status = domain.AcademicYearClosed
	previous.UpdatedAt = time.Now().UTC()
	if err := repo.Update(ctx, previous); err != nil {
		t.Fatalf("close fixture academic year: %v", err)
	}
	other := newTestAcademicYearWithDates(t, "2027/2028", time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC), time.Date(2028, 7, 1, 0, 0, 0, 0, time.UTC), domain.AcademicYearOpen)
	if err := repo.Create(ctx, other); err != nil {
		t.Fatalf("create second open fixture year: %v", err)
	}
	t.Cleanup(func() { deleteTestAcademicYear(t, db, other.ID) })
	return other
}

func newV17LeavingRequest(t *testing.T, cleanupDB *sql.DB, queryer database.DBTX, f savingsAccountFixture, cancelled bool) domain.StudentLeavingRequest {
	t.Helper()
	request, err := domain.NewStudentLeavingRequest(f.student.ID, f.academicYear.ID, f.academicYear.StartDate, f.user.ID)
	if err != nil {
		t.Fatal(err)
	}
	repo := postgres.NewStudentLeavingRequestRepository(queryer)
	if err := repo.Create(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if cancelled {
		if err := request.Cancel(f.user.ID, time.Now(), "proses dibatalkan"); err != nil {
			t.Fatal(err)
		}
		if err := repo.Update(context.Background(), request); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		if _, err := cleanupDB.ExecContext(context.Background(), `DELETE FROM student_leaving_requests WHERE id=$1`, request.ID); err != nil {
			t.Errorf("delete leaving request fixture: %v", err)
		}
	})
	return request
}

func v17FinancialCounts(t *testing.T, db *sql.DB, studentID uuid.UUID) v17Counts {
	t.Helper()
	var result v17Counts
	err := db.QueryRowContext(context.Background(), `SELECT (SELECT count(*) FROM savings_accounts WHERE student_id=$1), (SELECT count(*) FROM savings_settlements s JOIN savings_accounts a ON a.id=s.savings_account_id WHERE a.student_id=$1), (SELECT count(*) FROM transactions x JOIN savings_accounts a ON a.id=x.savings_account_id WHERE a.student_id=$1), (SELECT count(*) FROM audit_logs WHERE entity_id=$1)`, studentID).Scan(&result.accounts, &result.settlements, &result.transactions, &result.audits)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

type v17Counts struct{ accounts, settlements, transactions, audits int }

func (c v17Counts) equal(other v17Counts) bool { return c == other }

func mustV17Transaction(t *testing.T, accountID, actorID uuid.UUID, typ domain.TransactionType, amount domain.Money) domain.Transaction {
	t.Helper()
	transaction, err := domain.NewTransaction(accountID, typ, amount, time.Now(), actorID)
	if err != nil {
		t.Fatal(err)
	}
	return transaction
}
