package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/repository/postgres"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

func TestAcademicYearServiceIntegration_Create(t *testing.T) {
	db := openServiceScopeTestDatabase(t)
	ctx := context.Background()

	cleanupServiceScopeFixtures(t, db)

	academicYearRepo := postgres.NewAcademicYearRepository(db)

	svc, err := service.NewAcademicYearService(service.Dependencies{
		Repositories: repository.RepositorySet{
			AcademicYears: academicYearRepo,
		},
		UOW: postgres.NewUnitOfWorkManager(db),
	})
	if err != nil {
		t.Fatalf("NewAcademicYearService() error = %v", err)
	}

	actor := newServiceScopeUser(t, domain.RoleAdmin)

	startDate := time.Date(
		2026, 7, 1, 0, 0, 0, 0, time.UTC,
	)
	endDate := time.Date(
		2027, 7, 1, 0, 0, 0, 0, time.UTC,
	)

	name := "Integration " + uuid.NewString()

	got, err := svc.Create(ctx, service.CreateAcademicYearInput{
		Actor:     service.Actor{UserID: actor.ID, Role: domain.RoleAdmin},
		Name:      name,
		StartDate: startDate,
		EndDate:   endDate,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if got.AcademicYear == nil {
		t.Fatal("Create() returned nil AcademicYear")
	}

	if got.AcademicYear.Name != name {
		t.Fatalf(
			"Name = %q, want %q",
			got.AcademicYear.Name,
			name,
		)
	}

	if got.AcademicYear.Status != domain.AcademicYearOpen {
		t.Fatalf(
			"Status = %q, want %q",
			got.AcademicYear.Status,
			domain.AcademicYearOpen,
		)
	}

	persisted, err := academicYearRepo.GetByID(
		ctx,
		got.AcademicYear.ID,
	)
	if err != nil {
		t.Fatalf(
			"GetByID() after Create() error = %v",
			err,
		)
	}

	if persisted.ID != got.AcademicYear.ID {
		t.Fatalf(
			"persisted ID = %v, want %v",
			persisted.ID,
			got.AcademicYear.ID,
		)
	}

	if persisted.Name != name {
		t.Fatalf(
			"persisted Name = %q, want %q",
			persisted.Name,
			name,
		)
	}

	if persisted.Status != domain.AcademicYearOpen {
		t.Fatalf(
			"persisted Status = %q, want %q",
			persisted.Status,
			domain.AcademicYearOpen,
		)
	}
}

func TestAcademicYearServiceIntegration_GetAndList(t *testing.T) {
	db := openServiceScopeTestDatabase(t)
	ctx := context.Background()

	cleanupServiceScopeFixtures(t, db)

	academicYearRepo := postgres.NewAcademicYearRepository(db)

	svc, err := service.NewAcademicYearService(service.Dependencies{
		Repositories: repository.RepositorySet{
			AcademicYears: academicYearRepo,
		},
		UOW: postgres.NewUnitOfWorkManager(db),
	})
	if err != nil {
		t.Fatalf("NewAcademicYearService() error = %v", err)
	}

	admin := newServiceScopeUser(t, domain.RoleAdmin)

	first := newServiceScopeAcademicYear(
		t,
		"Integration First "+uuid.NewString(),
	)

	if err := academicYearRepo.Create(ctx, first); err != nil {
		t.Fatalf("create first academic year: %v", err)
	}

	adminActor := service.Actor{
		UserID: admin.ID,
		Role:   domain.RoleAdmin,
	}

	if _, err := svc.StartClosing(ctx, service.StartClosingAcademicYearInput{
		Actor:          adminActor,
		AcademicYearID: first.ID,
	}); err != nil {
		t.Fatalf("start closing first academic year: %v", err)
	}

	if _, err := svc.Close(ctx, service.CloseAcademicYearInput{
		Actor:          adminActor,
		AcademicYearID: first.ID,
	}); err != nil {
		t.Fatalf("close first academic year: %v", err)
	}

	second := newServiceScopeAcademicYear(
		t,
		"Integration Second "+uuid.NewString(),
	)

	if err := academicYearRepo.Create(ctx, second); err != nil {
		t.Fatalf("create second academic year: %v", err)
	}

	t.Run("get persisted academic year", func(t *testing.T) {
		got, err := svc.Get(ctx, service.GetAcademicYearInput{
			Actor: service.Actor{
				UserID: admin.ID,
				Role:   domain.RoleAdmin,
			},
			AcademicYearID: first.ID,
		})
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}

		if got.AcademicYear == nil {
			t.Fatal("Get() returned nil AcademicYear")
		}

		if got.AcademicYear.ID != first.ID {
			t.Fatalf(
				"ID = %v, want %v",
				got.AcademicYear.ID,
				first.ID,
			)
		}

		if got.AcademicYear.Name != first.Name {
			t.Fatalf(
				"Name = %q, want %q",
				got.AcademicYear.Name,
				first.Name,
			)
		}
	})

	t.Run("list persisted academic years", func(t *testing.T) {
		got, err := svc.List(ctx, service.ListAcademicYearsInput{
			Actor: service.Actor{
				UserID: admin.ID,
				Role:   domain.RoleAdmin,
			},
			Options: service.ListOptions{
				Limit:  10,
				Offset: 0,
			},
		})
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if len(got.AcademicYears) < 2 {
			t.Fatalf(
				"List() returned %d academic years, want at least 2",
				len(got.AcademicYears),
			)
		}

		foundFirst := false
		foundSecond := false

		for _, academicYear := range got.AcademicYears {
			if academicYear == nil {
				continue
			}

			switch academicYear.ID {
			case first.ID:
				foundFirst = true
			case second.ID:
				foundSecond = true
			}
		}

		if !foundFirst {
			t.Fatalf(
				"List() did not contain academic year %v",
				first.ID,
			)
		}

		if !foundSecond {
			t.Fatalf(
				"List() did not contain academic year %v",
				second.ID,
			)
		}
	})
}

func TestAcademicYearServiceIntegration_Lifecycle(t *testing.T) {
	db := openServiceScopeTestDatabase(t)
	ctx := context.Background()

	cleanupServiceScopeFixtures(t, db)

	academicYearRepo := postgres.NewAcademicYearRepository(db)

	svc, err := service.NewAcademicYearService(service.Dependencies{
		Repositories: repository.RepositorySet{
			AcademicYears: academicYearRepo,
		},
		UOW: postgres.NewUnitOfWorkManager(db),
	})
	if err != nil {
		t.Fatalf("NewAcademicYearService() error = %v", err)
	}

	admin := newServiceScopeUser(t, domain.RoleAdmin)

	academicYear := newServiceScopeAcademicYear(
		t,
		"Lifecycle "+uuid.NewString(),
	)

	if err := academicYearRepo.Create(ctx, academicYear); err != nil {
		t.Fatalf("create academic year: %v", err)
	}

	t.Run("initial state is OPEN", func(t *testing.T) {
		got, err := academicYearRepo.GetByID(
			ctx,
			academicYear.ID,
		)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}

		if got.Status != domain.AcademicYearOpen {
			t.Fatalf(
				"Status = %q, want %q",
				got.Status,
				domain.AcademicYearOpen,
			)
		}
	})

	t.Run("start closing persists CLOSING", func(t *testing.T) {
		got, err := svc.StartClosing(
			ctx,
			service.StartClosingAcademicYearInput{
				Actor: service.Actor{
					UserID: admin.ID,
					Role:   domain.RoleAdmin,
				},
				AcademicYearID: academicYear.ID,
			},
		)
		if err != nil {
			t.Fatalf("StartClosing() error = %v", err)
		}

		if got.AcademicYear == nil {
			t.Fatal("StartClosing() returned nil AcademicYear")
		}

		if got.AcademicYear.Status != domain.AcademicYearClosing {
			t.Fatalf(
				"Status = %q, want %q",
				got.AcademicYear.Status,
				domain.AcademicYearClosing,
			)
		}

		persisted, err := academicYearRepo.GetByID(
			ctx,
			academicYear.ID,
		)
		if err != nil {
			t.Fatalf(
				"GetByID() after StartClosing() error = %v",
				err,
			)
		}

		if persisted.Status != domain.AcademicYearClosing {
			t.Fatalf(
				"persisted Status = %q, want %q",
				persisted.Status,
				domain.AcademicYearClosing,
			)
		}
	})

	t.Run("close persists CLOSED", func(t *testing.T) {
		got, err := svc.Close(
			ctx,
			service.CloseAcademicYearInput{
				Actor: service.Actor{
					UserID: admin.ID,
					Role:   domain.RoleAdmin,
				},
				AcademicYearID: academicYear.ID,
			},
		)
		if err != nil {
			t.Fatalf("Close() error = %v", err)
		}

		if got.AcademicYear == nil {
			t.Fatal("Close() returned nil AcademicYear")
		}

		if got.AcademicYear.Status != domain.AcademicYearClosed {
			t.Fatalf(
				"Status = %q, want %q",
				got.AcademicYear.Status,
				domain.AcademicYearClosed,
			)
		}

		persisted, err := academicYearRepo.GetByID(
			ctx,
			academicYear.ID,
		)
		if err != nil {
			t.Fatalf(
				"GetByID() after Close() error = %v",
				err,
			)
		}

		if persisted.Status != domain.AcademicYearClosed {
			t.Fatalf(
				"persisted Status = %q, want %q",
				persisted.Status,
				domain.AcademicYearClosed,
			)
		}
	})

	t.Run("reopen persists OPEN", func(t *testing.T) {
		got, err := svc.Reopen(
			ctx,
			service.ReopenAcademicYearInput{
				Actor: service.Actor{
					UserID: admin.ID,
					Role:   domain.RoleAdmin,
				},
				AcademicYearID: academicYear.ID,
			},
		)
		if err != nil {
			t.Fatalf("Reopen() error = %v", err)
		}

		if got.AcademicYear == nil {
			t.Fatal("Reopen() returned nil AcademicYear")
		}

		if got.AcademicYear.Status != domain.AcademicYearOpen {
			t.Fatalf(
				"Status = %q, want %q",
				got.AcademicYear.Status,
				domain.AcademicYearOpen,
			)
		}

		persisted, err := academicYearRepo.GetByID(
			ctx,
			academicYear.ID,
		)
		if err != nil {
			t.Fatalf(
				"GetByID() after Reopen() error = %v",
				err,
			)
		}

		if persisted.Status != domain.AcademicYearOpen {
			t.Fatalf(
				"persisted Status = %q, want %q",
				persisted.Status,
				domain.AcademicYearOpen,
			)
		}
	})
}

func TestAcademicYearServiceIntegration_Authorization(t *testing.T) {
	db := openServiceScopeTestDatabase(t)
	ctx := context.Background()

	cleanupServiceScopeFixtures(t, db)

	academicYearRepo := postgres.NewAcademicYearRepository(db)

	svc, err := service.NewAcademicYearService(service.Dependencies{
		Repositories: repository.RepositorySet{
			AcademicYears: academicYearRepo,
		},
		UOW: postgres.NewUnitOfWorkManager(db),
	})
	if err != nil {
		t.Fatalf("NewAcademicYearService() error = %v", err)
	}

	startDate := time.Date(
		2026, 7, 1, 0, 0, 0, 0, time.UTC,
	)
	endDate := time.Date(
		2027, 7, 1, 0, 0, 0, 0, time.UTC,
	)

	for _, role := range []domain.UserRole{
		domain.RoleWaliKelas,
		domain.RoleSiswa,
	} {
		t.Run(string(role)+" cannot create", func(t *testing.T) {
			_, err := svc.Create(ctx, service.CreateAcademicYearInput{
				Actor: service.Actor{
					UserID: uuid.New(),
					Role:   role,
				},
				Name:      "Forbidden " + uuid.NewString(),
				StartDate: startDate,
				EndDate:   endDate,
			})

			if !errors.Is(err, service.ErrForbidden) {
				t.Fatalf(
					"Create() error = %v, want ErrForbidden",
					err,
				)
			}
		})
	}

	t.Run("wali kelas cannot start closing", func(t *testing.T) {
		academicYear := newServiceScopeAcademicYear(
			t,
			"Authorization "+uuid.NewString(),
		)

		if err := academicYearRepo.Create(ctx, academicYear); err != nil {
			t.Fatalf(
				"create academic year: %v",
				err,
			)
		}

		_, err := svc.StartClosing(
			ctx,
			service.StartClosingAcademicYearInput{
				Actor: service.Actor{
					UserID: uuid.New(),
					Role:   domain.RoleWaliKelas,
				},
				AcademicYearID: academicYear.ID,
			},
		)

		if !errors.Is(err, service.ErrForbidden) {
			t.Fatalf(
				"StartClosing() error = %v, want ErrForbidden",
				err,
			)
		}

		persisted, err := academicYearRepo.GetByID(
			ctx,
			academicYear.ID,
		)
		if err != nil {
			t.Fatalf(
				"GetByID() after forbidden operation error = %v",
				err,
			)
		}

		if persisted.Status != domain.AcademicYearOpen {
			t.Fatalf(
				"Status = %q, want %q after forbidden operation",
				persisted.Status,
				domain.AcademicYearOpen,
			)
		}
	})
}
