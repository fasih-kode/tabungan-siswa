package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

type userServiceFakeUserRepository struct {
	repository.UserRepository

	getByIDFn          func(context.Context, uuid.UUID) (domain.User, error)
	getByUsernameFn    func(context.Context, string) (domain.User, error)
	createFn           func(context.Context, domain.User) error
	updateFn           func(context.Context, domain.User) error
	existsByUsernameFn func(context.Context, string) (bool, error)
}

func (f *userServiceFakeUserRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (domain.User, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, id)
	}

	return domain.User{}, repository.ErrNotFound
}

func (f *userServiceFakeUserRepository) GetByUsername(
	ctx context.Context,
	username string,
) (domain.User, error) {
	if f.getByUsernameFn != nil {
		return f.getByUsernameFn(ctx, username)
	}

	return domain.User{}, repository.ErrNotFound
}

func (f *userServiceFakeUserRepository) Create(
	ctx context.Context,
	user domain.User,
) error {
	if f.createFn != nil {
		return f.createFn(ctx, user)
	}

	return nil
}

func (f *userServiceFakeUserRepository) Update(
	ctx context.Context,
	user domain.User,
) error {
	if f.updateFn != nil {
		return f.updateFn(ctx, user)
	}

	return nil
}

func (f *userServiceFakeUserRepository) ExistsByUsername(
	ctx context.Context,
	username string,
) (bool, error) {
	if f.existsByUsernameFn != nil {
		return f.existsByUsernameFn(ctx, username)
	}

	return false, nil
}

type userServiceFakeStudentRepository struct {
	repository.StudentRepository

	getByUserIDFn func(context.Context, uuid.UUID) (domain.Student, error)
}

func (f *userServiceFakeStudentRepository) GetByUserID(
	ctx context.Context,
	userID uuid.UUID,
) (domain.Student, error) {
	if f.getByUserIDFn != nil {
		return f.getByUserIDFn(ctx, userID)
	}

	return domain.Student{}, repository.ErrNotFound
}

type userServiceFakeStudentClassHistoryRepository struct {
	repository.StudentClassHistoryRepository

	listByStudentAndAcademicYearFn func(
		context.Context,
		uuid.UUID,
		uuid.UUID,
		repository.ListOptions,
	) ([]domain.StudentClassHistory, error)
}

func (f *userServiceFakeStudentClassHistoryRepository) ListByStudentAndAcademicYear(
	ctx context.Context,
	studentID uuid.UUID,
	academicYearID uuid.UUID,
	options repository.ListOptions,
) ([]domain.StudentClassHistory, error) {
	if f.listByStudentAndAcademicYearFn != nil {
		return f.listByStudentAndAcademicYearFn(
			ctx,
			studentID,
			academicYearID,
			options,
		)
	}

	return nil, nil
}

type userServiceFakeTeacherClassAssignmentRepository struct {
	repository.TeacherClassAssignmentRepository

	listByClassAndAcademicYearFn func(
		context.Context,
		uuid.UUID,
		uuid.UUID,
		repository.ListOptions,
	) ([]domain.TeacherClassAssignment, error)
}

func (f *userServiceFakeTeacherClassAssignmentRepository) ListByClassAndAcademicYear(
	ctx context.Context,
	classID uuid.UUID,
	academicYearID uuid.UUID,
	options repository.ListOptions,
) ([]domain.TeacherClassAssignment, error) {
	if f.listByClassAndAcademicYearFn != nil {
		return f.listByClassAndAcademicYearFn(
			ctx,
			classID,
			academicYearID,
			options,
		)
	}

	return nil, nil
}

type userServiceFakeUnitOfWork struct {
	repos         repository.RepositorySet
	commitErr     error
	rollbackErr   error
	commitCount   int
	rollbackCount int
}

func (f *userServiceFakeUnitOfWork) Repositories() repository.RepositorySet {
	return f.repos
}

func (f *userServiceFakeUnitOfWork) Commit() error {
	f.commitCount++
	return f.commitErr
}

func (f *userServiceFakeUnitOfWork) Rollback() error {
	f.rollbackCount++
	return f.rollbackErr
}

type userServiceFakeUnitOfWorkManager struct {
	uow        repository.UnitOfWork
	beginErr   error
	beginCount int
}

func (f *userServiceFakeUnitOfWorkManager) Begin(context.Context) (repository.UnitOfWork, error) {
	f.beginCount++

	if f.beginErr != nil {
		return nil, f.beginErr
	}

	return f.uow, nil
}

func newUserServiceForTest(
	t *testing.T,
	repos repository.RepositorySet,
	uowManager repository.UnitOfWorkManager,
) *userService {
	t.Helper()

	svc, err := NewUserService(Dependencies{
		Repositories: repos,
		UOW:          uowManager,
	})
	if err != nil {
		t.Fatalf("NewUserService() error = %v", err)
	}

	return svc
}

func newTestDomainUser(
	t *testing.T,
	id uuid.UUID,
	username string,
	role domain.UserRole,
) domain.User {
	t.Helper()

	now := time.Now().Add(-time.Hour)

	return domain.User{
		ID:           id,
		Username:     username,
		PasswordHash: "password-hash",
		Role:         role,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func TestNewUserService(t *testing.T) {
	uowManager := &userServiceFakeUnitOfWorkManager{}

	t.Run("valid dependencies", func(t *testing.T) {
		svc, err := NewUserService(Dependencies{
			UOW: uowManager,
		})
		if err != nil {
			t.Fatalf("NewUserService() error = %v, want nil", err)
		}

		if svc == nil {
			t.Fatal("NewUserService() service = nil, want non-nil")
		}
	})

	t.Run("invalid dependencies", func(t *testing.T) {
		svc, err := NewUserService(Dependencies{})

		if !errors.Is(err, ErrInvalidDependency) {
			t.Fatalf(
				"NewUserService() error = %v, want ErrInvalidDependency",
				err,
			)
		}

		if svc != nil {
			t.Fatal("NewUserService() service != nil on invalid dependencies")
		}
	})
}

func TestUserService_Get(t *testing.T) {
	ctx := context.Background()

	academicYearID := uuid.New()
	targetUserID := uuid.New()
	actorUserID := uuid.New()
	studentID := uuid.New()
	classID := uuid.New()

	targetUser := newTestDomainUser(
		t,
		targetUserID,
		"target-user",
		domain.RoleSiswa,
	)

	tests := []struct {
		name       string
		actor      Actor
		academicID uuid.UUID
		repos      repository.RepositorySet
		wantUser   bool
		wantErr    error
	}{
		{
			name: "admin allowed",
			actor: Actor{
				UserID: actorUserID,
				Role:   domain.RoleAdmin,
			},
			repos: repository.RepositorySet{
				Users: &userServiceFakeUserRepository{
					getByIDFn: func(context.Context, uuid.UUID) (domain.User, error) {
						return targetUser, nil
					},
				},
			},
			wantUser: true,
		},
		{
			name: "siswa self allowed",
			actor: Actor{
				UserID: targetUserID,
				Role:   domain.RoleSiswa,
			},
			repos: repository.RepositorySet{
				Users: &userServiceFakeUserRepository{
					getByIDFn: func(context.Context, uuid.UUID) (domain.User, error) {
						return targetUser, nil
					},
				},
			},
			wantUser: true,
		},
		{
			name: "siswa other user rejected",
			actor: Actor{
				UserID: actorUserID,
				Role:   domain.RoleSiswa,
			},
			repos: repository.RepositorySet{
				Users: &userServiceFakeUserRepository{
					getByIDFn: func(context.Context, uuid.UUID) (domain.User, error) {
						return targetUser, nil
					},
				},
			},
			wantErr: ErrScopeViolation,
		},
		{
			name:       "wali kelas in scope allowed",
			academicID: academicYearID,
			actor: Actor{
				UserID: actorUserID,
				Role:   domain.RoleWaliKelas,
			},
			repos: repository.RepositorySet{
				Users: &userServiceFakeUserRepository{
					getByIDFn: func(context.Context, uuid.UUID) (domain.User, error) {
						return targetUser, nil
					},
				},
				Students: &userServiceFakeStudentRepository{
					getByUserIDFn: func(context.Context, uuid.UUID) (domain.Student, error) {
						return domain.Student{
							ID:     studentID,
							UserID: &targetUserID,
						}, nil
					},
				},
				StudentClassHistories: &userServiceFakeStudentClassHistoryRepository{
					listByStudentAndAcademicYearFn: func(
						context.Context,
						uuid.UUID,
						uuid.UUID,
						repository.ListOptions,
					) ([]domain.StudentClassHistory, error) {
						return []domain.StudentClassHistory{
							{
								StudentID: studentID,
								ClassID:   classID,
							},
						}, nil
					},
				},
				TeacherClassAssignments: &userServiceFakeTeacherClassAssignmentRepository{
					listByClassAndAcademicYearFn: func(
						context.Context,
						uuid.UUID,
						uuid.UUID,
						repository.ListOptions,
					) ([]domain.TeacherClassAssignment, error) {
						return []domain.TeacherClassAssignment{
							{
								UserID:  actorUserID,
								ClassID: classID,
							},
						}, nil
					},
				},
			},
			wantUser: true,
		},
		{
			name:       "wali kelas outside scope rejected",
			academicID: academicYearID,
			actor: Actor{
				UserID: actorUserID,
				Role:   domain.RoleWaliKelas,
			},
			repos: repository.RepositorySet{
				Users: &userServiceFakeUserRepository{
					getByIDFn: func(context.Context, uuid.UUID) (domain.User, error) {
						return targetUser, nil
					},
				},
				Students: &userServiceFakeStudentRepository{
					getByUserIDFn: func(context.Context, uuid.UUID) (domain.Student, error) {
						return domain.Student{
							ID:     studentID,
							UserID: &targetUserID,
						}, nil
					},
				},
				StudentClassHistories: &userServiceFakeStudentClassHistoryRepository{
					listByStudentAndAcademicYearFn: func(
						context.Context,
						uuid.UUID,
						uuid.UUID,
						repository.ListOptions,
					) ([]domain.StudentClassHistory, error) {
						return []domain.StudentClassHistory{
							{
								StudentID: studentID,
								ClassID:   classID,
							},
						}, nil
					},
				},
				TeacherClassAssignments: &userServiceFakeTeacherClassAssignmentRepository{
					listByClassAndAcademicYearFn: func(
						context.Context,
						uuid.UUID,
						uuid.UUID,
						repository.ListOptions,
					) ([]domain.TeacherClassAssignment, error) {
						return []domain.TeacherClassAssignment{
							{
								UserID:  uuid.New(),
								ClassID: classID,
							},
						}, nil
					},
				},
			},
			wantErr: ErrScopeViolation,
		},
		{
			name: "wali kelas requires academic year",
			actor: Actor{
				UserID: actorUserID,
				Role:   domain.RoleWaliKelas,
			},
			repos: repository.RepositorySet{
				Users: &userServiceFakeUserRepository{
					getByIDFn: func(context.Context, uuid.UUID) (domain.User, error) {
						return targetUser, nil
					},
				},
			},
			wantErr: ErrScopeViolation,
		},
		{
			name: "repository error propagated",
			actor: Actor{
				UserID: actorUserID,
				Role:   domain.RoleAdmin,
			},
			repos: repository.RepositorySet{
				Users: &userServiceFakeUserRepository{
					getByIDFn: func(context.Context, uuid.UUID) (domain.User, error) {
						return domain.User{}, repository.ErrNotFound
					},
				},
			},
			wantErr: repository.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newUserServiceForTest(
				t,
				tt.repos,
				&userServiceFakeUnitOfWorkManager{},
			)

			got, err := svc.Get(ctx, GetUserInput{
				Actor:          tt.actor,
				UserID:         targetUserID,
				AcademicYearID: tt.academicID,
			})

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"Get() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}

			if tt.wantUser {
				if got.User == nil {
					t.Fatal("Get() User = nil, want user")
				}

				if got.User.ID != targetUser.ID {
					t.Fatalf(
						"Get() User.ID = %v, want %v",
						got.User.ID,
						targetUser.ID,
					)
				}
			} else if got.User != nil {
				t.Fatal("Get() User != nil on error")
			}
		})
	}
}

func TestUserService_GetRejectsInvalidActor(t *testing.T) {
	svc := newUserServiceForTest(
		t,
		repository.RepositorySet{
			Users: &userServiceFakeUserRepository{},
		},
		&userServiceFakeUnitOfWorkManager{},
	)

	_, err := svc.Get(context.Background(), GetUserInput{
		Actor:  Actor{},
		UserID: uuid.New(),
	})

	if !errors.Is(err, ErrInvalidActor) {
		t.Fatalf(
			"Get() error = %v, want ErrInvalidActor",
			err,
		)
	}
}

func TestUserService_GetByUsername(t *testing.T) {
	ctx := context.Background()

	user := newTestDomainUser(
		t,
		uuid.New(),
		"admin-user",
		domain.RoleAdmin,
	)

	t.Run("admin allowed", func(t *testing.T) {
		repos := repository.RepositorySet{
			Users: &userServiceFakeUserRepository{
				getByUsernameFn: func(
					context.Context,
					string,
				) (domain.User, error) {
					return user, nil
				},
			},
		}

		svc := newUserServiceForTest(
			t,
			repos,
			&userServiceFakeUnitOfWorkManager{},
		)

		got, err := svc.GetByUsername(
			ctx,
			GetUserByUsernameInput{
				Actor: Actor{
					UserID: uuid.New(),
					Role:   domain.RoleAdmin,
				},
				Username: user.Username,
			},
		)
		if err != nil {
			t.Fatalf(
				"GetByUsername() error = %v, want nil",
				err,
			)
		}

		if got.User == nil || got.User.ID != user.ID {
			t.Fatalf(
				"GetByUsername() User = %#v, want user %v",
				got.User,
				user.ID,
			)
		}
	})

	for _, role := range []domain.UserRole{
		domain.RoleWaliKelas,
		domain.RoleSiswa,
	} {
		t.Run(string(role)+" forbidden", func(t *testing.T) {
			svc := newUserServiceForTest(
				t,
				repository.RepositorySet{
					Users: &userServiceFakeUserRepository{},
				},
				&userServiceFakeUnitOfWorkManager{},
			)

			_, err := svc.GetByUsername(
				ctx,
				GetUserByUsernameInput{
					Actor: Actor{
						UserID: uuid.New(),
						Role:   role,
					},
					Username: user.Username,
				},
			)

			if !errors.Is(err, ErrForbidden) {
				t.Fatalf(
					"GetByUsername() error = %v, want ErrForbidden",
					err,
				)
			}
		})
	}

	t.Run("repository error propagated", func(t *testing.T) {
		svc := newUserServiceForTest(
			t,
			repository.RepositorySet{
				Users: &userServiceFakeUserRepository{
					getByUsernameFn: func(
						context.Context,
						string,
					) (domain.User, error) {
						return domain.User{}, repository.ErrNotFound
					},
				},
			},
			&userServiceFakeUnitOfWorkManager{},
		)

		_, err := svc.GetByUsername(
			ctx,
			GetUserByUsernameInput{
				Actor: Actor{
					UserID: uuid.New(),
					Role:   domain.RoleAdmin,
				},
				Username: "missing",
			},
		)

		if !errors.Is(err, repository.ErrNotFound) {
			t.Fatalf(
				"GetByUsername() error = %v, want ErrNotFound",
				err,
			)
		}
	})
}

func TestUserService_Create(t *testing.T) {
	ctx := context.Background()
	admin := Actor{
		UserID: uuid.New(),
		Role:   domain.RoleAdmin,
	}

	t.Run("admin creates user and commits", func(t *testing.T) {
		var created domain.User

		userRepo := &userServiceFakeUserRepository{
			createFn: func(
				_ context.Context,
				user domain.User,
			) error {
				created = user
				return nil
			},
		}

		uow := &userServiceFakeUnitOfWork{
			repos: repository.RepositorySet{
				Users: userRepo,
			},
		}

		uowManager := &userServiceFakeUnitOfWorkManager{
			uow: uow,
		}

		svc := newUserServiceForTest(
			t,
			repository.RepositorySet{
				Users: userRepo,
			},
			uowManager,
		)

		got, err := svc.Create(ctx, CreateUserInput{
			Actor:        admin,
			Username:     "new-user",
			PasswordHash: "password-hash",
			Role:         domain.RoleSiswa,
		})
		if err != nil {
			t.Fatalf("Create() error = %v, want nil", err)
		}

		if got.User == nil {
			t.Fatal("Create() User = nil, want user")
		}

		if created.ID != got.User.ID {
			t.Fatalf(
				"created ID = %v, want %v",
				created.ID,
				got.User.ID,
			)
		}

		if uow.commitCount != 1 {
			t.Fatalf(
				"Commit() count = %d, want 1",
				uow.commitCount,
			)
		}

		if uow.rollbackCount != 0 {
			t.Fatalf(
				"Rollback() count = %d, want 0",
				uow.rollbackCount,
			)
		}
	})

	t.Run("non admin forbidden", func(t *testing.T) {
		uowManager := &userServiceFakeUnitOfWorkManager{}

		svc := newUserServiceForTest(
			t,
			repository.RepositorySet{
				Users: &userServiceFakeUserRepository{},
			},
			uowManager,
		)

		_, err := svc.Create(ctx, CreateUserInput{
			Actor: Actor{
				UserID: uuid.New(),
				Role:   domain.RoleWaliKelas,
			},
			Username:     "new-user",
			PasswordHash: "password-hash",
			Role:         domain.RoleSiswa,
		})

		if !errors.Is(err, ErrForbidden) {
			t.Fatalf(
				"Create() error = %v, want ErrForbidden",
				err,
			)
		}

		if uowManager.beginCount != 0 {
			t.Fatalf(
				"Begin() count = %d, want 0",
				uowManager.beginCount,
			)
		}
	})

	t.Run("domain validation propagated", func(t *testing.T) {
		uowManager := &userServiceFakeUnitOfWorkManager{}

		svc := newUserServiceForTest(
			t,
			repository.RepositorySet{
				Users: &userServiceFakeUserRepository{},
			},
			uowManager,
		)

		_, err := svc.Create(ctx, CreateUserInput{
			Actor:        admin,
			Username:     "",
			PasswordHash: "password-hash",
			Role:         domain.RoleSiswa,
		})

		if !errors.Is(err, domain.ErrEmptyName) {
			t.Fatalf(
				"Create() error = %v, want ErrEmptyName",
				err,
			)
		}

		if uowManager.beginCount != 0 {
			t.Fatalf(
				"Begin() count = %d, want 0",
				uowManager.beginCount,
			)
		}
	})

	t.Run("username conflict", func(t *testing.T) {
		userRepo := &userServiceFakeUserRepository{
			existsByUsernameFn: func(
				context.Context,
				string,
			) (bool, error) {
				return true, nil
			},
		}

		uowManager := &userServiceFakeUnitOfWorkManager{}

		svc := newUserServiceForTest(
			t,
			repository.RepositorySet{
				Users: userRepo,
			},
			uowManager,
		)

		_, err := svc.Create(ctx, CreateUserInput{
			Actor:        admin,
			Username:     "existing-user",
			PasswordHash: "password-hash",
			Role:         domain.RoleSiswa,
		})

		if !errors.Is(err, repository.ErrConflict) {
			t.Fatalf(
				"Create() error = %v, want ErrConflict",
				err,
			)
		}

		if uowManager.beginCount != 0 {
			t.Fatalf(
				"Begin() count = %d, want 0",
				uowManager.beginCount,
			)
		}
	})

	t.Run("exists repository error propagated", func(t *testing.T) {
		repoErr := errors.New("exists failed")

		userRepo := &userServiceFakeUserRepository{
			existsByUsernameFn: func(
				context.Context,
				string,
			) (bool, error) {
				return false, repoErr
			},
		}

		svc := newUserServiceForTest(
			t,
			repository.RepositorySet{
				Users: userRepo,
			},
			&userServiceFakeUnitOfWorkManager{},
		)

		_, err := svc.Create(ctx, CreateUserInput{
			Actor:        admin,
			Username:     "new-user",
			PasswordHash: "password-hash",
			Role:         domain.RoleSiswa,
		})

		if !errors.Is(err, repoErr) {
			t.Fatalf(
				"Create() error = %v, want repository error",
				err,
			)
		}
	})

	t.Run("begin error wrapped", func(t *testing.T) {
		beginErr := errors.New("begin failed")

		uowManager := &userServiceFakeUnitOfWorkManager{
			beginErr: beginErr,
		}

		svc := newUserServiceForTest(
			t,
			repository.RepositorySet{
				Users: &userServiceFakeUserRepository{},
			},
			uowManager,
		)

		_, err := svc.Create(ctx, CreateUserInput{
			Actor:        admin,
			Username:     "new-user",
			PasswordHash: "password-hash",
			Role:         domain.RoleSiswa,
		})

		if !errors.Is(err, beginErr) {
			t.Fatalf(
				"Create() error = %v, want begin error",
				err,
			)
		}
	})

	t.Run("create repository error rolls back", func(t *testing.T) {
		createErr := errors.New("create failed")

		userRepo := &userServiceFakeUserRepository{
			createFn: func(
				context.Context,
				domain.User,
			) error {
				return createErr
			},
		}

		uow := &userServiceFakeUnitOfWork{
			repos: repository.RepositorySet{
				Users: userRepo,
			},
		}

		svc := newUserServiceForTest(
			t,
			repository.RepositorySet{
				Users: userRepo,
			},
			&userServiceFakeUnitOfWorkManager{
				uow: uow,
			},
		)

		_, err := svc.Create(ctx, CreateUserInput{
			Actor:        admin,
			Username:     "new-user",
			PasswordHash: "password-hash",
			Role:         domain.RoleSiswa,
		})

		if !errors.Is(err, createErr) {
			t.Fatalf(
				"Create() error = %v, want create error",
				err,
			)
		}

		if uow.rollbackCount != 1 {
			t.Fatalf(
				"Rollback() count = %d, want 1",
				uow.rollbackCount,
			)
		}

		if uow.commitCount != 0 {
			t.Fatalf(
				"Commit() count = %d, want 0",
				uow.commitCount,
			)
		}
	})

	t.Run("commit error rolls back", func(t *testing.T) {
		commitErr := errors.New("commit failed")

		userRepo := &userServiceFakeUserRepository{}

		uow := &userServiceFakeUnitOfWork{
			repos: repository.RepositorySet{
				Users: userRepo,
			},
			commitErr: commitErr,
		}

		svc := newUserServiceForTest(
			t,
			repository.RepositorySet{
				Users: userRepo,
			},
			&userServiceFakeUnitOfWorkManager{
				uow: uow,
			},
		)

		_, err := svc.Create(ctx, CreateUserInput{
			Actor:        admin,
			Username:     "new-user",
			PasswordHash: "password-hash",
			Role:         domain.RoleSiswa,
		})

		if !errors.Is(err, commitErr) {
			t.Fatalf(
				"Create() error = %v, want commit error",
				err,
			)
		}

		if uow.rollbackCount != 1 {
			t.Fatalf(
				"Rollback() count = %d, want 1",
				uow.rollbackCount,
			)
		}
	})
}

func TestUserService_Update(t *testing.T) {
	ctx := context.Background()

	admin := Actor{
		UserID: uuid.New(),
		Role:   domain.RoleAdmin,
	}

	userID := uuid.New()

	existing := newTestDomainUser(
		t,
		userID,
		"old-user",
		domain.RoleSiswa,
	)

	t.Run("admin updates user and changes updated at", func(t *testing.T) {
		var updated domain.User

		userRepo := &userServiceFakeUserRepository{
			getByIDFn: func(
				context.Context,
				uuid.UUID,
			) (domain.User, error) {
				return existing, nil
			},
			updateFn: func(
				_ context.Context,
				user domain.User,
			) error {
				updated = user
				return nil
			},
		}

		uow := &userServiceFakeUnitOfWork{
			repos: repository.RepositorySet{
				Users: userRepo,
			},
		}

		svc := newUserServiceForTest(
			t,
			repository.RepositorySet{
				Users: userRepo,
			},
			&userServiceFakeUnitOfWorkManager{
				uow: uow,
			},
		)

		before := time.Now()

		got, err := svc.Update(ctx, UpdateUserInput{
			Actor:        admin,
			UserID:       userID,
			Username:     "updated-user",
			PasswordHash: "updated-password",
			Role:         domain.RoleAdmin,
		})

		after := time.Now()

		if err != nil {
			t.Fatalf("Update() error = %v, want nil", err)
		}

		if got.User == nil {
			t.Fatal("Update() User = nil, want user")
		}

		if updated.Username != "updated-user" {
			t.Fatalf(
				"updated Username = %q, want updated-user",
				updated.Username,
			)
		}

		if updated.UpdatedAt.Before(before) ||
			updated.UpdatedAt.After(after) {
			t.Fatalf(
				"updated UpdatedAt = %v, want between %v and %v",
				updated.UpdatedAt,
				before,
				after,
			)
		}

		if !updated.UpdatedAt.After(existing.UpdatedAt) {
			t.Fatalf(
				"updated UpdatedAt = %v, want after existing UpdatedAt %v",
				updated.UpdatedAt,
				existing.UpdatedAt,
			)
		}

		if uow.commitCount != 1 {
			t.Fatalf(
				"Commit() count = %d, want 1",
				uow.commitCount,
			)
		}

		if uow.rollbackCount != 0 {
			t.Fatalf(
				"Rollback() count = %d, want 0",
				uow.rollbackCount,
			)
		}
	})

	t.Run("non admin forbidden", func(t *testing.T) {
		uowManager := &userServiceFakeUnitOfWorkManager{}

		svc := newUserServiceForTest(
			t,
			repository.RepositorySet{
				Users: &userServiceFakeUserRepository{},
			},
			uowManager,
		)

		_, err := svc.Update(ctx, UpdateUserInput{
			Actor: Actor{
				UserID: uuid.New(),
				Role:   domain.RoleWaliKelas,
			},
			UserID:       userID,
			Username:     "updated-user",
			PasswordHash: "updated-password",
			Role:         domain.RoleSiswa,
		})

		if !errors.Is(err, ErrForbidden) {
			t.Fatalf(
				"Update() error = %v, want ErrForbidden",
				err,
			)
		}

		if uowManager.beginCount != 0 {
			t.Fatalf(
				"Begin() count = %d, want 0",
				uowManager.beginCount,
			)
		}
	})

	t.Run("nil user id rejected", func(t *testing.T) {
		uowManager := &userServiceFakeUnitOfWorkManager{}

		svc := newUserServiceForTest(
			t,
			repository.RepositorySet{
				Users: &userServiceFakeUserRepository{},
			},
			uowManager,
		)

		_, err := svc.Update(ctx, UpdateUserInput{
			Actor:        admin,
			UserID:       uuid.Nil,
			Username:     "updated-user",
			PasswordHash: "updated-password",
			Role:         domain.RoleSiswa,
		})

		if !errors.Is(err, ErrScopeViolation) {
			t.Fatalf(
				"Update() error = %v, want ErrScopeViolation",
				err,
			)
		}

		if uowManager.beginCount != 0 {
			t.Fatalf(
				"Begin() count = %d, want 0",
				uowManager.beginCount,
			)
		}
	})

	t.Run("user not found rolls back", func(t *testing.T) {
		userRepo := &userServiceFakeUserRepository{
			getByIDFn: func(
				context.Context,
				uuid.UUID,
			) (domain.User, error) {
				return domain.User{}, repository.ErrNotFound
			},
		}

		uow := &userServiceFakeUnitOfWork{
			repos: repository.RepositorySet{
				Users: userRepo,
			},
		}

		svc := newUserServiceForTest(
			t,
			repository.RepositorySet{
				Users: userRepo,
			},
			&userServiceFakeUnitOfWorkManager{
				uow: uow,
			},
		)

		_, err := svc.Update(ctx, UpdateUserInput{
			Actor:        admin,
			UserID:       userID,
			Username:     "updated-user",
			PasswordHash: "updated-password",
			Role:         domain.RoleSiswa,
		})

		if !errors.Is(err, repository.ErrNotFound) {
			t.Fatalf(
				"Update() error = %v, want ErrNotFound",
				err,
			)
		}

		if uow.rollbackCount != 1 {
			t.Fatalf(
				"Rollback() count = %d, want 1",
				uow.rollbackCount,
			)
		}
	})

	t.Run("invalid update input", func(t *testing.T) {
		userRepo := &userServiceFakeUserRepository{
			getByIDFn: func(
				context.Context,
				uuid.UUID,
			) (domain.User, error) {
				return existing, nil
			},
		}

		uow := &userServiceFakeUnitOfWork{
			repos: repository.RepositorySet{
				Users: userRepo,
			},
		}

		svc := newUserServiceForTest(
			t,
			repository.RepositorySet{
				Users: userRepo,
			},
			&userServiceFakeUnitOfWorkManager{
				uow: uow,
			},
		)

		_, err := svc.Update(ctx, UpdateUserInput{
			Actor:        admin,
			UserID:       userID,
			Username:     "",
			PasswordHash: "updated-password",
			Role:         domain.RoleSiswa,
		})

		if !errors.Is(err, domain.ErrEmptyName) {
			t.Fatalf(
				"Update() error = %v, want ErrEmptyName",
				err,
			)
		}

		if uow.rollbackCount != 1 {
			t.Fatalf(
				"Rollback() count = %d, want 1",
				uow.rollbackCount,
			)
		}
	})

	t.Run("username conflict rolls back", func(t *testing.T) {
		userRepo := &userServiceFakeUserRepository{
			getByIDFn: func(
				context.Context,
				uuid.UUID,
			) (domain.User, error) {
				return existing, nil
			},
			existsByUsernameFn: func(
				context.Context,
				string,
			) (bool, error) {
				return true, nil
			},
		}

		uow := &userServiceFakeUnitOfWork{
			repos: repository.RepositorySet{
				Users: userRepo,
			},
		}

		svc := newUserServiceForTest(
			t,
			repository.RepositorySet{
				Users: userRepo,
			},
			&userServiceFakeUnitOfWorkManager{
				uow: uow,
			},
		)

		_, err := svc.Update(ctx, UpdateUserInput{
			Actor:        admin,
			UserID:       userID,
			Username:     "another-user",
			PasswordHash: "updated-password",
			Role:         domain.RoleSiswa,
		})

		if !errors.Is(err, repository.ErrConflict) {
			t.Fatalf(
				"Update() error = %v, want ErrConflict",
				err,
			)
		}

		if uow.rollbackCount != 1 {
			t.Fatalf(
				"Rollback() count = %d, want 1",
				uow.rollbackCount,
			)
		}
	})

	t.Run("update repository error rolls back", func(t *testing.T) {
		updateErr := errors.New("update failed")

		userRepo := &userServiceFakeUserRepository{
			getByIDFn: func(
				context.Context,
				uuid.UUID,
			) (domain.User, error) {
				return existing, nil
			},
			updateFn: func(
				context.Context,
				domain.User,
			) error {
				return updateErr
			},
		}

		uow := &userServiceFakeUnitOfWork{
			repos: repository.RepositorySet{
				Users: userRepo,
			},
		}

		svc := newUserServiceForTest(
			t,
			repository.RepositorySet{
				Users: userRepo,
			},
			&userServiceFakeUnitOfWorkManager{
				uow: uow,
			},
		)

		_, err := svc.Update(ctx, UpdateUserInput{
			Actor:        admin,
			UserID:       userID,
			Username:     "updated-user",
			PasswordHash: "updated-password",
			Role:         domain.RoleSiswa,
		})

		if !errors.Is(err, updateErr) {
			t.Fatalf(
				"Update() error = %v, want update error",
				err,
			)
		}

		if uow.rollbackCount != 1 {
			t.Fatalf(
				"Rollback() count = %d, want 1",
				uow.rollbackCount,
			)
		}
	})

	t.Run("commit error rolls back", func(t *testing.T) {
		commitErr := errors.New("commit failed")

		userRepo := &userServiceFakeUserRepository{
			getByIDFn: func(
				context.Context,
				uuid.UUID,
			) (domain.User, error) {
				return existing, nil
			},
		}

		uow := &userServiceFakeUnitOfWork{
			repos: repository.RepositorySet{
				Users: userRepo,
			},
			commitErr: commitErr,
		}

		svc := newUserServiceForTest(
			t,
			repository.RepositorySet{
				Users: userRepo,
			},
			&userServiceFakeUnitOfWorkManager{
				uow: uow,
			},
		)

		_, err := svc.Update(ctx, UpdateUserInput{
			Actor:        admin,
			UserID:       userID,
			Username:     "updated-user",
			PasswordHash: "updated-password",
			Role:         domain.RoleSiswa,
		})

		if !errors.Is(err, commitErr) {
			t.Fatalf(
				"Update() error = %v, want commit error",
				err,
			)
		}

		if uow.rollbackCount != 1 {
			t.Fatalf(
				"Rollback() count = %d, want 1",
				uow.rollbackCount,
			)
		}
	})
}
