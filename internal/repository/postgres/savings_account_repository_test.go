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

func TestSavingsAccountRepository_CreateAndGetByID(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewSavingsAccountRepository(db)
	ctx := context.Background()

	fixture := newSavingsAccountFixture(t, db)

	account, err := domain.NewSavingsAccount(
		fixture.student.ID,
		fixture.academicYear.ID,
	)
	if err != nil {
		t.Fatalf("NewSavingsAccount() error = %v", err)
	}

	if err := repo.Create(ctx, account); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestSavingsAccount(t, db, account.ID)
	})

	got, err := repo.GetByID(ctx, account.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	assertSavingsAccountEqual(t, got, account)
}

func TestSavingsAccountRepository_GetByIDNotFound(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewSavingsAccountRepository(db)

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

func TestSavingsAccountRepository_GetByStudentAndAcademicYear(
	t *testing.T,
) {
	db := openTestDatabase(t)
	repo := postgres.NewSavingsAccountRepository(db)
	ctx := context.Background()

	fixture := newSavingsAccountFixture(t, db)

	account, err := domain.NewSavingsAccount(
		fixture.student.ID,
		fixture.academicYear.ID,
	)
	if err != nil {
		t.Fatalf("NewSavingsAccount() error = %v", err)
	}

	if err := repo.Create(ctx, account); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestSavingsAccount(t, db, account.ID)
	})

	got, err := repo.GetByStudentAndAcademicYear(
		ctx,
		fixture.student.ID,
		fixture.academicYear.ID,
	)
	if err != nil {
		t.Fatalf(
			"GetByStudentAndAcademicYear() error = %v",
			err,
		)
	}

	assertSavingsAccountEqual(t, got, account)
}

func TestSavingsAccountRepository_GetByStudentAndAcademicYearNotFound(
	t *testing.T,
) {
	db := openTestDatabase(t)
	repo := postgres.NewSavingsAccountRepository(db)

	fixture := newSavingsAccountFixture(t, db)

	_, err := repo.GetByStudentAndAcademicYear(
		context.Background(),
		fixture.student.ID,
		fixture.academicYear.ID,
	)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf(
			"GetByStudentAndAcademicYear() error = %v, want ErrNotFound",
			err,
		)
	}
}

func TestSavingsAccountRepository_CreateDuplicateStudentAcademicYear(
	t *testing.T,
) {
	db := openTestDatabase(t)
	repo := postgres.NewSavingsAccountRepository(db)
	ctx := context.Background()

	fixture := newSavingsAccountFixture(t, db)

	first, err := domain.NewSavingsAccount(
		fixture.student.ID,
		fixture.academicYear.ID,
	)
	if err != nil {
		t.Fatalf("NewSavingsAccount(first) error = %v", err)
	}

	second, err := domain.NewSavingsAccount(
		fixture.student.ID,
		fixture.academicYear.ID,
	)
	if err != nil {
		t.Fatalf("NewSavingsAccount(second) error = %v", err)
	}

	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestSavingsAccount(t, db, first.ID)
	})

	err = repo.Create(ctx, second)
	if !errors.Is(err, repository.ErrConflict) {
		t.Fatalf(
			"Create(second) error = %v, want ErrConflict",
			err,
		)
	}
}

func TestSavingsAccountRepository_Update(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewSavingsAccountRepository(db)
	ctx := context.Background()

	fixture := newSavingsAccountFixture(t, db)

	account, err := domain.NewSavingsAccount(
		fixture.student.ID,
		fixture.academicYear.ID,
	)
	if err != nil {
		t.Fatalf("NewSavingsAccount() error = %v", err)
	}

	if err := repo.Create(ctx, account); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestSavingsAccount(t, db, account.ID)
	})

	settledAt := time.Date(
		2026,
		9,
		25,
		10,
		0,
		0,
		0,
		time.UTC,
	)

	if err := account.Settle(settledAt); err != nil {
		t.Fatalf("Settle() error = %v", err)
	}

	if err := repo.Update(ctx, account); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, err := repo.GetByID(ctx, account.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	assertSavingsAccountEqual(t, got, account)
}

func TestSavingsAccountRepository_UpdateReopen(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewSavingsAccountRepository(db)
	ctx := context.Background()

	fixture := newSavingsAccountFixture(t, db)

	account, err := domain.NewSavingsAccount(
		fixture.student.ID,
		fixture.academicYear.ID,
	)
	if err != nil {
		t.Fatalf("NewSavingsAccount() error = %v", err)
	}

	if err := repo.Create(ctx, account); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestSavingsAccount(t, db, account.ID)
	})

	settledAt := time.Date(
		2026,
		9,
		25,
		10,
		0,
		0,
		0,
		time.UTC,
	)

	if err := account.Settle(settledAt); err != nil {
		t.Fatalf("Settle() error = %v", err)
	}

	if err := repo.Update(ctx, account); err != nil {
		t.Fatalf("Update(settled) error = %v", err)
	}

	if err := account.Reopen(); err != nil {
		t.Fatalf("Reopen() error = %v", err)
	}

	if err := repo.Update(ctx, account); err != nil {
		t.Fatalf("Update(reopen) error = %v", err)
	}

	got, err := repo.GetByID(ctx, account.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	assertSavingsAccountEqual(t, got, account)

	if got.SettledAt != nil {
		t.Fatalf(
			"SettledAt = %v, want nil",
			*got.SettledAt,
		)
	}
}

func TestSavingsAccountRepository_UpdateNotFound(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewSavingsAccountRepository(db)

	account, err := domain.NewSavingsAccount(
		uuid.New(),
		uuid.New(),
	)
	if err != nil {
		t.Fatalf("NewSavingsAccount() error = %v", err)
	}

	err = repo.Update(
		context.Background(),
		account,
	)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf(
			"Update() error = %v, want ErrNotFound",
			err,
		)
	}
}

func TestSavingsAccountRepository_GetByIDForUpdate(
	t *testing.T,
) {
	db := openTestDatabase(t)
	ctx := context.Background()

	fixture := newSavingsAccountFixture(t, db)

	account, err := domain.NewSavingsAccount(
		fixture.student.ID,
		fixture.academicYear.ID,
	)
	if err != nil {
		t.Fatalf("NewSavingsAccount() error = %v", err)
	}

	if err := postgres.NewSavingsAccountRepository(db).Create(
		ctx,
		account,
	); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestSavingsAccount(t, db, account.ID)
	})

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	repo := postgres.NewSavingsAccountRepository(tx)

	got, err := repo.GetByIDForUpdate(ctx, account.ID)
	if err != nil {
		t.Fatalf(
			"GetByIDForUpdate() error = %v",
			err,
		)
	}

	assertSavingsAccountEqual(t, got, account)
}

func TestSavingsAccountRepository_GetByIDForUpdateNotFound(
	t *testing.T,
) {
	db := openTestDatabase(t)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	repo := postgres.NewSavingsAccountRepository(tx)

	_, err = repo.GetByIDForUpdate(
		ctx,
		uuid.New(),
	)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf(
			"GetByIDForUpdate() error = %v, want ErrNotFound",
			err,
		)
	}
}

type savingsAccountFixture struct {
	user         domain.User
	student      domain.Student
	academicYear domain.AcademicYear
}

func newSavingsAccountFixture(
	t *testing.T,
	db *sql.DB,
) savingsAccountFixture {
	t.Helper()

	ctx := context.Background()

	userRepo := postgres.NewUserRepository(db)
	studentRepo := postgres.NewStudentRepository(db)
	academicYearRepo := postgres.NewAcademicYearRepository(db)

	user := newTestUser(t)
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf(
			"create fixture user: %v",
			err,
		)
	}

	t.Cleanup(func() {
		deleteTestUser(t, db, user.ID)
	})

	student := newTestStudentWithUserID(
		t,
		"Siswa Savings Account",
		user.ID,
	)
	if err := studentRepo.Create(ctx, student); err != nil {
		t.Fatalf(
			"create fixture student: %v",
			err,
		)
	}

	t.Cleanup(func() {
		deleteTestStudent(t, db, student.ID)
	})

	academicYear := newTestAcademicYear(t)
	academicYear.Name = "2026/2027"
	academicYear.StartDate = time.Date(
		2026,
		7,
		1,
		0,
		0,
		0,
		0,
		time.UTC,
	)
	academicYear.EndDate = time.Date(
		2027,
		7,
		1,
		0,
		0,
		0,
		0,
		time.UTC,
	)

	if err := academicYearRepo.Create(
		ctx,
		academicYear,
	); err != nil {
		t.Fatalf(
			"create fixture academic year: %v",
			err,
		)
	}

	t.Cleanup(func() {
		deleteTestAcademicYear(
			t,
			db,
			academicYear.ID,
		)
	})

	return savingsAccountFixture{
		user:         user,
		student:      student,
		academicYear: academicYear,
	}
}

func assertSavingsAccountEqual(
	t *testing.T,
	got domain.SavingsAccount,
	want domain.SavingsAccount,
) {
	t.Helper()

	if got.ID != want.ID {
		t.Errorf(
			"ID = %v, want %v",
			got.ID,
			want.ID,
		)
	}

	if got.StudentID != want.StudentID {
		t.Errorf(
			"StudentID = %v, want %v",
			got.StudentID,
			want.StudentID,
		)
	}

	if got.AcademicYearID != want.AcademicYearID {
		t.Errorf(
			"AcademicYearID = %v, want %v",
			got.AcademicYearID,
			want.AcademicYearID,
		)
	}

	if got.Status != want.Status {
		t.Errorf(
			"Status = %v, want %v",
			got.Status,
			want.Status,
		)
	}

	if got.SettledAt == nil && want.SettledAt == nil {
	} else if got.SettledAt == nil || want.SettledAt == nil {
		t.Errorf(
			"SettledAt = %v, want %v",
			got.SettledAt,
			want.SettledAt,
		)
	} else if !got.SettledAt.Equal(*want.SettledAt) {
		t.Errorf(
			"SettledAt = %v, want %v",
			*got.SettledAt,
			*want.SettledAt,
		)
	}

	if !got.CreatedAt.Truncate(time.Microsecond).Equal(
		want.CreatedAt.Truncate(time.Microsecond),
	) {
		t.Errorf(
			"CreatedAt = %v, want %v",
			got.CreatedAt,
			want.CreatedAt,
		)
	}

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

func deleteTestSavingsAccount(
	t *testing.T,
	db *sql.DB,
	id uuid.UUID,
) {
	t.Helper()

	_, err := db.ExecContext(
		context.Background(),
		"DELETE FROM savings_accounts WHERE id = $1",
		id,
	)
	if err != nil {
		t.Errorf(
			"delete test savings account: %v",
			err,
		)
	}
}
