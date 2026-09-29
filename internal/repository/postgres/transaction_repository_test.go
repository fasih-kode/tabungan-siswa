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

func TestTransactionRepository_CreateAndGetByID(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewTransactionRepository(db)
	ctx := context.Background()

	fixture := newTransactionFixture(t, db)

	transaction, err := domain.NewDeposit(
		fixture.account.ID,
		100_000,
		time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC),
		fixture.user.ID,
	)
	if err != nil {
		t.Fatalf("NewDeposit() error = %v", err)
	}

	if err := repo.Create(ctx, transaction); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestTransaction(t, db, transaction.ID)
	})

	got, err := repo.GetByID(ctx, transaction.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	assertTransactionEqual(t, got, transaction)
}

func TestTransactionRepository_GetByIDNotFound(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewTransactionRepository(db)

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

func TestTransactionRepository_ListBySavingsAccount(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewTransactionRepository(db)
	ctx := context.Background()

	fixture := newTransactionFixture(t, db)

	first, err := domain.NewDeposit(
		fixture.account.ID,
		100_000,
		time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC),
		fixture.user.ID,
	)
	if err != nil {
		t.Fatalf("NewDeposit(first) error = %v", err)
	}

	second, err := domain.NewWithdrawal(
		fixture.account.ID,
		25_000,
		time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC),
		fixture.user.ID,
	)
	if err != nil {
		t.Fatalf("NewWithdrawal(second) error = %v", err)
	}

	otherAccount, otherUser := newTransactionAccountFixture(
		t,
		db,
		fixture.year.ID,
	)
	other, err := domain.NewDeposit(
		otherAccount.ID,
		999_000,
		time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC),
		otherUser.ID,
	)
	if err != nil {
		t.Fatalf("NewDeposit(other) error = %v", err)
	}

	for _, transaction := range []domain.Transaction{first, second, other} {
		if err := repo.Create(ctx, transaction); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		t.Cleanup(func() {
			deleteTestTransaction(t, db, transaction.ID)
		})
	}

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

	if got[0].ID != second.ID {
		t.Fatalf("got[0].ID = %v, want %v", got[0].ID, second.ID)
	}

	if got[1].ID != first.ID {
		t.Fatalf("got[1].ID = %v, want %v", got[1].ID, first.ID)
	}
}

func TestTransactionRepository_ListBySavingsAccountEmpty(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewTransactionRepository(db)

	fixture := newTransactionFixture(t, db)

	got, err := repo.ListBySavingsAccount(
		context.Background(),
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

func TestTransactionRepository_ListBySavingsAccountPagination(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewTransactionRepository(db)
	ctx := context.Background()

	fixture := newTransactionFixture(t, db)

	transactions := make([]domain.Transaction, 0, 3)

	for i := 0; i < 3; i++ {
		transaction, err := domain.NewDeposit(
			fixture.account.ID,
			domain.Money(100_000+i),
			time.Date(
				2026,
				time.September,
				20+i,
				0,
				0,
				0,
				0,
				time.UTC,
			),
			fixture.user.ID,
		)
		if err != nil {
			t.Fatalf("NewDeposit() error = %v", err)
		}

		if err := repo.Create(ctx, transaction); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		transactions = append(transactions, transaction)

		t.Cleanup(func() {
			deleteTestTransaction(t, db, transaction.ID)
		})
	}

	got, err := repo.ListBySavingsAccount(
		ctx,
		fixture.account.ID,
		repository.ListOptions{
			Limit:  1,
			Offset: 1,
		},
	)
	if err != nil {
		t.Fatalf("ListBySavingsAccount() error = %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}

	if got[0].ID != transactions[1].ID {
		t.Fatalf(
			"got[0].ID = %v, want %v",
			got[0].ID,
			transactions[1].ID,
		)
	}
}

func TestTransactionRepository_ListActiveBySavingsAccount(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewTransactionRepository(db)
	ctx := context.Background()

	fixture := newTransactionFixture(t, db)

	active, err := domain.NewDeposit(
		fixture.account.ID,
		100_000,
		time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC),
		fixture.user.ID,
	)
	if err != nil {
		t.Fatalf("NewDeposit(active) error = %v", err)
	}

	cancelled, err := domain.NewWithdrawal(
		fixture.account.ID,
		25_000,
		time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC),
		fixture.user.ID,
	)
	if err != nil {
		t.Fatalf("NewWithdrawal(cancelled) error = %v", err)
	}

	if err := cancelled.Cancel(); err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}

	for _, transaction := range []domain.Transaction{active, cancelled} {
		if err := repo.Create(ctx, transaction); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		t.Cleanup(func() {
			deleteTestTransaction(t, db, transaction.ID)
		})
	}

	got, err := repo.ListActiveBySavingsAccount(
		ctx,
		fixture.account.ID,
	)
	if err != nil {
		t.Fatalf(
			"ListActiveBySavingsAccount() error = %v",
			err,
		)
	}

	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}

	assertTransactionEqual(t, got[0], active)
}

func TestTransactionRepository_ListActiveBySavingsAccountEmpty(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewTransactionRepository(db)

	fixture := newTransactionFixture(t, db)

	got, err := repo.ListActiveBySavingsAccount(
		context.Background(),
		fixture.account.ID,
	)
	if err != nil {
		t.Fatalf(
			"ListActiveBySavingsAccount() error = %v",
			err,
		)
	}

	if got == nil {
		t.Fatal("ListActiveBySavingsAccount() returned nil slice")
	}

	if len(got) != 0 {
		t.Fatalf("len(got) = %d, want 0", len(got))
	}
}

func TestTransactionRepository_Update(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewTransactionRepository(db)
	ctx := context.Background()

	fixture := newTransactionFixture(t, db)

	transaction, err := domain.NewDeposit(
		fixture.account.ID,
		100_000,
		time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC),
		fixture.user.ID,
	)
	if err != nil {
		t.Fatalf("NewDeposit() error = %v", err)
	}

	if err := repo.Create(ctx, transaction); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestTransaction(t, db, transaction.ID)
	})

	transaction.Type = domain.TransactionWithdrawal
	transaction.Amount = 50_000
	transaction.TransactionDate = time.Date(
		2026,
		9,
		25,
		0,
		0,
		0,
		0,
		time.UTC,
	)
	transaction.UpdatedAt = time.Now()

	if err := repo.Update(ctx, transaction); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, err := repo.GetByID(ctx, transaction.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	assertTransactionEqual(t, got, transaction)
}

func TestTransactionRepository_UpdateCancel(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewTransactionRepository(db)
	ctx := context.Background()

	fixture := newTransactionFixture(t, db)

	transaction, err := domain.NewDeposit(
		fixture.account.ID,
		100_000,
		time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC),
		fixture.user.ID,
	)
	if err != nil {
		t.Fatalf("NewDeposit() error = %v", err)
	}

	if err := repo.Create(ctx, transaction); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestTransaction(t, db, transaction.ID)
	})

	if err := transaction.Cancel(); err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}

	if err := repo.Update(ctx, transaction); err != nil {
		t.Fatalf("Update(cancelled) error = %v", err)
	}

	got, err := repo.GetByID(ctx, transaction.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	assertTransactionEqual(t, got, transaction)
}

func TestTransactionRepository_UpdateNotFound(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewTransactionRepository(db)

	transaction, err := domain.NewDeposit(
		uuid.New(),
		100_000,
		time.Now(),
		uuid.New(),
	)
	if err != nil {
		t.Fatalf("NewDeposit() error = %v", err)
	}

	err = repo.Update(
		context.Background(),
		transaction,
	)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf(
			"Update() error = %v, want ErrNotFound",
			err,
		)
	}
}

type transactionFixture struct {
	user    domain.User
	student domain.Student
	year    domain.AcademicYear
	account domain.SavingsAccount
}

func newTransactionFixture(
	t *testing.T,
	db *sql.DB,
) transactionFixture {
	t.Helper()

	ctx := context.Background()

	userRepo := postgres.NewUserRepository(db)
	studentRepo := postgres.NewStudentRepository(db)
	academicYearRepo := postgres.NewAcademicYearRepository(db)
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
		"Siswa Transaction",
		user.ID,
	)
	if err := studentRepo.Create(ctx, student); err != nil {
		t.Fatalf("create fixture student: %v", err)
	}

	t.Cleanup(func() {
		deleteTestStudent(t, db, student.ID)
	})

	year := newTestAcademicYear(t)
	year.Name = "2026/2027"
	year.StartDate = time.Date(
		2026,
		7,
		1,
		0,
		0,
		0,
		0,
		time.UTC,
	)
	year.EndDate = time.Date(
		2027,
		7,
		1,
		0,
		0,
		0,
		0,
		time.UTC,
	)

	if err := academicYearRepo.Create(ctx, year); err != nil {
		t.Fatalf("create fixture academic year: %v", err)
	}

	t.Cleanup(func() {
		deleteTestAcademicYear(t, db, year.ID)
	})

	account, err := domain.NewSavingsAccount(
		student.ID,
		year.ID,
	)
	if err != nil {
		t.Fatalf("NewSavingsAccount() error = %v", err)
	}

	if err := accountRepo.Create(ctx, account); err != nil {
		t.Fatalf("create fixture savings account: %v", err)
	}

	t.Cleanup(func() {
		deleteTestSavingsAccount(t, db, account.ID)
	})

	return transactionFixture{
		user:    user,
		student: student,
		year:    year,
		account: account,
	}
}

func newTransactionAccountFixture(
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
		t.Fatalf("create fixture user: %v", err)
	}

	t.Cleanup(func() {
		deleteTestUser(t, db, user.ID)
	})

	student := newTestStudentWithUserID(
		t,
		"Siswa Transaction Other",
		user.ID,
	)
	if err := studentRepo.Create(ctx, student); err != nil {
		t.Fatalf("create fixture student: %v", err)
	}

	t.Cleanup(func() {
		deleteTestStudent(t, db, student.ID)
	})

	account, err := domain.NewSavingsAccount(
		student.ID,
		academicYearID,
	)
	if err != nil {
		t.Fatalf("NewSavingsAccount() error = %v", err)
	}

	if err := accountRepo.Create(ctx, account); err != nil {
		t.Fatalf("create fixture savings account: %v", err)
	}

	t.Cleanup(func() {
		deleteTestSavingsAccount(t, db, account.ID)
	})

	return account, user
}

func assertTransactionEqual(
	t *testing.T,
	got domain.Transaction,
	want domain.Transaction,
) {
	t.Helper()

	if got.ID != want.ID {
		t.Errorf("ID = %v, want %v", got.ID, want.ID)
	}

	if got.SavingsAccountID != want.SavingsAccountID {
		t.Errorf(
			"SavingsAccountID = %v, want %v",
			got.SavingsAccountID,
			want.SavingsAccountID,
		)
	}

	if got.Type != want.Type {
		t.Errorf("Type = %v, want %v", got.Type, want.Type)
	}

	if got.Amount != want.Amount {
		t.Errorf("Amount = %v, want %v", got.Amount, want.Amount)
	}

	if !got.TransactionDate.Equal(want.TransactionDate) {
		t.Errorf(
			"TransactionDate = %v, want %v",
			got.TransactionDate,
			want.TransactionDate,
		)
	}

	if got.Status != want.Status {
		t.Errorf(
			"Status = %v, want %v",
			got.Status,
			want.Status,
		)
	}

	if got.CreatedBy != want.CreatedBy {
		t.Errorf(
			"CreatedBy = %v, want %v",
			got.CreatedBy,
			want.CreatedBy,
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

func deleteTestTransaction(
	t *testing.T,
	db *sql.DB,
	id uuid.UUID,
) {
	t.Helper()

	_, err := db.ExecContext(
		context.Background(),
		"DELETE FROM transactions WHERE id = $1",
		id,
	)
	if err != nil {
		t.Errorf("delete test transaction: %v", err)
	}
}
