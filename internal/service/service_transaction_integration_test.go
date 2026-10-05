package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/repository/postgres"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

func TestSavingsAndTransactionServicesIntegration_CrossServiceCommit(t *testing.T) {
	db := openServiceScopeTestDatabase(t)
	ctx := context.Background()

	cleanupServiceScopeFixtures(t, db)

	userRepo := postgres.NewUserRepository(db)
	studentRepo := postgres.NewStudentRepository(db)
	academicYearRepo := postgres.NewAcademicYearRepository(db)
	savingsAccountRepo := postgres.NewSavingsAccountRepository(db)
	transactionRepo := postgres.NewTransactionRepository(db)
	auditRepo := postgres.NewAuditRepository(db)

	actorUser := newServiceScopeUser(t, domain.RoleAdmin)
	if err := userRepo.Create(ctx, actorUser); err != nil {
		t.Fatalf("create actor: %v", err)
	}

	student := newServiceScopeStudent(t, "Cross-Service Commit "+uuid.NewString())
	if err := studentRepo.Create(ctx, student); err != nil {
		t.Fatalf("create student: %v", err)
	}

	academicYear := newServiceScopeAcademicYear(
		t,
		"Cross-Service Commit "+uuid.NewString(),
	)
	if err := academicYearRepo.Create(ctx, academicYear); err != nil {
		t.Fatalf("create academic year: %v", err)
	}

	manager := postgres.NewUnitOfWorkManager(db)

	savingsService, err := service.NewSavingsAccountService(service.Dependencies{
		Repositories: repository.RepositorySet{
			Students: studentRepo,
		},
		UOW: manager,
	})
	if err != nil {
		t.Fatalf("NewSavingsAccountService() error = %v", err)
	}

	transactionService, err := service.NewTransactionService(service.Dependencies{
		Repositories: repository.RepositorySet{
			Students:        studentRepo,
			SavingsAccounts: savingsAccountRepo,
			Transactions:    transactionRepo,
		},
		UOW: manager,
	})
	if err != nil {
		t.Fatalf("NewTransactionService() error = %v", err)
	}

	actor := service.Actor{
		UserID: actorUser.ID,
		Role:   domain.RoleAdmin,
	}

	var (
		accountID     uuid.UUID
		transactionID uuid.UUID
	)

	err = service.WithinTransaction(
		ctx,
		manager,
		func(txCtx context.Context, _ repository.RepositorySet) error {
			accountOutput, err := savingsService.Create(
				txCtx,
				service.CreateSavingsAccountInput{
					Actor:          actor,
					StudentID:      student.ID,
					AcademicYearID: academicYear.ID,
				},
			)
			if err != nil {
				return err
			}

			accountID = accountOutput.Account.ID

			transactionOutput, err := transactionService.Deposit(
				txCtx,
				service.DepositInput{
					Actor:            actor,
					SavingsAccountID: accountID,
					Amount:           domain.Money(100_000),
					TransactionDate:  time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
				},
			)
			if err != nil {
				return err
			}

			transactionID = transactionOutput.Transaction.ID
			return nil
		},
	)
	if err != nil {
		t.Fatalf("WithinTransaction() error = %v", err)
	}

	account, err := savingsAccountRepo.GetByID(ctx, accountID)
	if err != nil {
		t.Fatalf("GetByID() account after commit: %v", err)
	}

	if account.StudentID != student.ID {
		t.Fatalf("account StudentID = %v, want %v", account.StudentID, student.ID)
	}

	transaction, err := transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		t.Fatalf("GetByID() transaction after commit: %v", err)
	}

	if transaction.SavingsAccountID != accountID {
		t.Fatalf(
			"transaction SavingsAccountID = %v, want %v",
			transaction.SavingsAccountID,
			accountID,
		)
	}

	accountAudits, err := auditRepo.ListByEntity(
		ctx,
		domain.AuditEntitySavingsAccount,
		accountID,
		repository.ListOptions{Limit: 10},
	)
	if err != nil {
		t.Fatalf("ListByEntity() account audit: %v", err)
	}

	if len(accountAudits) != 1 {
		t.Fatalf("account audit count = %d, want 1", len(accountAudits))
	}

	transactionAudits, err := auditRepo.ListByEntity(
		ctx,
		domain.AuditEntityTransaction,
		transactionID,
		repository.ListOptions{Limit: 10},
	)
	if err != nil {
		t.Fatalf("ListByEntity() transaction audit: %v", err)
	}

	if len(transactionAudits) != 1 {
		t.Fatalf("transaction audit count = %d, want 1", len(transactionAudits))
	}

	if accountAudits[0].Action != "CREATE" {
		t.Fatalf("account audit action = %q, want CREATE", accountAudits[0].Action)
	}

	if transactionAudits[0].Action != "CREATE" {
		t.Fatalf(
			"transaction audit action = %q, want CREATE",
			transactionAudits[0].Action,
		)
	}
}

func TestSavingsAndTransactionServicesIntegration_CrossServiceRollbackIsAtomic(t *testing.T) {
	db := openServiceScopeTestDatabase(t)
	ctx := context.Background()

	cleanupServiceScopeFixtures(t, db)

	userRepo := postgres.NewUserRepository(db)
	studentRepo := postgres.NewStudentRepository(db)
	academicYearRepo := postgres.NewAcademicYearRepository(db)
	savingsAccountRepo := postgres.NewSavingsAccountRepository(db)
	transactionRepo := postgres.NewTransactionRepository(db)
	auditRepo := postgres.NewAuditRepository(db)

	actorUser := newServiceScopeUser(t, domain.RoleAdmin)
	if err := userRepo.Create(ctx, actorUser); err != nil {
		t.Fatalf("create actor: %v", err)
	}

	student := newServiceScopeStudent(t, "Cross-Service Rollback "+uuid.NewString())
	if err := studentRepo.Create(ctx, student); err != nil {
		t.Fatalf("create student: %v", err)
	}

	academicYear := newServiceScopeAcademicYear(
		t,
		"Cross-Service Rollback "+uuid.NewString(),
	)
	if err := academicYearRepo.Create(ctx, academicYear); err != nil {
		t.Fatalf("create academic year: %v", err)
	}

	manager := postgres.NewUnitOfWorkManager(db)

	savingsService, err := service.NewSavingsAccountService(service.Dependencies{
		Repositories: repository.RepositorySet{
			Students: studentRepo,
		},
		UOW: manager,
	})
	if err != nil {
		t.Fatalf("NewSavingsAccountService() error = %v", err)
	}

	transactionService, err := service.NewTransactionService(service.Dependencies{
		Repositories: repository.RepositorySet{
			Students:        studentRepo,
			SavingsAccounts: savingsAccountRepo,
			Transactions:    transactionRepo,
		},
		UOW: manager,
	})
	if err != nil {
		t.Fatalf("NewTransactionService() error = %v", err)
	}

	actor := service.Actor{
		UserID: actorUser.ID,
		Role:   domain.RoleAdmin,
	}

	wantErr := errors.New("force cross-service rollback")

	var accountID uuid.UUID
	var transactionID uuid.UUID

	err = service.WithinTransaction(
		ctx,
		manager,
		func(txCtx context.Context, _ repository.RepositorySet) error {
			accountOutput, err := savingsService.Create(
				txCtx,
				service.CreateSavingsAccountInput{
					Actor:          actor,
					StudentID:      student.ID,
					AcademicYearID: academicYear.ID,
				},
			)
			if err != nil {
				return err
			}

			accountID = accountOutput.Account.ID

			transactionOutput, err := transactionService.Deposit(
				txCtx,
				service.DepositInput{
					Actor:            actor,
					SavingsAccountID: accountID,
					Amount:           domain.Money(75_000),
					TransactionDate:  time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC),
				},
			)
			if err != nil {
				return err
			}

			transactionID = transactionOutput.Transaction.ID

			return wantErr
		},
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("WithinTransaction() error = %v, want %v", err, wantErr)
	}

	if _, err := savingsAccountRepo.GetByID(ctx, accountID); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf(
			"GetByID() account error = %v, want repository.ErrNotFound",
			err,
		)
	}

	if _, err := transactionRepo.GetByID(ctx, transactionID); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf(
			"GetByID() transaction error = %v, want repository.ErrNotFound",
			err,
		)
	}

	accountAudits, err := auditRepo.ListByEntity(
		ctx,
		domain.AuditEntitySavingsAccount,
		accountID,
		repository.ListOptions{Limit: 10},
	)
	if err != nil {
		t.Fatalf("ListByEntity() account audit: %v", err)
	}

	if len(accountAudits) != 0 {
		t.Fatalf("account audit count = %d, want 0 after rollback", len(accountAudits))
	}

	transactionAudits, err := auditRepo.ListByEntity(
		ctx,
		domain.AuditEntityTransaction,
		transactionID,
		repository.ListOptions{Limit: 10},
	)
	if err != nil {
		t.Fatalf("ListByEntity() transaction audit: %v", err)
	}

	if len(transactionAudits) != 0 {
		t.Fatalf(
			"transaction audit count = %d, want 0 after rollback",
			len(transactionAudits),
		)
	}
}
