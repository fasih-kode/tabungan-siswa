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

type academicYearServiceFakeAcademicYearRepository struct {
	repository.AcademicYearRepository

	createFn  func(context.Context, domain.AcademicYear) error
	getByIDFn func(context.Context, uuid.UUID) (domain.AcademicYear, error)
	listFn    func(context.Context, repository.ListOptions) ([]domain.AcademicYear, error)
	updateFn  func(context.Context, domain.AcademicYear) error
	getOpenFn func(context.Context) (domain.AcademicYear, error)

	created      domain.AcademicYear
	updated      domain.AcademicYear
	createCalls  int
	getByIDCalls int
	listCalls    int
	updateCalls  int
}

func (f *academicYearServiceFakeAcademicYearRepository) Create(
	ctx context.Context,
	academicYear domain.AcademicYear,
) error {
	f.createCalls++
	f.created = academicYear

	if f.createFn != nil {
		return f.createFn(ctx, academicYear)
	}

	return nil
}

func (f *academicYearServiceFakeAcademicYearRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (domain.AcademicYear, error) {
	f.getByIDCalls++

	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, id)
	}

	return domain.AcademicYear{}, repository.ErrNotFound
}

func (f *academicYearServiceFakeAcademicYearRepository) List(
	ctx context.Context,
	options repository.ListOptions,
) ([]domain.AcademicYear, error) {
	f.listCalls++

	if f.listFn != nil {
		return f.listFn(ctx, options)
	}

	return nil, nil
}

func (f *academicYearServiceFakeAcademicYearRepository) Update(
	ctx context.Context,
	academicYear domain.AcademicYear,
) error {
	f.updateCalls++
	f.updated = academicYear

	if f.updateFn != nil {
		return f.updateFn(ctx, academicYear)
	}

	return nil
}

func (f *academicYearServiceFakeAcademicYearRepository) GetOpen(
	ctx context.Context,
) (domain.AcademicYear, error) {
	if f.getOpenFn != nil {
		return f.getOpenFn(ctx)
	}

	return domain.AcademicYear{}, repository.ErrNotFound
}

type academicYearServiceFakeUnitOfWork struct {
	repos         repository.RepositorySet
	commitErr     error
	rollbackErr   error
	commitCount   int
	rollbackCount int
}

func (f *academicYearServiceFakeUnitOfWork) Repositories() repository.RepositorySet {
	return f.repos
}

func (f *academicYearServiceFakeUnitOfWork) Commit() error {
	f.commitCount++
	return f.commitErr
}

func (f *academicYearServiceFakeUnitOfWork) Rollback() error {
	f.rollbackCount++
	return f.rollbackErr
}

type academicYearServiceFakeUnitOfWorkManager struct {
	uow        repository.UnitOfWork
	beginErr   error
	beginCount int
}

func (f *academicYearServiceFakeUnitOfWorkManager) Begin(
	context.Context,
) (repository.UnitOfWork, error) {
	f.beginCount++

	if f.beginErr != nil {
		return nil, f.beginErr
	}

	return f.uow, nil
}

func newAcademicYearServiceForTest(
	t *testing.T,
	repos repository.RepositorySet,
	uowManager repository.UnitOfWorkManager,
) *academicYearService {
	t.Helper()

	svc, err := NewAcademicYearService(Dependencies{
		Repositories: repos,
		UOW:          uowManager,
	})
	if err != nil {
		t.Fatalf("NewAcademicYearService() error = %v", err)
	}

	return svc
}

func newTestAcademicYear(
	id uuid.UUID,
	status domain.AcademicYearStatus,
) domain.AcademicYear {
	now := time.Now().Add(-time.Hour)

	return domain.AcademicYear{
		ID:        id,
		Name:      "2026/2027",
		StartDate: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2027, 6, 30, 0, 0, 0, 0, time.UTC),
		Status:    status,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestNewAcademicYearService(t *testing.T) {
	uowManager := &academicYearServiceFakeUnitOfWorkManager{}

	t.Run("valid dependencies", func(t *testing.T) {
		svc, err := NewAcademicYearService(
			Dependencies{UOW: uowManager},
		)

		if err != nil {
			t.Fatalf(
				"NewAcademicYearService() error = %v, want nil",
				err,
			)
		}

		if svc == nil {
			t.Fatal(
				"NewAcademicYearService() service = nil, want non-nil",
			)
		}
	})

	t.Run("invalid dependencies", func(t *testing.T) {
		svc, err := NewAcademicYearService(Dependencies{})

		if !errors.Is(err, ErrInvalidDependency) {
			t.Fatalf(
				"NewAcademicYearService() error = %v, want ErrInvalidDependency",
				err,
			)
		}

		if svc != nil {
			t.Fatal(
				"NewAcademicYearService() service != nil on invalid dependencies",
			)
		}
	})
}

func TestAcademicYearService_Create(t *testing.T) {
	ctx := context.Background()

	admin := Actor{
		UserID: uuid.New(),
		Role:   domain.RoleAdmin,
	}

	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2027, 6, 30, 0, 0, 0, 0, time.UTC)

	t.Run("admin creates and commits", func(t *testing.T) {
		repo := &academicYearServiceFakeAcademicYearRepository{}

		uow := &academicYearServiceFakeUnitOfWork{
			repos: repository.RepositorySet{
				AcademicYears: repo,
			},
		}

		manager := &academicYearServiceFakeUnitOfWorkManager{
			uow: uow,
		}

		svc := newAcademicYearServiceForTest(
			t,
			repository.RepositorySet{
				AcademicYears: repo,
			},
			manager,
		)

		got, err := svc.Create(ctx, CreateAcademicYearInput{
			Actor:     admin,
			Name:      "  2026/2027  ",
			StartDate: start,
			EndDate:   end,
		})

		if err != nil {
			t.Fatalf("Create() error = %v, want nil", err)
		}

		if got.AcademicYear == nil {
			t.Fatal("Create() AcademicYear = nil, want non-nil")
		}

		if got.AcademicYear.Name != "2026/2027" {
			t.Fatalf(
				"Name = %q, want %q",
				got.AcademicYear.Name,
				"2026/2027",
			)
		}

		if repo.createCalls != 1 ||
			uow.commitCount != 1 ||
			uow.rollbackCount != 0 ||
			manager.beginCount != 1 {
			t.Fatalf(
				"counts = create:%d begin:%d commit:%d rollback:%d, want 1/1/1/0",
				repo.createCalls,
				manager.beginCount,
				uow.commitCount,
				uow.rollbackCount,
			)
		}
	})

	for _, role := range []domain.UserRole{
		domain.RoleWaliKelas,
		domain.RoleSiswa,
	} {
		t.Run(string(role)+" forbidden", func(t *testing.T) {
			repo := &academicYearServiceFakeAcademicYearRepository{}
			manager := &academicYearServiceFakeUnitOfWorkManager{}

			svc := newAcademicYearServiceForTest(
				t,
				repository.RepositorySet{
					AcademicYears: repo,
				},
				manager,
			)

			_, err := svc.Create(ctx, CreateAcademicYearInput{
				Actor: Actor{
					UserID: uuid.New(),
					Role:   role,
				},
				Name:      "2026/2027",
				StartDate: start,
				EndDate:   end,
			})

			if !errors.Is(err, ErrForbidden) {
				t.Fatalf(
					"Create() error = %v, want ErrForbidden",
					err,
				)
			}

			if manager.beginCount != 0 || repo.createCalls != 0 {
				t.Fatal(
					"forbidden Create() touched transaction/repository",
				)
			}
		})
	}

	t.Run("domain validation error", func(t *testing.T) {
		repo := &academicYearServiceFakeAcademicYearRepository{}
		manager := &academicYearServiceFakeUnitOfWorkManager{}

		svc := newAcademicYearServiceForTest(
			t,
			repository.RepositorySet{
				AcademicYears: repo,
			},
			manager,
		)

		_, err := svc.Create(ctx, CreateAcademicYearInput{
			Actor:     admin,
			Name:      " ",
			StartDate: start,
			EndDate:   end,
		})

		if !errors.Is(err, domain.ErrEmptyName) {
			t.Fatalf(
				"Create() error = %v, want ErrEmptyName",
				err,
			)
		}

		if manager.beginCount != 0 {
			t.Fatal("domain validation started transaction")
		}
	})

	t.Run("repository error rolls back", func(t *testing.T) {
		repoErr := errors.New("create failed")

		repo := &academicYearServiceFakeAcademicYearRepository{
			createFn: func(
				context.Context,
				domain.AcademicYear,
			) error {
				return repoErr
			},
		}

		uow := &academicYearServiceFakeUnitOfWork{
			repos: repository.RepositorySet{
				AcademicYears: repo,
			},
		}

		svc := newAcademicYearServiceForTest(
			t,
			repository.RepositorySet{
				AcademicYears: repo,
			},
			&academicYearServiceFakeUnitOfWorkManager{
				uow: uow,
			},
		)

		_, err := svc.Create(ctx, CreateAcademicYearInput{
			Actor:     admin,
			Name:      "2026/2027",
			StartDate: start,
			EndDate:   end,
		})

		if !errors.Is(err, repoErr) {
			t.Fatalf(
				"Create() error = %v, want repository error",
				err,
			)
		}

		if uow.rollbackCount != 1 || uow.commitCount != 0 {
			t.Fatalf(
				"commit:%d rollback:%d, want 0/1",
				uow.commitCount,
				uow.rollbackCount,
			)
		}
	})

	t.Run("begin error wrapped", func(t *testing.T) {
		beginErr := errors.New("begin failed")

		manager := &academicYearServiceFakeUnitOfWorkManager{
			beginErr: beginErr,
		}

		svc := newAcademicYearServiceForTest(
			t,
			repository.RepositorySet{
				AcademicYears: &academicYearServiceFakeAcademicYearRepository{},
			},
			manager,
		)

		_, err := svc.Create(ctx, CreateAcademicYearInput{
			Actor:     admin,
			Name:      "2026/2027",
			StartDate: start,
			EndDate:   end,
		})

		if !errors.Is(err, beginErr) {
			t.Fatalf(
				"Create() error = %v, want wrapped begin error",
				err,
			)
		}
	})

	t.Run("commit error rolls back", func(t *testing.T) {
		commitErr := errors.New("commit failed")

		repo := &academicYearServiceFakeAcademicYearRepository{}

		uow := &academicYearServiceFakeUnitOfWork{
			repos: repository.RepositorySet{
				AcademicYears: repo,
			},
			commitErr: commitErr,
		}

		svc := newAcademicYearServiceForTest(
			t,
			repository.RepositorySet{
				AcademicYears: repo,
			},
			&academicYearServiceFakeUnitOfWorkManager{
				uow: uow,
			},
		)

		_, err := svc.Create(ctx, CreateAcademicYearInput{
			Actor:     admin,
			Name:      "2026/2027",
			StartDate: start,
			EndDate:   end,
		})

		if !errors.Is(err, commitErr) {
			t.Fatalf(
				"Create() error = %v, want wrapped commit error",
				err,
			)
		}

		if uow.rollbackCount != 1 || uow.commitCount != 1 {
			t.Fatalf(
				"commit:%d rollback:%d, want 1/1",
				uow.commitCount,
				uow.rollbackCount,
			)
		}
	})
}

func TestAcademicYearService_Get(t *testing.T) {
	ctx := context.Background()

	academicYearID := uuid.New()
	academicYear := newTestAcademicYear(
		academicYearID,
		domain.AcademicYearOpen,
	)

	for _, role := range []domain.UserRole{
		domain.RoleAdmin,
		domain.RoleWaliKelas,
		domain.RoleSiswa,
	} {
		t.Run(string(role)+" allowed", func(t *testing.T) {
			repo := &academicYearServiceFakeAcademicYearRepository{
				getByIDFn: func(
					context.Context,
					uuid.UUID,
				) (domain.AcademicYear, error) {
					return academicYear, nil
				},
			}

			svc := newAcademicYearServiceForTest(
				t,
				repository.RepositorySet{
					AcademicYears: repo,
				},
				&academicYearServiceFakeUnitOfWorkManager{},
			)

			got, err := svc.Get(ctx, GetAcademicYearInput{
				Actor: Actor{
					UserID: uuid.New(),
					Role:   role,
				},
				AcademicYearID: academicYearID,
			})

			if err != nil {
				t.Fatalf("Get() error = %v, want nil", err)
			}

			if got.AcademicYear == nil ||
				got.AcademicYear.ID != academicYearID {
				t.Fatalf(
					"Get() AcademicYear = %#v, want ID %v",
					got.AcademicYear,
					academicYearID,
				)
			}
		})
	}

	t.Run("invalid ID", func(t *testing.T) {
		repo := &academicYearServiceFakeAcademicYearRepository{}

		svc := newAcademicYearServiceForTest(
			t,
			repository.RepositorySet{
				AcademicYears: repo,
			},
			&academicYearServiceFakeUnitOfWorkManager{},
		)

		_, err := svc.Get(ctx, GetAcademicYearInput{
			Actor: Actor{
				UserID: uuid.New(),
				Role:   domain.RoleAdmin,
			},
		})

		if !errors.Is(err, domain.ErrInvalidID) {
			t.Fatalf(
				"Get() error = %v, want ErrInvalidID",
				err,
			)
		}

		if repo.getByIDCalls != 0 {
			t.Fatal("invalid ID reached repository")
		}
	})

	t.Run("repository error propagated", func(t *testing.T) {
		repo := &academicYearServiceFakeAcademicYearRepository{
			getByIDFn: func(
				context.Context,
				uuid.UUID,
			) (domain.AcademicYear, error) {
				return domain.AcademicYear{}, repository.ErrNotFound
			},
		}

		svc := newAcademicYearServiceForTest(
			t,
			repository.RepositorySet{
				AcademicYears: repo,
			},
			&academicYearServiceFakeUnitOfWorkManager{},
		)

		_, err := svc.Get(ctx, GetAcademicYearInput{
			Actor: Actor{
				UserID: uuid.New(),
				Role:   domain.RoleAdmin,
			},
			AcademicYearID: academicYearID,
		})

		if !errors.Is(err, repository.ErrNotFound) {
			t.Fatalf(
				"Get() error = %v, want ErrNotFound",
				err,
			)
		}
	})
}

func TestAcademicYearService_List(t *testing.T) {
	ctx := context.Background()

	academicYears := []domain.AcademicYear{
		newTestAcademicYear(
			uuid.New(),
			domain.AcademicYearOpen,
		),
		newTestAcademicYear(
			uuid.New(),
			domain.AcademicYearClosed,
		),
	}

	for _, role := range []domain.UserRole{
		domain.RoleAdmin,
		domain.RoleWaliKelas,
		domain.RoleSiswa,
	} {
		t.Run(string(role)+" allowed and maps options", func(t *testing.T) {
			var gotOptions repository.ListOptions

			repo := &academicYearServiceFakeAcademicYearRepository{
				listFn: func(
					_ context.Context,
					options repository.ListOptions,
				) ([]domain.AcademicYear, error) {
					gotOptions = options
					return academicYears, nil
				},
			}

			svc := newAcademicYearServiceForTest(
				t,
				repository.RepositorySet{
					AcademicYears: repo,
				},
				&academicYearServiceFakeUnitOfWorkManager{},
			)

			got, err := svc.List(ctx, ListAcademicYearsInput{
				Actor: Actor{
					UserID: uuid.New(),
					Role:   role,
				},
				Options: ListOptions{
					Limit:  7,
					Offset: 14,
				},
			})

			if err != nil {
				t.Fatalf("List() error = %v, want nil", err)
			}

			if gotOptions.Limit != 7 ||
				gotOptions.Offset != 14 {
				t.Fatalf(
					"repository options = %+v, want limit 7 offset 14",
					gotOptions,
				)
			}

			if len(got.AcademicYears) != 2 ||
				got.AcademicYears[0] == nil ||
				got.AcademicYears[1] == nil {
				t.Fatalf(
					"AcademicYears = %#v, want 2 non-nil pointers",
					got.AcademicYears,
				)
			}
		})
	}

	t.Run("repository error propagated", func(t *testing.T) {
		repoErr := errors.New("list failed")

		repo := &academicYearServiceFakeAcademicYearRepository{
			listFn: func(
				context.Context,
				repository.ListOptions,
			) ([]domain.AcademicYear, error) {
				return nil, repoErr
			},
		}

		svc := newAcademicYearServiceForTest(
			t,
			repository.RepositorySet{
				AcademicYears: repo,
			},
			&academicYearServiceFakeUnitOfWorkManager{},
		)

		_, err := svc.List(ctx, ListAcademicYearsInput{
			Actor: Actor{
				UserID: uuid.New(),
				Role:   domain.RoleAdmin,
			},
		})

		if !errors.Is(err, repoErr) {
			t.Fatalf(
				"List() error = %v, want repository error",
				err,
			)
		}
	})
}

func TestAcademicYearService_StartClosing(t *testing.T) {
	testAcademicYearLifecycleService(
		t,
		"StartClosing",
		domain.AcademicYearOpen,
		domain.AcademicYearClosing,
		func(
			svc *academicYearService,
			ctx context.Context,
			actor Actor,
			id uuid.UUID,
		) (domain.AcademicYear, error) {
			out, err := svc.StartClosing(
				ctx,
				StartClosingAcademicYearInput{
					Actor:          actor,
					AcademicYearID: id,
				},
			)

			return dereferenceAcademicYear(out.AcademicYear), err
		},
	)
}

func TestAcademicYearService_Close(t *testing.T) {
	testAcademicYearLifecycleService(
		t,
		"Close",
		domain.AcademicYearClosing,
		domain.AcademicYearClosed,
		func(
			svc *academicYearService,
			ctx context.Context,
			actor Actor,
			id uuid.UUID,
		) (domain.AcademicYear, error) {
			out, err := svc.Close(
				ctx,
				CloseAcademicYearInput{
					Actor:          actor,
					AcademicYearID: id,
				},
			)

			return dereferenceAcademicYear(out.AcademicYear), err
		},
	)
}

func TestAcademicYearService_Reopen(t *testing.T) {
	testAcademicYearLifecycleService(
		t,
		"Reopen",
		domain.AcademicYearClosed,
		domain.AcademicYearOpen,
		func(
			svc *academicYearService,
			ctx context.Context,
			actor Actor,
			id uuid.UUID,
		) (domain.AcademicYear, error) {
			out, err := svc.Reopen(
				ctx,
				ReopenAcademicYearInput{
					Actor:          actor,
					AcademicYearID: id,
				},
			)

			return dereferenceAcademicYear(out.AcademicYear), err
		},
	)
}

func dereferenceAcademicYear(
	value *domain.AcademicYear,
) domain.AcademicYear {
	if value == nil {
		return domain.AcademicYear{}
	}

	return *value
}

func testAcademicYearLifecycleService(
	t *testing.T,
	name string,
	initialStatus domain.AcademicYearStatus,
	finalStatus domain.AcademicYearStatus,
	call func(
		*academicYearService,
		context.Context,
		Actor,
		uuid.UUID,
	) (domain.AcademicYear, error),
) {
	t.Helper()

	ctx := context.Background()

	admin := Actor{
		UserID: uuid.New(),
		Role:   domain.RoleAdmin,
	}

	academicYearID := uuid.New()

	t.Run("admin transitions and commits", func(t *testing.T) {
		repo := &academicYearServiceFakeAcademicYearRepository{
			getByIDFn: func(
				context.Context,
				uuid.UUID,
			) (domain.AcademicYear, error) {
				return newTestAcademicYear(
					academicYearID,
					initialStatus,
				), nil
			},
		}

		uow := &academicYearServiceFakeUnitOfWork{
			repos: repository.RepositorySet{
				AcademicYears: repo,
			},
		}

		svc := newAcademicYearServiceForTest(
			t,
			repository.RepositorySet{
				AcademicYears: repo,
			},
			&academicYearServiceFakeUnitOfWorkManager{
				uow: uow,
			},
		)

		got, err := call(
			svc,
			ctx,
			admin,
			academicYearID,
		)

		if err != nil {
			t.Fatalf(
				"%s() error = %v, want nil",
				name,
				err,
			)
		}

		if got.Status != finalStatus {
			t.Fatalf(
				"%s() status = %v, want %v",
				name,
				got.Status,
				finalStatus,
			)
		}

		if repo.updateCalls != 1 ||
			uow.commitCount != 1 ||
			uow.rollbackCount != 0 {
			t.Fatalf(
				"update:%d commit:%d rollback:%d, want 1/1/0",
				repo.updateCalls,
				uow.commitCount,
				uow.rollbackCount,
			)
		}
	})

	for _, role := range []domain.UserRole{
		domain.RoleWaliKelas,
		domain.RoleSiswa,
	} {
		t.Run(string(role)+" forbidden", func(t *testing.T) {
			repo := &academicYearServiceFakeAcademicYearRepository{}
			manager := &academicYearServiceFakeUnitOfWorkManager{}

			svc := newAcademicYearServiceForTest(
				t,
				repository.RepositorySet{
					AcademicYears: repo,
				},
				manager,
			)

			_, err := call(
				svc,
				ctx,
				Actor{
					UserID: uuid.New(),
					Role:   role,
				},
				academicYearID,
			)

			if !errors.Is(err, ErrForbidden) {
				t.Fatalf(
					"%s() error = %v, want ErrForbidden",
					name,
					err,
				)
			}

			if manager.beginCount != 0 {
				t.Fatal(
					"forbidden lifecycle operation started transaction",
				)
			}
		})
	}

	t.Run("invalid ID", func(t *testing.T) {
		repo := &academicYearServiceFakeAcademicYearRepository{}
		manager := &academicYearServiceFakeUnitOfWorkManager{}

		svc := newAcademicYearServiceForTest(
			t,
			repository.RepositorySet{
				AcademicYears: repo,
			},
			manager,
		)

		_, err := call(
			svc,
			ctx,
			admin,
			uuid.Nil,
		)

		if !errors.Is(err, domain.ErrInvalidID) {
			t.Fatalf(
				"%s() error = %v, want ErrInvalidID",
				name,
				err,
			)
		}

		if manager.beginCount != 0 {
			t.Fatal("invalid ID started transaction")
		}
	})

	t.Run("not found rolls back", func(t *testing.T) {
		repo := &academicYearServiceFakeAcademicYearRepository{
			getByIDFn: func(
				context.Context,
				uuid.UUID,
			) (domain.AcademicYear, error) {
				return domain.AcademicYear{}, repository.ErrNotFound
			},
		}

		uow := &academicYearServiceFakeUnitOfWork{
			repos: repository.RepositorySet{
				AcademicYears: repo,
			},
		}

		svc := newAcademicYearServiceForTest(
			t,
			repository.RepositorySet{
				AcademicYears: repo,
			},
			&academicYearServiceFakeUnitOfWorkManager{
				uow: uow,
			},
		)

		_, err := call(
			svc,
			ctx,
			admin,
			academicYearID,
		)

		if !errors.Is(err, repository.ErrNotFound) {
			t.Fatalf(
				"%s() error = %v, want ErrNotFound",
				name,
				err,
			)
		}

		if uow.rollbackCount != 1 ||
			uow.commitCount != 0 {
			t.Fatalf(
				"commit:%d rollback:%d, want 0/1",
				uow.commitCount,
				uow.rollbackCount,
			)
		}
	})

	t.Run("invalid domain transition rolls back", func(t *testing.T) {
		invalidStatus := domain.AcademicYearClosed

		if name == "Close" {
			invalidStatus = domain.AcademicYearOpen
		}

		if name == "Reopen" {
			invalidStatus = domain.AcademicYearOpen
		}

		repo := &academicYearServiceFakeAcademicYearRepository{
			getByIDFn: func(
				context.Context,
				uuid.UUID,
			) (domain.AcademicYear, error) {
				return newTestAcademicYear(
					academicYearID,
					invalidStatus,
				), nil
			},
		}

		uow := &academicYearServiceFakeUnitOfWork{
			repos: repository.RepositorySet{
				AcademicYears: repo,
			},
		}

		svc := newAcademicYearServiceForTest(
			t,
			repository.RepositorySet{
				AcademicYears: repo,
			},
			&academicYearServiceFakeUnitOfWorkManager{
				uow: uow,
			},
		)

		_, err := call(
			svc,
			ctx,
			admin,
			academicYearID,
		)

		if !errors.Is(err, domain.ErrInvalidTransition) {
			t.Fatalf(
				"%s() error = %v, want ErrInvalidTransition",
				name,
				err,
			)
		}

		if uow.rollbackCount != 1 ||
			repo.updateCalls != 0 ||
			uow.commitCount != 0 {
			t.Fatalf(
				"update:%d commit:%d rollback:%d, want 0/0/1",
				repo.updateCalls,
				uow.commitCount,
				uow.rollbackCount,
			)
		}
	})

	t.Run("update error rolls back", func(t *testing.T) {
		updateErr := errors.New("update failed")

		repo := &academicYearServiceFakeAcademicYearRepository{
			getByIDFn: func(
				context.Context,
				uuid.UUID,
			) (domain.AcademicYear, error) {
				return newTestAcademicYear(
					academicYearID,
					initialStatus,
				), nil
			},
			updateFn: func(
				context.Context,
				domain.AcademicYear,
			) error {
				return updateErr
			},
		}

		uow := &academicYearServiceFakeUnitOfWork{
			repos: repository.RepositorySet{
				AcademicYears: repo,
			},
		}

		svc := newAcademicYearServiceForTest(
			t,
			repository.RepositorySet{
				AcademicYears: repo,
			},
			&academicYearServiceFakeUnitOfWorkManager{
				uow: uow,
			},
		)

		_, err := call(
			svc,
			ctx,
			admin,
			academicYearID,
		)

		if !errors.Is(err, updateErr) {
			t.Fatalf(
				"%s() error = %v, want update error",
				name,
				err,
			)
		}

		if uow.rollbackCount != 1 ||
			uow.commitCount != 0 {
			t.Fatalf(
				"commit:%d rollback:%d, want 0/1",
				uow.commitCount,
				uow.rollbackCount,
			)
		}
	})

	t.Run("begin error wrapped", func(t *testing.T) {
		beginErr := errors.New("begin failed")

		manager := &academicYearServiceFakeUnitOfWorkManager{
			beginErr: beginErr,
		}

		svc := newAcademicYearServiceForTest(
			t,
			repository.RepositorySet{
				AcademicYears: &academicYearServiceFakeAcademicYearRepository{},
			},
			manager,
		)

		_, err := call(
			svc,
			ctx,
			admin,
			academicYearID,
		)

		if !errors.Is(err, beginErr) {
			t.Fatalf(
				"%s() error = %v, want wrapped begin error",
				name,
				err,
			)
		}
	})

	t.Run("commit error rolls back", func(t *testing.T) {
		commitErr := errors.New("commit failed")

		repo := &academicYearServiceFakeAcademicYearRepository{
			getByIDFn: func(
				context.Context,
				uuid.UUID,
			) (domain.AcademicYear, error) {
				return newTestAcademicYear(
					academicYearID,
					initialStatus,
				), nil
			},
		}

		uow := &academicYearServiceFakeUnitOfWork{
			repos: repository.RepositorySet{
				AcademicYears: repo,
			},
			commitErr: commitErr,
		}

		svc := newAcademicYearServiceForTest(
			t,
			repository.RepositorySet{
				AcademicYears: repo,
			},
			&academicYearServiceFakeUnitOfWorkManager{
				uow: uow,
			},
		)

		_, err := call(
			svc,
			ctx,
			admin,
			academicYearID,
		)

		if !errors.Is(err, commitErr) {
			t.Fatalf(
				"%s() error = %v, want wrapped commit error",
				name,
				err,
			)
		}

		if uow.commitCount != 1 ||
			uow.rollbackCount != 1 {
			t.Fatalf(
				"commit:%d rollback:%d, want 1/1",
				uow.commitCount,
				uow.rollbackCount,
			)
		}
	})
}
