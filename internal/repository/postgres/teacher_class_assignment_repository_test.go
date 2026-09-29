package postgres_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/repository/postgres"
	"github.com/google/uuid"
)

func TestTeacherClassAssignmentRepository_CreateAndGetByID(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewTeacherClassAssignmentRepository(db)
	ctx := context.Background()

	fixture := newTeacherClassAssignmentFixture(t, db)

	assignment := newTestTeacherClassAssignment(
		t,
		fixture.user.ID,
		fixture.academicYear.ID,
		fixture.class.ID,
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC),
	)

	if err := repo.Create(ctx, assignment); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestTeacherClassAssignment(t, db, assignment.ID)
	})

	got, err := repo.GetByID(ctx, assignment.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	assertTeacherClassAssignmentEqual(t, got, assignment)
}

func TestTeacherClassAssignmentRepository_GetByIDNotFound(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewTeacherClassAssignmentRepository(db)

	_, err := repo.GetByID(context.Background(), uuid.New())
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("GetByID() error = %v, want ErrNotFound", err)
	}
}

func TestTeacherClassAssignmentRepository_ListByUserAndAcademicYear(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewTeacherClassAssignmentRepository(db)
	ctx := context.Background()

	fixture := newTeacherClassAssignmentFixture(t, db)
	secondClass, err := domain.NewClass("8A", 8)
	if err != nil {
		t.Fatalf("NewClass() error = %v", err)
	}

	if err := postgres.NewClassRepository(db).Create(ctx, secondClass); err != nil {
		t.Fatalf("create second class error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestClass(t, db, secondClass.ID)
	})

	first := newTestTeacherClassAssignment(
		t,
		fixture.user.ID,
		fixture.academicYear.ID,
		fixture.class.ID,
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
	)

	second := newTestTeacherClassAssignment(
		t,
		fixture.user.ID,
		fixture.academicYear.ID,
		secondClass.ID,
		time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
	)

	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}

	if err := repo.Create(ctx, second); err != nil {
		t.Fatalf("Create(second) error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestTeacherClassAssignment(t, db, first.ID)
		deleteTestTeacherClassAssignment(t, db, second.ID)
	})

	got, err := repo.ListByUserAndAcademicYear(
		ctx,
		fixture.user.ID,
		fixture.academicYear.ID,
		repository.ListOptions{
			Limit:  10,
			Offset: 0,
		},
	)
	if err != nil {
		t.Fatalf("ListByUserAndAcademicYear() error = %v", err)
	}

	if got == nil {
		t.Fatal("ListByUserAndAcademicYear() returned nil slice, want empty/non-nil slice")
	}

	if len(got) != 2 {
		t.Fatalf(
			"ListByUserAndAcademicYear() length = %d, want 2",
			len(got),
		)
	}

	assertTeacherClassAssignmentEqual(t, got[0], first)
	assertTeacherClassAssignmentEqual(t, got[1], second)
}

func TestTeacherClassAssignmentRepository_ListByUserAndAcademicYearEmpty(
	t *testing.T,
) {
	db := openTestDatabase(t)
	repo := postgres.NewTeacherClassAssignmentRepository(db)

	fixture := newTeacherClassAssignmentFixture(t, db)

	got, err := repo.ListByUserAndAcademicYear(
		context.Background(),
		fixture.user.ID,
		fixture.academicYear.ID,
		repository.ListOptions{
			Limit:  10,
			Offset: 0,
		},
	)
	if err != nil {
		t.Fatalf("ListByUserAndAcademicYear() error = %v", err)
	}

	if got == nil {
		t.Fatal("ListByUserAndAcademicYear() returned nil slice, want empty slice")
	}

	if len(got) != 0 {
		t.Fatalf(
			"ListByUserAndAcademicYear() length = %d, want 0",
			len(got),
		)
	}
}

func TestTeacherClassAssignmentRepository_ListByClassAndAcademicYear(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewTeacherClassAssignmentRepository(db)
	ctx := context.Background()

	fixture := newTeacherClassAssignmentFixture(t, db)

	secondUser := newTestUser(t)
	if err := postgres.NewUserRepository(db).Create(ctx, secondUser); err != nil {
		t.Fatalf("create second user error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestUser(t, db, secondUser.ID)
	})

	first := newTestTeacherClassAssignment(
		t,
		fixture.user.ID,
		fixture.academicYear.ID,
		fixture.class.ID,
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
	)

	second := newTestTeacherClassAssignment(
		t,
		secondUser.ID,
		fixture.academicYear.ID,
		fixture.class.ID,
		time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
	)

	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}

	if err := repo.Create(ctx, second); err != nil {
		t.Fatalf("Create(second) error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestTeacherClassAssignment(t, db, first.ID)
		deleteTestTeacherClassAssignment(t, db, second.ID)
	})

	got, err := repo.ListByClassAndAcademicYear(
		ctx,
		fixture.class.ID,
		fixture.academicYear.ID,
		repository.ListOptions{
			Limit:  10,
			Offset: 0,
		},
	)
	if err != nil {
		t.Fatalf("ListByClassAndAcademicYear() error = %v", err)
	}

	if got == nil {
		t.Fatal("ListByClassAndAcademicYear() returned nil slice, want empty slice")
	}

	if len(got) != 2 {
		t.Fatalf(
			"ListByClassAndAcademicYear() length = %d, want 2",
			len(got),
		)
	}

	assertTeacherClassAssignmentEqual(t, got[0], first)
	assertTeacherClassAssignmentEqual(t, got[1], second)
}

func TestTeacherClassAssignmentRepository_ListByClassAndAcademicYearEmpty(
	t *testing.T,
) {
	db := openTestDatabase(t)
	repo := postgres.NewTeacherClassAssignmentRepository(db)

	fixture := newTeacherClassAssignmentFixture(t, db)

	got, err := repo.ListByClassAndAcademicYear(
		context.Background(),
		fixture.class.ID,
		fixture.academicYear.ID,
		repository.ListOptions{
			Limit:  10,
			Offset: 0,
		},
	)
	if err != nil {
		t.Fatalf("ListByClassAndAcademicYear() error = %v", err)
	}

	if got == nil {
		t.Fatal("ListByClassAndAcademicYear() returned nil slice, want empty slice")
	}

	if len(got) != 0 {
		t.Fatalf(
			"ListByClassAndAcademicYear() length = %d, want 0",
			len(got),
		)
	}
}

func TestTeacherClassAssignmentRepository_ListPagination(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewTeacherClassAssignmentRepository(db)
	ctx := context.Background()

	fixture := newTeacherClassAssignmentFixture(t, db)

	secondClass, err := domain.NewClass("8A", 8)
	if err != nil {
		t.Fatalf("NewClass() error = %v", err)
	}

	if err := postgres.NewClassRepository(db).Create(ctx, secondClass); err != nil {
		t.Fatalf("create second class error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestClass(t, db, secondClass.ID)
	})

	thirdClass, err := domain.NewClass("9A", 9)
	if err != nil {
		t.Fatalf("NewClass() error = %v", err)
	}

	if err := postgres.NewClassRepository(db).Create(ctx, thirdClass); err != nil {
		t.Fatalf("create third class error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestClass(t, db, thirdClass.ID)
	})

	first := newTestTeacherClassAssignment(
		t,
		fixture.user.ID,
		fixture.academicYear.ID,
		fixture.class.ID,
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
	)

	second := newTestTeacherClassAssignment(
		t,
		fixture.user.ID,
		fixture.academicYear.ID,
		secondClass.ID,
		time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
	)

	third := newTestTeacherClassAssignment(
		t,
		fixture.user.ID,
		fixture.academicYear.ID,
		thirdClass.ID,
		time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC),
	)

	for _, assignment := range []domain.TeacherClassAssignment{
		first,
		second,
		third,
	} {
		if err := repo.Create(ctx, assignment); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	t.Cleanup(func() {
		deleteTestTeacherClassAssignment(t, db, first.ID)
		deleteTestTeacherClassAssignment(t, db, second.ID)
		deleteTestTeacherClassAssignment(t, db, third.ID)
	})

	got, err := repo.ListByUserAndAcademicYear(
		ctx,
		fixture.user.ID,
		fixture.academicYear.ID,
		repository.ListOptions{
			Limit:  1,
			Offset: 1,
		},
	)
	if err != nil {
		t.Fatalf("ListByUserAndAcademicYear() error = %v", err)
	}

	if len(got) != 1 {
		t.Fatalf(
			"ListByUserAndAcademicYear() length = %d, want 1",
			len(got),
		)
	}

	assertTeacherClassAssignmentEqual(t, got[0], second)
}

func TestTeacherClassAssignmentRepository_GetCurrentByClassAndAcademicYear(
	t *testing.T,
) {
	db := openTestDatabase(t)
	repo := postgres.NewTeacherClassAssignmentRepository(db)
	ctx := context.Background()

	fixture := newTeacherClassAssignmentFixture(t, db)

	first := newTestTeacherClassAssignment(
		t,
		fixture.user.ID,
		fixture.academicYear.ID,
		fixture.class.ID,
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
	)

	secondUser := newTestUser(t)
	if err := postgres.NewUserRepository(db).Create(ctx, secondUser); err != nil {
		t.Fatalf("create second user error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestUser(t, db, secondUser.ID)
	})

	second := newTestTeacherClassAssignment(
		t,
		secondUser.ID,
		fixture.academicYear.ID,
		fixture.class.ID,
		time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
	)

	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}

	if err := repo.Create(ctx, second); err != nil {
		t.Fatalf("Create(second) error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestTeacherClassAssignment(t, db, first.ID)
		deleteTestTeacherClassAssignment(t, db, second.ID)
	})

	got, err := repo.GetCurrentByClassAndAcademicYear(
		ctx,
		fixture.class.ID,
		fixture.academicYear.ID,
		time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("GetCurrentByClassAndAcademicYear() error = %v", err)
	}

	assertTeacherClassAssignmentEqual(t, got, second)
}

func TestTeacherClassAssignmentRepository_GetCurrentByClassAndAcademicYearNotFound(
	t *testing.T,
) {
	db := openTestDatabase(t)
	repo := postgres.NewTeacherClassAssignmentRepository(db)

	fixture := newTeacherClassAssignmentFixture(t, db)

	_, err := repo.GetCurrentByClassAndAcademicYear(
		context.Background(),
		fixture.class.ID,
		fixture.academicYear.ID,
		time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC),
	)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf(
			"GetCurrentByClassAndAcademicYear() error = %v, want ErrNotFound",
			err,
		)
	}
}

func TestTeacherClassAssignmentRepository_CreateOverlappingPeriod(
	t *testing.T,
) {
	db := openTestDatabase(t)
	repo := postgres.NewTeacherClassAssignmentRepository(db)
	ctx := context.Background()

	fixture := newTeacherClassAssignmentFixture(t, db)

	first := newTestTeacherClassAssignment(
		t,
		fixture.user.ID,
		fixture.academicYear.ID,
		fixture.class.ID,
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
	)

	secondUser := newTestUser(t)
	if err := postgres.NewUserRepository(db).Create(ctx, secondUser); err != nil {
		t.Fatalf("create second user error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestUser(t, db, secondUser.ID)
	})

	second := newTestTeacherClassAssignment(
		t,
		secondUser.ID,
		fixture.academicYear.ID,
		fixture.class.ID,
		time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC),
	)

	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestTeacherClassAssignment(t, db, first.ID)
	})

	err := repo.Create(ctx, second)
	if !errors.Is(err, repository.ErrConflict) {
		t.Fatalf(
			"Create(second) error = %v, want ErrConflict",
			err,
		)
	}
}

func TestTeacherClassAssignmentRepository_CreateOpenEndedPeriod(
	t *testing.T,
) {
	db := openTestDatabase(t)
	repo := postgres.NewTeacherClassAssignmentRepository(db)
	ctx := context.Background()

	fixture := newTeacherClassAssignmentFixture(t, db)

	assignment := newTestTeacherClassAssignmentOpenEnded(
		t,
		fixture.user.ID,
		fixture.academicYear.ID,
		fixture.class.ID,
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
	)

	if err := repo.Create(ctx, assignment); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestTeacherClassAssignment(t, db, assignment.ID)
	})

	got, err := repo.GetByID(ctx, assignment.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	assertTeacherClassAssignmentEqual(t, got, assignment)

	if got.Period.To != nil {
		t.Fatalf("Period.To = %v, want nil", got.Period.To)
	}
}

func TestTeacherClassAssignmentRepository_CreateDifferentClassSamePeriod(
	t *testing.T,
) {
	db := openTestDatabase(t)
	repo := postgres.NewTeacherClassAssignmentRepository(db)
	ctx := context.Background()

	fixture := newTeacherClassAssignmentFixture(t, db)

	secondClass, err := domain.NewClass("8A", 8)
	if err != nil {
		t.Fatalf("NewClass() error = %v", err)
	}

	if err := postgres.NewClassRepository(db).Create(ctx, secondClass); err != nil {
		t.Fatalf("create second class error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestClass(t, db, secondClass.ID)
	})

	first := newTestTeacherClassAssignment(
		t,
		fixture.user.ID,
		fixture.academicYear.ID,
		fixture.class.ID,
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC),
	)

	second := newTestTeacherClassAssignment(
		t,
		fixture.user.ID,
		fixture.academicYear.ID,
		secondClass.ID,
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC),
	)

	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}

	if err := repo.Create(ctx, second); err != nil {
		t.Fatalf("Create(second) error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestTeacherClassAssignment(t, db, first.ID)
		deleteTestTeacherClassAssignment(t, db, second.ID)
	})
}

func TestTeacherClassAssignmentRepository_CreateDifferentAcademicYearSamePeriod(
	t *testing.T,
) {
	db := openTestDatabase(t)
	repo := postgres.NewTeacherClassAssignmentRepository(db)
	ctx := context.Background()

	fixture := newTeacherClassAssignmentFixture(t, db)

	secondAcademicYear := newTestAcademicYear(t)
	secondAcademicYear.Name = "2027/2028"
	secondAcademicYear.StartDate = time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC)
	secondAcademicYear.EndDate = time.Date(2028, 7, 1, 0, 0, 0, 0, time.UTC)
	secondAcademicYear.Status = domain.AcademicYearClosed

	if err := postgres.NewAcademicYearRepository(db).Create(
		ctx,
		secondAcademicYear,
	); err != nil {
		t.Fatalf("create second academic year error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestAcademicYear(t, db, secondAcademicYear.ID)
	})

	first := newTestTeacherClassAssignment(
		t,
		fixture.user.ID,
		fixture.academicYear.ID,
		fixture.class.ID,
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC),
	)

	second := newTestTeacherClassAssignment(
		t,
		fixture.user.ID,
		secondAcademicYear.ID,
		fixture.class.ID,
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC),
	)

	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}

	if err := repo.Create(ctx, second); err != nil {
		t.Fatalf("Create(second) error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestTeacherClassAssignment(t, db, first.ID)
		deleteTestTeacherClassAssignment(t, db, second.ID)
	})
}

func TestTeacherClassAssignmentRepository_GetCurrentUsesHalfOpenPeriod(
	t *testing.T,
) {
	db := openTestDatabase(t)
	repo := postgres.NewTeacherClassAssignmentRepository(db)
	ctx := context.Background()

	fixture := newTeacherClassAssignmentFixture(t, db)

	first := newTestTeacherClassAssignment(
		t,
		fixture.user.ID,
		fixture.academicYear.ID,
		fixture.class.ID,
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
	)

	second := newTestTeacherClassAssignmentOpenEnded(
		t,
		fixture.user.ID,
		fixture.academicYear.ID,
		fixture.class.ID,
		time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
	)

	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}

	if err := repo.Create(ctx, second); err != nil {
		t.Fatalf("Create(second) error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestTeacherClassAssignment(t, db, first.ID)
		deleteTestTeacherClassAssignment(t, db, second.ID)
	})

	got, err := repo.GetCurrentByClassAndAcademicYear(
		ctx,
		fixture.class.ID,
		fixture.academicYear.ID,
		time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("GetCurrentByClassAndAcademicYear() error = %v", err)
	}

	assertTeacherClassAssignmentEqual(t, got, second)
}

type teacherClassAssignmentFixture struct {
	user         domain.User
	academicYear domain.AcademicYear
	class        domain.Class
}

func newTeacherClassAssignmentFixture(
	t *testing.T,
	db *sql.DB,
) teacherClassAssignmentFixture {
	t.Helper()

	ctx := context.Background()

	userRepo := postgres.NewUserRepository(db)
	academicYearRepo := postgres.NewAcademicYearRepository(db)
	classRepo := postgres.NewClassRepository(db)

	user := newTestUser(t)
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("create fixture user: %v", err)
	}

	t.Cleanup(func() {
		deleteTestUser(t, db, user.ID)
	})

	academicYear := newTestAcademicYear(t)
	academicYear.Name = "2026/2027"
	academicYear.StartDate = time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	academicYear.EndDate = time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC)

	if err := academicYearRepo.Create(ctx, academicYear); err != nil {
		t.Fatalf("create fixture academic year: %v", err)
	}

	t.Cleanup(func() {
		deleteTestAcademicYear(t, db, academicYear.ID)
	})

	class, err := domain.NewClass("7A", 7)
	if err != nil {
		t.Fatalf("NewClass() error = %v", err)
	}

	if err := classRepo.Create(ctx, class); err != nil {
		t.Fatalf("create fixture class: %v", err)
	}

	t.Cleanup(func() {
		deleteTestClass(t, db, class.ID)
	})

	return teacherClassAssignmentFixture{
		user:         user,
		academicYear: academicYear,
		class:        class,
	}
}

func newTestTeacherClassAssignment(
	t *testing.T,
	userID uuid.UUID,
	academicYearID uuid.UUID,
	classID uuid.UUID,
	from time.Time,
	to time.Time,
) domain.TeacherClassAssignment {
	t.Helper()

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

func newTestTeacherClassAssignmentOpenEnded(
	t *testing.T,
	userID uuid.UUID,
	academicYearID uuid.UUID,
	classID uuid.UUID,
	from time.Time,
) domain.TeacherClassAssignment {
	t.Helper()

	period, err := domain.NewDateRange(from, nil)
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

func assertTeacherClassAssignmentEqual(
	t *testing.T,
	got domain.TeacherClassAssignment,
	want domain.TeacherClassAssignment,
) {
	t.Helper()

	if got.ID != want.ID {
		t.Errorf("ID = %v, want %v", got.ID, want.ID)
	}

	if got.UserID != want.UserID {
		t.Errorf("UserID = %v, want %v", got.UserID, want.UserID)
	}

	if got.AcademicYearID != want.AcademicYearID {
		t.Errorf(
			"AcademicYearID = %v, want %v",
			got.AcademicYearID,
			want.AcademicYearID,
		)
	}

	if got.ClassID != want.ClassID {
		t.Errorf("ClassID = %v, want %v", got.ClassID, want.ClassID)
	}

	if !got.Period.From.Equal(want.Period.From) {
		t.Errorf(
			"Period.From = %v, want %v",
			got.Period.From,
			want.Period.From,
		)
	}

	if got.Period.To == nil && want.Period.To == nil {
	} else if got.Period.To == nil || want.Period.To == nil {
		t.Errorf(
			"Period.To = %v, want %v",
			got.Period.To,
			want.Period.To,
		)
	} else if !got.Period.To.Equal(*want.Period.To) {
		t.Errorf(
			"Period.To = %v, want %v",
			got.Period.To,
			want.Period.To,
		)
	}

	if !got.CreatedAt.Truncate(time.Microsecond).Equal(
		want.CreatedAt.Truncate(time.Microsecond),
	) {
		t.Errorf(
			"CreatedAt = %v, want %v",
			got.CreatedAt,
			want.CreatedAt,
		)
	}

	if !got.UpdatedAt.Truncate(time.Microsecond).Equal(
		want.UpdatedAt.Truncate(time.Microsecond),
	) {
		t.Errorf(
			"UpdatedAt = %v, want %v",
			got.UpdatedAt,
			want.UpdatedAt,
		)
	}
}

func deleteTestTeacherClassAssignment(
	t *testing.T,
	db *sql.DB,
	id uuid.UUID,
) {
	t.Helper()

	_, err := db.ExecContext(
		context.Background(),
		"DELETE FROM teacher_class_assignments WHERE id = $1",
		id,
	)
	if err != nil {
		t.Errorf("delete test teacher class assignment: %v", err)
	}
}
