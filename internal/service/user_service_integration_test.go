package service_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/repository/postgres"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

func TestUserServiceIntegration_Get(t *testing.T) {
	db := openServiceScopeTestDatabase(t)
	ctx := context.Background()

	cleanupServiceScopeFixtures(t, db)

	userRepo := postgres.NewUserRepository(db)
	studentRepo := postgres.NewStudentRepository(db)
	academicYearRepo := postgres.NewAcademicYearRepository(db)
	classRepo := postgres.NewClassRepository(db)
	assignmentRepo := postgres.NewTeacherClassAssignmentRepository(db)
	historyRepo := postgres.NewStudentClassHistoryRepository(db)

	repos := repository.RepositorySet{
		Users:                   userRepo,
		Students:                studentRepo,
		AcademicYears:           academicYearRepo,
		Classes:                 classRepo,
		TeacherClassAssignments: assignmentRepo,
		StudentClassHistories:   historyRepo,
	}

	svc, err := service.NewUserService(service.Dependencies{
		Repositories: repos,
		UOW:          postgres.NewUnitOfWorkManager(db),
	})
	if err != nil {
		t.Fatalf("NewUserService() error = %v", err)
	}

	admin := newServiceScopeUser(t, domain.RoleAdmin)
	wali := newServiceScopeUser(t, domain.RoleWaliKelas)
	studentUser := newServiceScopeUser(t, domain.RoleSiswa)
	otherUser := newServiceScopeUser(t, domain.RoleSiswa)

	for _, user := range []domain.User{admin, wali, studentUser, otherUser} {
		if err := userRepo.Create(ctx, user); err != nil {
			t.Fatalf("create user %s: %v", user.Username, err)
		}
	}

	academicYear := newServiceScopeAcademicYear(t, "2026/2027")
	if err := academicYearRepo.Create(ctx, academicYear); err != nil {
		t.Fatalf("create academic year: %v", err)
	}

	classA := newServiceScopeClass(t, "7A", 7)
	classB := newServiceScopeClass(t, "7B", 7)

	if err := classRepo.Create(ctx, classA); err != nil {
		t.Fatalf("create class A: %v", err)
	}

	if err := classRepo.Create(ctx, classB); err != nil {
		t.Fatalf("create class B: %v", err)
	}

	assignment := newServiceScopeAssignment(
		t,
		wali.ID,
		academicYear.ID,
		classA.ID,
	)

	if err := assignmentRepo.Create(ctx, assignment); err != nil {
		t.Fatalf("create assignment: %v", err)
	}

	student := newServiceScopeStudent(t, "Student Scope")
	student.UserID = &studentUser.ID

	if err := studentRepo.Create(ctx, student); err != nil {
		t.Fatalf("create student: %v", err)
	}

	history := newServiceScopeStudentHistory(
		t,
		student.ID,
		academicYear.ID,
		classA.ID,
	)

	if err := historyRepo.Create(ctx, history); err != nil {
		t.Fatalf("create student history: %v", err)
	}

	t.Run("admin can get any user", func(t *testing.T) {
		got, err := svc.Get(ctx, service.GetUserInput{
			Actor: service.Actor{
				UserID: admin.ID,
				Role:   domain.RoleAdmin,
			},
			UserID:         otherUser.ID,
			AcademicYearID: academicYear.ID,
		})
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}

		if got.User == nil {
			t.Fatal("Get() returned nil user")
		}

		if got.User.ID != otherUser.ID {
			t.Fatalf("User.ID = %v, want %v", got.User.ID, otherUser.ID)
		}
	})

	t.Run("siswa can get self", func(t *testing.T) {
		got, err := svc.Get(ctx, service.GetUserInput{
			Actor: service.Actor{
				UserID: studentUser.ID,
				Role:   domain.RoleSiswa,
			},
			UserID:         studentUser.ID,
			AcademicYearID: academicYear.ID,
		})
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}

		if got.User == nil || got.User.ID != studentUser.ID {
			t.Fatalf("Get() returned unexpected user: %#v", got.User)
		}
	})

	t.Run("siswa cannot get another user", func(t *testing.T) {
		_, err := svc.Get(ctx, service.GetUserInput{
			Actor: service.Actor{
				UserID: studentUser.ID,
				Role:   domain.RoleSiswa,
			},
			UserID:         otherUser.ID,
			AcademicYearID: academicYear.ID,
		})
		if !errors.Is(err, service.ErrScopeViolation) {
			t.Fatalf("Get() error = %v, want ErrScopeViolation", err)
		}
	})

	t.Run("wali kelas can get student in scope", func(t *testing.T) {
		got, err := svc.Get(ctx, service.GetUserInput{
			Actor: service.Actor{
				UserID: wali.ID,
				Role:   domain.RoleWaliKelas,
			},
			UserID:         studentUser.ID,
			AcademicYearID: academicYear.ID,
		})
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}

		if got.User == nil || got.User.ID != studentUser.ID {
			t.Fatalf("Get() returned unexpected user: %#v", got.User)
		}
	})

	t.Run("wali kelas cannot get student outside scope", func(t *testing.T) {
		otherStudentUser := newServiceScopeUser(t, domain.RoleSiswa)

		if err := userRepo.Create(ctx, otherStudentUser); err != nil {
			t.Fatalf("create other student user: %v", err)
		}

		otherStudent := newServiceScopeStudent(t, "Student Outside")
		otherStudent.UserID = &otherStudentUser.ID

		if err := studentRepo.Create(ctx, otherStudent); err != nil {
			t.Fatalf("create other student: %v", err)
		}

		otherHistory := newServiceScopeStudentHistory(
			t,
			otherStudent.ID,
			academicYear.ID,
			classB.ID,
		)

		if err := historyRepo.Create(ctx, otherHistory); err != nil {
			t.Fatalf("create other student history: %v", err)
		}

		_, err := svc.Get(ctx, service.GetUserInput{
			Actor: service.Actor{
				UserID: wali.ID,
				Role:   domain.RoleWaliKelas,
			},
			UserID:         otherStudentUser.ID,
			AcademicYearID: academicYear.ID,
		})
		if !errors.Is(err, service.ErrScopeViolation) {
			t.Fatalf("Get() error = %v, want ErrScopeViolation", err)
		}
	})
}

func TestUserServiceIntegration_GetByUsername(t *testing.T) {
	db := openServiceScopeTestDatabase(t)
	ctx := context.Background()

	cleanupServiceScopeFixtures(t, db)

	userRepo := postgres.NewUserRepository(db)

	repos := repository.RepositorySet{
		Users: userRepo,
	}

	svc, err := service.NewUserService(service.Dependencies{
		Repositories: repos,
		UOW:          postgres.NewUnitOfWorkManager(db),
	})
	if err != nil {
		t.Fatalf("NewUserService() error = %v", err)
	}

	target := newServiceScopeUser(t, domain.RoleSiswa)

	if err := userRepo.Create(ctx, target); err != nil {
		t.Fatalf("create target user: %v", err)
	}

	t.Run("admin can get by username", func(t *testing.T) {
		got, err := svc.GetByUsername(ctx, service.GetUserByUsernameInput{
			Actor: service.Actor{
				UserID: uuid.New(),
				Role:   domain.RoleAdmin,
			},
			Username: target.Username,
		})
		if err != nil {
			t.Fatalf("GetByUsername() error = %v", err)
		}

		if got.User == nil || got.User.ID != target.ID {
			t.Fatalf("GetByUsername() returned unexpected user: %#v", got.User)
		}
	})

	t.Run("wali kelas is forbidden", func(t *testing.T) {
		_, err := svc.GetByUsername(ctx, service.GetUserByUsernameInput{
			Actor: service.Actor{
				UserID: uuid.New(),
				Role:   domain.RoleWaliKelas,
			},
			Username: target.Username,
		})
		if !errors.Is(err, service.ErrForbidden) {
			t.Fatalf("GetByUsername() error = %v, want ErrForbidden", err)
		}
	})
}

func TestUserServiceIntegration_Create(t *testing.T) {
	db := openServiceScopeTestDatabase(t)
	ctx := context.Background()

	cleanupServiceScopeFixtures(t, db)

	userRepo := postgres.NewUserRepository(db)

	repos := repository.RepositorySet{
		Users: userRepo,
	}

	svc, err := service.NewUserService(service.Dependencies{
		Repositories: repos,
		UOW:          postgres.NewUnitOfWorkManager(db),
	})
	if err != nil {
		t.Fatalf("NewUserService() error = %v", err)
	}

	actor := newServiceScopeUser(t, domain.RoleAdmin)

	if err := userRepo.Create(ctx, actor); err != nil {
		t.Fatalf("create actor: %v", err)
	}

	username := "integration-create-" + uuid.NewString()

	got, err := svc.Create(ctx, service.CreateUserInput{
		Actor: service.Actor{
			UserID: actor.ID,
			Role:   domain.RoleAdmin,
		},
		Username:     username,
		PasswordHash: "integration-password-hash",
		Role:         domain.RoleSiswa,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if got.User == nil {
		t.Fatal("Create() returned nil user")
	}

	if got.User.Username != username {
		t.Fatalf("Username = %q, want %q", got.User.Username, username)
	}

	persisted, err := userRepo.GetByID(ctx, got.User.ID)
	if err != nil {
		t.Fatalf("GetByID() after Create() error = %v", err)
	}

	if persisted.Username != username {
		t.Fatalf("persisted Username = %q, want %q", persisted.Username, username)
	}

	if persisted.Role != domain.RoleSiswa {
		t.Fatalf("persisted Role = %q, want %q", persisted.Role, domain.RoleSiswa)
	}
}

func TestUserServiceIntegration_Update(t *testing.T) {
	db := openServiceScopeTestDatabase(t)
	ctx := context.Background()

	cleanupServiceScopeFixtures(t, db)

	userRepo := postgres.NewUserRepository(db)

	repos := repository.RepositorySet{
		Users: userRepo,
	}

	svc, err := service.NewUserService(service.Dependencies{
		Repositories: repos,
		UOW:          postgres.NewUnitOfWorkManager(db),
	})
	if err != nil {
		t.Fatalf("NewUserService() error = %v", err)
	}

	actor := newServiceScopeUser(t, domain.RoleAdmin)
	target := newServiceScopeUser(t, domain.RoleSiswa)

	if err := userRepo.Create(ctx, actor); err != nil {
		t.Fatalf("create actor: %v", err)
	}

	if err := userRepo.Create(ctx, target); err != nil {
		t.Fatalf("create target: %v", err)
	}

	originalUpdatedAt := target.UpdatedAt
	time.Sleep(2 * time.Millisecond)

	newUsername := "integration-update-" + uuid.NewString()

	got, err := svc.Update(ctx, service.UpdateUserInput{
		Actor: service.Actor{
			UserID: actor.ID,
			Role:   domain.RoleAdmin,
		},
		UserID:       target.ID,
		Username:     newUsername,
		PasswordHash: "updated-password-hash",
		Role:         domain.RoleSiswa,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if got.User == nil {
		t.Fatal("Update() returned nil user")
	}

	if got.User.Username != newUsername {
		t.Fatalf("Username = %q, want %q", got.User.Username, newUsername)
	}

	if got.User.Role != domain.RoleSiswa {
		t.Fatalf("Role = %q, want %q", got.User.Role, domain.RoleSiswa)
	}

	if !got.User.UpdatedAt.After(originalUpdatedAt) {
		t.Fatalf(
			"UpdatedAt = %v, want after %v",
			got.User.UpdatedAt,
			originalUpdatedAt,
		)
	}

	persisted, err := userRepo.GetByID(ctx, target.ID)
	if err != nil {
		t.Fatalf("GetByID() after Update() error = %v", err)
	}

	if persisted.Username != newUsername {
		t.Fatalf("persisted Username = %q, want %q", persisted.Username, newUsername)
	}

	if persisted.PasswordHash != "updated-password-hash" {
		t.Fatalf(
			"persisted PasswordHash = %q, want %q",
			persisted.PasswordHash,
			"updated-password-hash",
		)
	}

	if persisted.Role != domain.RoleSiswa {
		t.Fatalf("persisted Role = %q, want %q", persisted.Role, domain.RoleSiswa)
	}

	if !persisted.UpdatedAt.After(originalUpdatedAt) {
		t.Fatalf(
			"persisted UpdatedAt = %v, want after %v",
			persisted.UpdatedAt,
			originalUpdatedAt,
		)
	}
}

func TestUserServiceIntegration_ResetAndDeactivateRevokeSessions(t *testing.T) {
	db := openServiceScopeTestDatabase(t)
	ctx := context.Background()
	cleanupServiceScopeFixtures(t, db)
	userRepo := postgres.NewUserRepository(db)
	sessionRepo := postgres.NewSessionRepository(db)
	admin, target := newServiceScopeUser(t, domain.RoleAdmin), newServiceScopeUser(t, domain.RoleWaliKelas)
	for _, user := range []domain.User{admin, target} {
		if err := userRepo.Create(ctx, user); err != nil {
			t.Fatalf("create user: %v", err)
		}
	}
	repos := repository.RepositorySet{Users: userRepo, Sessions: sessionRepo}
	svc, err := service.NewUserService(service.Dependencies{Repositories: repos, UOW: postgres.NewUnitOfWorkManager(db)})
	if err != nil {
		t.Fatal(err)
	}
	createSession := func(token string) domain.Session {
		s, err := domain.NewSession(target.ID, token, time.Now().Add(time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		if err := sessionRepo.Create(ctx, s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	first := createSession("reset-" + uuid.NewString())
	_, err = svc.Update(ctx, service.UpdateUserInput{Actor: service.Actor{UserID: admin.ID, Role: domain.RoleAdmin}, UserID: target.ID, Username: target.Username, Role: target.Role, PasswordHash: "reset-hash", PasswordChangeRequired: true, RevokeSessions: true})
	if err != nil {
		t.Fatalf("reset update: %v", err)
	}
	got, err := sessionRepo.GetByTokenHash(ctx, first.TokenHash)
	if err != nil || got.RevokedAt == nil {
		t.Fatalf("session after reset = %#v, err=%v; want revoked", got, err)
	}
	second := createSession("deactivate-" + uuid.NewString())
	if err := svc.SetActive(ctx, service.SetUserActiveInput{Actor: service.Actor{UserID: admin.ID, Role: domain.RoleAdmin}, UserID: target.ID, Active: false}); err != nil {
		t.Fatalf("deactivate: %v", err)
	}
	updated, err := userRepo.GetByID(ctx, target.ID)
	if err != nil || updated.Active {
		t.Fatalf("user active=%v, err=%v; want inactive", updated.Active, err)
	}
	got, err = sessionRepo.GetByTokenHash(ctx, second.TokenHash)
	if err != nil || got.RevokedAt == nil {
		t.Fatalf("session after deactivation = %#v, err=%v; want revoked", got, err)
	}
}

func TestUserServiceIntegration_RollsBackPasswordAndDeactivationWhenSessionRevokeFails(t *testing.T) {
	db := openServiceScopeTestDatabase(t)
	ctx := context.Background()
	cleanupServiceScopeFixtures(t, db)
	userRepo := postgres.NewUserRepository(db)
	sessionRepo := postgres.NewSessionRepository(db)
	admin, target := newServiceScopeUser(t, domain.RoleAdmin), newServiceScopeUser(t, domain.RoleWaliKelas)
	for _, user := range []domain.User{admin, target} {
		if err := userRepo.Create(ctx, user); err != nil {
			t.Fatalf("create user: %v", err)
		}
	}
	session, err := domain.NewSession(target.ID, "rollback-"+uuid.NewString(), time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if err := sessionRepo.Create(ctx, session); err != nil {
		t.Fatal(err)
	}
	identifier := strings.ReplaceAll(target.ID.String(), "-", "")
	functionName := "codex_f24_fail_revoke_" + identifier
	triggerName := "codex_f24_fail_revoke_" + identifier
	_, err = db.ExecContext(ctx, fmt.Sprintf(`CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'forced FASE24 test failure'; END $$`, functionName))
	if err != nil {
		t.Fatalf("create failure function: %v", err)
	}
	_, err = db.ExecContext(ctx, fmt.Sprintf(`CREATE TRIGGER %s BEFORE UPDATE OF revoked_at ON sessions FOR EACH ROW WHEN (OLD.user_id = '%s'::uuid AND OLD.revoked_at IS NULL) EXECUTE FUNCTION %s()`, triggerName, target.ID, functionName))
	if err != nil {
		t.Fatalf("create failure trigger: %v", err)
	}
	t.Cleanup(func() {
		if _, err := db.ExecContext(context.Background(), fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON sessions", triggerName)); err != nil {
			t.Errorf("drop failure trigger: %v", err)
		}
		if _, err := db.ExecContext(context.Background(), fmt.Sprintf("DROP FUNCTION IF EXISTS %s()", functionName)); err != nil {
			t.Errorf("drop failure function: %v", err)
		}
	})
	svc, err := service.NewUserService(service.Dependencies{Repositories: repository.RepositorySet{Users: userRepo, Sessions: sessionRepo}, UOW: postgres.NewUnitOfWorkManager(db)})
	if err != nil {
		t.Fatal(err)
	}
	actor := service.Actor{UserID: admin.ID, Role: domain.RoleAdmin}
	_, err = svc.Update(ctx, service.UpdateUserInput{Actor: actor, UserID: target.ID, Username: target.Username, Role: target.Role, PasswordHash: "must-roll-back", PasswordChangeRequired: true, RevokeSessions: true})
	if err == nil {
		t.Fatal("password reset succeeded despite session revocation failure")
	}
	gotUser, err := userRepo.GetByID(ctx, target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotUser.PasswordHash != target.PasswordHash || gotUser.PasswordChangeRequired {
		t.Fatalf("password reset persisted despite rollback: %#v", gotUser)
	}
	err = svc.SetActive(ctx, service.SetUserActiveInput{Actor: actor, UserID: target.ID, Active: false})
	if err == nil {
		t.Fatal("deactivation succeeded despite session revocation failure")
	}
	gotUser, err = userRepo.GetByID(ctx, target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !gotUser.Active {
		t.Fatal("deactivation persisted despite rollback")
	}
	gotSession, err := sessionRepo.GetByTokenHash(ctx, session.TokenHash)
	if err != nil {
		t.Fatal(err)
	}
	if gotSession.RevokedAt != nil {
		t.Fatal("session revocation persisted despite transaction rollback")
	}
}
