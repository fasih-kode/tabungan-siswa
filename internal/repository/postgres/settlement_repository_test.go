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

func TestSettlementRepository_CreateAndGetByID(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewSettlementRepository(db)
	ctx := context.Background()

	fixture := newSettlementFixture(t, db)

	executedAt := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	settlement, err := domain.NewSavingsSettlement(
		fixture.account.ID,
		domain.SettlementRegularYearEnd,
		100_000,
		fixture.user.ID,
		executedAt,
		nil,
	)
	if err != nil {
		t.Fatalf("NewSavingsSettlement() error = %v", err)
	}

	if err := repo.Create(ctx, settlement); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	t.Cleanup(deleteTestSettlement(t, db, settlement.ID))

	got, err := repo.GetByID(ctx, settlement.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	assertSavingsSettlementEqual(t, settlement, got)
}

func TestSettlementRepository_GetByIDNotFound(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewSettlementRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, uuid.New())
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("GetByID() error = %v, want ErrNotFound", err)
	}
}

func TestSettlementRepository_GetCompletedBySavingsAccount(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewSettlementRepository(db)
	ctx := context.Background()

	fixture := newSettlementFixture(t, db)

	first, err := domain.NewSavingsSettlement(
		fixture.account.ID,
		domain.SettlementRegularYearEnd,
		100_000,
		fixture.user.ID,
		time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC),
		nil,
	)
	if err != nil {
		t.Fatalf("NewSavingsSettlement(first) error = %v", err)
	}

	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}
	t.Cleanup(deleteTestSettlement(t, db, first.ID))

	if err := first.Supersede(); err != nil {
		t.Fatalf("Supersede(first) error = %v", err)
	}

	if err := repo.Update(ctx, first); err != nil {
		t.Fatalf("Update(first) error = %v", err)
	}

	replacement, err := domain.NewSavingsSettlement(
		fixture.account.ID,
		domain.SettlementRegularYearEnd,
		125_000,
		fixture.user.ID,
		time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC),
		&first.ID,
	)
	if err != nil {
		t.Fatalf("NewSavingsSettlement(replacement) error = %v", err)
	}

	if err := repo.Create(ctx, replacement); err != nil {
		t.Fatalf("Create(replacement) error = %v", err)
	}
	t.Cleanup(deleteTestSettlement(t, db, replacement.ID))

	got, err := repo.GetCompletedBySavingsAccount(ctx, fixture.account.ID)
	if err != nil {
		t.Fatalf("GetCompletedBySavingsAccount() error = %v", err)
	}

	assertSavingsSettlementEqual(t, replacement, got)
}

func TestSettlementRepository_GetCompletedBySavingsAccountNotFound(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewSettlementRepository(db)
	ctx := context.Background()

	fixture := newSettlementFixture(t, db)

	_, err := repo.GetCompletedBySavingsAccount(ctx, fixture.account.ID)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf(
			"GetCompletedBySavingsAccount() error = %v, want ErrNotFound",
			err,
		)
	}
}

func TestSettlementRepository_ListBySavingsAccount(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewSettlementRepository(db)
	ctx := context.Background()

	fixture := newSettlementFixture(t, db)

	first, err := domain.NewSavingsSettlement(
		fixture.account.ID,
		domain.SettlementRegularYearEnd,
		100_000,
		fixture.user.ID,
		time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC),
		nil,
	)
	if err != nil {
		t.Fatalf("NewSavingsSettlement(first) error = %v", err)
	}

	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}
	t.Cleanup(deleteTestSettlement(t, db, first.ID))

	if err := first.Supersede(); err != nil {
		t.Fatalf("Supersede(first) error = %v", err)
	}

	if err := repo.Update(ctx, first); err != nil {
		t.Fatalf("Update(first) error = %v", err)
	}

	second, err := domain.NewSavingsSettlement(
		fixture.account.ID,
		domain.SettlementRegularYearEnd,
		125_000,
		fixture.user.ID,
		time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC),
		&first.ID,
	)
	if err != nil {
		t.Fatalf("NewSavingsSettlement(second) error = %v", err)
	}

	if err := repo.Create(ctx, second); err != nil {
		t.Fatalf("Create(second) error = %v", err)
	}
	t.Cleanup(deleteTestSettlement(t, db, second.ID))

	otherAccount, otherUser := newSettlementAccountFixture(
		t,
		db,
		fixture.year.ID,
	)

	other, err := domain.NewSavingsSettlement(
		otherAccount.ID,
		domain.SettlementStudentLeaving,
		999_000,
		otherUser.ID,
		time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC),
		nil,
	)
	if err != nil {
		t.Fatalf("NewSavingsSettlement(other) error = %v", err)
	}

	if err := repo.Create(ctx, other); err != nil {
		t.Fatalf("Create(other) error = %v", err)
	}
	t.Cleanup(deleteTestSettlement(t, db, other.ID))

	got, err := repo.ListBySavingsAccount(
		ctx,
		fixture.account.ID,
		repository.ListOptions{
			Limit:  10,
			Offset: 0,
		},
	)
	if err != nil {
		t.Fatalf("ListBySavingsAccount() error = %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}

	assertSavingsSettlementEqual(t, second, got[0])
	assertSavingsSettlementEqual(t, first, got[1])
}

func TestSettlementRepository_ListBySavingsAccountEmpty(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewSettlementRepository(db)
	ctx := context.Background()

	fixture := newSettlementFixture(t, db)

	got, err := repo.ListBySavingsAccount(
		ctx,
		fixture.account.ID,
		repository.ListOptions{
			Limit:  10,
			Offset: 0,
		},
	)
	if err != nil {
		t.Fatalf("ListBySavingsAccount() error = %v", err)
	}

	if got == nil {
		t.Fatal("ListBySavingsAccount() returned nil slice")
	}

	if len(got) != 0 {
		t.Fatalf("len(got) = %d, want 0", len(got))
	}
}

func TestSettlementRepository_ListBySavingsAccountPagination(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewSettlementRepository(db)
	ctx := context.Background()

	fixture := newSettlementFixture(t, db)

	settlements := make([]domain.SavingsSettlement, 0, 4)

	for i := 0; i < 4; i++ {
		executedAt := time.Date(
			2026,
			time.Month(7+i),
			1,
			10,
			0,
			0,
			0,
			time.UTC,
		)

		settlement, err := domain.NewSavingsSettlement(
			fixture.account.ID,
			domain.SettlementStudentLeaving,
			domain.Money(100_000+i),
			fixture.user.ID,
			executedAt,
			nil,
		)
		if err != nil {
			t.Fatalf("NewSavingsSettlement() error = %v", err)
		}

		if err := repo.Create(ctx, settlement); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		t.Cleanup(deleteTestSettlement(t, db, settlement.ID))

		if i < 3 {
			if err := settlement.Supersede(); err != nil {
				t.Fatalf("Supersede() error = %v", err)
			}

			if err := repo.Update(ctx, settlement); err != nil {
				t.Fatalf("Update() error = %v", err)
			}
		}

		settlements = append(settlements, settlement)
	}

	got, err := repo.ListBySavingsAccount(
		ctx,
		fixture.account.ID,
		repository.ListOptions{
			Limit:  2,
			Offset: 1,
		},
	)
	if err != nil {
		t.Fatalf("ListBySavingsAccount() error = %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}

	assertSavingsSettlementEqual(t, settlements[2], got[0])
	assertSavingsSettlementEqual(t, settlements[1], got[1])
}

func TestSettlementRepository_Update(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewSettlementRepository(db)
	ctx := context.Background()

	fixture := newSettlementFixture(t, db)

	settlement, err := domain.NewSavingsSettlement(
		fixture.account.ID,
		domain.SettlementStudentLeaving,
		100_000,
		fixture.user.ID,
		time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC),
		nil,
	)
	if err != nil {
		t.Fatalf("NewSavingsSettlement() error = %v", err)
	}

	if err := repo.Create(ctx, settlement); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	t.Cleanup(deleteTestSettlement(t, db, settlement.ID))

	settlement.Amount = 150_000
	settlement.ExecutedAt = time.Date(2026, 9, 25, 11, 0, 0, 0, time.UTC)

	if err := repo.Update(ctx, settlement); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, err := repo.GetByID(ctx, settlement.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	assertSavingsSettlementEqual(t, settlement, got)
}

func TestSettlementRepository_UpdateSupersede(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewSettlementRepository(db)
	ctx := context.Background()

	fixture := newSettlementFixture(t, db)

	settlement, err := domain.NewSavingsSettlement(
		fixture.account.ID,
		domain.SettlementRegularYearEnd,
		100_000,
		fixture.user.ID,
		time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC),
		nil,
	)
	if err != nil {
		t.Fatalf("NewSavingsSettlement() error = %v", err)
	}

	if err := repo.Create(ctx, settlement); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	t.Cleanup(deleteTestSettlement(t, db, settlement.ID))

	if err := settlement.Supersede(); err != nil {
		t.Fatalf("Supersede() error = %v", err)
	}

	if err := repo.Update(ctx, settlement); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, err := repo.GetByID(ctx, settlement.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	assertSavingsSettlementEqual(t, settlement, got)
}

func TestSettlementRepository_UpdateNotFound(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewSettlementRepository(db)
	ctx := context.Background()

	settlement, err := domain.NewSavingsSettlement(
		uuid.New(),
		domain.SettlementStudentLeaving,
		0,
		uuid.New(),
		time.Now().UTC(),
		nil,
	)
	if err != nil {
		t.Fatalf("NewSavingsSettlement() error = %v", err)
	}

	err = repo.Update(ctx, settlement)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("Update() error = %v, want ErrNotFound", err)
	}
}

func TestSettlementRepository_CreateConflict(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewSettlementRepository(db)
	ctx := context.Background()

	fixture := newSettlementFixture(t, db)

	first, err := domain.NewSavingsSettlement(
		fixture.account.ID,
		domain.SettlementRegularYearEnd,
		100_000,
		fixture.user.ID,
		time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC),
		nil,
	)
	if err != nil {
		t.Fatalf("NewSavingsSettlement(first) error = %v", err)
	}

	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}
	t.Cleanup(deleteTestSettlement(t, db, first.ID))

	second, err := domain.NewSavingsSettlement(
		fixture.account.ID,
		domain.SettlementRegularYearEnd,
		125_000,
		fixture.user.ID,
		time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC),
		nil,
	)
	if err != nil {
		t.Fatalf("NewSavingsSettlement(second) error = %v", err)
	}

	err = repo.Create(ctx, second)
	if !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("Create(second) error = %v, want ErrConflict", err)
	}
}

func TestSettlementRepository_CreateReplacement(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewSettlementRepository(db)
	ctx := context.Background()

	fixture := newSettlementFixture(t, db)

	first, err := domain.NewSavingsSettlement(
		fixture.account.ID,
		domain.SettlementRegularYearEnd,
		100_000,
		fixture.user.ID,
		time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC),
		nil,
	)
	if err != nil {
		t.Fatalf("NewSavingsSettlement(first) error = %v", err)
	}

	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}
	t.Cleanup(deleteTestSettlement(t, db, first.ID))

	if err := first.Supersede(); err != nil {
		t.Fatalf("Supersede(first) error = %v", err)
	}

	if err := repo.Update(ctx, first); err != nil {
		t.Fatalf("Update(first) error = %v", err)
	}

	replacement, err := domain.NewSavingsSettlement(
		fixture.account.ID,
		domain.SettlementRegularYearEnd,
		125_000,
		fixture.user.ID,
		time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC),
		&first.ID,
	)
	if err != nil {
		t.Fatalf("NewSavingsSettlement(replacement) error = %v", err)
	}

	if err := repo.Create(ctx, replacement); err != nil {
		t.Fatalf("Create(replacement) error = %v", err)
	}
	t.Cleanup(deleteTestSettlement(t, db, replacement.ID))

	got, err := repo.GetByID(ctx, replacement.ID)
	if err != nil {
		t.Fatalf("GetByID(replacement) error = %v", err)
	}

	assertSavingsSettlementEqual(t, replacement, got)
}

type settlementFixture struct {
	user    domain.User
	student domain.Student
	year    domain.AcademicYear
	account domain.SavingsAccount
}

func newSettlementFixture(t *testing.T, db *sql.DB) settlementFixture {
	t.Helper()

	ctx := context.Background()

	userRepo := postgres.NewUserRepository(db)
	studentRepo := postgres.NewStudentRepository(db)
	yearRepo := postgres.NewAcademicYearRepository(db)
	accountRepo := postgres.NewSavingsAccountRepository(db)

	user := newTestUser(t)
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("create fixture user: %v", err)
	}
	t.Cleanup(func() {
		deleteTestUser(t, db, user.ID)
	})

	student := newTestStudentWithUserID(
		t,
		"Siswa Settlement",
		user.ID,
	)
	if err := studentRepo.Create(ctx, student); err != nil {
		t.Fatalf("create fixture student: %v", err)
	}
	t.Cleanup(func() {
		deleteTestStudent(t, db, student.ID)
	})

	year := newTestAcademicYear(t)
	if err := yearRepo.Create(ctx, year); err != nil {
		t.Fatalf("create fixture academic year: %v", err)
	}
	t.Cleanup(func() {
		deleteTestAcademicYear(t, db, year.ID)
	})

	account, err := domain.NewSavingsAccount(student.ID, year.ID)
	if err != nil {
		t.Fatalf("NewSavingsAccount() error = %v", err)
	}

	if err := accountRepo.Create(ctx, account); err != nil {
		t.Fatalf("create fixture savings account: %v", err)
	}
	t.Cleanup(func() {
		deleteTestSavingsAccount(t, db, account.ID)
	})

	return settlementFixture{
		user:    user,
		student: student,
		year:    year,
		account: account,
	}
}

func newSettlementAccountFixture(
	t *testing.T,
	db *sql.DB,
	academicYearID uuid.UUID,
) (domain.SavingsAccount, domain.User) {
	t.Helper()

	ctx := context.Background()

	userRepo := postgres.NewUserRepository(db)
	studentRepo := postgres.NewStudentRepository(db)
	accountRepo := postgres.NewSavingsAccountRepository(db)

	user := newTestUser(t)
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("create other fixture user: %v", err)
	}
	t.Cleanup(func() {
		deleteTestUser(t, db, user.ID)
	})

	student := newTestStudentWithUserID(
		t,
		"Siswa Settlement Other",
		user.ID,
	)
	if err := studentRepo.Create(ctx, student); err != nil {
		t.Fatalf("create other fixture student: %v", err)
	}
	t.Cleanup(func() {
		deleteTestStudent(t, db, student.ID)
	})

	account, err := domain.NewSavingsAccount(student.ID, academicYearID)
	if err != nil {
		t.Fatalf("NewSavingsAccount(other) error = %v", err)
	}

	if err := accountRepo.Create(ctx, account); err != nil {
		t.Fatalf("create other fixture savings account: %v", err)
	}
	t.Cleanup(func() {
		deleteTestSavingsAccount(t, db, account.ID)
	})

	return account, user
}

func deleteTestSettlement(t *testing.T, db *sql.DB, id uuid.UUID) func() {
	t.Helper()

	return func() {
		t.Helper()

		ctx := context.Background()

		if _, err := db.ExecContext(
			ctx,
			`DELETE FROM savings_settlements WHERE id = $1`,
			id,
		); err != nil {
			t.Fatalf("delete test settlement %s: %v", id, err)
		}
	}
}

func assertSavingsSettlementEqual(
	t *testing.T,
	want domain.SavingsSettlement,
	got domain.SavingsSettlement,
) {
	t.Helper()

	if got.ID != want.ID {
		t.Fatalf("ID = %v, want %v", got.ID, want.ID)
	}

	if got.SavingsAccountID != want.SavingsAccountID {
		t.Fatalf(
			"SavingsAccountID = %v, want %v",
			got.SavingsAccountID,
			want.SavingsAccountID,
		)
	}

	if got.Type != want.Type {
		t.Fatalf("Type = %v, want %v", got.Type, want.Type)
	}

	if got.Amount != want.Amount {
		t.Fatalf("Amount = %v, want %v", got.Amount, want.Amount)
	}

	if got.ExecutedBy != want.ExecutedBy {
		t.Fatalf(
			"ExecutedBy = %v, want %v",
			got.ExecutedBy,
			want.ExecutedBy,
		)
	}

	if !got.ExecutedAt.Equal(want.ExecutedAt) {
		t.Fatalf(
			"ExecutedAt = %v, want %v",
			got.ExecutedAt,
			want.ExecutedAt,
		)
	}

	if got.Status != want.Status {
		t.Fatalf("Status = %v, want %v", got.Status, want.Status)
	}

	switch {
	case want.ReplacesSettlementID == nil && got.ReplacesSettlementID == nil:
	case want.ReplacesSettlementID != nil && got.ReplacesSettlementID != nil:
		if *got.ReplacesSettlementID != *want.ReplacesSettlementID {
			t.Fatalf(
				"ReplacesSettlementID = %v, want %v",
				*got.ReplacesSettlementID,
				*want.ReplacesSettlementID,
			)
		}
	default:
		t.Fatalf(
			"ReplacesSettlementID = %v, want %v",
			got.ReplacesSettlementID,
			want.ReplacesSettlementID,
		)
	}
}
