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

func TestStudentClassHistoryRepository_CreateAndGetByID(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewStudentClassHistoryRepository(db)
	ctx := context.Background()

	fixture := newStudentClassHistoryFixture(t, db)

	history := newTestStudentClassHistory(
		t,
		fixture.student.ID,
		fixture.academicYear.ID,
		fixture.class.ID,
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC),
	)

	if err := repo.Create(ctx, history); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestStudentClassHistory(t, db, history.ID)
	})

	got, err := repo.GetByID(ctx, history.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	assertStudentClassHistoryEqual(t, got, history)
}

func TestStudentClassHistoryRepository_GetByIDNotFound(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewStudentClassHistoryRepository(db)

	_, err := repo.GetByID(context.Background(), uuid.New())
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("GetByID() error = %v, want ErrNotFound", err)
	}
}

func TestStudentClassHistoryRepository_ListByStudentAndAcademicYear(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewStudentClassHistoryRepository(db)
	ctx := context.Background()

	fixture := newStudentClassHistoryFixture(t, db)

	first := newTestStudentClassHistory(
		t,
		fixture.student.ID,
		fixture.academicYear.ID,
		fixture.class.ID,
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
	)

	second := newTestStudentClassHistory(
		t,
		fixture.student.ID,
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
		deleteTestStudentClassHistory(t, db, first.ID)
		deleteTestStudentClassHistory(t, db, second.ID)
	})

	got, err := repo.ListByStudentAndAcademicYear(
		ctx,
		fixture.student.ID,
		fixture.academicYear.ID,
		repository.ListOptions{
			Limit:  10,
			Offset: 0,
		},
	)
	if err != nil {
		t.Fatalf("ListByStudentAndAcademicYear() error = %v", err)
	}

	if len(got) != 2 {
		t.Fatalf(
			"ListByStudentAndAcademicYear() length = %d, want 2",
			len(got),
		)
	}

	assertStudentClassHistoryEqual(t, got[0], first)
	assertStudentClassHistoryEqual(t, got[1], second)
}

func TestStudentClassHistoryRepository_ListByStudentAndAcademicYearEmpty(
	t *testing.T,
) {
	db := openTestDatabase(t)
	repo := postgres.NewStudentClassHistoryRepository(db)

	fixture := newStudentClassHistoryFixture(t, db)

	got, err := repo.ListByStudentAndAcademicYear(
		context.Background(),
		fixture.student.ID,
		fixture.academicYear.ID,
		repository.ListOptions{
			Limit:  10,
			Offset: 0,
		},
	)
	if err != nil {
		t.Fatalf("ListByStudentAndAcademicYear() error = %v", err)
	}

	if got == nil {
		t.Fatal("ListByStudentAndAcademicYear() returned nil slice, want empty slice")
	}

	if len(got) != 0 {
		t.Fatalf(
			"ListByStudentAndAcademicYear() length = %d, want 0",
			len(got),
		)
	}
}

func TestStudentClassHistoryRepository_ListPagination(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewStudentClassHistoryRepository(db)
	ctx := context.Background()

	fixture := newStudentClassHistoryFixture(t, db)

	first := newTestStudentClassHistory(
		t,
		fixture.student.ID,
		fixture.academicYear.ID,
		fixture.class.ID,
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
	)

	second := newTestStudentClassHistory(
		t,
		fixture.student.ID,
		fixture.academicYear.ID,
		fixture.class.ID,
		time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
	)

	third := newTestStudentClassHistory(
		t,
		fixture.student.ID,
		fixture.academicYear.ID,
		fixture.class.ID,
		time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC),
	)

	for _, history := range []domain.StudentClassHistory{
		first,
		second,
		third,
	} {
		if err := repo.Create(ctx, history); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	t.Cleanup(func() {
		deleteTestStudentClassHistory(t, db, first.ID)
		deleteTestStudentClassHistory(t, db, second.ID)
		deleteTestStudentClassHistory(t, db, third.ID)
	})

	got, err := repo.ListByStudentAndAcademicYear(
		ctx,
		fixture.student.ID,
		fixture.academicYear.ID,
		repository.ListOptions{
			Limit:  1,
			Offset: 1,
		},
	)
	if err != nil {
		t.Fatalf("ListByStudentAndAcademicYear() error = %v", err)
	}

	if len(got) != 1 {
		t.Fatalf(
			"ListByStudentAndAcademicYear() length = %d, want 1",
			len(got),
		)
	}

	assertStudentClassHistoryEqual(t, got[0], second)
}

func TestStudentClassHistoryRepository_CloseCurrentByStudentAndAcademicYear(
	t *testing.T,
) {
	db := openTestDatabase(t)
	repo := postgres.NewStudentClassHistoryRepository(db)
	ctx := context.Background()

	fixture := newStudentClassHistoryFixture(t, db)

	startDate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)

	history := newTestStudentClassHistoryOpenEnded(
		t,
		fixture.student.ID,
		fixture.academicYear.ID,
		fixture.class.ID,
		startDate,
	)

	if err := repo.Create(ctx, history); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestStudentClassHistory(t, db, history.ID)
	})

	got, err := repo.CloseCurrentByStudentAndAcademicYear(
		ctx,
		fixture.student.ID,
		fixture.academicYear.ID,
		endDate,
	)
	if err != nil {
		t.Fatalf(
			"CloseCurrentByStudentAndAcademicYear() error = %v",
			err,
		)
	}

	if got.ID != history.ID {
		t.Fatalf(
			"ID = %v, want %v",
			got.ID,
			history.ID,
		)
	}

	if got.StudentID != history.StudentID {
		t.Fatalf(
			"StudentID = %v, want %v",
			got.StudentID,
			history.StudentID,
		)
	}

	if got.AcademicYearID != history.AcademicYearID {
		t.Fatalf(
			"AcademicYearID = %v, want %v",
			got.AcademicYearID,
			history.AcademicYearID,
		)
	}

	if got.ClassID != history.ClassID {
		t.Fatalf(
			"ClassID = %v, want %v",
			got.ClassID,
			history.ClassID,
		)
	}

	if !got.Period.From.Equal(startDate) {
		t.Fatalf(
			"Period.From = %v, want %v",
			got.Period.From,
			startDate,
		)
	}

	if got.Period.To == nil {
		t.Fatal("Period.To = nil, want closed period")
	}

	if !got.Period.To.Equal(endDate) {
		t.Fatalf(
			"Period.To = %v, want %v",
			*got.Period.To,
			endDate,
		)
	}

	if !got.UpdatedAt.After(history.UpdatedAt) {
		t.Fatalf(
			"UpdatedAt = %v, want after %v",
			got.UpdatedAt,
			history.UpdatedAt,
		)
	}
}

func TestStudentClassHistoryRepository_CloseCurrentByStudentAndAcademicYearNotFound(
	t *testing.T,
) {
	db := openTestDatabase(t)
	repo := postgres.NewStudentClassHistoryRepository(db)

	_, err := repo.CloseCurrentByStudentAndAcademicYear(
		context.Background(),
		uuid.New(),
		uuid.New(),
		time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC),
	)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf(
			"CloseCurrentByStudentAndAcademicYear() error = %v, want ErrNotFound",
			err,
		)
	}
}

func TestStudentClassHistoryRepository_GetCurrentByStudentAndAcademicYear(
	t *testing.T,
) {
	db := openTestDatabase(t)
	repo := postgres.NewStudentClassHistoryRepository(db)
	ctx := context.Background()

	fixture := newStudentClassHistoryFixture(t, db)

	first := newTestStudentClassHistory(
		t,
		fixture.student.ID,
		fixture.academicYear.ID,
		fixture.class.ID,
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
	)

	second := newTestStudentClassHistory(
		t,
		fixture.student.ID,
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
		deleteTestStudentClassHistory(t, db, first.ID)
		deleteTestStudentClassHistory(t, db, second.ID)
	})

	got, err := repo.GetCurrentByStudentAndAcademicYear(
		ctx,
		fixture.student.ID,
		fixture.academicYear.ID,
		time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("GetCurrentByStudentAndAcademicYear() error = %v", err)
	}

	assertStudentClassHistoryEqual(t, got, second)
}

func TestStudentClassHistoryRepository_GetCurrentByStudentAndAcademicYearNotFound(
	t *testing.T,
) {
	db := openTestDatabase(t)
	repo := postgres.NewStudentClassHistoryRepository(db)

	fixture := newStudentClassHistoryFixture(t, db)

	_, err := repo.GetCurrentByStudentAndAcademicYear(
		context.Background(),
		fixture.student.ID,
		fixture.academicYear.ID,
		time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC),
	)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf(
			"GetCurrentByStudentAndAcademicYear() error = %v, want ErrNotFound",
			err,
		)
	}
}

func TestStudentClassHistoryRepository_CreateOverlappingPeriod(
	t *testing.T,
) {
	db := openTestDatabase(t)
	repo := postgres.NewStudentClassHistoryRepository(db)
	ctx := context.Background()

	fixture := newStudentClassHistoryFixture(t, db)

	first := newTestStudentClassHistory(
		t,
		fixture.student.ID,
		fixture.academicYear.ID,
		fixture.class.ID,
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
	)

	second := newTestStudentClassHistory(
		t,
		fixture.student.ID,
		fixture.academicYear.ID,
		fixture.class.ID,
		time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC),
	)

	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestStudentClassHistory(t, db, first.ID)
	})

	err := repo.Create(ctx, second)
	if !errors.Is(err, repository.ErrConflict) {
		t.Fatalf(
			"Create(second) error = %v, want ErrConflict",
			err,
		)
	}
}

func TestStudentClassHistoryRepository_CreateOpenEndedPeriod(
	t *testing.T,
) {
	db := openTestDatabase(t)
	repo := postgres.NewStudentClassHistoryRepository(db)
	ctx := context.Background()

	fixture := newStudentClassHistoryFixture(t, db)

	history := newTestStudentClassHistoryOpenEnded(
		t,
		fixture.student.ID,
		fixture.academicYear.ID,
		fixture.class.ID,
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
	)

	if err := repo.Create(ctx, history); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestStudentClassHistory(t, db, history.ID)
	})

	got, err := repo.GetByID(ctx, history.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	assertStudentClassHistoryEqual(t, got, history)

	if got.Period.To != nil {
		t.Fatalf("Period.To = %v, want nil", got.Period.To)
	}
}

func TestStudentClassHistoryRepository_CreateDifferentStudentSamePeriod(
	t *testing.T,
) {
	db := openTestDatabase(t)
	repo := postgres.NewStudentClassHistoryRepository(db)
	ctx := context.Background()

	fixture := newStudentClassHistoryFixture(t, db)
	secondStudent := newTestStudent(t, "Budi")

	if err := postgres.NewStudentRepository(db).Create(
		ctx,
		secondStudent,
	); err != nil {
		t.Fatalf("create second student error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestStudent(t, db, secondStudent.ID)
	})

	first := newTestStudentClassHistory(
		t,
		fixture.student.ID,
		fixture.academicYear.ID,
		fixture.class.ID,
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC),
	)

	second := newTestStudentClassHistory(
		t,
		secondStudent.ID,
		fixture.academicYear.ID,
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
		deleteTestStudentClassHistory(t, db, first.ID)
		deleteTestStudentClassHistory(t, db, second.ID)
	})
}

func TestStudentClassHistoryRepository_CreateDifferentAcademicYearSamePeriod(
	t *testing.T,
) {
	db := openTestDatabase(t)
	repo := postgres.NewStudentClassHistoryRepository(db)
	ctx := context.Background()

	fixture := newStudentClassHistoryFixture(t, db)

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

	first := newTestStudentClassHistory(
		t,
		fixture.student.ID,
		fixture.academicYear.ID,
		fixture.class.ID,
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC),
	)

	second := newTestStudentClassHistory(
		t,
		fixture.student.ID,
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
		deleteTestStudentClassHistory(t, db, first.ID)
		deleteTestStudentClassHistory(t, db, second.ID)
	})
}

func TestStudentClassHistoryRepository_GetCurrentUsesHalfOpenPeriod(
	t *testing.T,
) {
	db := openTestDatabase(t)
	repo := postgres.NewStudentClassHistoryRepository(db)
	ctx := context.Background()

	fixture := newStudentClassHistoryFixture(t, db)

	first := newTestStudentClassHistory(
		t,
		fixture.student.ID,
		fixture.academicYear.ID,
		fixture.class.ID,
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
	)

	second := newTestStudentClassHistoryOpenEnded(
		t,
		fixture.student.ID,
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
		deleteTestStudentClassHistory(t, db, first.ID)
		deleteTestStudentClassHistory(t, db, second.ID)
	})

	got, err := repo.GetCurrentByStudentAndAcademicYear(
		ctx,
		fixture.student.ID,
		fixture.academicYear.ID,
		time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("GetCurrentByStudentAndAcademicYear() error = %v", err)
	}

	assertStudentClassHistoryEqual(t, got, second)
}

type studentClassHistoryFixture struct {
	user         domain.User
	student      domain.Student
	academicYear domain.AcademicYear
	class        domain.Class
}

func newStudentClassHistoryFixture(
	t *testing.T,
	db *sql.DB,
) studentClassHistoryFixture {
	t.Helper()

	ctx := context.Background()

	userRepo := postgres.NewUserRepository(db)
	studentRepo := postgres.NewStudentRepository(db)
	academicYearRepo := postgres.NewAcademicYearRepository(db)
	classRepo := postgres.NewClassRepository(db)

	user := newTestUser(t)
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("create fixture user: %v", err)
	}

	t.Cleanup(func() {
		deleteTestUser(t, db, user.ID)
	})

	student := newTestStudentWithUserID(t, "Ahmad", user.ID)
	if err := studentRepo.Create(ctx, student); err != nil {
		t.Fatalf("create fixture student: %v", err)
	}

	t.Cleanup(func() {
		deleteTestStudent(t, db, student.ID)
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

	return studentClassHistoryFixture{
		user:         user,
		student:      student,
		academicYear: academicYear,
		class:        class,
	}
}

func newTestStudentClassHistory(
	t *testing.T,
	studentID uuid.UUID,
	academicYearID uuid.UUID,
	classID uuid.UUID,
	from time.Time,
	to time.Time,
) domain.StudentClassHistory {
	t.Helper()

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

func newTestStudentClassHistoryOpenEnded(
	t *testing.T,
	studentID uuid.UUID,
	academicYearID uuid.UUID,
	classID uuid.UUID,
	from time.Time,
) domain.StudentClassHistory {
	t.Helper()

	period, err := domain.NewDateRange(from, nil)
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

func assertStudentClassHistoryEqual(
	t *testing.T,
	got domain.StudentClassHistory,
	want domain.StudentClassHistory,
) {
	t.Helper()

	if got.ID != want.ID {
		t.Errorf("ID = %v, want %v", got.ID, want.ID)
	}

	if got.StudentID != want.StudentID {
		t.Errorf("StudentID = %v, want %v", got.StudentID, want.StudentID)
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

func deleteTestStudentClassHistory(
	t *testing.T,
	db *sql.DB,
	id uuid.UUID,
) {
	t.Helper()

	_, err := db.ExecContext(
		context.Background(),
		"DELETE FROM student_class_histories WHERE id = $1",
		id,
	)
	if err != nil {
		t.Errorf("delete test student class history: %v", err)
	}
}
