package service

import (
	"context"
	"errors"
	"testing"

	"github.com/fasih/tabungan-siswa/internal/repository"
)

type transactionScopeFakeUOW struct {
	repos         repository.RepositorySet
	commitCount   int
	rollbackCount int
}

func (f *transactionScopeFakeUOW) Repositories() repository.RepositorySet {
	return f.repos
}

func (f *transactionScopeFakeUOW) Commit() error {
	f.commitCount++
	return nil
}

func (f *transactionScopeFakeUOW) Rollback() error {
	f.rollbackCount++
	return nil
}

type transactionScopeFakeUOWManager struct {
	uow        repository.UnitOfWork
	beginCount int
	beginErr   error
}

func (f *transactionScopeFakeUOWManager) Begin(context.Context) (repository.UnitOfWork, error) {
	f.beginCount++
	if f.beginErr != nil {
		return nil, f.beginErr
	}
	return f.uow, nil
}

func TestWithinTransactionStartsAndCommitsOuterTransaction(t *testing.T) {
	uow := &transactionScopeFakeUOW{}
	manager := &transactionScopeFakeUOWManager{uow: uow}

	var callbackRepos repository.RepositorySet
	var callbackContext context.Context

	err := WithinTransaction(
		context.Background(),
		manager,
		func(ctx context.Context, repos repository.RepositorySet) error {
			callbackContext = ctx
			callbackRepos = repos
			return nil
		},
	)
	if err != nil {
		t.Fatalf("WithinTransaction returned error: %v", err)
	}

	if manager.beginCount != 1 {
		t.Fatalf("expected one Begin call, got %d", manager.beginCount)
	}
	if uow.commitCount != 1 {
		t.Fatalf("expected one Commit call, got %d", uow.commitCount)
	}
	if uow.rollbackCount != 0 {
		t.Fatalf("expected zero Rollback calls, got %d", uow.rollbackCount)
	}
	if callbackContext == nil {
		t.Fatal("expected transaction context")
	}
	if callbackRepos != uow.repos {
		t.Fatal("expected callback to receive transaction-bound repositories")
	}
}

func TestWithinTransactionRollsBackWhenCallbackFails(t *testing.T) {
	uow := &transactionScopeFakeUOW{}
	manager := &transactionScopeFakeUOWManager{uow: uow}
	wantErr := errors.New("callback failed")

	err := WithinTransaction(
		context.Background(),
		manager,
		func(context.Context, repository.RepositorySet) error {
			return wantErr
		},
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected callback error, got %v", err)
	}

	if manager.beginCount != 1 {
		t.Fatalf("expected one Begin call, got %d", manager.beginCount)
	}
	if uow.commitCount != 0 {
		t.Fatalf("expected zero Commit calls, got %d", uow.commitCount)
	}
	if uow.rollbackCount != 1 {
		t.Fatalf("expected one Rollback call, got %d", uow.rollbackCount)
	}
}

func TestWithinTransactionReusesExistingTransaction(t *testing.T) {
	uow := &transactionScopeFakeUOW{}
	manager := &transactionScopeFakeUOWManager{uow: uow}

	err := WithinTransaction(
		context.Background(),
		manager,
		func(ctx context.Context, repos repository.RepositorySet) error {
			return WithinTransaction(
				ctx,
				manager,
				func(nestedCtx context.Context, nestedRepos repository.RepositorySet) error {
					if nestedCtx != ctx {
						t.Fatal("expected nested operation to reuse transaction context")
					}
					if nestedRepos != repos {
						t.Fatal("expected nested operation to reuse transaction repositories")
					}
					return nil
				},
			)
		},
	)
	if err != nil {
		t.Fatalf("WithinTransaction returned error: %v", err)
	}

	if manager.beginCount != 1 {
		t.Fatalf("expected one Begin call, got %d", manager.beginCount)
	}
	if uow.commitCount != 1 {
		t.Fatalf("expected outer transaction to commit once, got %d", uow.commitCount)
	}
	if uow.rollbackCount != 0 {
		t.Fatalf("expected zero Rollback calls, got %d", uow.rollbackCount)
	}
}

func TestBeginTransactionStartsNewScope(t *testing.T) {
	uow := &transactionScopeFakeUOW{}
	manager := &transactionScopeFakeUOWManager{uow: uow}

	gotUOW, gotRepos, gotCtx, owns, err := beginTransaction(
		context.Background(),
		manager,
	)
	if err != nil {
		t.Fatalf("beginTransaction returned error: %v", err)
	}

	if gotUOW != uow {
		t.Fatal("expected returned UnitOfWork to be manager UnitOfWork")
	}
	if gotRepos != uow.repos {
		t.Fatal("expected returned repositories to be transaction-bound")
	}
	if gotCtx == nil {
		t.Fatal("expected transaction context")
	}
	if !owns {
		t.Fatal("expected outer caller to own new transaction")
	}
}

func TestBeginTransactionReusesExistingScope(t *testing.T) {
	uow := &transactionScopeFakeUOW{}
	manager := &transactionScopeFakeUOWManager{uow: uow}

	ctx := context.Background()
	_, _, txCtx, owns, err := beginTransaction(ctx, manager)
	if err != nil {
		t.Fatalf("initial beginTransaction returned error: %v", err)
	}
	if !owns {
		t.Fatal("expected initial transaction to be owned by caller")
	}

	manager.beginCount = 0

	gotUOW, gotRepos, gotNestedCtx, nestedOwns, err := beginTransaction(
		txCtx,
		manager,
	)
	if err != nil {
		t.Fatalf("nested beginTransaction returned error: %v", err)
	}

	if gotUOW != uow {
		t.Fatal("expected nested call to reuse UnitOfWork")
	}
	if gotRepos != uow.repos {
		t.Fatal("expected nested call to reuse repositories")
	}
	if gotNestedCtx != txCtx {
		t.Fatal("expected nested call to reuse context")
	}
	if nestedOwns {
		t.Fatal("expected nested caller not to own transaction")
	}
	if manager.beginCount != 0 {
		t.Fatalf("expected no nested Begin call, got %d", manager.beginCount)
	}
}

func TestWithinTransactionRejectsInvalidArguments(t *testing.T) {
	manager := &transactionScopeFakeUOWManager{
		uow: &transactionScopeFakeUOW{},
	}

	tests := []struct {
		name string
		ctx  context.Context
		mgr  repository.UnitOfWorkManager
		fn   func(context.Context, repository.RepositorySet) error
	}{
		{
			name: "nil context",
			ctx:  nil,
			mgr:  manager,
			fn:   func(context.Context, repository.RepositorySet) error { return nil },
		},
		{
			name: "nil manager",
			ctx:  context.Background(),
			mgr:  nil,
			fn:   func(context.Context, repository.RepositorySet) error { return nil },
		},
		{
			name: "nil function",
			ctx:  context.Background(),
			mgr:  manager,
			fn:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := WithinTransaction(tt.ctx, tt.mgr, tt.fn); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestBeginTransactionPropagatesBeginError(t *testing.T) {
	wantErr := errors.New("begin failed")
	manager := &transactionScopeFakeUOWManager{beginErr: wantErr}

	_, _, _, _, err := beginTransaction(
		context.Background(),
		manager,
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected begin error, got %v", err)
	}
}

func TestBeginTransactionRejectsNilUnitOfWork(t *testing.T) {
	manager := &transactionScopeFakeUOWManager{}

	_, _, _, _, err := beginTransaction(
		context.Background(),
		manager,
	)
	if err == nil {
		t.Fatal("expected nil UnitOfWork error")
	}
}
