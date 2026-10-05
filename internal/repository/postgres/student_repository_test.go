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

func TestStudentRepository_CreateAndGetByID(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewStudentRepository(db)
	ctx := context.Background()

	student := newTestStudent(t, "Ahmad")

	if err := repo.Create(ctx, student); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestStudent(t, db, student.ID)
	})

	got, err := repo.GetByID(ctx, student.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	assertStudentEqual(t, got, student)
}

func TestStudentRepository_GetByIDNotFound(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewStudentRepository(db)

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

func TestStudentRepository_GetByNIS(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewStudentRepository(db)
	ctx := context.Background()

	nis := "NIS-001"
	student := newTestStudentWithNIS(t, "Ahmad", nis)

	if err := repo.Create(ctx, student); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestStudent(t, db, student.ID)
	})

	got, err := repo.GetByNIS(ctx, nis)
	if err != nil {
		t.Fatalf("GetByNIS() error = %v", err)
	}

	assertStudentEqual(t, got, student)
}

func TestStudentRepository_GetByNISNotFound(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewStudentRepository(db)

	_, err := repo.GetByNIS(
		context.Background(),
		"NIS-NOT-FOUND",
	)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf(
			"GetByNIS() error = %v, want ErrNotFound",
			err,
		)
	}
}

func TestStudentRepository_GetByNISN(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewStudentRepository(db)
	ctx := context.Background()

	nisn := "NISN-001"
	student := newTestStudentWithNISN(t, "Ahmad", nisn)

	if err := repo.Create(ctx, student); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestStudent(t, db, student.ID)
	})

	got, err := repo.GetByNISN(ctx, nisn)
	if err != nil {
		t.Fatalf("GetByNISN() error = %v", err)
	}

	assertStudentEqual(t, got, student)
}

func TestStudentRepository_GetByNISNNotFound(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewStudentRepository(db)

	_, err := repo.GetByNISN(
		context.Background(),
		"NISN-NOT-FOUND",
	)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf(
			"GetByNISN() error = %v, want ErrNotFound",
			err,
		)
	}
}

func TestStudentRepository_GetByUserID(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewStudentRepository(db)
	userRepo := postgres.NewUserRepository(db)
	ctx := context.Background()

	user := newTestUser(t)

	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("userRepo.Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestUser(t, db, user.ID)
	})

	student := newTestStudentWithUserID(t, "Ahmad", user.ID)

	if err := repo.Create(ctx, student); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestStudent(t, db, student.ID)
	})

	got, err := repo.GetByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetByUserID() error = %v", err)
	}

	assertStudentEqual(t, got, student)
}

func TestStudentRepository_GetByUserIDNotFound(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewStudentRepository(db)

	_, err := repo.GetByUserID(
		context.Background(),
		uuid.New(),
	)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf(
			"GetByUserID() error = %v, want ErrNotFound",
			err,
		)
	}
}

func TestStudentRepository_List(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewStudentRepository(db)
	ctx := context.Background()

	cleanupStudents(t, db)

	first := newTestStudent(t, "Budi")
	second := newTestStudent(t, "Ahmad")
	third := newTestStudent(t, "Citra")

	for _, student := range []domain.Student{
		first,
		second,
		third,
	} {
		if err := repo.Create(ctx, student); err != nil {
			t.Fatalf("Create(%s) error = %v", student.Name, err)
		}
	}

	t.Cleanup(func() {
		deleteTestStudent(t, db, first.ID)
		deleteTestStudent(t, db, second.ID)
		deleteTestStudent(t, db, third.ID)
	})

	got, err := repo.List(ctx, repository.ListOptions{
		Limit:  10,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("List() length = %d, want 3", len(got))
	}

	assertStudentEqual(t, got[0], second)
	assertStudentEqual(t, got[1], first)
	assertStudentEqual(t, got[2], third)
}

func TestStudentRepository_ListPagination(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewStudentRepository(db)
	ctx := context.Background()

	cleanupStudents(t, db)

	first := newTestStudent(t, "Ahmad")
	second := newTestStudent(t, "Budi")
	third := newTestStudent(t, "Citra")

	for _, student := range []domain.Student{
		first,
		second,
		third,
	} {
		if err := repo.Create(ctx, student); err != nil {
			t.Fatalf("Create(%s) error = %v", student.Name, err)
		}
	}

	t.Cleanup(func() {
		deleteTestStudent(t, db, first.ID)
		deleteTestStudent(t, db, second.ID)
		deleteTestStudent(t, db, third.ID)
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

	assertStudentEqual(t, got[0], second)
}

func TestStudentRepository_Update(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewStudentRepository(db)
	ctx := context.Background()

	student := newTestStudent(t, "Ahmad")

	if err := repo.Create(ctx, student); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestStudent(t, db, student.ID)
	})

	updated := student
	updated.Name = "Ahmad Fauzan"
	updated.Status = domain.StudentLeft
	updated.UpdatedAt = time.Now()

	if err := repo.Update(ctx, updated); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, err := repo.GetByID(ctx, student.ID)
	if err != nil {
		t.Fatalf("GetByID() after Update() error = %v", err)
	}

	assertStudentEqual(t, got, updated)
}

func TestStudentRepository_UpdateNotFound(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewStudentRepository(db)

	student := newTestStudent(t, "Ahmad")

	err := repo.Update(
		context.Background(),
		student,
	)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf(
			"Update() error = %v, want ErrNotFound",
			err,
		)
	}
}

func TestStudentRepository_CreateDuplicateNIS(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewStudentRepository(db)
	ctx := context.Background()

	cleanupStudents(t, db)

	first := newTestStudentWithNIS(t, "Ahmad", "NIS-001")

	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("first Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestStudent(t, db, first.ID)
	})

	second := newTestStudentWithNIS(t, "Budi", "NIS-001")

	err := repo.Create(ctx, second)
	if !errors.Is(err, repository.ErrConflict) {
		t.Fatalf(
			"second Create() error = %v, want ErrConflict",
			err,
		)
	}
}

func TestStudentRepository_CreateDuplicateNISN(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewStudentRepository(db)
	ctx := context.Background()

	cleanupStudents(t, db)

	first := newTestStudentWithNISN(t, "Ahmad", "NISN-001")

	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("first Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestStudent(t, db, first.ID)
	})

	second := newTestStudentWithNISN(t, "Budi", "NISN-001")

	err := repo.Create(ctx, second)
	if !errors.Is(err, repository.ErrConflict) {
		t.Fatalf(
			"second Create() error = %v, want ErrConflict",
			err,
		)
	}
}

func TestStudentRepository_CreateDuplicateUserID(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewStudentRepository(db)
	userRepo := postgres.NewUserRepository(db)
	ctx := context.Background()

	cleanupStudents(t, db)

	user := newTestUser(t)

	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("userRepo.Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestUser(t, db, user.ID)
	})

	first := newTestStudentWithUserID(t, "Ahmad", user.ID)

	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("first Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestStudent(t, db, first.ID)
	})

	second := newTestStudentWithUserID(t, "Budi", user.ID)

	err := repo.Create(ctx, second)
	if !errors.Is(err, repository.ErrConflict) {
		t.Fatalf(
			"second Create() error = %v, want ErrConflict",
			err,
		)
	}
}

func TestStudentRepository_UpdateDuplicateUniqueField(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewStudentRepository(db)
	ctx := context.Background()

	cleanupStudents(t, db)

	first := newTestStudentWithNIS(t, "Ahmad", "NIS-001")
	second := newTestStudentWithNIS(t, "Budi", "NIS-002")

	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("first Create() error = %v", err)
	}

	if err := repo.Create(ctx, second); err != nil {
		t.Fatalf("second Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestStudent(t, db, first.ID)
		deleteTestStudent(t, db, second.ID)
	})

	second.NIS = first.NIS
	second.UpdatedAt = time.Now()

	err := repo.Update(ctx, second)
	if !errors.Is(err, repository.ErrConflict) {
		t.Fatalf(
			"Update() error = %v, want ErrConflict",
			err,
		)
	}
}

func TestStudentRepository_ListByClassAndAcademicYear(t *testing.T) {
	db := openTestDatabase(t)
	cleanupScopedRepositoryFixtures(t, db)
	studentRepo := postgres.NewStudentRepository(db)
	classRepo := postgres.NewClassRepository(db)
	academicYearRepo := postgres.NewAcademicYearRepository(db)
	historyRepo := postgres.NewStudentClassHistoryRepository(db)
	ctx := context.Background()

	academicYear := newTestAcademicYear(t)
	if err := academicYearRepo.Create(ctx, academicYear); err != nil {
		t.Fatalf("Create() academic year error = %v", err)
	}

	secondAcademicYear := newTestAcademicYearWithDates(
		t,
		"2026/2027",
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 6, 30, 0, 0, 0, 0, time.UTC),
		domain.AcademicYearClosed,
	)
	if err := academicYearRepo.Create(ctx, secondAcademicYear); err != nil {
		t.Fatalf("Create() second academic year error = %v", err)
	}

	classA := newTestClass(t, "7A-student-scope", 7)
	classB := newTestClass(t, "8A-student-scope", 8)

	if err := classRepo.Create(ctx, classA); err != nil {
		t.Fatalf("Create() classA error = %v", err)
	}
	if err := classRepo.Create(ctx, classB); err != nil {
		t.Fatalf("Create() classB error = %v", err)
	}

	studentA := newTestStudent(t, "Budi Scope")
	studentB := newTestStudent(t, "Citra Scope")
	studentC := newTestStudent(t, "Dedi Scope")
	studentD := newTestStudent(t, "Eka Scope")

	for _, student := range []domain.Student{
		studentA,
		studentB,
		studentC,
		studentD,
	} {
		if err := studentRepo.Create(ctx, student); err != nil {
			t.Fatalf("Create() student %q error = %v", student.Name, err)
		}
	}

	// Student A has two sequential histories in the same class and
	// academic year. The scoped student query must still return A once.
	historyA1 := newTestStudentClassHistory(
		t,
		studentA.ID,
		academicYear.ID,
		classA.ID,
		time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	)
	historyA2 := newTestStudentClassHistory(
		t,
		studentA.ID,
		academicYear.ID,
		classA.ID,
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC),
	)
	historyB := newTestStudentClassHistory(
		t,
		studentB.ID,
		academicYear.ID,
		classA.ID,
		time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC),
	)
	historyC := newTestStudentClassHistory(
		t,
		studentC.ID,
		academicYear.ID,
		classB.ID,
		time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC),
	)
	historyD := newTestStudentClassHistory(
		t,
		studentD.ID,
		secondAcademicYear.ID,
		classA.ID,
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 6, 30, 0, 0, 0, 0, time.UTC),
	)

	histories := []domain.StudentClassHistory{
		historyA1,
		historyA2,
		historyB,
		historyC,
		historyD,
	}

	for _, history := range histories {
		if err := historyRepo.Create(ctx, history); err != nil {
			t.Fatalf("Create() history error = %v", err)
		}
	}

	t.Cleanup(func() {
		deleteTestAcademicYear(t, db, secondAcademicYear.ID)
		deleteTestAcademicYear(t, db, academicYear.ID)

		deleteTestClass(t, db, classB.ID)
		deleteTestClass(t, db, classA.ID)

		deleteTestStudent(t, db, studentD.ID)
		deleteTestStudent(t, db, studentC.ID)
		deleteTestStudent(t, db, studentB.ID)
		deleteTestStudent(t, db, studentA.ID)
	})

	t.Cleanup(func() {
		for i := len(histories) - 1; i >= 0; i-- {
			deleteTestStudentClassHistory(t, db, histories[i].ID)
		}
	})

	got, err := studentRepo.ListByClassAndAcademicYear(
		ctx,
		classA.ID,
		academicYear.ID,
		repository.ListOptions{Limit: 10, Offset: 0},
	)
	if err != nil {
		t.Fatalf("ListByClassAndAcademicYear() error = %v", err)
	}

	if len(got) != 2 {
		t.Fatalf(
			"ListByClassAndAcademicYear() returned %d students, want 2",
			len(got),
		)
	}

	if got[0].ID != studentA.ID {
		t.Errorf("got[0].ID = %v, want %v", got[0].ID, studentA.ID)
	}

	if got[1].ID != studentB.ID {
		t.Errorf("got[1].ID = %v, want %v", got[1].ID, studentB.ID)
	}
}

func TestStudentRepository_ListByClassAndAcademicYearPagination(t *testing.T) {
	db := openTestDatabase(t)
	cleanupScopedRepositoryFixtures(t, db)
	studentRepo := postgres.NewStudentRepository(db)
	classRepo := postgres.NewClassRepository(db)
	academicYearRepo := postgres.NewAcademicYearRepository(db)
	historyRepo := postgres.NewStudentClassHistoryRepository(db)
	ctx := context.Background()

	academicYear := newTestAcademicYear(t)
	if err := academicYearRepo.Create(ctx, academicYear); err != nil {
		t.Fatalf("Create() academic year error = %v", err)
	}

	class := newTestClass(t, "7A-student-page", 7)
	if err := classRepo.Create(ctx, class); err != nil {
		t.Fatalf("Create() class error = %v", err)
	}

	first := newTestStudent(t, "Budi Page")
	second := newTestStudent(t, "Citra Page")

	if err := studentRepo.Create(ctx, first); err != nil {
		t.Fatalf("Create() first student error = %v", err)
	}
	if err := studentRepo.Create(ctx, second); err != nil {
		t.Fatalf("Create() second student error = %v", err)
	}

	histories := make([]domain.StudentClassHistory, 0, 2)

	for _, student := range []domain.Student{first, second} {
		history := newTestStudentClassHistory(
			t,
			student.ID,
			academicYear.ID,
			class.ID,
			academicYear.StartDate,
			academicYear.EndDate,
		)

		if err := historyRepo.Create(ctx, history); err != nil {
			t.Fatalf("Create() history error = %v", err)
		}

		histories = append(histories, history)
	}

	t.Cleanup(func() {
		deleteTestAcademicYear(t, db, academicYear.ID)
		deleteTestClass(t, db, class.ID)
		deleteTestStudent(t, db, second.ID)
		deleteTestStudent(t, db, first.ID)
	})

	t.Cleanup(func() {
		for i := len(histories) - 1; i >= 0; i-- {
			deleteTestStudentClassHistory(t, db, histories[i].ID)
		}
	})

	got, err := studentRepo.ListByClassAndAcademicYear(
		ctx,
		class.ID,
		academicYear.ID,
		repository.ListOptions{Limit: 1, Offset: 1},
	)
	if err != nil {
		t.Fatalf("ListByClassAndAcademicYear() error = %v", err)
	}

	if len(got) != 1 {
		t.Fatalf(
			"ListByClassAndAcademicYear() returned %d students, want 1",
			len(got),
		)
	}

	if got[0].ID != second.ID {
		t.Errorf("got[0].ID = %v, want %v", got[0].ID, second.ID)
	}
}

func assertStudentEqual(
	t *testing.T,
	got domain.Student,
	want domain.Student,
) {
	t.Helper()

	if got.ID != want.ID {
		t.Errorf("ID = %v, want %v", got.ID, want.ID)
	}

	assertOptionalUUIDEqual(t, "UserID", got.UserID, want.UserID)
	assertOptionalStringEqual(t, "NIS", got.NIS, want.NIS)
	assertOptionalStringEqual(t, "NISN", got.NISN, want.NISN)

	if got.Name != want.Name {
		t.Errorf("Name = %q, want %q", got.Name, want.Name)
	}

	if got.Status != want.Status {
		t.Errorf("Status = %q, want %q", got.Status, want.Status)
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

func assertOptionalUUIDEqual(
	t *testing.T,
	field string,
	got *uuid.UUID,
	want *uuid.UUID,
) {
	t.Helper()

	if got == nil && want == nil {
		return
	}

	if got == nil || want == nil {
		t.Errorf("%s = %v, want %v", field, got, want)
		return
	}

	if *got != *want {
		t.Errorf("%s = %v, want %v", field, *got, *want)
	}
}

func assertOptionalStringEqual(
	t *testing.T,
	field string,
	got *string,
	want *string,
) {
	t.Helper()

	if got == nil && want == nil {
		return
	}

	if got == nil || want == nil {
		t.Errorf("%s = %v, want %v", field, got, want)
		return
	}

	if *got != *want {
		t.Errorf("%s = %q, want %q", field, *got, *want)
	}
}

func newTestStudent(
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

func newTestStudentWithNIS(
	t *testing.T,
	name string,
	nis string,
) domain.Student {
	t.Helper()

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

func newTestStudentWithNISN(
	t *testing.T,
	name string,
	nisn string,
) domain.Student {
	t.Helper()

	nis := "NIS-" + uuid.NewString()
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

func newTestStudentWithUserID(
	t *testing.T,
	name string,
	userID uuid.UUID,
) domain.Student {
	t.Helper()

	nis := "NIS-" + uuid.NewString()
	nisn := "NISN-" + uuid.NewString()
	student, err := domain.NewStudent(
		name,
		&userID,
		&nis,
		&nisn,
	)
	if err != nil {
		t.Fatalf("NewStudent() error = %v", err)
	}

	return student
}

func deleteTestStudent(
	t *testing.T,
	db *sql.DB,
	id uuid.UUID,
) {
	t.Helper()

	_, err := db.ExecContext(
		context.Background(),
		"DELETE FROM students WHERE id = $1",
		id,
	)
	if err != nil {
		t.Errorf("delete test student: %v", err)
	}
}

func cleanupStudents(
	t *testing.T,
	db *sql.DB,
) {
	t.Helper()

	_, err := db.ExecContext(
		context.Background(),
		"DELETE FROM students",
	)
	if err != nil {
		t.Fatalf("cleanup students: %v", err)
	}
}
