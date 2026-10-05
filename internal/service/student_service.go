package service

import (
	"context"
	"fmt"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

var _ StudentService = (*studentService)(nil)

func (s *studentService) Create(
	ctx context.Context,
	input CreateStudentInput,
) (CreateStudentOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleAdmin); err != nil {
		return CreateStudentOutput{}, err
	}

	student, err := domain.NewStudent(
		input.Name,
		input.UserID,
		input.NIS,
		input.NISN,
	)
	if err != nil {
		return CreateStudentOutput{}, err
	}

	uow, err := s.deps.UOW.Begin(ctx)
	if err != nil {
		return CreateStudentOutput{}, fmt.Errorf(
			"begin create student transaction: %w",
			err,
		)
	}

	committed := false
	defer func() {
		if !committed {
			_ = uow.Rollback()
		}
	}()

	if err := uow.Repositories().Students.Create(ctx, student); err != nil {
		return CreateStudentOutput{}, err
	}

	if err := uow.Commit(); err != nil {
		return CreateStudentOutput{}, fmt.Errorf(
			"commit create student transaction: %w",
			err,
		)
	}

	committed = true

	return CreateStudentOutput{
		Student: &student,
	}, nil
}

func (s *studentService) Get(
	ctx context.Context,
	input GetStudentInput,
) (GetStudentOutput, error) {
	if err := input.Actor.Validate(); err != nil {
		return GetStudentOutput{}, err
	}

	if input.StudentID == uuid.Nil {
		return GetStudentOutput{}, ErrScopeViolation
	}

	student, err := s.deps.Repositories.Students.GetByID(
		ctx,
		input.StudentID,
	)
	if err != nil {
		return GetStudentOutput{}, err
	}

	switch input.Actor.Role {
	case domain.RoleAdmin:
		return GetStudentOutput{
			Student: &student,
		}, nil

	case domain.RoleSiswa:
		if student.UserID == nil {
			return GetStudentOutput{}, ErrScopeViolation
		}

		if err := RequireSelf(input.Actor, *student.UserID); err != nil {
			return GetStudentOutput{}, err
		}

		return GetStudentOutput{
			Student: &student,
		}, nil

	case domain.RoleWaliKelas:
		if input.AcademicYearID == uuid.Nil {
			return GetStudentOutput{}, ErrScopeViolation
		}

		if err := RequireStudentScope(
			ctx,
			s.deps.Repositories,
			input.Actor,
			student.ID,
			input.AcademicYearID,
		); err != nil {
			return GetStudentOutput{}, err
		}

		return GetStudentOutput{
			Student: &student,
		}, nil

	default:
		return GetStudentOutput{}, ErrForbidden
	}
}

func (s *studentService) GetByNIS(
	ctx context.Context,
	input GetStudentByNISInput,
) (GetStudentByNISOutput, error) {
	if err := input.Actor.Validate(); err != nil {
		return GetStudentByNISOutput{}, err
	}

	if input.NIS == "" {
		return GetStudentByNISOutput{}, domain.ErrInvalidValue
	}

	student, err := s.deps.Repositories.Students.GetByNIS(
		ctx,
		input.NIS,
	)
	if err != nil {
		return GetStudentByNISOutput{}, err
	}

	switch input.Actor.Role {
	case domain.RoleAdmin:
		return GetStudentByNISOutput{
			Student: &student,
		}, nil

	case domain.RoleSiswa:
		if student.UserID == nil {
			return GetStudentByNISOutput{}, ErrScopeViolation
		}

		if err := RequireSelf(input.Actor, *student.UserID); err != nil {
			return GetStudentByNISOutput{}, err
		}

		return GetStudentByNISOutput{
			Student: &student,
		}, nil

	case domain.RoleWaliKelas:
		if input.AcademicYearID == uuid.Nil {
			return GetStudentByNISOutput{}, ErrScopeViolation
		}

		if err := RequireStudentScope(
			ctx,
			s.deps.Repositories,
			input.Actor,
			student.ID,
			input.AcademicYearID,
		); err != nil {
			return GetStudentByNISOutput{}, err
		}

		return GetStudentByNISOutput{
			Student: &student,
		}, nil

	default:
		return GetStudentByNISOutput{}, ErrForbidden
	}
}

func (s *studentService) GetByNISN(
	ctx context.Context,
	input GetStudentByNISNInput,
) (GetStudentByNISNOutput, error) {
	if err := input.Actor.Validate(); err != nil {
		return GetStudentByNISNOutput{}, err
	}

	if input.NISN == "" {
		return GetStudentByNISNOutput{}, domain.ErrInvalidValue
	}

	student, err := s.deps.Repositories.Students.GetByNISN(
		ctx,
		input.NISN,
	)
	if err != nil {
		return GetStudentByNISNOutput{}, err
	}

	switch input.Actor.Role {
	case domain.RoleAdmin:
		return GetStudentByNISNOutput{
			Student: &student,
		}, nil

	case domain.RoleSiswa:
		if student.UserID == nil {
			return GetStudentByNISNOutput{}, ErrScopeViolation
		}

		if err := RequireSelf(input.Actor, *student.UserID); err != nil {
			return GetStudentByNISNOutput{}, err
		}

		return GetStudentByNISNOutput{
			Student: &student,
		}, nil

	case domain.RoleWaliKelas:
		if input.AcademicYearID == uuid.Nil {
			return GetStudentByNISNOutput{}, ErrScopeViolation
		}

		if err := RequireStudentScope(
			ctx,
			s.deps.Repositories,
			input.Actor,
			student.ID,
			input.AcademicYearID,
		); err != nil {
			return GetStudentByNISNOutput{}, err
		}

		return GetStudentByNISNOutput{
			Student: &student,
		}, nil

	default:
		return GetStudentByNISNOutput{}, ErrForbidden
	}
}

func (s *studentService) GetByUserID(
	ctx context.Context,
	input GetStudentByUserIDInput,
) (GetStudentByUserIDOutput, error) {
	if err := input.Actor.Validate(); err != nil {
		return GetStudentByUserIDOutput{}, err
	}

	if input.UserID == uuid.Nil {
		return GetStudentByUserIDOutput{}, ErrScopeViolation
	}

	student, err := s.deps.Repositories.Students.GetByUserID(
		ctx,
		input.UserID,
	)
	if err != nil {
		return GetStudentByUserIDOutput{}, err
	}

	switch input.Actor.Role {
	case domain.RoleAdmin:
		return GetStudentByUserIDOutput{
			Student: &student,
		}, nil

	case domain.RoleSiswa:
		if err := RequireSelf(input.Actor, input.UserID); err != nil {
			return GetStudentByUserIDOutput{}, err
		}

		return GetStudentByUserIDOutput{
			Student: &student,
		}, nil

	case domain.RoleWaliKelas:
		if input.AcademicYearID == uuid.Nil {
			return GetStudentByUserIDOutput{}, ErrScopeViolation
		}

		if err := RequireStudentScope(
			ctx,
			s.deps.Repositories,
			input.Actor,
			student.ID,
			input.AcademicYearID,
		); err != nil {
			return GetStudentByUserIDOutput{}, err
		}

		return GetStudentByUserIDOutput{
			Student: &student,
		}, nil

	default:
		return GetStudentByUserIDOutput{}, ErrForbidden
	}
}

func (s *studentService) List(
	ctx context.Context,
	input ListStudentsInput,
) (ListStudentsOutput, error) {
	if err := input.Actor.Validate(); err != nil {
		return ListStudentsOutput{}, err
	}

	if input.AcademicYearID == uuid.Nil {
		return ListStudentsOutput{}, ErrScopeViolation
	}

	if input.ClassID == uuid.Nil {
		return ListStudentsOutput{}, ErrScopeViolation
	}

	options := repository.ListOptions{
		Limit:  input.Options.Limit,
		Offset: input.Options.Offset,
	}

	switch input.Actor.Role {
	case domain.RoleAdmin:
		students, err := s.deps.Repositories.Students.
			ListByClassAndAcademicYear(
				ctx,
				input.ClassID,
				input.AcademicYearID,
				options,
			)
		if err != nil {
			return ListStudentsOutput{}, err
		}

		return ListStudentsOutput{
			Students: studentPointers(students),
		}, nil

	case domain.RoleWaliKelas:
		if err := RequireClassScope(
			ctx,
			s.deps.Repositories,
			input.Actor,
			input.ClassID,
			input.AcademicYearID,
		); err != nil {
			return ListStudentsOutput{}, err
		}

		students, err := s.deps.Repositories.Students.
			ListByClassAndAcademicYear(
				ctx,
				input.ClassID,
				input.AcademicYearID,
				options,
			)
		if err != nil {
			return ListStudentsOutput{}, err
		}

		return ListStudentsOutput{
			Students: studentPointers(students),
		}, nil

	case domain.RoleSiswa:
		return ListStudentsOutput{}, ErrForbidden

	default:
		return ListStudentsOutput{}, ErrForbidden
	}
}

func (s *studentService) Update(
	ctx context.Context,
	input UpdateStudentInput,
) (UpdateStudentOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleAdmin); err != nil {
		return UpdateStudentOutput{}, err
	}

	if input.StudentID == uuid.Nil {
		return UpdateStudentOutput{}, ErrScopeViolation
	}

	uow, err := s.deps.UOW.Begin(ctx)
	if err != nil {
		return UpdateStudentOutput{}, fmt.Errorf(
			"begin update student transaction: %w",
			err,
		)
	}

	committed := false
	defer func() {
		if !committed {
			_ = uow.Rollback()
		}
	}()

	repos := uow.Repositories()

	existing, err := repos.Students.GetByID(
		ctx,
		input.StudentID,
	)
	if err != nil {
		return UpdateStudentOutput{}, err
	}

	student := existing
	student.UserID = input.UserID
	student.NIS = input.NIS
	student.NISN = input.NISN
	student.Name = input.Name
	student.UpdatedAt = time.Now()

	if err := validateStudentUpdate(student); err != nil {
		return UpdateStudentOutput{}, err
	}

	if err := repos.Students.Update(ctx, student); err != nil {
		return UpdateStudentOutput{}, err
	}

	if err := uow.Commit(); err != nil {
		return UpdateStudentOutput{}, fmt.Errorf(
			"commit update student transaction: %w",
			err,
		)
	}

	committed = true

	return UpdateStudentOutput{
		Student: &student,
	}, nil
}

func (s *studentService) Leave(
	ctx context.Context,
	input LeaveStudentInput,
) (LeaveStudentOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleAdmin); err != nil {
		return LeaveStudentOutput{}, err
	}

	if input.StudentID == uuid.Nil {
		return LeaveStudentOutput{}, ErrScopeViolation
	}

	uow, err := s.deps.UOW.Begin(ctx)
	if err != nil {
		return LeaveStudentOutput{}, fmt.Errorf(
			"begin leave student transaction: %w",
			err,
		)
	}

	committed := false
	defer func() {
		if !committed {
			_ = uow.Rollback()
		}
	}()

	repos := uow.Repositories()

	student, err := repos.Students.GetByID(
		ctx,
		input.StudentID,
	)
	if err != nil {
		return LeaveStudentOutput{}, err
	}

	if err := student.Leave(); err != nil {
		return LeaveStudentOutput{}, err
	}

	if err := repos.Students.Update(ctx, student); err != nil {
		return LeaveStudentOutput{}, err
	}

	if err := uow.Commit(); err != nil {
		return LeaveStudentOutput{}, fmt.Errorf(
			"commit leave student transaction: %w",
			err,
		)
	}

	committed = true

	return LeaveStudentOutput{
		Student: &student,
	}, nil
}

func validateStudentUpdate(student domain.Student) error {
	if student.ID == uuid.Nil {
		return ErrScopeViolation
	}

	if student.NIS == nil {
		return domain.ErrInvalidValue
	}

	if student.NISN == nil {
		return domain.ErrInvalidValue
	}

	if student.Name == "" {
		return domain.ErrEmptyName
	}

	if !student.Status.IsValid() {
		return domain.ErrInvalidValue
	}

	return nil
}

func studentPointers(students []domain.Student) []*domain.Student {
	result := make([]*domain.Student, 0, len(students))

	for i := range students {
		student := students[i]
		result = append(result, &student)
	}

	return result
}
