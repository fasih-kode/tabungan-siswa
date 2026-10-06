package service_test

import (
	"context"
	"testing"

	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/repository/postgres"
	"github.com/fasih/tabungan-siswa/internal/service"
)

type serviceFactoryFakeUOWManager struct{}

func (serviceFactoryFakeUOWManager) Begin(
	context.Context,
) (repository.UnitOfWork, error) {
	return nil, nil
}

type serviceFactoryFakePasswordHasher struct{}

func (serviceFactoryFakePasswordHasher) Hash(string) (string, error) {
	return "", nil
}

func (serviceFactoryFakePasswordHasher) Compare(
	string,
	string,
) error {
	return nil
}

func TestNewServiceSetBuildsAllServices(t *testing.T) {
	deps := service.Dependencies{
		Repositories:   postgres.NewRepositorySet(nil),
		UOW:            serviceFactoryFakeUOWManager{},
		PasswordHasher: serviceFactoryFakePasswordHasher{},
	}

	services, err := service.NewServiceSet(deps)
	if err != nil {
		t.Fatalf("NewServiceSet() error = %v", err)
	}

	if services.Authentication == nil {
		t.Error("ServiceSet.Authentication is nil")
	}
	if services.User == nil {
		t.Error("ServiceSet.User is nil")
	}
	if services.AcademicYear == nil {
		t.Error("ServiceSet.AcademicYear is nil")
	}
	if services.Class == nil {
		t.Error("ServiceSet.Class is nil")
	}
	if services.Student == nil {
		t.Error("ServiceSet.Student is nil")
	}
	if services.ClassAssignment == nil {
		t.Error("ServiceSet.ClassAssignment is nil")
	}
	if services.ClassTransfer == nil {
		t.Error("ServiceSet.ClassTransfer is nil")
	}
	if services.SavingsAccount == nil {
		t.Error("ServiceSet.SavingsAccount is nil")
	}
	if services.Transaction == nil {
		t.Error("ServiceSet.Transaction is nil")
	}
	if services.Settlement == nil {
		t.Error("ServiceSet.Settlement is nil")
	}
	if services.Audit == nil {
		t.Error("ServiceSet.Audit is nil")
	}
}

func TestNewServiceSetRejectsInvalidDependencies(t *testing.T) {
	_, err := service.NewServiceSet(service.Dependencies{})

	if err == nil {
		t.Fatal("NewServiceSet() error = nil, want error")
	}
}
