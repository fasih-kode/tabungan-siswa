package postgres_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/repository/postgres"
	"github.com/google/uuid"
)

func TestAcademicYearRepository_CreateAndGetByID(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewAcademicYearRepository(db)
	ctx := context.Background()

	academicYear := newTestAcademicYear(t)

	if err := repo.Create(ctx, academicYear); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestAcademicYear(t, db, academicYear.ID)
	})

	got, err := repo.GetByID(ctx, academicYear.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	assertAcademicYearEqual(t, got, academicYear)
}

func TestAcademicYearRepository_GetByIDForUpdateSerializesAccountCreation(t *testing.T) {
	db := openTestDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	fixture := newSavingsAccountFixture(t, db)

	closeTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin close transaction: %v", err)
	}
	defer func() { _ = closeTx.Rollback() }()

	lockedYear, err := postgres.NewAcademicYearRepository(closeTx).GetByIDForUpdate(
		ctx, fixture.academicYear.ID,
	)
	if err != nil {
		t.Fatalf("GetByIDForUpdate() error = %v", err)
	}

	account, err := domain.NewSavingsAccount(fixture.student.ID, fixture.academicYear.ID)
	if err != nil {
		t.Fatalf("NewSavingsAccount() error = %v", err)
	}

	type createResult struct{ err error }
	started := make(chan struct{})
	created := make(chan createResult, 1)
	go func() {
		insertTx, beginErr := db.BeginTx(ctx, nil)
		if beginErr != nil {
			started <- struct{}{}
			created <- createResult{err: beginErr}
			return
		}
		defer func() { _ = insertTx.Rollback() }()
		started <- struct{}{}
		createErr := postgres.NewSavingsAccountRepository(insertTx).Create(ctx, account)
		if createErr == nil {
			createErr = insertTx.Commit()
		}
		created <- createResult{err: createErr}
	}()
	<-started

	// The INSERT trigger must wait on the year row's FOR SHARE lock while this
	// transaction holds FOR UPDATE. It must not complete before close commits.
	select {
	case result := <-created:
		t.Fatalf("account creation completed before close transaction released its lock: %v", result.err)
	case <-time.After(150 * time.Millisecond):
	}

	lockedYear.Status = domain.AcademicYearClosed
	lockedYear.UpdatedAt = time.Now().UTC()
	if err := postgres.NewAcademicYearRepository(closeTx).Update(ctx, lockedYear); err != nil {
		t.Fatalf("close year while holding lock: %v", err)
	}
	if err := closeTx.Commit(); err != nil {
		t.Fatalf("commit close transaction: %v", err)
	}

	select {
	case result := <-created:
		if !errors.Is(result.err, domain.ErrInvalidTransition) {
			t.Fatalf("account creation error = %v, want ErrInvalidTransition", result.err)
		}
	case <-ctx.Done():
		t.Fatalf("account creation did not finish after close commit: %v", ctx.Err())
	}

	// Assert the invariant in persisted database state, not only through the
	// insert error: a CLOSED year must not contain a newly created OPEN account.
	var persistedStatus domain.AcademicYearStatus
	if err := db.QueryRowContext(
		ctx,
		`SELECT status FROM academic_years WHERE id = $1`,
		fixture.academicYear.ID,
	).Scan(&persistedStatus); err != nil {
		t.Fatalf("read persisted academic year status: %v", err)
	}
	if persistedStatus != domain.AcademicYearClosed {
		t.Fatalf("persisted academic year status = %q, want CLOSED", persistedStatus)
	}

	var accountCount int
	if err := db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM savings_accounts WHERE academic_year_id = $1`,
		fixture.academicYear.ID,
	).Scan(&accountCount); err != nil {
		t.Fatalf("count persisted savings accounts: %v", err)
	}
	if accountCount != 0 {
		t.Fatalf("persisted savings account count = %d, want 0", accountCount)
	}
}

func TestAcademicYearRepository_GetByIDForShareBlocksLifecycleUpdate(t *testing.T) {
	db := openTestDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	year := newTestAcademicYear(t)
	repo := postgres.NewAcademicYearRepository(db)
	if err := repo.Create(ctx, year); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	t.Cleanup(func() { deleteTestAcademicYear(t, db, year.ID) })

	sharedTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin shared transaction: %v", err)
	}
	defer func() { _ = sharedTx.Rollback() }()
	if _, err := postgres.NewAcademicYearRepository(sharedTx).GetByIDForShare(ctx, year.ID); err != nil {
		t.Fatalf("GetByIDForShare() error = %v", err)
	}

	started := make(chan struct{})
	updated := make(chan error, 1)
	go func() {
		updateTx, beginErr := db.BeginTx(ctx, nil)
		if beginErr != nil {
			started <- struct{}{}
			updated <- beginErr
			return
		}
		defer func() { _ = updateTx.Rollback() }()
		started <- struct{}{}
		locked, lockErr := postgres.NewAcademicYearRepository(updateTx).GetByIDForUpdate(ctx, year.ID)
		if lockErr != nil {
			updated <- lockErr
			return
		}
		if err := locked.StartClosing(); err != nil {
			updated <- err
			return
		}
		if err := postgres.NewAcademicYearRepository(updateTx).Update(ctx, locked); err != nil {
			updated <- err
			return
		}
		updated <- updateTx.Commit()
	}()
	<-started
	select {
	case err := <-updated:
		t.Fatalf("year lifecycle update completed while FOR SHARE was held: %v", err)
	case <-time.After(150 * time.Millisecond):
	}
	if err := sharedTx.Commit(); err != nil {
		t.Fatalf("release shared year lock: %v", err)
	}
	select {
	case err := <-updated:
		if err != nil {
			t.Fatalf("lifecycle update after shared lock release: %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("lifecycle update did not finish after lock release: %v", ctx.Err())
	}
	got, err := repo.GetByID(ctx, year.ID)
	if err != nil || got.Status != domain.AcademicYearClosing {
		t.Fatalf("persisted year = %q, err=%v; want CLOSING", got.Status, err)
	}
}

func TestAcademicYearRepository_ListForUpdateBlocksLifecycleUpdate(t *testing.T) {
	db := openTestDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cleanupAcademicYears(t, db)
	year := newTestAcademicYear(t)
	repo := postgres.NewAcademicYearRepository(db)
	if err := repo.Create(ctx, year); err != nil {
		t.Fatalf("Create() year: %v", err)
	}
	t.Cleanup(func() { deleteTestAcademicYear(t, db, year.ID) })

	lockTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin list-lock transaction: %v", err)
	}
	defer func() { _ = lockTx.Rollback() }()
	lockedYears, err := postgres.NewAcademicYearRepository(lockTx).ListForUpdate(ctx)
	if err != nil || len(lockedYears) != 1 || lockedYears[0].ID != year.ID {
		t.Fatalf("ListForUpdate() = %+v, err=%v", lockedYears, err)
	}

	started := make(chan struct{})
	updated := make(chan error, 1)
	go func() {
		updateTx, beginErr := db.BeginTx(ctx, nil)
		if beginErr != nil {
			started <- struct{}{}
			updated <- beginErr
			return
		}
		defer func() { _ = updateTx.Rollback() }()
		started <- struct{}{}
		locked, lockErr := postgres.NewAcademicYearRepository(updateTx).GetByIDForUpdate(ctx, year.ID)
		if lockErr != nil {
			updated <- lockErr
			return
		}
		if err := locked.StartClosing(); err != nil {
			updated <- err
			return
		}
		if err := postgres.NewAcademicYearRepository(updateTx).Update(ctx, locked); err != nil {
			updated <- err
			return
		}
		updated <- updateTx.Commit()
	}()
	<-started
	select {
	case err := <-updated:
		t.Fatalf("year lifecycle update completed while ListForUpdate lock was held: %v", err)
	case <-time.After(150 * time.Millisecond):
	}
	if err := lockTx.Commit(); err != nil {
		t.Fatalf("release ListForUpdate lock: %v", err)
	}
	select {
	case err := <-updated:
		if err != nil {
			t.Fatalf("lifecycle update after ListForUpdate release: %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("lifecycle update did not finish after lock release: %v", ctx.Err())
	}
}

func TestAcademicYearRepository_GetByIDNotFound(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewAcademicYearRepository(db)

	_, err := repo.GetByID(
		context.Background(),
		uuid.New(),
	)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf(
			"GetByID() error = %v, want ErrNotFound",
			err,
		)
	}
}

func TestAcademicYearRepository_GetOpen(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewAcademicYearRepository(db)
	ctx := context.Background()

	academicYear := newTestAcademicYear(t)

	if err := repo.Create(ctx, academicYear); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestAcademicYear(t, db, academicYear.ID)
	})

	got, err := repo.GetOpen(ctx)
	if err != nil {
		t.Fatalf("GetOpen() error = %v", err)
	}

	assertAcademicYearEqual(t, got, academicYear)
}

func TestAcademicYearRepository_GetOpenNotFound(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewAcademicYearRepository(db)

	cleanupAcademicYears(t, db)

	_, err := repo.GetOpen(context.Background())
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf(
			"GetOpen() error = %v, want ErrNotFound",
			err,
		)
	}
}

func TestAcademicYearRepository_List(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewAcademicYearRepository(db)
	ctx := context.Background()

	cleanupAcademicYears(t, db)

	first := newTestAcademicYearWithDates(
		t,
		"2024/2025",
		time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
		domain.AcademicYearClosed,
	)
	second := newTestAcademicYearWithDates(
		t,
		"2025/2026",
		time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		domain.AcademicYearClosed,
	)

	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}

	if err := repo.Create(ctx, second); err != nil {
		t.Fatalf("Create(second) error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestAcademicYear(t, db, first.ID)
		deleteTestAcademicYear(t, db, second.ID)
	})

	got, err := repo.List(ctx, repository.ListOptions{
		Limit:  10,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("List() length = %d, want 2", len(got))
	}

	assertAcademicYearEqual(t, got[0], second)
	assertAcademicYearEqual(t, got[1], first)
}

func TestAcademicYearRepository_ListPagination(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewAcademicYearRepository(db)
	ctx := context.Background()

	cleanupAcademicYears(t, db)

	first := newTestAcademicYearWithDates(
		t,
		"2023/2024",
		time.Date(2023, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC),
		domain.AcademicYearClosed,
	)
	second := newTestAcademicYearWithDates(
		t,
		"2024/2025",
		time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
		domain.AcademicYearClosed,
	)
	third := newTestAcademicYearWithDates(
		t,
		"2025/2026",
		time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		domain.AcademicYearClosed,
	)

	for _, academicYear := range []domain.AcademicYear{
		first,
		second,
		third,
	} {
		if err := repo.Create(ctx, academicYear); err != nil {
			t.Fatalf("Create(%s) error = %v", academicYear.Name, err)
		}
	}

	t.Cleanup(func() {
		deleteTestAcademicYear(t, db, first.ID)
		deleteTestAcademicYear(t, db, second.ID)
		deleteTestAcademicYear(t, db, third.ID)
	})

	got, err := repo.List(ctx, repository.ListOptions{
		Limit:  1,
		Offset: 1,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("List() length = %d, want 1", len(got))
	}

	assertAcademicYearEqual(t, got[0], second)
}

func TestAcademicYearRepository_Update(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewAcademicYearRepository(db)
	ctx := context.Background()

	academicYear := newTestAcademicYear(t)

	if err := repo.Create(ctx, academicYear); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestAcademicYear(t, db, academicYear.ID)
	})

	updated := academicYear
	updated.Name = "Updated " + uuid.NewString()
	updated.StartDate = time.Date(
		2025,
		7,
		1,
		0,
		0,
		0,
		0,
		time.UTC,
	)
	updated.EndDate = time.Date(
		2026,
		6,
		30,
		0,
		0,
		0,
		0,
		time.UTC,
	)
	updated.Status = domain.AcademicYearClosing
	updated.UpdatedAt = time.Now()

	if err := repo.Update(ctx, updated); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, err := repo.GetByID(ctx, academicYear.ID)
	if err != nil {
		t.Fatalf("GetByID() after Update() error = %v", err)
	}

	assertAcademicYearEqual(t, got, updated)
}

func TestAcademicYearRepository_UpdateNotFound(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewAcademicYearRepository(db)

	academicYear := newTestAcademicYear(t)

	err := repo.Update(
		context.Background(),
		academicYear,
	)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf(
			"Update() error = %v, want ErrNotFound",
			err,
		)
	}
}

func TestAcademicYearRepository_PeriodsDoNotOverlap(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewAcademicYearRepository(db)
	ctx := context.Background()

	cleanupAcademicYears(t, db)

	first := newTestAcademicYearWithDates(
		t,
		"2020/2021",
		time.Date(2020, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2021, 7, 1, 0, 0, 0, 0, time.UTC),
		domain.AcademicYearClosed,
	)
	adjacent := newTestAcademicYearWithDates(
		t,
		"2021/2022",
		time.Date(2021, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2022, 7, 1, 0, 0, 0, 0, time.UTC),
		domain.AcademicYearClosed,
	)

	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}
	t.Cleanup(func() {
		deleteTestAcademicYear(t, db, first.ID)
		deleteTestAcademicYear(t, db, adjacent.ID)
	})
	if err := repo.Create(ctx, adjacent); err != nil {
		t.Fatalf("Create(adjacent) error = %v; adjacent half-open periods should be allowed", err)
	}

	overlapping := newTestAcademicYearWithDates(
		t,
		"2021/2022-overlap",
		time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2022, 6, 1, 0, 0, 0, 0, time.UTC),
		domain.AcademicYearClosed,
	)
	if err := repo.Create(ctx, overlapping); !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("Create(overlapping) error = %v, want ErrConflict", err)
	}

	updated := adjacent
	updated.StartDate = time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC)
	updated.UpdatedAt = time.Now()
	if err := repo.Update(ctx, updated); !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("Update(overlapping) error = %v, want ErrConflict", err)
	}
}

func closeTestAcademicYear(t *testing.T, db *sql.DB, year domain.AcademicYear) {
	t.Helper()
	repo := postgres.NewAcademicYearRepository(db)
	if year.Status == domain.AcademicYearOpen {
		if err := year.StartClosing(); err != nil {
			t.Fatalf("start closing test academic year: %v", err)
		}
		if err := repo.Update(context.Background(), year); err != nil {
			t.Fatalf("update test academic year to CLOSING: %v", err)
		}
	}
	if year.Status == domain.AcademicYearClosing {
		if err := year.Close(); err != nil {
			t.Fatalf("close test academic year: %v", err)
		}
		if err := repo.Update(context.Background(), year); err != nil {
			t.Fatalf("update test academic year to CLOSED: %v", err)
		}
	}
}

func TestAcademicYearRepository_CreateDuplicateOpen(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewAcademicYearRepository(db)
	ctx := context.Background()

	cleanupAcademicYears(t, db)

	first := newTestAcademicYear(t)

	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("first Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestAcademicYear(t, db, first.ID)
	})

	second := newTestAcademicYearWithDates(
		t,
		"2026/2027",
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC),
		domain.AcademicYearOpen,
	)

	err := repo.Create(ctx, second)
	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf(
			"second Create() error = %v, want ErrInvalidTransition from closed-year prerequisite",
			err,
		)
	}
}

func TestAcademicYearRepository_CreateTriggerRequiresPreviousYearClosed(t *testing.T) {
	db := openTestDatabase(t)
	ctx := context.Background()
	cleanupAcademicYears(t, db)
	repo := postgres.NewAcademicYearRepository(db)
	first := newTestAcademicYear(t)
	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("create first year: %v", err)
	}
	t.Cleanup(func() { deleteTestAcademicYear(t, db, first.ID) })
	second := newTestAcademicYearWithDates(t, "Next", time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC), time.Date(2028, 7, 1, 0, 0, 0, 0, time.UTC), domain.AcademicYearOpen)
	if err := repo.Create(ctx, second); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("Create() with OPEN predecessor = %v, want ErrInvalidTransition", err)
	}
	if err := first.StartClosing(); err != nil {
		t.Fatalf("StartClosing(): %v", err)
	}
	if err := repo.Update(ctx, first); err != nil {
		t.Fatalf("update CLOSING predecessor: %v", err)
	}
	if err := repo.Create(ctx, second); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("Create() with CLOSING predecessor = %v, want ErrInvalidTransition", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("Close(): %v", err)
	}
	if err := repo.Update(ctx, first); err != nil {
		t.Fatalf("update CLOSED predecessor: %v", err)
	}
	if err := repo.Create(ctx, second); err != nil {
		t.Fatalf("Create() with CLOSED predecessor: %v", err)
	}
	t.Cleanup(func() { deleteTestAcademicYear(t, db, second.ID) })
}

func assertAcademicYearEqual(
	t *testing.T,
	got domain.AcademicYear,
	want domain.AcademicYear,
) {
	t.Helper()

	if got.ID != want.ID {
		t.Errorf("ID = %v, want %v", got.ID, want.ID)
	}

	if got.Name != want.Name {
		t.Errorf("Name = %q, want %q", got.Name, want.Name)
	}

	if !got.StartDate.Equal(want.StartDate) {
		t.Errorf(
			"StartDate = %v, want %v",
			got.StartDate,
			want.StartDate,
		)
	}

	if !got.EndDate.Equal(want.EndDate) {
		t.Errorf(
			"EndDate = %v, want %v",
			got.EndDate,
			want.EndDate,
		)
	}

	if got.Status != want.Status {
		t.Errorf("Status = %q, want %q", got.Status, want.Status)
	}

	if !got.CreatedAt.Equal(want.CreatedAt) {
		if !got.CreatedAt.Truncate(time.Microsecond).Equal(
			want.CreatedAt.Truncate(time.Microsecond),
		) {
			t.Errorf(
				"CreatedAt = %v, want %v",
				got.CreatedAt,
				want.CreatedAt,
			)
		}
	}

	if !got.UpdatedAt.Equal(want.UpdatedAt) {
		if !got.UpdatedAt.Truncate(time.Microsecond).Equal(
			want.UpdatedAt.Truncate(time.Microsecond),
		) {
			t.Errorf(
				"UpdatedAt = %v, want %v",
				got.UpdatedAt,
				want.UpdatedAt,
			)
		}
	}
}

func newTestAcademicYear(t *testing.T) domain.AcademicYear {
	t.Helper()

	return newTestAcademicYearWithDates(
		t,
		"2025/2026",
		time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		domain.AcademicYearOpen,
	)
}

func newTestAcademicYearWithDates(
	t *testing.T,
	name string,
	startDate time.Time,
	endDate time.Time,
	status domain.AcademicYearStatus,
) domain.AcademicYear {
	t.Helper()

	academicYear, err := domain.NewAcademicYear(
		name,
		startDate,
		endDate,
	)
	if err != nil {
		t.Fatalf("NewAcademicYear() error = %v", err)
	}

	academicYear.Status = status
	return academicYear
}

func deleteTestAcademicYear(
	t *testing.T,
	db *sql.DB,
	id uuid.UUID,
) {
	t.Helper()

	_, err := db.ExecContext(
		context.Background(),
		"DELETE FROM academic_years WHERE id = $1",
		id,
	)
	if err != nil {
		t.Errorf("delete test academic year: %v", err)
	}
}

func cleanupScopedRepositoryFixtures(
	t *testing.T,
	db *sql.DB,
) {
	t.Helper()

	statements := []struct {
		name  string
		query string
	}{
		{
			name:  "transactions",
			query: "DELETE FROM transactions",
		},
		{
			name:  "savings settlements",
			query: "DELETE FROM savings_settlements",
		},
		{
			name:  "savings accounts",
			query: "DELETE FROM savings_accounts",
		},
		{
			name:  "class transfer requests",
			query: "DELETE FROM class_transfer_requests",
		},
		{
			name:  "student class histories",
			query: "DELETE FROM student_class_histories",
		},
		{
			name:  "teacher class assignments",
			query: "DELETE FROM teacher_class_assignments",
		},
		{
			name:  "audit logs",
			query: "DELETE FROM audit_logs",
		},
		{
			name:  "students",
			query: "DELETE FROM students",
		},
		{
			name:  "classes",
			query: "DELETE FROM classes",
		},
		{
			name:  "academic years",
			query: "DELETE FROM academic_years",
		},
		{
			name:  "sessions",
			query: "DELETE FROM sessions",
		},
		{
			name:  "users",
			query: "DELETE FROM users",
		},
	}

	ctx := context.Background()

	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement.query); err != nil {
			t.Fatalf(
				"cleanup %s: %v",
				statement.name,
				err,
			)
		}
	}
}

func cleanupAcademicYears(
	t *testing.T,
	db *sql.DB,
) {
	t.Helper()

	statements := []struct {
		name  string
		query string
	}{
		{
			name:  "transactions",
			query: "DELETE FROM transactions",
		},
		{
			name:  "savings settlements",
			query: "DELETE FROM savings_settlements",
		},
		{
			name:  "savings accounts",
			query: "DELETE FROM savings_accounts",
		},
		{
			name:  "class transfer requests",
			query: "DELETE FROM class_transfer_requests",
		},
		{
			name:  "student class histories",
			query: "DELETE FROM student_class_histories",
		},
		{
			name:  "teacher class assignments",
			query: "DELETE FROM teacher_class_assignments",
		},
		{
			name:  "audit logs",
			query: "DELETE FROM audit_logs",
		},
		{
			name:  "students",
			query: "DELETE FROM students",
		},
		{
			name:  "classes",
			query: "DELETE FROM classes",
		},
		{
			name:  "academic years",
			query: "DELETE FROM academic_years",
		},
		{
			name:  "sessions",
			query: "DELETE FROM sessions",
		},
		{
			name:  "users",
			query: "DELETE FROM users",
		},
	}

	ctx := context.Background()

	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement.query); err != nil {
			t.Fatalf(
				"cleanup %s: %v",
				statement.name,
				err,
			)
		}
	}
}
