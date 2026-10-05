package service

import (
	"context"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

// RequireClassScope memastikan actor memiliki akses terhadap class
// pada academic year tertentu.
//
// ADMIN selalu memiliki akses.
// WALI_KELAS hanya memiliki akses terhadap class yang ditugaskan kepadanya.
// SISWA tidak memiliki class scope administratif.
func RequireClassScope(
	ctx context.Context,
	repos repository.RepositorySet,
	actor Actor,
	classID uuid.UUID,
	academicYearID uuid.UUID,
) error {
	if err := actor.Validate(); err != nil {
		return err
	}

	if classID == uuid.Nil || academicYearID == uuid.Nil {
		return ErrScopeViolation
	}

	if actor.Role == domain.RoleAdmin {
		return nil
	}

	if actor.Role != domain.RoleWaliKelas {
		return ErrForbidden
	}

	assignments, err := repos.TeacherClassAssignments.ListByClassAndAcademicYear(
		ctx,
		classID,
		academicYearID,
		repository.ListOptions{
			Limit:  100,
			Offset: 0,
		},
	)
	if err != nil {
		return err
	}

	for _, assignment := range assignments {
		if assignment.UserID == actor.UserID {
			return nil
		}
	}

	return ErrScopeViolation
}

// RequireStudentScope memastikan actor memiliki akses terhadap student
// pada academic year tertentu.
//
// ADMIN selalu memiliki akses.
// WALI_KELAS hanya memiliki akses apabila student berada pada class
// yang menjadi scope actor pada academic year tersebut.
// SISWA tidak menggunakan scope WALI_KELAS.
func RequireStudentScope(
	ctx context.Context,
	repos repository.RepositorySet,
	actor Actor,
	studentID uuid.UUID,
	academicYearID uuid.UUID,
) error {
	if err := actor.Validate(); err != nil {
		return err
	}

	if studentID == uuid.Nil || academicYearID == uuid.Nil {
		return ErrScopeViolation
	}

	if actor.Role == domain.RoleAdmin {
		return nil
	}

	if actor.Role != domain.RoleWaliKelas {
		return ErrForbidden
	}

	histories, err := repos.StudentClassHistories.ListByStudentAndAcademicYear(
		ctx,
		studentID,
		academicYearID,
		repository.ListOptions{
			Limit:  100,
			Offset: 0,
		},
	)
	if err != nil {
		return err
	}

	for _, history := range histories {
		if err := RequireClassScope(
			ctx,
			repos,
			actor,
			history.ClassID,
			academicYearID,
		); err == nil {
			return nil
		}
	}

	return ErrScopeViolation
}
