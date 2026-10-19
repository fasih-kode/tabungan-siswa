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

func (s *userService) ListByRole(
	ctx context.Context,
	input ListUsersByRoleInput,
) (ListUsersByRoleOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleAdmin); err != nil {
		return ListUsersByRoleOutput{}, err
	}
	if !input.Role.IsValid() {
		return ListUsersByRoleOutput{}, domain.ErrInvalidValue
	}
	if input.Options.Limit < 1 || input.Options.Limit > 100 || input.Options.Offset < 0 {
		return ListUsersByRoleOutput{}, domain.ErrInvalidValue
	}

	users, err := s.deps.Repositories.Users.ListByRole(
		ctx,
		input.Role,
		repository.ListOptions{
			Limit:  input.Options.Limit,
			Offset: input.Options.Offset,
		},
	)
	if err != nil {
		return ListUsersByRoleOutput{}, err
	}

	result := make([]*domain.User, 0, len(users))
	for i := range users {
		user := users[i]
		// Password hash is never needed by callers listing users.
		user.PasswordHash = ""
		result = append(result, &user)
	}
	total := 0
	if input.Options.IncludeTotal {
		if s.deps.Repositories.ListTotals == nil {
			return ListUsersByRoleOutput{}, ErrInvalidDependency
		}
		total, err = s.deps.Repositories.ListTotals.Users(ctx, input.Role)
		if err != nil {
			return ListUsersByRoleOutput{}, err
		}
	}
	return ListUsersByRoleOutput{Users: result, Total: total}, nil
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
	user.PasswordChangeRequired = input.PasswordChangeRequired

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

func (s *userService) CreateStudentAccount(
	ctx context.Context,
	input CreateStudentAccountInput,
) (CreateStudentAccountOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleAdmin); err != nil {
		return CreateStudentAccountOutput{}, err
	}
	if input.StudentID == uuid.Nil {
		return CreateStudentAccountOutput{}, ErrScopeViolation
	}

	uow, err := s.deps.UOW.Begin(ctx)
	if err != nil {
		return CreateStudentAccountOutput{}, fmt.Errorf("begin create student account: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = uow.Rollback()
		}
	}()

	repos := uow.Repositories()
	student, err := repos.Students.GetByIDForUpdate(ctx, input.StudentID)
	if err != nil {
		return CreateStudentAccountOutput{}, err
	}
	if !student.IsActive() {
		return CreateStudentAccountOutput{}, domain.ErrInvalidTransition
	}
	if student.UserID != nil {
		return CreateStudentAccountOutput{}, repository.ErrConflict
	}
	user, err := domain.NewUser(input.Username, input.PasswordHash, domain.RoleSiswa)
	if err != nil {
		return CreateStudentAccountOutput{}, err
	}
	exists, err := repos.Users.ExistsByUsername(ctx, user.Username)
	if err != nil {
		return CreateStudentAccountOutput{}, err
	}
	if exists {
		return CreateStudentAccountOutput{}, repository.ErrConflict
	}
	if err := repos.Users.Create(ctx, user); err != nil {
		return CreateStudentAccountOutput{}, err
	}
	student.UserID = &user.ID
	student.UpdatedAt = time.Now()
	if err := repos.Students.Update(ctx, student); err != nil {
		return CreateStudentAccountOutput{}, err
	}
	if err := uow.Commit(); err != nil {
		return CreateStudentAccountOutput{}, fmt.Errorf("commit create student account: %w", err)
	}
	committed = true
	return CreateStudentAccountOutput{User: &user, Student: &student}, nil
}

func (s *userService) Update(
	ctx context.Context,
	input UpdateUserInput,
) (UpdateUserOutput, error) {
	if err := input.Actor.Validate(); err != nil {
		return UpdateUserOutput{}, err
	}
	isAdmin := input.Actor.Role == domain.RoleAdmin
	selfChange := input.Actor.UserID == input.UserID && input.CurrentPassword != ""
	if !isAdmin && !selfChange {
		return UpdateUserOutput{}, ErrForbidden
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
	if selfChange {
		if input.Username != existing.Username || input.Role != existing.Role || s.deps.PasswordHasher == nil {
			return UpdateUserOutput{}, ErrForbidden
		}
		if err := s.deps.PasswordHasher.Compare(existing.PasswordHash, input.CurrentPassword); err != nil {
			return UpdateUserOutput{}, ErrInvalidCredentials
		}
	} else if !isAdmin {
		return UpdateUserOutput{}, ErrForbidden
	}

	user := existing
	user.Username = input.Username
	user.PasswordHash = input.PasswordHash
	user.Role = input.Role
	if selfChange {
		user.PasswordChangeRequired = false
	} else if input.PasswordChangeRequired {
		user.PasswordChangeRequired = true
	}
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
