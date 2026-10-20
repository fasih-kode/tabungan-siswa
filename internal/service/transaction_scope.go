package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

type transactionScopeContextKey struct{}

type transactionScope struct {
	uow   repository.UnitOfWork
	repos repository.RepositorySet
}

type approvedCorrectionContextKey struct{}

type approvedCorrectionScope struct {
	RequestID uuid.UUID
	AccountID uuid.UUID
}

// withApprovedCorrectionScope is reserved for the FASE 23 settlement service.
// Its caller must first validate an approved, active reopen request and the
// responsible Wali Kelas. HTTP handlers cannot construct this scope.
func withApprovedCorrectionScope(ctx context.Context, requestID, accountID uuid.UUID) context.Context {
	return context.WithValue(ctx, approvedCorrectionContextKey{}, approvedCorrectionScope{RequestID: requestID, AccountID: accountID})
}

func approvedCorrectionScopeFromContext(ctx context.Context) (approvedCorrectionScope, bool) {
	scope, ok := ctx.Value(approvedCorrectionContextKey{}).(approvedCorrectionScope)
	return scope, ok && scope.RequestID != uuid.Nil && scope.AccountID != uuid.Nil
}

// WithinTransaction executes fn inside one UnitOfWork.
//
// If ctx already belongs to an active transaction scope, the existing
// UnitOfWork is reused. The nested caller never commits or rolls back the
// transaction; the outermost caller owns that boundary.
//
// If ctx does not contain an active transaction scope, this function starts
// one, commits after fn succeeds, and rolls back when fn returns an error.
func WithinTransaction(
	ctx context.Context,
	manager repository.UnitOfWorkManager,
	fn func(context.Context, repository.RepositorySet) error,
) error {
	if ctx == nil {
		return errors.New("transaction context is nil")
	}
	if manager == nil {
		return errors.New("unit of work manager is nil")
	}
	if fn == nil {
		return errors.New("transaction function is nil")
	}

	if scope, ok := transactionScopeFromContext(ctx); ok {
		return fn(ctx, scope.repos)
	}

	uow, err := manager.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	if uow == nil {
		return errors.New("unit of work is nil")
	}

	scope := transactionScope{
		uow:   uow,
		repos: uow.Repositories(),
	}
	txCtx := context.WithValue(ctx, transactionScopeContextKey{}, scope)

	if err := fn(txCtx, scope.repos); err != nil {
		_ = uow.Rollback()
		return err
	}

	if err := uow.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// beginTransaction starts a transaction when the caller is the outermost
// transaction boundary. When a transaction already exists in ctx, it returns
// that same transaction without creating a nested UnitOfWork.
//
// Service mutation methods use this helper so they can participate in a
// cross-service transaction without changing their public contracts.
func beginTransaction(
	ctx context.Context,
	manager repository.UnitOfWorkManager,
) (
	repository.UnitOfWork,
	repository.RepositorySet,
	context.Context,
	bool,
	error,
) {
	if ctx == nil {
		return nil, repository.RepositorySet{}, nil, false, errors.New(
			"transaction context is nil",
		)
	}
	if manager == nil {
		return nil, repository.RepositorySet{}, nil, false, errors.New(
			"unit of work manager is nil",
		)
	}

	if scope, ok := transactionScopeFromContext(ctx); ok {
		return scope.uow, scope.repos, ctx, false, nil
	}

	uow, err := manager.Begin(ctx)
	if err != nil {
		return nil, repository.RepositorySet{}, nil, false, fmt.Errorf(
			"begin transaction: %w",
			err,
		)
	}
	if uow == nil {
		return nil, repository.RepositorySet{}, nil, false, errors.New(
			"unit of work is nil",
		)
	}

	scope := transactionScope{
		uow:   uow,
		repos: uow.Repositories(),
	}
	txCtx := context.WithValue(ctx, transactionScopeContextKey{}, scope)

	return uow, scope.repos, txCtx, true, nil
}

func transactionScopeFromContext(
	ctx context.Context,
) (transactionScope, bool) {
	scope, ok := ctx.Value(transactionScopeContextKey{}).(transactionScope)
	return scope, ok
}
