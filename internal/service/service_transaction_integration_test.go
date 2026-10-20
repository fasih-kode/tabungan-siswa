package service_test

import (
	"context"
	"database/sql"
	"encoding/json"
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

	actorUser := newServiceScopeUser(t, domain.RoleWaliKelas)
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

	assignFinancialIntegrationScope(t, db, actorUser.ID, student.ID, academicYear.ID)
	manager := postgres.NewUnitOfWorkManager(db)

	savingsService, err := service.NewSavingsAccountService(service.Dependencies{
		Repositories: repository.RepositorySet{
			Students:                studentRepo,
			StudentClassHistories:   postgres.NewStudentClassHistoryRepository(db),
			TeacherClassAssignments: postgres.NewTeacherClassAssignmentRepository(db), AcademicYears: academicYearRepo,
			SavingsAccounts: savingsAccountRepo, Transactions: transactionRepo, AuditLogs: auditRepo,
		},
		UOW: manager,
	})
	if err != nil {
		t.Fatalf("NewSavingsAccountService() error = %v", err)
	}

	transactionService, err := service.NewTransactionService(service.Dependencies{
		Repositories: repository.RepositorySet{
			Students:                studentRepo,
			StudentClassHistories:   postgres.NewStudentClassHistoryRepository(db),
			TeacherClassAssignments: postgres.NewTeacherClassAssignmentRepository(db),
			SavingsAccounts:         savingsAccountRepo,
			Transactions:            transactionRepo,
		},
		UOW: manager,
	})
	if err != nil {
		t.Fatalf("NewTransactionService() error = %v", err)
	}

	actor := service.Actor{
		UserID: actorUser.ID,
		Role:   domain.RoleWaliKelas,
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
	passbook, err := savingsService.GetPassbook(ctx, service.GetSavingsAccountPassbookInput{Actor: actor, SavingsAccountID: accountID})
	if err != nil {
		t.Fatalf("GetPassbook() error = %v", err)
	}
	if passbook.Balance != domain.Money(100_000) || len(passbook.Entries) != 1 || passbook.Entries[0].RunningBalance != domain.Money(100_000) {
		t.Fatalf("passbook balance/entries = %d/%+v, want 100000/one entry", passbook.Balance, passbook.Entries)
	}

	secondStudent := newServiceScopeStudent(t, "Cross-Service Separate Student "+uuid.NewString())
	if err := studentRepo.Create(ctx, secondStudent); err != nil {
		t.Fatalf("create second student: %v", err)
	}
	assignFinancialIntegrationScope(t, db, actorUser.ID, secondStudent.ID, academicYear.ID)
	secondAccount, err := savingsService.Create(ctx, service.CreateSavingsAccountInput{Actor: actor, StudentID: secondStudent.ID, AcademicYearID: academicYear.ID})
	if err != nil {
		t.Fatalf("create second student's savings account: %v", err)
	}
	if _, err := transactionService.Deposit(ctx, service.DepositInput{Actor: actor, SavingsAccountID: secondAccount.Account.ID, Amount: domain.Money(25_000), TransactionDate: time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)}); err != nil {
		t.Fatalf("deposit to second student's account: %v", err)
	}
	// An admin without any class assignment can supervise both classes.
	admin := service.Actor{UserID: uuid.New(), Role: domain.RoleAdmin}
	firstBalance, err := savingsService.GetBalance(ctx, service.GetSavingsAccountBalanceInput{Actor: admin, SavingsAccountID: accountID})
	if err != nil {
		t.Fatal(err)
	}
	secondBalance, err := savingsService.GetBalance(ctx, service.GetSavingsAccountBalanceInput{Actor: admin, SavingsAccountID: secondAccount.Account.ID})
	if err != nil {
		t.Fatal(err)
	}
	if firstBalance.Balance != domain.Money(100_000) || secondBalance.Balance != domain.Money(25_000) {
		t.Fatalf("balances crossed savings accounts: first=%d second=%d", firstBalance.Balance, secondBalance.Balance)
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

	actorUser := newServiceScopeUser(t, domain.RoleWaliKelas)
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

	assignFinancialIntegrationScope(t, db, actorUser.ID, student.ID, academicYear.ID)
	manager := postgres.NewUnitOfWorkManager(db)

	savingsService, err := service.NewSavingsAccountService(service.Dependencies{
		Repositories: repository.RepositorySet{
			Students:                studentRepo,
			StudentClassHistories:   postgres.NewStudentClassHistoryRepository(db),
			TeacherClassAssignments: postgres.NewTeacherClassAssignmentRepository(db),
		},
		UOW: manager,
	})
	if err != nil {
		t.Fatalf("NewSavingsAccountService() error = %v", err)
	}

	transactionService, err := service.NewTransactionService(service.Dependencies{
		Repositories: repository.RepositorySet{
			Students:                studentRepo,
			StudentClassHistories:   postgres.NewStudentClassHistoryRepository(db),
			TeacherClassAssignments: postgres.NewTeacherClassAssignmentRepository(db),
			SavingsAccounts:         savingsAccountRepo,
			Transactions:            transactionRepo,
		},
		UOW: manager,
	})
	if err != nil {
		t.Fatalf("NewTransactionService() error = %v", err)
	}

	actor := service.Actor{
		UserID: actorUser.ID,
		Role:   domain.RoleWaliKelas,
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

func TestTransactionServiceIntegration_DateBoundsConcurrentWithdrawalsAndLifecycleLocks(t *testing.T) {
	db := openServiceScopeTestDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cleanupServiceScopeFixtures(t, db)

	userRepo := postgres.NewUserRepository(db)
	studentRepo := postgres.NewStudentRepository(db)
	yearRepo := postgres.NewAcademicYearRepository(db)
	accountRepo := postgres.NewSavingsAccountRepository(db)
	transactionRepo := postgres.NewTransactionRepository(db)
	settlementRepo := postgres.NewSettlementRepository(db)
	auditRepo := postgres.NewAuditRepository(db)
	historyRepo := postgres.NewStudentClassHistoryRepository(db)
	assignmentRepo := postgres.NewTeacherClassAssignmentRepository(db)
	transferRepo := postgres.NewClassTransferRequestRepository(db)
	manager := postgres.NewUnitOfWorkManager(db)

	teacher := newServiceScopeUser(t, domain.RoleWaliKelas)
	admin := newServiceScopeUser(t, domain.RoleAdmin)
	for _, user := range []domain.User{teacher, admin} {
		if err := userRepo.Create(ctx, user); err != nil {
			t.Fatalf("create user: %v", err)
		}
	}
	student := newServiceScopeStudent(t, "Transaction Lifecycle "+uuid.NewString())
	if err := studentRepo.Create(ctx, student); err != nil {
		t.Fatalf("create student: %v", err)
	}
	year := newServiceScopeAcademicYear(t, "Transaction Lifecycle "+uuid.NewString())
	if err := yearRepo.Create(ctx, year); err != nil {
		t.Fatalf("create year: %v", err)
	}
	assignFinancialIntegrationScope(t, db, teacher.ID, student.ID, year.ID)
	account, err := domain.NewSavingsAccount(student.ID, year.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := accountRepo.Create(ctx, account); err != nil {
		t.Fatalf("create account: %v", err)
	}

	repos := repository.RepositorySet{
		AcademicYears: yearRepo, SavingsAccounts: accountRepo, Transactions: transactionRepo,
		Students: studentRepo, StudentClassHistories: historyRepo,
		TeacherClassAssignments: assignmentRepo, Settlements: settlementRepo,
		AuditLogs: auditRepo, ClassTransferRequests: transferRepo,
	}
	txService, err := service.NewTransactionService(service.Dependencies{Repositories: repos, UOW: manager})
	if err != nil {
		t.Fatal(err)
	}
	settlementService, err := service.NewSettlementService(service.Dependencies{Repositories: repos, UOW: manager})
	if err != nil {
		t.Fatal(err)
	}
	yearService, err := service.NewAcademicYearService(service.Dependencies{Repositories: repos, UOW: manager})
	if err != nil {
		t.Fatal(err)
	}
	actor := service.Actor{UserID: teacher.ID, Role: domain.RoleWaliKelas}
	adminActor := service.Actor{UserID: admin.ID, Role: domain.RoleAdmin}
	firstDate := year.StartDate
	lastValidDate := year.EndDate.AddDate(0, 0, -1)
	endExclusiveDate := year.EndDate

	if _, err := txService.Deposit(ctx, service.DepositInput{Actor: actor, SavingsAccountID: account.ID, Amount: 100_000, TransactionDate: firstDate}); err != nil {
		t.Fatalf("deposit on inclusive start date: %v", err)
	}
	if _, err := txService.Deposit(ctx, service.DepositInput{Actor: actor, SavingsAccountID: account.ID, Amount: 1_000, TransactionDate: endExclusiveDate}); !errors.Is(err, service.ErrTransactionDateOutOfAcademicYear) {
		t.Fatalf("deposit on exclusive end date error = %v", err)
	}
	if _, err := txService.Deposit(ctx, service.DepositInput{Actor: adminActor, SavingsAccountID: account.ID, Amount: 1_000, TransactionDate: lastValidDate}); err != nil {
		t.Fatalf("admin deposit without class scope on last valid date: %v", err)
	}

	withdrawalDate := year.StartDate.AddDate(0, 0, 2)
	results := make(chan error, 2)
	for range 2 {
		go func() {
			_, err := txService.Withdrawal(ctx, service.WithdrawalInput{Actor: actor, SavingsAccountID: account.ID, Amount: 75_000, TransactionDate: withdrawalDate})
			results <- err
		}()
	}
	successes, rejected := 0, 0
	for range 2 {
		err := <-results
		switch {
		case err == nil:
			successes++
		case errors.Is(err, domain.ErrNegativeBalance):
			rejected++
		default:
			t.Fatalf("concurrent withdrawal error = %v", err)
		}
	}
	if successes != 1 || rejected != 1 {
		t.Fatalf("concurrent withdrawals success/rejected = %d/%d, want 1/1", successes, rejected)
	}
	if _, err := txService.Withdrawal(ctx, service.WithdrawalInput{Actor: actor, SavingsAccountID: account.ID, Amount: 30_000, TransactionDate: year.StartDate.AddDate(0, 0, 1)}); !errors.Is(err, domain.ErrNegativeBalance) {
		t.Fatalf("backdated withdrawal invalidating a later balance error = %v, want negative balance", err)
	}

	// A deposit that has acquired both locks finishes before settlement can
	// lock the same account; the resulting snapshot must include that deposit.
	student.Status = domain.StudentLeft
	student.UpdatedAt = time.Now().UTC()
	if err := studentRepo.Update(ctx, student); err != nil {
		t.Fatalf("mark student LEFT: %v", err)
	}
	depositReady, releaseDeposit := make(chan struct{}), make(chan struct{})
	depositDone := make(chan error, 1)
	go func() {
		depositDone <- service.WithinTransaction(ctx, manager, func(txCtx context.Context, _ repository.RepositorySet) error {
			if _, err := txService.Deposit(txCtx, service.DepositInput{Actor: actor, SavingsAccountID: account.ID, Amount: 5_000, TransactionDate: year.StartDate.AddDate(0, 0, 3)}); err != nil {
				return err
			}
			close(depositReady)
			<-releaseDeposit
			return nil
		})
	}()
	select {
	case <-depositReady:
	case <-ctx.Done():
		t.Fatalf("deposit did not acquire account/year locks: %v", ctx.Err())
	}
	settlementDone := make(chan error, 1)
	go func() {
		_, err := settlementService.SettleStudentLeaving(ctx, service.SettleStudentLeavingInput{Actor: actor, SavingsAccountID: account.ID})
		settlementDone <- err
	}()
	select {
	case err := <-settlementDone:
		t.Fatalf("settlement completed while transaction held account lock: %v", err)
	case <-time.After(150 * time.Millisecond):
	}
	close(releaseDeposit)
	if err := <-depositDone; err != nil {
		t.Fatalf("commit locked deposit: %v", err)
	}
	if err := <-settlementDone; err != nil {
		t.Fatalf("settle after deposit: %v", err)
	}
	settledAccount, err := accountRepo.GetByID(ctx, account.ID)
	if err != nil || !settledAccount.IsSettled() {
		t.Fatalf("account after settlement = %+v, err=%v", settledAccount, err)
	}
	settlement, err := settlementRepo.GetCompletedBySavingsAccount(ctx, account.ID)
	if err != nil {
		t.Fatalf("get settlement: %v", err)
	}
	passbook, err := txService.List(ctx, service.ListTransactionsInput{Actor: actor, SavingsAccountID: account.ID, Options: service.ListOptions{Limit: 10}})
	if err != nil {
		t.Fatalf("list transactions after settlement: %v", err)
	}
	if settlement.Amount != passbook.Balance || settlement.Amount != 31_000 {
		t.Fatalf("settlement snapshot=%d transaction balance=%d, want 31000", settlement.Amount, passbook.Balance)
	}

	// The inverse interleaving: settlement holds the account lock first; a
	// waiting transaction must observe SETTLED and leave no new row.
	student2 := newServiceScopeStudent(t, "Settlement Wins "+uuid.NewString())
	if err := studentRepo.Create(ctx, student2); err != nil {
		t.Fatalf("create second student: %v", err)
	}
	assignFinancialIntegrationScope(t, db, teacher.ID, student2.ID, year.ID)
	account2, err := domain.NewSavingsAccount(student2.ID, year.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := accountRepo.Create(ctx, account2); err != nil {
		t.Fatalf("create second account: %v", err)
	}
	student2.Status = domain.StudentLeft
	student2.UpdatedAt = time.Now().UTC()
	if err := studentRepo.Update(ctx, student2); err != nil {
		t.Fatalf("mark second student LEFT: %v", err)
	}
	settlementReady, releaseSettlement := make(chan struct{}), make(chan struct{})
	settlementTxDone := make(chan error, 1)
	go func() {
		settlementTxDone <- service.WithinTransaction(ctx, manager, func(txCtx context.Context, _ repository.RepositorySet) error {
			if _, err := settlementService.SettleStudentLeaving(txCtx, service.SettleStudentLeavingInput{Actor: actor, SavingsAccountID: account2.ID}); err != nil {
				return err
			}
			close(settlementReady)
			<-releaseSettlement
			return nil
		})
	}()
	select {
	case <-settlementReady:
	case <-ctx.Done():
		t.Fatalf("settlement did not acquire account lock: %v", ctx.Err())
	}
	depositAfterSettlement := make(chan error, 1)
	go func() {
		_, err := txService.Deposit(ctx, service.DepositInput{Actor: actor, SavingsAccountID: account2.ID, Amount: 1_000, TransactionDate: year.StartDate})
		depositAfterSettlement <- err
	}()
	select {
	case err := <-depositAfterSettlement:
		t.Fatalf("deposit completed before settlement committed: %v", err)
	case <-time.After(150 * time.Millisecond):
	}
	close(releaseSettlement)
	if err := <-settlementTxDone; err != nil {
		t.Fatalf("commit account settlement: %v", err)
	}
	if err := <-depositAfterSettlement; !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("deposit after settlement error = %v, want invalid transition", err)
	}

	// A separate OPEN account verifies that year closing waits for a transaction
	// holding FOR SHARE, then prevents any later write once CLOSING is committed.
	student3 := newServiceScopeStudent(t, "Year Lock "+uuid.NewString())
	if err := studentRepo.Create(ctx, student3); err != nil {
		t.Fatalf("create third student: %v", err)
	}
	assignFinancialIntegrationScope(t, db, teacher.ID, student3.ID, year.ID)
	account3, err := domain.NewSavingsAccount(student3.ID, year.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := accountRepo.Create(ctx, account3); err != nil {
		t.Fatalf("create third account: %v", err)
	}
	writeReady, releaseWrite := make(chan struct{}), make(chan struct{})
	writeDone := make(chan error, 1)
	go func() {
		writeDone <- service.WithinTransaction(ctx, manager, func(txCtx context.Context, _ repository.RepositorySet) error {
			if _, err := txService.Deposit(txCtx, service.DepositInput{Actor: actor, SavingsAccountID: account3.ID, Amount: 8_000, TransactionDate: lastValidDate}); err != nil {
				return err
			}
			close(writeReady)
			<-releaseWrite
			return nil
		})
	}()
	select {
	case <-writeReady:
	case <-ctx.Done():
		t.Fatalf("second deposit did not acquire year lock: %v", ctx.Err())
	}
	closingStarted, closingDone := make(chan struct{}), make(chan error, 1)
	go func() {
		close(closingStarted)
		_, err := yearService.StartClosing(ctx, service.StartClosingAcademicYearInput{Actor: adminActor, AcademicYearID: year.ID})
		closingDone <- err
	}()
	<-closingStarted
	select {
	case err := <-closingDone:
		t.Fatalf("StartClosing completed while transaction held FOR SHARE: %v", err)
	case <-time.After(150 * time.Millisecond):
	}
	close(releaseWrite)
	if err := <-writeDone; err != nil {
		t.Fatalf("commit transaction before closing: %v", err)
	}
	if err := <-closingDone; err != nil {
		t.Fatalf("StartClosing after transaction commit: %v", err)
	}
	if _, err := txService.Deposit(ctx, service.DepositInput{Actor: adminActor, SavingsAccountID: account3.ID, Amount: 1_000, TransactionDate: lastValidDate}); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("deposit in CLOSING year error = %v, want invalid transition", err)
	}
}

func TestTransactionCorrectionAuditIntegration(t *testing.T) {
	db := openServiceScopeTestDatabase(t)
	ctx := context.Background()
	cleanupServiceScopeFixtures(t, db)
	userRepo := postgres.NewUserRepository(db)
	studentRepo := postgres.NewStudentRepository(db)
	yearRepo := postgres.NewAcademicYearRepository(db)
	accountRepo := postgres.NewSavingsAccountRepository(db)
	txRepo := postgres.NewTransactionRepository(db)
	auditRepo := postgres.NewAuditRepository(db)
	historyRepo := postgres.NewStudentClassHistoryRepository(db)
	assignmentRepo := postgres.NewTeacherClassAssignmentRepository(db)
	manager := postgres.NewUnitOfWorkManager(db)
	teacher := newServiceScopeUser(t, domain.RoleWaliKelas)
	otherTeacher := newServiceScopeUser(t, domain.RoleWaliKelas)
	admin := newServiceScopeUser(t, domain.RoleAdmin)
	for _, user := range []domain.User{teacher, otherTeacher, admin} {
		if err := userRepo.Create(ctx, user); err != nil {
			t.Fatal(err)
		}
	}
	student := newServiceScopeStudent(t, "Correction audit "+uuid.NewString())
	if err := studentRepo.Create(ctx, student); err != nil {
		t.Fatal(err)
	}
	year := newServiceScopeAcademicYear(t, "Correction audit "+uuid.NewString())
	if err := yearRepo.Create(ctx, year); err != nil {
		t.Fatal(err)
	}
	assignFinancialIntegrationScope(t, db, teacher.ID, student.ID, year.ID)
	account, err := domain.NewSavingsAccount(student.ID, year.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := accountRepo.Create(ctx, account); err != nil {
		t.Fatal(err)
	}
	repos := repository.RepositorySet{
		AcademicYears: yearRepo, SavingsAccounts: accountRepo, Transactions: txRepo,
		Students: studentRepo, StudentClassHistories: historyRepo,
		TeacherClassAssignments: assignmentRepo, AuditLogs: auditRepo,
	}
	txService, err := service.NewTransactionService(service.Dependencies{Repositories: repos, UOW: manager})
	if err != nil {
		t.Fatal(err)
	}
	auditService, err := service.NewAuditService(service.Dependencies{Repositories: repos, UOW: manager})
	if err != nil {
		t.Fatal(err)
	}
	actor := service.Actor{UserID: teacher.ID, Role: domain.RoleWaliKelas}
	adminActor := service.Actor{UserID: admin.ID, Role: domain.RoleAdmin}
	created, err := txService.Deposit(ctx, service.DepositInput{Actor: actor, SavingsAccountID: account.ID, Amount: 100_000, TransactionDate: year.StartDate})
	if err != nil {
		t.Fatal(err)
	}
	correctedDate := year.StartDate.AddDate(0, 0, 1)
	if _, err := txService.Edit(ctx, service.EditTransactionInput{Actor: actor, TransactionID: created.Transaction.ID, Amount: 75_000, TransactionDate: correctedDate, Reason: "Jumlah salah input"}); err != nil {
		t.Fatalf("Edit(): %v", err)
	}
	logs, err := auditService.ListByEntity(ctx, service.ListAuditByEntityInput{Actor: actor, EntityType: domain.AuditEntityTransaction, EntityID: created.Transaction.ID, Options: service.ListOptions{Limit: 10}})
	if err != nil {
		t.Fatalf("Wali audit read: %v", err)
	}
	if len(logs.AuditLogs) != 2 || logs.AuditLogs[0].Action != "UPDATE" || logs.AuditLogs[1].Action != "CREATE" {
		t.Fatalf("audit timeline = %#v", logs.AuditLogs)
	}
	var after struct {
		Reason      string             `json:"reason"`
		Transaction domain.Transaction `json:"transaction"`
	}
	if err := json.Unmarshal(logs.AuditLogs[0].AfterData, &after); err != nil {
		t.Fatal(err)
	}
	if after.Reason != "Jumlah salah input" || after.Transaction.Amount != 75_000 || !after.Transaction.TransactionDate.Equal(correctedDate) {
		t.Fatalf("audit after snapshot = %+v", after)
	}
	if _, err := auditService.ListByEntity(ctx, service.ListAuditByEntityInput{Actor: service.Actor{UserID: otherTeacher.ID, Role: domain.RoleWaliKelas}, EntityType: domain.AuditEntityTransaction, EntityID: created.Transaction.ID}); !errors.Is(err, service.ErrScopeViolation) {
		t.Fatalf("out-of-scope Wali audit error = %v", err)
	}
	if _, err := auditService.ListByEntity(ctx, service.ListAuditByEntityInput{Actor: adminActor, EntityType: domain.AuditEntityTransaction, EntityID: created.Transaction.ID}); err != nil {
		t.Fatalf("Admin audit read: %v", err)
	}
	if _, err := txService.Cancel(ctx, service.CancelTransactionInput{Actor: actor, TransactionID: created.Transaction.ID, Reason: "Setoran rangkap"}); err != nil {
		t.Fatalf("Cancel(): %v", err)
	}
	listed, err := txService.List(ctx, service.ListTransactionsInput{Actor: actor, SavingsAccountID: account.ID, Options: service.ListOptions{Limit: 10}})
	if err != nil {
		t.Fatal(err)
	}
	if listed.Total != 1 || listed.Balance != 0 || len(listed.Transactions) != 1 || listed.Transactions[0].Status != domain.TransactionCancelled {
		t.Fatalf("history after cancel: total=%d balance=%d rows=%+v", listed.Total, listed.Balance, listed.Transactions)
	}
}

// Each student is placed in a distinct class assigned to the acting teacher.
func assignFinancialIntegrationScope(t *testing.T, db *sql.DB, teacherID, studentID, yearID uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	class := newServiceScopeClass(t, "Financial "+uuid.NewString(), 8)
	if err := postgres.NewClassRepository(db).Create(ctx, class); err != nil {
		t.Fatal(err)
	}
	assignment := newServiceScopeAssignment(t, teacherID, yearID, class.ID)
	if err := postgres.NewTeacherClassAssignmentRepository(db).Create(ctx, assignment); err != nil {
		t.Fatal(err)
	}
	history := newServiceScopeStudentHistory(t, studentID, yearID, class.ID)
	if err := postgres.NewStudentClassHistoryRepository(db).Create(ctx, history); err != nil {
		t.Fatal(err)
	}
}
