package service

import (
	"context"
	"errors"
	"testing"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

type foundationFakeUOWManager struct{}

func (foundationFakeUOWManager) Begin(context.Context) (repository.UnitOfWork, error) {
	return nil, nil
}

func TestFoundationActorValidation(t *testing.T) {
	tests := []struct {
		name  string
		actor Actor
	}{
		{
			name: "admin",
			actor: Actor{
				UserID: uuid.New(),
				Role:   domain.RoleAdmin,
			},
		},
		{
			name: "wali kelas",
			actor: Actor{
				UserID: uuid.New(),
				Role:   domain.RoleWaliKelas,
			},
		},
		{
			name: "siswa",
			actor: Actor{
				UserID: uuid.New(),
				Role:   domain.RoleSiswa,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.actor.Validate(); err != nil {
				t.Fatalf("Validate() error = %v, want nil", err)
			}
		})
	}
}

func TestFoundationActorRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name  string
		actor Actor
	}{
		{
			name: "nil user id",
			actor: Actor{
				UserID: uuid.Nil,
				Role:   domain.RoleAdmin,
			},
		},
		{
			name: "invalid role",
			actor: Actor{
				UserID: uuid.New(),
				Role:   domain.UserRole("INVALID"),
			},
		},
		{
			name:  "zero actor",
			actor: Actor{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.actor.Validate()

			if !errors.Is(err, ErrInvalidActor) {
				t.Fatalf(
					"Validate() error = %v, want ErrInvalidActor",
					err,
				)
			}
		})
	}
}

func TestFoundationDependenciesValidation(t *testing.T) {
	valid := Dependencies{
		UOW: foundationFakeUOWManager{},
	}

	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}

	invalid := Dependencies{}

	if !errors.Is(invalid.Validate(), ErrInvalidDependency) {
		t.Fatalf("Validate() error = %v, want ErrInvalidDependency", invalid.Validate())
	}
}

func TestFoundationConstructorsRejectInvalidDependencies(t *testing.T) {
	deps := Dependencies{}

	tests := []struct {
		name string
		fn   func(Dependencies) error
	}{
		{
			name: "user",
			fn: func(d Dependencies) error {
				_, err := NewUserService(d)
				return err
			},
		},
		{
			name: "academic year",
			fn: func(d Dependencies) error {
				_, err := NewAcademicYearService(d)
				return err
			},
		},
		{
			name: "class",
			fn: func(d Dependencies) error {
				_, err := NewClassService(d)
				return err
			},
		},
		{
			name: "student",
			fn: func(d Dependencies) error {
				_, err := NewStudentService(d)
				return err
			},
		},
		{
			name: "class assignment",
			fn: func(d Dependencies) error {
				_, err := NewClassAssignmentService(d)
				return err
			},
		},
		{
			name: "class transfer",
			fn: func(d Dependencies) error {
				_, err := NewClassTransferService(d)
				return err
			},
		},
		{
			name: "savings account",
			fn: func(d Dependencies) error {
				_, err := NewSavingsAccountService(d)
				return err
			},
		},
		{
			name: "transaction",
			fn: func(d Dependencies) error {
				_, err := NewTransactionService(d)
				return err
			},
		},
		{
			name: "settlement",
			fn: func(d Dependencies) error {
				_, err := NewSettlementService(d)
				return err
			},
		},
		{
			name: "audit",
			fn: func(d Dependencies) error {
				_, err := NewAuditService(d)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fn(deps); !errors.Is(err, ErrInvalidDependency) {
				t.Fatalf(
					"constructor error = %v, want ErrInvalidDependency",
					err,
				)
			}
		})
	}
}
