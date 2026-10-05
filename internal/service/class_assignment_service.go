package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

var _ ClassAssignmentService = (*classAssignmentService)(nil)

func (s *classAssignmentService) Assign(
	ctx context.Context,
	input AssignTeacherInput,
) (AssignTeacherOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleAdmin); err != nil {
		return AssignTeacherOutput{}, err
	}

	if input.UserID == uuid.Nil ||
		input.ClassID == uuid.Nil ||
		input.AcademicYearID == uuid.Nil {
		return AssignTeacherOutput{}, domain.ErrInvalidID
	}

	user, err := s.deps.Repositories.Users.GetByID(ctx, input.UserID)
	if err != nil {
		return AssignTeacherOutput{}, err
	}

	if user.Role != domain.RoleWaliKelas {
		return AssignTeacherOutput{}, ErrInvalidDependency
	}

	_, err = s.deps.Repositories.Classes.GetByID(
		ctx,
		input.ClassID,
	)
	if err != nil {
		return AssignTeacherOutput{}, err
	}

	academicYear, err := s.deps.Repositories.AcademicYears.GetByID(
		ctx,
		input.AcademicYearID,
	)
	if err != nil {
		return AssignTeacherOutput{}, err
	}

	period, err := domain.NewDateRange(
		academicYear.StartDate,
		&academicYear.EndDate,
	)
	if err != nil {
		return AssignTeacherOutput{}, err
	}

	assignment, err := domain.NewTeacherClassAssignment(
		input.UserID,
		input.AcademicYearID,
		input.ClassID,
		period,
	)
	if err != nil {
		return AssignTeacherOutput{}, err
	}

	uow, err := s.deps.UOW.Begin(ctx)
	if err != nil {
		return AssignTeacherOutput{}, fmt.Errorf(
			"begin assign teacher transaction: %w",
			err,
		)
	}

	committed := false
	defer func() {
		if !committed {
			_ = uow.Rollback()
		}
	}()

	if err := uow.Repositories().TeacherClassAssignments.Create(
		ctx,
		assignment,
	); err != nil {
		return AssignTeacherOutput{}, err
	}

	if err := uow.Commit(); err != nil {
		return AssignTeacherOutput{}, fmt.Errorf(
			"commit assign teacher transaction: %w",
			err,
		)
	}

	committed = true

	return AssignTeacherOutput{
		Assignment: &assignment,
	}, nil
}

func (s *classAssignmentService) Get(
	ctx context.Context,
	input GetTeacherAssignmentInput,
) (GetTeacherAssignmentOutput, error) {
	if err := input.Actor.Validate(); err != nil {
		return GetTeacherAssignmentOutput{}, err
	}

	if input.AssignmentID == uuid.Nil {
		return GetTeacherAssignmentOutput{}, domain.ErrInvalidID
	}

	assignment, err := s.deps.Repositories.TeacherClassAssignments.GetByID(
		ctx,
		input.AssignmentID,
	)
	if err != nil {
		return GetTeacherAssignmentOutput{}, err
	}

	if err := s.requireAssignmentScope(
		ctx,
		input.Actor,
		assignment,
	); err != nil {
		return GetTeacherAssignmentOutput{}, err
	}

	return GetTeacherAssignmentOutput{
		Assignment: &assignment,
	}, nil
}

func (s *classAssignmentService) ListByTeacher(
	ctx context.Context,
	input ListTeacherAssignmentsInput,
) (ListTeacherAssignmentsOutput, error) {
	if err := input.Actor.Validate(); err != nil {
		return ListTeacherAssignmentsOutput{}, err
	}

	if input.UserID == uuid.Nil ||
		input.AcademicYearID == uuid.Nil {
		return ListTeacherAssignmentsOutput{}, domain.ErrInvalidID
	}

	assignments, err := s.deps.Repositories.TeacherClassAssignments.
		ListByUserAndAcademicYear(
			ctx,
			input.UserID,
			input.AcademicYearID,
			repository.ListOptions{
				Limit:  input.Options.Limit,
				Offset: input.Options.Offset,
			},
		)
	if err != nil {
		return ListTeacherAssignmentsOutput{}, err
	}

	if input.Actor.Role == domain.RoleAdmin {
		return teacherAssignmentOutput(assignments), nil
	}

	if input.Actor.Role == domain.RoleWaliKelas {
		return s.filterAssignmentsByClassScope(
			ctx,
			input.Actor,
			assignments,
		)
	}

	if input.Actor.Role == domain.RoleSiswa {
		student, err := s.deps.Repositories.Students.GetByUserID(
			ctx,
			input.Actor.UserID,
		)
		if err != nil {
			return ListTeacherAssignmentsOutput{}, err
		}

		return s.filterAssignmentsByStudentScope(
			ctx,
			input.Actor,
			student.ID,
			assignments,
		)
	}

	return ListTeacherAssignmentsOutput{}, ErrForbidden
}

func (s *classAssignmentService) ListByClass(
	ctx context.Context,
	input ListClassAssignmentsInput,
) (ListClassAssignmentsOutput, error) {
	if err := input.Actor.Validate(); err != nil {
		return ListClassAssignmentsOutput{}, err
	}

	if input.ClassID == uuid.Nil ||
		input.AcademicYearID == uuid.Nil {
		return ListClassAssignmentsOutput{}, domain.ErrInvalidID
	}

	if input.Actor.Role == domain.RoleAdmin {
		return s.listByClass(
			ctx,
			input.ClassID,
			input.AcademicYearID,
			input.Options,
		)
	}

	if input.Actor.Role == domain.RoleWaliKelas {
		if err := RequireClassScope(
			ctx,
			s.deps.Repositories,
			input.Actor,
			input.ClassID,
			input.AcademicYearID,
		); err != nil {
			return ListClassAssignmentsOutput{}, err
		}

		return s.listByClass(
			ctx,
			input.ClassID,
			input.AcademicYearID,
			input.Options,
		)
	}

	if input.Actor.Role == domain.RoleSiswa {
		student, err := s.deps.Repositories.Students.GetByUserID(
			ctx,
			input.Actor.UserID,
		)
		if err != nil {
			return ListClassAssignmentsOutput{}, err
		}

		if err := s.requireStudentClassScope(
			ctx,
			input.Actor,
			student.ID,
			input.ClassID,
			input.AcademicYearID,
		); err != nil {
			return ListClassAssignmentsOutput{}, err
		}

		return s.listByClass(
			ctx,
			input.ClassID,
			input.AcademicYearID,
			input.Options,
		)
	}

	return ListClassAssignmentsOutput{}, ErrForbidden
}

func (s *classAssignmentService) GetCurrentByClass(
	ctx context.Context,
	input GetCurrentClassAssignmentInput,
) (GetCurrentClassAssignmentOutput, error) {
	if err := input.Actor.Validate(); err != nil {
		return GetCurrentClassAssignmentOutput{}, err
	}

	if input.ClassID == uuid.Nil ||
		input.AcademicYearID == uuid.Nil {
		return GetCurrentClassAssignmentOutput{}, domain.ErrInvalidID
	}

	if input.Date.IsZero() {
		return GetCurrentClassAssignmentOutput{}, domain.ErrInvalidValue
	}

	if input.Actor.Role == domain.RoleAdmin {
		return s.getCurrentByClass(
			ctx,
			input.ClassID,
			input.AcademicYearID,
			input.Date,
		)
	}

	if input.Actor.Role == domain.RoleWaliKelas {
		if err := RequireClassScope(
			ctx,
			s.deps.Repositories,
			input.Actor,
			input.ClassID,
			input.AcademicYearID,
		); err != nil {
			return GetCurrentClassAssignmentOutput{}, err
		}

		return s.getCurrentByClass(
			ctx,
			input.ClassID,
			input.AcademicYearID,
			input.Date,
		)
	}

	if input.Actor.Role == domain.RoleSiswa {
		student, err := s.deps.Repositories.Students.GetByUserID(
			ctx,
			input.Actor.UserID,
		)
		if err != nil {
			return GetCurrentClassAssignmentOutput{}, err
		}

		if err := s.requireStudentClassScope(
			ctx,
			input.Actor,
			student.ID,
			input.ClassID,
			input.AcademicYearID,
		); err != nil {
			return GetCurrentClassAssignmentOutput{}, err
		}

		return s.getCurrentByClass(
			ctx,
			input.ClassID,
			input.AcademicYearID,
			input.Date,
		)
	}

	return GetCurrentClassAssignmentOutput{}, ErrForbidden
}

func (s *classAssignmentService) listByClass(
	ctx context.Context,
	classID uuid.UUID,
	academicYearID uuid.UUID,
	options ListOptions,
) (ListClassAssignmentsOutput, error) {
	assignments, err := s.deps.Repositories.TeacherClassAssignments.
		ListByClassAndAcademicYear(
			ctx,
			classID,
			academicYearID,
			repository.ListOptions{
				Limit:  options.Limit,
				Offset: options.Offset,
			},
		)
	if err != nil {
		return ListClassAssignmentsOutput{}, err
	}

	return ListClassAssignmentsOutput{
		Assignments: teacherAssignmentPointers(assignments),
	}, nil
}

func (s *classAssignmentService) getCurrentByClass(
	ctx context.Context,
	classID uuid.UUID,
	academicYearID uuid.UUID,
	dateValue time.Time,
) (GetCurrentClassAssignmentOutput, error) {
	assignment, err := s.deps.Repositories.TeacherClassAssignments.
		GetCurrentByClassAndAcademicYear(
			ctx,
			classID,
			academicYearID,
			dateValue,
		)
	if err != nil {
		return GetCurrentClassAssignmentOutput{}, err
	}

	return GetCurrentClassAssignmentOutput{
		Assignment: &assignment,
	}, nil
}

func (s *classAssignmentService) requireAssignmentScope(
	ctx context.Context,
	actor Actor,
	assignment domain.TeacherClassAssignment,
) error {
	switch actor.Role {
	case domain.RoleAdmin:
		return nil

	case domain.RoleWaliKelas:
		return RequireClassScope(
			ctx,
			s.deps.Repositories,
			actor,
			assignment.ClassID,
			assignment.AcademicYearID,
		)

	case domain.RoleSiswa:
		student, err := s.deps.Repositories.Students.GetByUserID(
			ctx,
			actor.UserID,
		)
		if err != nil {
			return err
		}

		return s.requireStudentClassScope(
			ctx,
			actor,
			student.ID,
			assignment.ClassID,
			assignment.AcademicYearID,
		)

	default:
		return ErrForbidden
	}
}

func (s *classAssignmentService) filterAssignmentsByClassScope(
	ctx context.Context,
	actor Actor,
	assignments []domain.TeacherClassAssignment,
) (ListTeacherAssignmentsOutput, error) {
	result := make([]*domain.TeacherClassAssignment, 0, len(assignments))

	for _, assignment := range assignments {
		err := RequireClassScope(
			ctx,
			s.deps.Repositories,
			actor,
			assignment.ClassID,
			assignment.AcademicYearID,
		)

		if err == nil {
			assignmentCopy := assignment
			result = append(result, &assignmentCopy)
			continue
		}

		if errors.Is(err, ErrScopeViolation) {
			continue
		}

		return ListTeacherAssignmentsOutput{}, err
	}

	return ListTeacherAssignmentsOutput{
		Assignments: result,
	}, nil
}

func (s *classAssignmentService) filterAssignmentsByStudentScope(
	ctx context.Context,
	actor Actor,
	studentID uuid.UUID,
	assignments []domain.TeacherClassAssignment,
) (ListTeacherAssignmentsOutput, error) {
	result := make([]*domain.TeacherClassAssignment, 0, len(assignments))

	for _, assignment := range assignments {
		err := s.requireStudentClassScope(
			ctx,
			actor,
			studentID,
			assignment.ClassID,
			assignment.AcademicYearID,
		)

		if err == nil {
			assignmentCopy := assignment
			result = append(result, &assignmentCopy)
			continue
		}

		if errors.Is(err, ErrScopeViolation) {
			continue
		}

		return ListTeacherAssignmentsOutput{}, err
	}

	return ListTeacherAssignmentsOutput{
		Assignments: result,
	}, nil
}

func (s *classAssignmentService) requireStudentClassScope(
	ctx context.Context,
	actor Actor,
	studentID uuid.UUID,
	classID uuid.UUID,
	academicYearID uuid.UUID,
) error {
	histories, err := s.deps.Repositories.StudentClassHistories.
		ListByStudentAndAcademicYear(
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
		if history.ClassID == classID {
			return nil
		}
	}

	return ErrScopeViolation
}

func teacherAssignmentOutput(
	assignments []domain.TeacherClassAssignment,
) ListTeacherAssignmentsOutput {
	return ListTeacherAssignmentsOutput{
		Assignments: teacherAssignmentPointers(assignments),
	}
}

func teacherAssignmentPointers(
	assignments []domain.TeacherClassAssignment,
) []*domain.TeacherClassAssignment {
	result := make([]*domain.TeacherClassAssignment, 0, len(assignments))

	for _, assignment := range assignments {
		assignmentCopy := assignment
		result = append(result, &assignmentCopy)
	}

	return result
}
