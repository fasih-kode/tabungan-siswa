package service_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/platform/database"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/repository/postgres"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

func TestRequireClassScopeIntegration(t *testing.T) {
	db := openServiceScopeTestDatabase(t)
	ctx := context.Background()

	cleanupServiceScopeFixtures(t, db)

	userRepo := postgres.NewUserRepository(db)
	academicYearRepo := postgres.NewAcademicYearRepository(db)
	classRepo := postgres.NewClassRepository(db)
	assignmentRepo := postgres.NewTeacherClassAssignmentRepository(db)

	waliA := newServiceScopeUser(t, domain.RoleWaliKelas)
	waliB := newServiceScopeUser(t, domain.RoleWaliKelas)

	if err := userRepo.Create(ctx, waliA); err != nil {
		t.Fatalf("create wali A: %v", err)
	}

	if err := userRepo.Create(ctx, waliB); err != nil {
		t.Fatalf("create wali B: %v", err)
	}

	academicYear := newServiceScopeAcademicYear(t, "2026/2027")
	if err := academicYearRepo.Create(ctx, academicYear); err != nil {
		t.Fatalf("create academic year: %v", err)
	}

	classA := newServiceScopeClass(t, "7A", 7)
	classB := newServiceScopeClass(t, "7B", 7)
	classC := newServiceScopeClass(t, "7C", 7)

	if err := classRepo.Create(ctx, classA); err != nil {
		t.Fatalf("create class A: %v", err)
	}

	if err := classRepo.Create(ctx, classB); err != nil {
		t.Fatalf("create class B: %v", err)
	}

	if err := classRepo.Create(ctx, classC); err != nil {
		t.Fatalf("create class C: %v", err)
	}

	assignmentA1 := newServiceScopeAssignment(
		t,
		waliA.ID,
		academicYear.ID,
		classA.ID,
	)

	assignmentA2 := newServiceScopeAssignment(
		t,
		waliA.ID,
		academicYear.ID,
		classB.ID,
	)

	assignmentB := newServiceScopeAssignment(
		t,
		waliB.ID,
		academicYear.ID,
		classC.ID,
	)

	if err := assignmentRepo.Create(ctx, assignmentA1); err != nil {
		t.Fatalf("create assignment A1: %v", err)
	}

	if err := assignmentRepo.Create(ctx, assignmentA2); err != nil {
		t.Fatalf("create assignment A2: %v", err)
	}

	if err := assignmentRepo.Create(ctx, assignmentB); err != nil {
		t.Fatalf("create assignment B: %v", err)
	}

	admin := service.Actor{
		UserID: uuid.New(),
		Role:   domain.RoleAdmin,
	}

	waliActorA := service.Actor{
		UserID: waliA.ID,
		Role:   domain.RoleWaliKelas,
	}

	t.Run("admin can access class A", func(t *testing.T) {
		err := service.RequireClassScope(
			ctx,
			serviceScopeRepositories(db),
			admin,
			classA.ID,
			academicYear.ID,
		)

		if err != nil {
			t.Fatalf("RequireClassScope() error = %v, want nil", err)
		}
	})

	t.Run("admin can access class B", func(t *testing.T) {
		err := service.RequireClassScope(
			ctx,
			serviceScopeRepositories(db),
			admin,
			classB.ID,
			academicYear.ID,
		)

		if err != nil {
			t.Fatalf("RequireClassScope() error = %v, want nil", err)
		}
	})

	t.Run("wali A can access class A", func(t *testing.T) {
		err := service.RequireClassScope(
			ctx,
			serviceScopeRepositories(db),
			waliActorA,
			classA.ID,
			academicYear.ID,
		)

		if err != nil {
			t.Fatalf("RequireClassScope() error = %v, want nil", err)
		}
	})

	t.Run("wali A can access class B", func(t *testing.T) {
		err := service.RequireClassScope(
			ctx,
			serviceScopeRepositories(db),
			waliActorA,
			classB.ID,
			academicYear.ID,
		)

		if err != nil {
			t.Fatalf("RequireClassScope() error = %v, want nil", err)
		}
	})

	t.Run("wali A cannot access class C", func(t *testing.T) {
		err := service.RequireClassScope(
			ctx,
			serviceScopeRepositories(db),
			waliActorA,
			classC.ID,
			academicYear.ID,
		)

		if !errors.Is(err, service.ErrScopeViolation) {
			t.Fatalf(
				"RequireClassScope() error = %v, want ErrScopeViolation",
				err,
			)
		}
	})

	t.Run("wali A cannot access class A in another academic year", func(t *testing.T) {
		otherYear := newServiceScopeAcademicYear(t, "2027/2028")

		if err := otherYear.StartClosing(); err != nil {
			t.Fatalf("start closing other academic year: %v", err)
		}

		if err := otherYear.Close(); err != nil {
			t.Fatalf("close other academic year: %v", err)
		}

		if err := academicYearRepo.Create(ctx, otherYear); err != nil {
			t.Fatalf("create other academic year: %v", err)
		}

		err := service.RequireClassScope(
			ctx,
			serviceScopeRepositories(db),
			waliActorA,
			classA.ID,
			otherYear.ID,
		)

		if !errors.Is(err, service.ErrScopeViolation) {
			t.Fatalf(
				"RequireClassScope() error = %v, want ErrScopeViolation",
				err,
			)
		}

		if _, err := db.ExecContext(
			ctx,
			"DELETE FROM academic_years WHERE id = $1",
			otherYear.ID,
		); err != nil {
			t.Fatalf("cleanup other academic year: %v", err)
		}
	})
}

func TestRequireStudentScopeIntegration(t *testing.T) {
	db := openServiceScopeTestDatabase(t)
	ctx := context.Background()

	cleanupServiceScopeFixtures(t, db)

	userRepo := postgres.NewUserRepository(db)
	academicYearRepo := postgres.NewAcademicYearRepository(db)
	classRepo := postgres.NewClassRepository(db)
	studentRepo := postgres.NewStudentRepository(db)
	assignmentRepo := postgres.NewTeacherClassAssignmentRepository(db)
	historyRepo := postgres.NewStudentClassHistoryRepository(db)

	waliA := newServiceScopeUser(t, domain.RoleWaliKelas)
	waliB := newServiceScopeUser(t, domain.RoleWaliKelas)

	if err := userRepo.Create(ctx, waliA); err != nil {
		t.Fatalf("create wali A: %v", err)
	}

	if err := userRepo.Create(ctx, waliB); err != nil {
		t.Fatalf("create wali B: %v", err)
	}

	academicYear := newServiceScopeAcademicYear(t, "2026/2027")
	if err := academicYearRepo.Create(ctx, academicYear); err != nil {
		t.Fatalf("create academic year: %v", err)
	}

	classA := newServiceScopeClass(t, "8A", 8)
	classB := newServiceScopeClass(t, "8B", 8)

	if err := classRepo.Create(ctx, classA); err != nil {
		t.Fatalf("create class A: %v", err)
	}

	if err := classRepo.Create(ctx, classB); err != nil {
		t.Fatalf("create class B: %v", err)
	}

	assignmentA := newServiceScopeAssignment(
		t,
		waliA.ID,
		academicYear.ID,
		classA.ID,
	)

	assignmentB := newServiceScopeAssignment(
		t,
		waliB.ID,
		academicYear.ID,
		classB.ID,
	)

	if err := assignmentRepo.Create(ctx, assignmentA); err != nil {
		t.Fatalf("create assignment A: %v", err)
	}

	if err := assignmentRepo.Create(ctx, assignmentB); err != nil {
		t.Fatalf("create assignment B: %v", err)
	}

	studentA := newServiceScopeStudent(t, "Student A")
	studentB := newServiceScopeStudent(t, "Student B")

	if err := studentRepo.Create(ctx, studentA); err != nil {
		t.Fatalf("create student A: %v", err)
	}

	if err := studentRepo.Create(ctx, studentB); err != nil {
		t.Fatalf("create student B: %v", err)
	}

	historyA := newServiceScopeStudentHistory(
		t,
		studentA.ID,
		academicYear.ID,
		classA.ID,
	)

	historyB := newServiceScopeStudentHistory(
		t,
		studentB.ID,
		academicYear.ID,
		classB.ID,
	)

	if err := historyRepo.Create(ctx, historyA); err != nil {
		t.Fatalf("create history A: %v", err)
	}

	if err := historyRepo.Create(ctx, historyB); err != nil {
		t.Fatalf("create history B: %v", err)
	}

	admin := service.Actor{
		UserID: uuid.New(),
		Role:   domain.RoleAdmin,
	}

	waliActorA := service.Actor{
		UserID: waliA.ID,
		Role:   domain.RoleWaliKelas,
	}

	t.Run("admin can access student A", func(t *testing.T) {
		err := service.RequireStudentScope(
			ctx,
			serviceScopeRepositories(db),
			admin,
			studentA.ID,
			academicYear.ID,
		)

		if err != nil {
			t.Fatalf("RequireStudentScope() error = %v, want nil", err)
		}
	})

	t.Run("admin can access student B", func(t *testing.T) {
		err := service.RequireStudentScope(
			ctx,
			serviceScopeRepositories(db),
			admin,
			studentB.ID,
			academicYear.ID,
		)

		if err != nil {
			t.Fatalf("RequireStudentScope() error = %v, want nil", err)
		}
	})

	t.Run("wali A can access student A", func(t *testing.T) {
		err := service.RequireStudentScope(
			ctx,
			serviceScopeRepositories(db),
			waliActorA,
			studentA.ID,
			academicYear.ID,
		)

		if err != nil {
			t.Fatalf("RequireStudentScope() error = %v, want nil", err)
		}
	})

	t.Run("wali A cannot access student B", func(t *testing.T) {
		err := service.RequireStudentScope(
			ctx,
			serviceScopeRepositories(db),
			waliActorA,
			studentB.ID,
			academicYear.ID,
		)

		if !errors.Is(err, service.ErrScopeViolation) {
			t.Fatalf(
				"RequireStudentScope() error = %v, want ErrScopeViolation",
				err,
			)
		}
	})
}

func openServiceScopeTestDatabase(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Fatal("DATABASE_URL is not set")
	}

	db, err := database.Open(dsn)
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}

	if err := database.Ping(context.Background(), db); err != nil {
		_ = db.Close()
		t.Fatalf("database.Ping() error = %v", err)
	}

	t.Cleanup(func() {
		if err := database.Close(db); err != nil {
			t.Errorf("database.Close() error = %v", err)
		}
	})

	return db
}

func serviceScopeRepositories(db *sql.DB) repository.RepositorySet {
	return repository.RepositorySet{
		Classes:                 postgres.NewClassRepository(db),
		Students:                postgres.NewStudentRepository(db),
		StudentClassHistories:   postgres.NewStudentClassHistoryRepository(db),
		TeacherClassAssignments: postgres.NewTeacherClassAssignmentRepository(db),
	}
}

func cleanupServiceScopeFixtures(t *testing.T, db *sql.DB) {
	t.Helper()

	statements := []string{
		"DELETE FROM transactions",
		"DELETE FROM savings_settlements",
		"DELETE FROM savings_accounts",
		"DELETE FROM class_transfer_requests",
		"DELETE FROM student_class_histories",
		"DELETE FROM teacher_class_assignments",
		"DELETE FROM audit_logs",
		"DELETE FROM students",
		"DELETE FROM classes",
		"DELETE FROM academic_years",
		"DELETE FROM users",
	}

	ctx := context.Background()

	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatalf("cleanup %q: %v", statement, err)
		}
	}
}

func newServiceScopeUser(
	t *testing.T,
	role domain.UserRole,
) domain.User {
	t.Helper()

	user, err := domain.NewUser(
		"scope-"+uuid.NewString(),
		"test-password-hash",
		role,
	)
	if err != nil {
		t.Fatalf("NewUser() error = %v", err)
	}

	return user
}

func newServiceScopeAcademicYear(
	t *testing.T,
	name string,
) domain.AcademicYear {
	t.Helper()

	academicYear, err := domain.NewAcademicYear(
		name,
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("NewAcademicYear() error = %v", err)
	}

	return academicYear
}

func newServiceScopeClass(
	t *testing.T,
	name string,
	level int,
) domain.Class {
	t.Helper()

	class, err := domain.NewClass(name, level)
	if err != nil {
		t.Fatalf("NewClass() error = %v", err)
	}

	return class
}

func newServiceScopeStudent(
	t *testing.T,
	name string,
) domain.Student {
	t.Helper()

	nis := "NIS-" + uuid.NewString()
	nisn := "NISN-" + uuid.NewString()
	student, err := domain.NewStudent(
		name,
		nil,
		&nis,
		&nisn,
	)
	if err != nil {
		t.Fatalf("NewStudent() error = %v", err)
	}

	return student
}

func newServiceScopeAssignment(
	t *testing.T,
	userID uuid.UUID,
	academicYearID uuid.UUID,
	classID uuid.UUID,
) domain.TeacherClassAssignment {
	t.Helper()

	from := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC)

	period, err := domain.NewDateRange(from, &to)
	if err != nil {
		t.Fatalf("NewDateRange() error = %v", err)
	}

	assignment, err := domain.NewTeacherClassAssignment(
		userID,
		academicYearID,
		classID,
		period,
	)
	if err != nil {
		t.Fatalf("NewTeacherClassAssignment() error = %v", err)
	}

	return assignment
}

func newServiceScopeStudentHistory(
	t *testing.T,
	studentID uuid.UUID,
	academicYearID uuid.UUID,
	classID uuid.UUID,
) domain.StudentClassHistory {
	t.Helper()

	from := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC)

	period, err := domain.NewDateRange(from, &to)
	if err != nil {
		t.Fatalf("NewDateRange() error = %v", err)
	}

	history, err := domain.NewStudentClassHistory(
		studentID,
		academicYearID,
		classID,
		period,
	)
	if err != nil {
		t.Fatalf("NewStudentClassHistory() error = %v", err)
	}

	return history
}
