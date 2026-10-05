package service_test

import (
	"context"
	"testing"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

func TestRequireClassScopeRejectsInvalidActor(t *testing.T) {
	err := service.RequireClassScope(
		context.Background(),
		repository.RepositorySet{},
		service.Actor{},
		uuid.New(),
		uuid.New(),
	)

	if err != service.ErrInvalidActor {
		t.Fatalf("RequireClassScope() error = %v, want ErrInvalidActor", err)
	}
}

func TestRequireClassScopeRejectsNilIDs(t *testing.T) {
	actor := service.Actor{
		UserID: uuid.New(),
		Role:   domain.RoleAdmin,
	}

	tests := []struct {
		name           string
		classID        uuid.UUID
		academicYearID uuid.UUID
	}{
		{
			name:           "nil class",
			classID:        uuid.Nil,
			academicYearID: uuid.New(),
		},
		{
			name:           "nil academic year",
			classID:        uuid.New(),
			academicYearID: uuid.Nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.RequireClassScope(
				context.Background(),
				repository.RepositorySet{},
				actor,
				tt.classID,
				tt.academicYearID,
			)

			if err != service.ErrScopeViolation {
				t.Fatalf(
					"RequireClassScope() error = %v, want ErrScopeViolation",
					err,
				)
			}
		})
	}
}

func TestRequireClassScopeRejectsStudentRole(t *testing.T) {
	actor := service.Actor{
		UserID: uuid.New(),
		Role:   domain.RoleSiswa,
	}

	err := service.RequireClassScope(
		context.Background(),
		repository.RepositorySet{},
		actor,
		uuid.New(),
		uuid.New(),
	)

	if err != service.ErrForbidden {
		t.Fatalf("RequireClassScope() error = %v, want ErrForbidden", err)
	}
}

func TestRequireStudentScopeRejectsInvalidActor(t *testing.T) {
	err := service.RequireStudentScope(
		context.Background(),
		repository.RepositorySet{},
		service.Actor{},
		uuid.New(),
		uuid.New(),
	)

	if err != service.ErrInvalidActor {
		t.Fatalf(
			"RequireStudentScope() error = %v, want ErrInvalidActor",
			err,
		)
	}
}

func TestRequireStudentScopeRejectsNilIDs(t *testing.T) {
	actor := service.Actor{
		UserID: uuid.New(),
		Role:   domain.RoleAdmin,
	}

	tests := []struct {
		name           string
		studentID      uuid.UUID
		academicYearID uuid.UUID
	}{
		{
			name:           "nil student",
			studentID:      uuid.Nil,
			academicYearID: uuid.New(),
		},
		{
			name:           "nil academic year",
			studentID:      uuid.New(),
			academicYearID: uuid.Nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.RequireStudentScope(
				context.Background(),
				repository.RepositorySet{},
				actor,
				tt.studentID,
				tt.academicYearID,
			)

			if err != service.ErrScopeViolation {
				t.Fatalf(
					"RequireStudentScope() error = %v, want ErrScopeViolation",
					err,
				)
			}
		})
	}
}

func TestRequireStudentScopeRejectsStudentRole(t *testing.T) {
	actor := service.Actor{
		UserID: uuid.New(),
		Role:   domain.RoleSiswa,
	}

	err := service.RequireStudentScope(
		context.Background(),
		repository.RepositorySet{},
		actor,
		uuid.New(),
		uuid.New(),
	)

	if err != service.ErrForbidden {
		t.Fatalf(
			"RequireStudentScope() error = %v, want ErrForbidden",
			err,
		)
	}
}
