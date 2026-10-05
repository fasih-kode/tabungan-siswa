package service

import (
	"context"
	"fmt"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

type userService struct {
	deps Dependencies
}

var _ UserService = (*userService)(nil)

func NewUserService(deps Dependencies) (*userService, error) {
	if err := deps.Validate(); err != nil {
		return nil, err
	}

	return &userService{
		deps: deps,
	}, nil
}

func (s *userService) Get(
	ctx context.Context,
	input GetUserInput,
) (GetUserOutput, error) {
	if err := input.Actor.Validate(); err != nil {
		return GetUserOutput{}, err
	}

	if input.UserID == uuid.Nil {
		return GetUserOutput{}, ErrScopeViolation
	}

	user, err := s.deps.Repositories.Users.GetByID(ctx, input.UserID)
	if err != nil {
		return GetUserOutput{}, err
	}

	switch input.Actor.Role {
	case domain.RoleAdmin:
		return GetUserOutput{
			User: &user,
		}, nil

	case domain.RoleSiswa:
		if err := RequireSelf(input.Actor, user.ID); err != nil {
			return GetUserOutput{}, err
		}

		return GetUserOutput{
			User: &user,
		}, nil

	case domain.RoleWaliKelas:
		if input.AcademicYearID == uuid.Nil {
			return GetUserOutput{}, ErrScopeViolation
		}

		student, err := s.deps.Repositories.Students.GetByUserID(
			ctx,
			user.ID,
		)
		if err != nil {
			return GetUserOutput{}, err
		}

		if err := RequireStudentScope(
			ctx,
			s.deps.Repositories,
			input.Actor,
			student.ID,
			input.AcademicYearID,
		); err != nil {
			return GetUserOutput{}, err
		}

		return GetUserOutput{
			User: &user,
		}, nil

	default:
		return GetUserOutput{}, ErrForbidden
	}
}

func (s *userService) GetByUsername(
	ctx context.Context,
	input GetUserByUsernameInput,
) (GetUserByUsernameOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleAdmin); err != nil {
		return GetUserByUsernameOutput{}, err
	}

	user, err := s.deps.Repositories.Users.GetByUsername(
		ctx,
		input.Username,
	)
	if err != nil {
		return GetUserByUsernameOutput{}, err
	}

	return GetUserByUsernameOutput{
		User: &user,
	}, nil
}

func (s *userService) Create(
	ctx context.Context,
	input CreateUserInput,
) (CreateUserOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleAdmin); err != nil {
		return CreateUserOutput{}, err
	}

	user, err := domain.NewUser(
		input.Username,
		input.PasswordHash,
		input.Role,
	)
	if err != nil {
		return CreateUserOutput{}, err
	}

	exists, err := s.deps.Repositories.Users.ExistsByUsername(
		ctx,
		user.Username,
	)
	if err != nil {
		return CreateUserOutput{}, err
	}

	if exists {
		return CreateUserOutput{}, repository.ErrConflict
	}

	uow, err := s.deps.UOW.Begin(ctx)
	if err != nil {
		return CreateUserOutput{}, fmt.Errorf("begin create user transaction: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_ = uow.Rollback()
		}
	}()

	if err := uow.Repositories().Users.Create(ctx, user); err != nil {
		return CreateUserOutput{}, err
	}

	if err := uow.Commit(); err != nil {
		return CreateUserOutput{}, fmt.Errorf("commit create user transaction: %w", err)
	}

	committed = true

	return CreateUserOutput{
		User: &user,
	}, nil
}

func (s *userService) Update(
	ctx context.Context,
	input UpdateUserInput,
) (UpdateUserOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleAdmin); err != nil {
		return UpdateUserOutput{}, err
	}

	if input.UserID == uuid.Nil {
		return UpdateUserOutput{}, ErrScopeViolation
	}

	uow, err := s.deps.UOW.Begin(ctx)
	if err != nil {
		return UpdateUserOutput{}, fmt.Errorf("begin update user transaction: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_ = uow.Rollback()
		}
	}()

	repos := uow.Repositories()

	existing, err := repos.Users.GetByID(ctx, input.UserID)
	if err != nil {
		return UpdateUserOutput{}, err
	}

	user := existing
	user.Username = input.Username
	user.PasswordHash = input.PasswordHash
	user.Role = input.Role
	user.UpdatedAt = time.Now()

	if err := validateUserUpdate(user); err != nil {
		return UpdateUserOutput{}, err
	}

	if user.Username != existing.Username {
		exists, err := repos.Users.ExistsByUsername(
			ctx,
			user.Username,
		)
		if err != nil {
			return UpdateUserOutput{}, err
		}

		if exists {
			return UpdateUserOutput{}, repository.ErrConflict
		}
	}

	if err := repos.Users.Update(ctx, user); err != nil {
		return UpdateUserOutput{}, err
	}

	if err := uow.Commit(); err != nil {
		return UpdateUserOutput{}, fmt.Errorf("commit update user transaction: %w", err)
	}

	committed = true

	return UpdateUserOutput{
		User: &user,
	}, nil
}

func validateUserUpdate(user domain.User) error {
	if user.ID == uuid.Nil {
		return ErrScopeViolation
	}

	if user.Username == "" {
		return domain.ErrEmptyName
	}

	if user.PasswordHash == "" {
		return domain.ErrInvalidPasswordHash
	}

	if !user.Role.IsValid() {
		return domain.ErrInvalidValue
	}

	return nil
}
