package service

import (
	"context"
	"errors"
	"testing"

	"github.com/fasih/tabungan-siswa/internal/repository"
)

type fakeUnitOfWorkManager struct{}

func (fakeUnitOfWorkManager) Begin(context.Context) (repository.UnitOfWork, error) {
	return nil, nil
}

func TestDependencies_ValidateAcceptsValidDependencies(t *testing.T) {
	deps := Dependencies{
		UOW: fakeUnitOfWorkManager{},
	}

	if err := deps.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestDependencies_ValidateRejectsNilUnitOfWorkManager(t *testing.T) {
	deps := Dependencies{}

	err := deps.Validate()
	if !errors.Is(err, ErrInvalidDependency) {
		t.Fatalf("Validate() error = %v, want ErrInvalidDependency", err)
	}
}

func TestDependencies_ValidateDoesNotRequireEveryRepository(t *testing.T) {
	deps := Dependencies{
		Repositories: repository.RepositorySet{},
		UOW:          fakeUnitOfWorkManager{},
	}

	if err := deps.Validate(); err != nil {
		t.Fatalf(
			"Validate() error = %v, want nil; Dependencies must not require every repository",
			err,
		)
	}
}
