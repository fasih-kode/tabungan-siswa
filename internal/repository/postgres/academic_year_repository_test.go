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

func TestAcademicYearRepository_CreateAndGetByID(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewAcademicYearRepository(db)
	ctx := context.Background()

	academicYear := newTestAcademicYear(t)

	if err := repo.Create(ctx, academicYear); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestAcademicYear(t, db, academicYear.ID)
	})

	got, err := repo.GetByID(ctx, academicYear.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	assertAcademicYearEqual(t, got, academicYear)
}

func TestAcademicYearRepository_GetByIDNotFound(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewAcademicYearRepository(db)

	_, err := repo.GetByID(
		context.Background(),
		uuid.New(),
	)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf(
			"GetByID() error = %v, want ErrNotFound",
			err,
		)
	}
}

func TestAcademicYearRepository_GetOpen(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewAcademicYearRepository(db)
	ctx := context.Background()

	academicYear := newTestAcademicYear(t)

	if err := repo.Create(ctx, academicYear); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestAcademicYear(t, db, academicYear.ID)
	})

	got, err := repo.GetOpen(ctx)
	if err != nil {
		t.Fatalf("GetOpen() error = %v", err)
	}

	assertAcademicYearEqual(t, got, academicYear)
}

func TestAcademicYearRepository_GetOpenNotFound(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewAcademicYearRepository(db)

	cleanupAcademicYears(t, db)

	_, err := repo.GetOpen(context.Background())
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf(
			"GetOpen() error = %v, want ErrNotFound",
			err,
		)
	}
}

func TestAcademicYearRepository_List(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewAcademicYearRepository(db)
	ctx := context.Background()

	cleanupAcademicYears(t, db)

	first := newTestAcademicYearWithDates(
		t,
		"2024/2025",
		time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC),
		domain.AcademicYearClosed,
	)
	second := newTestAcademicYearWithDates(
		t,
		"2025/2026",
		time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC),
		domain.AcademicYearClosing,
	)

	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}

	if err := repo.Create(ctx, second); err != nil {
		t.Fatalf("Create(second) error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestAcademicYear(t, db, first.ID)
		deleteTestAcademicYear(t, db, second.ID)
	})

	got, err := repo.List(ctx, repository.ListOptions{
		Limit:  10,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("List() length = %d, want 2", len(got))
	}

	assertAcademicYearEqual(t, got[0], second)
	assertAcademicYearEqual(t, got[1], first)
}

func TestAcademicYearRepository_ListPagination(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewAcademicYearRepository(db)
	ctx := context.Background()

	cleanupAcademicYears(t, db)

	first := newTestAcademicYearWithDates(
		t,
		"2023/2024",
		time.Date(2023, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC),
		domain.AcademicYearClosed,
	)
	second := newTestAcademicYearWithDates(
		t,
		"2024/2025",
		time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC),
		domain.AcademicYearClosed,
	)
	third := newTestAcademicYearWithDates(
		t,
		"2025/2026",
		time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC),
		domain.AcademicYearClosed,
	)

	for _, academicYear := range []domain.AcademicYear{
		first,
		second,
		third,
	} {
		if err := repo.Create(ctx, academicYear); err != nil {
			t.Fatalf("Create(%s) error = %v", academicYear.Name, err)
		}
	}

	t.Cleanup(func() {
		deleteTestAcademicYear(t, db, first.ID)
		deleteTestAcademicYear(t, db, second.ID)
		deleteTestAcademicYear(t, db, third.ID)
	})

	got, err := repo.List(ctx, repository.ListOptions{
		Limit:  1,
		Offset: 1,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("List() length = %d, want 1", len(got))
	}

	assertAcademicYearEqual(t, got[0], second)
}

func TestAcademicYearRepository_Update(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewAcademicYearRepository(db)
	ctx := context.Background()

	academicYear := newTestAcademicYear(t)

	if err := repo.Create(ctx, academicYear); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestAcademicYear(t, db, academicYear.ID)
	})

	updated := academicYear
	updated.Name = "Updated " + uuid.NewString()
	updated.StartDate = time.Date(
		2025,
		7,
		1,
		0,
		0,
		0,
		0,
		time.UTC,
	)
	updated.EndDate = time.Date(
		2026,
		6,
		30,
		0,
		0,
		0,
		0,
		time.UTC,
	)
	updated.Status = domain.AcademicYearClosing
	updated.UpdatedAt = time.Now()

	if err := repo.Update(ctx, updated); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, err := repo.GetByID(ctx, academicYear.ID)
	if err != nil {
		t.Fatalf("GetByID() after Update() error = %v", err)
	}

	assertAcademicYearEqual(t, got, updated)
}

func TestAcademicYearRepository_UpdateNotFound(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewAcademicYearRepository(db)

	academicYear := newTestAcademicYear(t)

	err := repo.Update(
		context.Background(),
		academicYear,
	)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf(
			"Update() error = %v, want ErrNotFound",
			err,
		)
	}
}

func TestAcademicYearRepository_CreateDuplicateOpen(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewAcademicYearRepository(db)
	ctx := context.Background()

	cleanupAcademicYears(t, db)

	first := newTestAcademicYear(t)

	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("first Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestAcademicYear(t, db, first.ID)
	})

	second := newTestAcademicYearWithDates(
		t,
		"2026/2027",
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 6, 30, 0, 0, 0, 0, time.UTC),
		domain.AcademicYearOpen,
	)

	err := repo.Create(ctx, second)
	if !errors.Is(err, repository.ErrConflict) {
		t.Fatalf(
			"second Create() error = %v, want ErrConflict",
			err,
		)
	}
}

func assertAcademicYearEqual(
	t *testing.T,
	got domain.AcademicYear,
	want domain.AcademicYear,
) {
	t.Helper()

	if got.ID != want.ID {
		t.Errorf("ID = %v, want %v", got.ID, want.ID)
	}

	if got.Name != want.Name {
		t.Errorf("Name = %q, want %q", got.Name, want.Name)
	}

	if !got.StartDate.Equal(want.StartDate) {
		t.Errorf(
			"StartDate = %v, want %v",
			got.StartDate,
			want.StartDate,
		)
	}

	if !got.EndDate.Equal(want.EndDate) {
		t.Errorf(
			"EndDate = %v, want %v",
			got.EndDate,
			want.EndDate,
		)
	}

	if got.Status != want.Status {
		t.Errorf("Status = %q, want %q", got.Status, want.Status)
	}

	if !got.CreatedAt.Equal(want.CreatedAt) {
		if !got.CreatedAt.Truncate(time.Microsecond).Equal(
			want.CreatedAt.Truncate(time.Microsecond),
		) {
			t.Errorf(
				"CreatedAt = %v, want %v",
				got.CreatedAt,
				want.CreatedAt,
			)
		}
	}

	if !got.UpdatedAt.Equal(want.UpdatedAt) {
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
}

func newTestAcademicYear(t *testing.T) domain.AcademicYear {
	t.Helper()

	return newTestAcademicYearWithDates(
		t,
		"2025/2026",
		time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC),
		domain.AcademicYearOpen,
	)
}

func newTestAcademicYearWithDates(
	t *testing.T,
	name string,
	startDate time.Time,
	endDate time.Time,
	status domain.AcademicYearStatus,
) domain.AcademicYear {
	t.Helper()

	academicYear, err := domain.NewAcademicYear(
		name,
		startDate,
		endDate,
	)
	if err != nil {
		t.Fatalf("NewAcademicYear() error = %v", err)
	}

	academicYear.Status = status
	return academicYear
}

func deleteTestAcademicYear(
	t *testing.T,
	db *sql.DB,
	id uuid.UUID,
) {
	t.Helper()

	_, err := db.ExecContext(
		context.Background(),
		"DELETE FROM academic_years WHERE id = $1",
		id,
	)
	if err != nil {
		t.Errorf("delete test academic year: %v", err)
	}
}

func cleanupScopedRepositoryFixtures(
	t *testing.T,
	db *sql.DB,
) {
	t.Helper()

	statements := []struct {
		name  string
		query string
	}{
		{
			name:  "transactions",
			query: "DELETE FROM transactions",
		},
		{
			name:  "savings settlements",
			query: "DELETE FROM savings_settlements",
		},
		{
			name:  "savings accounts",
			query: "DELETE FROM savings_accounts",
		},
		{
			name:  "class transfer requests",
			query: "DELETE FROM class_transfer_requests",
		},
		{
			name:  "student class histories",
			query: "DELETE FROM student_class_histories",
		},
		{
			name:  "teacher class assignments",
			query: "DELETE FROM teacher_class_assignments",
		},
		{
			name:  "audit logs",
			query: "DELETE FROM audit_logs",
		},
		{
			name:  "students",
			query: "DELETE FROM students",
		},
		{
			name:  "classes",
			query: "DELETE FROM classes",
		},
		{
			name:  "academic years",
			query: "DELETE FROM academic_years",
		},
		{
			name:  "users",
			query: "DELETE FROM users",
		},
	}

	ctx := context.Background()

	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement.query); err != nil {
			t.Fatalf(
				"cleanup %s: %v",
				statement.name,
				err,
			)
		}
	}
}

func cleanupAcademicYears(
	t *testing.T,
	db *sql.DB,
) {
	t.Helper()

	statements := []struct {
		name  string
		query string
	}{
		{
			name:  "transactions",
			query: "DELETE FROM transactions",
		},
		{
			name:  "savings settlements",
			query: "DELETE FROM savings_settlements",
		},
		{
			name:  "savings accounts",
			query: "DELETE FROM savings_accounts",
		},
		{
			name:  "class transfer requests",
			query: "DELETE FROM class_transfer_requests",
		},
		{
			name:  "student class histories",
			query: "DELETE FROM student_class_histories",
		},
		{
			name:  "teacher class assignments",
			query: "DELETE FROM teacher_class_assignments",
		},
		{
			name:  "audit logs",
			query: "DELETE FROM audit_logs",
		},
		{
			name:  "students",
			query: "DELETE FROM students",
		},
		{
			name:  "classes",
			query: "DELETE FROM classes",
		},
		{
			name:  "academic years",
			query: "DELETE FROM academic_years",
		},
		{
			name:  "users",
			query: "DELETE FROM users",
		},
	}

	ctx := context.Background()

	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement.query); err != nil {
			t.Fatalf(
				"cleanup %s: %v",
				statement.name,
				err,
			)
		}
	}
}
