package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/repository/postgres"
	"github.com/google/uuid"
)

func TestStudentLeavingRequestRepositoryV15(t *testing.T) {
	db := openTestDatabase(t)
	ctx := context.Background()

	var version int
	var dirty bool
	if err := db.QueryRowContext(ctx, `SELECT version, dirty FROM schema_migrations LIMIT 1`).Scan(&version, &dirty); err != nil {
		t.Fatalf("read migration version: %v", err)
	}
	if version != 15 || dirty {
		t.Skipf("requires clean schema v15; found version=%d dirty=%t", version, dirty)
	}

	repo := postgres.NewStudentLeavingRequestRepository(db)
	fixture := newSettlementFixture(t, db)
	t.Cleanup(func() {
		if _, err := db.ExecContext(ctx, `DELETE FROM student_leaving_requests WHERE student_id = $1`, fixture.student.ID); err != nil {
			t.Errorf("delete student leaving request fixtures: %v", err)
		}
	})

	initial, err := domain.NewStudentLeavingRequest(
		fixture.student.ID,
		fixture.year.ID,
		time.Date(2026, 10, 27, 19, 0, 0, 0, time.FixedZone("UTC+7", 7*60*60)),
		fixture.user.ID,
	)
	if err != nil {
		t.Fatalf("NewStudentLeavingRequest() error = %v", err)
	}
	if err := repo.Create(ctx, initial); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := repo.GetByID(ctx, initial.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	assertStudentLeavingRequestEqual(t, initial, got)
	if got.FinalizedBy != nil || got.FinalizedAt != nil || got.CancelledBy != nil || got.CancelledAt != nil || got.CancellationReason != nil {
		t.Fatalf("new request nullable metadata must be nil: %#v", got)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}
	lockedRepo := postgres.NewStudentLeavingRequestRepository(tx)
	locked, err := lockedRepo.GetByIDForUpdate(ctx, initial.ID)
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("GetByIDForUpdate() error = %v", err)
	}
	if locked.ID != initial.ID {
		t.Fatalf("GetByIDForUpdate() ID = %s, want %s", locked.ID, initial.ID)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}

	active, err := repo.GetInProgressByStudent(ctx, fixture.student.ID)
	if err != nil || active.ID != initial.ID {
		t.Fatalf("GetInProgressByStudent() = %#v, %v; want %s, nil", active, err, initial.ID)
	}
	tx, err = db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx() for active lock error = %v", err)
	}
	active, err = postgres.NewStudentLeavingRequestRepository(tx).GetInProgressByStudentForUpdate(ctx, fixture.student.ID)
	if err != nil || active.ID != initial.ID {
		_ = tx.Rollback()
		t.Fatalf("GetInProgressByStudentForUpdate() = %#v, %v; want %s, nil", active, err, initial.ID)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback() for active lock error = %v", err)
	}

	list, err := repo.ListByStudent(ctx, fixture.student.ID)
	if err != nil || len(list) != 1 || list[0].ID != initial.ID {
		t.Fatalf("ListByStudent() = %#v, %v; want the created request", list, err)
	}
	if _, err := repo.GetByID(ctx, uuid.New()); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("GetByID(missing) error = %v, want ErrNotFound", err)
	}
	duplicate, err := domain.NewStudentLeavingRequest(fixture.student.ID, fixture.year.ID, time.Now(), fixture.user.ID)
	if err != nil {
		t.Fatalf("NewStudentLeavingRequest(duplicate) error = %v", err)
	}
	if err := repo.Create(ctx, duplicate); !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("Create(duplicate active) error = %v, want ErrConflict", err)
	}

	firstCopy, err := repo.GetByID(ctx, initial.ID)
	if err != nil {
		t.Fatalf("GetByID(first stale copy) error = %v", err)
	}
	staleCopy, err := repo.GetByID(ctx, initial.ID)
	if err != nil {
		t.Fatalf("GetByID(second stale copy) error = %v", err)
	}
	finalizedAt := time.Date(2026, 10, 28, 9, 0, 0, 0, time.UTC)
	if err := firstCopy.Finalize(fixture.user.ID, finalizedAt); err != nil {
		t.Fatalf("Finalize() error = %v", err)
	}
	if err := repo.Update(ctx, firstCopy); err != nil {
		t.Fatalf("Update(finalized) error = %v", err)
	}
	if err := staleCopy.Cancel(fixture.user.ID, finalizedAt.Add(time.Hour), "Dibatalkan"); err != nil {
		t.Fatalf("Cancel(stale copy) error = %v", err)
	}
	if err := repo.Update(ctx, staleCopy); !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("Update(stale) error = %v, want ErrConflict", err)
	}
	got, err = repo.GetByID(ctx, initial.ID)
	if err != nil || got.Status != domain.StudentLeavingFinalized || got.FinalizedBy == nil || got.CancelledBy != nil {
		t.Fatalf("GetByID(after stale update) = %#v, %v; finalized state must remain", got, err)
	}
	if _, err := repo.GetInProgressByStudent(ctx, fixture.student.ID); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("GetInProgressByStudent(after finalize) error = %v, want ErrNotFound", err)
	}

	secondCycle, err := domain.NewStudentLeavingRequest(fixture.student.ID, fixture.year.ID, time.Now(), fixture.user.ID)
	if err != nil {
		t.Fatalf("NewStudentLeavingRequest(second process) error = %v", err)
	}
	if err := repo.Create(ctx, secondCycle); err != nil {
		t.Fatalf("Create(second process after finalize) error = %v", err)
	}
	if err := secondCycle.Cancel(fixture.user.ID, time.Now(), "  Proses dibatalkan  "); err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}
	if err := repo.Update(ctx, secondCycle); err != nil {
		t.Fatalf("Update(cancelled) error = %v", err)
	}
	got, err = repo.GetByID(ctx, secondCycle.ID)
	if err != nil || got.Status != domain.StudentLeavingCancelled || got.CancelledBy == nil || got.CancelledAt == nil || got.CancellationReason == nil || *got.CancellationReason != "Proses dibatalkan" {
		t.Fatalf("GetByID(cancelled) = %#v, %v; nullable cancellation metadata not restored", got, err)
	}
	list, err = repo.ListByStudent(ctx, fixture.student.ID)
	if err != nil || len(list) != 2 {
		t.Fatalf("ListByStudent(after transitions) = %#v, %v; want two requests", list, err)
	}
	if list[0].CreatedAt.Before(list[1].CreatedAt) ||
		(list[0].CreatedAt.Equal(list[1].CreatedAt) && list[0].ID.String() < list[1].ID.String()) {
		t.Fatalf("ListByStudent() is not ordered by created_at DESC, id DESC: %#v", list)
	}

	manager := postgres.NewUnitOfWorkManager(db)
	uow, err := manager.Begin(ctx)
	if err != nil {
		t.Fatalf("UnitOfWork Begin() error = %v", err)
	}
	rollbackRequest, err := domain.NewStudentLeavingRequest(fixture.student.ID, fixture.year.ID, time.Now(), fixture.user.ID)
	if err != nil {
		_ = uow.Rollback()
		t.Fatalf("NewStudentLeavingRequest(rollback) error = %v", err)
	}
	if err := uow.Repositories().StudentLeavingRequests.Create(ctx, rollbackRequest); err != nil {
		_ = uow.Rollback()
		t.Fatalf("UOW repository Create() error = %v", err)
	}
	if err := uow.Rollback(); err != nil {
		t.Fatalf("UnitOfWork Rollback() error = %v", err)
	}
	if _, err := repo.GetByID(ctx, rollbackRequest.ID); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("GetByID(rolled-back request) error = %v, want ErrNotFound", err)
	}

	t.Run("concurrent active creation", func(t *testing.T) {
		concurrentAccount, concurrentUser := newSettlementAccountFixture(t, db, fixture.year.ID)
		t.Cleanup(func() {
			if _, err := db.ExecContext(ctx, `DELETE FROM student_leaving_requests WHERE student_id = $1`, concurrentAccount.StudentID); err != nil {
				t.Errorf("delete concurrent request fixture: %v", err)
			}
		})
		requestA, err := domain.NewStudentLeavingRequest(concurrentAccount.StudentID, fixture.year.ID, time.Now(), concurrentUser.ID)
		if err != nil {
			t.Fatalf("NewStudentLeavingRequest(A) error = %v", err)
		}
		requestB, err := domain.NewStudentLeavingRequest(concurrentAccount.StudentID, fixture.year.ID, time.Now(), concurrentUser.ID)
		if err != nil {
			t.Fatalf("NewStudentLeavingRequest(B) error = %v", err)
		}

		transactionA, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("BeginTx(A) error = %v", err)
		}
		transactionB, err := db.BeginTx(ctx, nil)
		if err != nil {
			_ = transactionA.Rollback()
			t.Fatalf("BeginTx(B) error = %v", err)
		}
		defer transactionB.Rollback()
		var transactionAPID int
		if err := transactionA.QueryRowContext(ctx, `SELECT pg_backend_pid()`).Scan(&transactionAPID); err != nil {
			_ = transactionA.Rollback()
			t.Fatalf("read transaction A PID: %v", err)
		}
		if err := postgres.NewStudentLeavingRequestRepository(transactionA).Create(ctx, requestA); err != nil {
			_ = transactionA.Rollback()
			t.Fatalf("Create(A) error = %v", err)
		}
		var transactionBPID int
		if err := transactionB.QueryRowContext(ctx, `SELECT pg_backend_pid()`).Scan(&transactionBPID); err != nil {
			_ = transactionA.Rollback()
			t.Fatalf("read transaction B PID: %v", err)
		}

		createResult := make(chan error, 1)
		go func() {
			createResult <- postgres.NewStudentLeavingRequestRepository(transactionB).Create(ctx, requestB)
		}()

		deadline := time.Now().Add(5 * time.Second)
		blocked := false
		for time.Now().Before(deadline) {
			var isBlocked bool
			queryErr := db.QueryRowContext(ctx, `
				SELECT wait_event_type = 'Lock' AND pg_blocking_pids(pid) @> ARRAY[$2]::integer[]
				FROM pg_stat_activity
				WHERE pid = $1
			`, transactionBPID, transactionAPID).Scan(&isBlocked)
			if queryErr != nil {
				_ = transactionA.Rollback()
				t.Fatalf("inspect concurrent transaction: %v", queryErr)
			}
			if isBlocked {
				blocked = true
				break
			}
			select {
			case err := <-createResult:
				_ = transactionA.Rollback()
				t.Fatalf("Create(B) completed before unique conflict lock was observable: %v", err)
			case <-time.After(20 * time.Millisecond):
			}
		}
		if !blocked {
			_ = transactionA.Rollback()
			t.Fatal("Create(B) did not become observably blocked by the unique partial index")
		}
		if err := transactionA.Commit(); err != nil {
			t.Fatalf("Commit(A) error = %v", err)
		}
		select {
		case err := <-createResult:
			if !errors.Is(err, repository.ErrConflict) {
				t.Fatalf("Create(B) error = %v, want ErrConflict", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("Create(B) did not finish after transaction A committed")
		}

		var activeCount int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM student_leaving_requests WHERE student_id = $1 AND status = 'IN_PROGRESS'`, concurrentAccount.StudentID).Scan(&activeCount); err != nil {
			t.Fatalf("count active requests: %v", err)
		}
		if activeCount != 1 {
			t.Fatalf("active request count = %d, want 1", activeCount)
		}
	})
}

func assertStudentLeavingRequestEqual(t *testing.T, want, got domain.StudentLeavingRequest) {
	t.Helper()
	if want.ID != got.ID || want.StudentID != got.StudentID || want.AcademicYearID != got.AcademicYearID ||
		!want.EffectiveLeavingDate.Equal(got.EffectiveLeavingDate) || want.Status != got.Status ||
		want.InitiatedBy != got.InitiatedBy || !want.CreatedAt.Truncate(time.Microsecond).Equal(got.CreatedAt) {
		t.Fatalf("StudentLeavingRequest mismatch:\nwant: %#v\n got: %#v", want, got)
	}
}

func TestNewRepositorySetWiresStudentLeavingRequests(t *testing.T) {
	if repositories := postgres.NewRepositorySet(nil); repositories.StudentLeavingRequests == nil {
		t.Fatal("NewRepositorySet(nil).StudentLeavingRequests is nil")
	}
}
