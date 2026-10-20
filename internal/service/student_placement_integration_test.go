package service_test

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/repository/postgres"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

type studentPlacementFixture struct {
	db                *sql.DB
	repos             repository.RepositorySet
	manager           repository.UnitOfWorkManager
	students          service.StudentService
	classes           service.ClassService
	actor             service.Actor
	year, otherYear   domain.AcademicYear
	class, otherClass domain.Class
}

func newStudentPlacementFixture(t *testing.T) studentPlacementFixture {
	t.Helper()
	db := openServiceScopeTestDatabase(t)
	cleanupServiceScopeFixtures(t, db)
	t.Cleanup(func() { cleanupServiceScopeFixtures(t, db) })
	f := studentPlacementFixture{db: db, repos: postgres.NewRepositorySet(db), manager: postgres.NewUnitOfWorkManager(db)}
	user := newServiceScopeUser(t, domain.RoleAdmin)
	if err := f.repos.Users.Create(context.Background(), user); err != nil {
		t.Fatal(err)
	}
	f.actor = service.Actor{UserID: user.ID, Role: domain.RoleAdmin}
	f.otherYear = newServiceScopeAcademicYearForStartYear(t, "2026/2027", 2026)
	f.otherYear.Status = domain.AcademicYearClosed
	f.year = newServiceScopeAcademicYearForStartYear(t, "2027/2028", 2027)
	for _, year := range []domain.AcademicYear{f.otherYear, f.year} {
		if err := f.repos.AcademicYears.Create(context.Background(), year); err != nil {
			t.Fatal(err)
		}
	}
	var err error
	deps := service.Dependencies{Repositories: f.repos, UOW: f.manager}
	f.students, err = service.NewStudentService(deps)
	if err != nil {
		t.Fatal(err)
	}
	f.classes, err = service.NewClassService(deps)
	if err != nil {
		t.Fatal(err)
	}
	for i, year := range []domain.AcademicYear{f.otherYear, f.year} {
		output, err := f.classes.Create(context.Background(), service.CreateClassInput{Actor: f.actor, Name: "VII A", Level: 7, AcademicYearID: year.ID, EffectiveDate: year.StartDate.AddDate(0, 0, 14)})
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			f.otherClass = *output.Class
		} else {
			f.class = *output.Class
		}
	}
	return f
}

func (f studentPlacementFixture) createInput(nis string) service.CreateStudentInput {
	nisn := "00" + nis
	return service.CreateStudentInput{Actor: f.actor, Name: "Siswa " + nis, NIS: &nis, NISN: &nisn, AcademicYearID: f.year.ID, ClassID: f.class.ID}
}

func (f studentPlacementFixture) importRow(row int, nis string) service.StudentImportRow {
	return service.StudentImportRow{SpreadsheetRow: row, Name: "Siswa " + nis, NIS: nis, NISN: "00" + nis, AcademicYear: f.year.Name, Class: f.class.Name}
}

func TestStudentPlacementIntegrationManualAndDirectory(t *testing.T) {
	f := newStudentPlacementFixture(t)
	ctx := context.Background()
	output, err := f.students.Create(ctx, f.createInput("1001"))
	if err != nil {
		t.Fatal(err)
	}
	if output.Student.UserID != nil {
		t.Fatal("manual student unexpectedly has a login account")
	}
	histories, err := f.repos.StudentClassHistories.ListByStudentAndAcademicYear(ctx, output.Student.ID, f.year.ID, repository.ListOptions{Limit: 10})
	if err != nil || len(histories) != 1 {
		t.Fatalf("histories=%+v err=%v", histories, err)
	}
	history := histories[0]
	if history.ClassID != f.class.ID || !history.Period.From.Equal(f.class.EffectiveDate) || history.Period.To == nil || !history.Period.To.Equal(f.year.EndDate) {
		t.Fatalf("placement=%+v", history)
	}
	// A changed configuration affects the next placement; existing history is immutable.
	updatedDate := f.class.EffectiveDate.AddDate(0, 0, 1)
	_, err = f.classes.Update(ctx, service.UpdateClassInput{Actor: f.actor, ClassID: f.class.ID, Name: f.class.Name, Level: 7, AcademicYearID: f.year.ID, EffectiveDate: updatedDate})
	if err != nil {
		t.Fatal(err)
	}
	unchanged, err := f.repos.StudentClassHistories.GetByID(ctx, history.ID)
	if err != nil || !unchanged.Period.From.Equal(history.Period.From) {
		t.Fatalf("existing history changed: %+v %v", unchanged, err)
	}
	// The same student may be placed in a later year using that year's configured date.
	_, err = f.students.AssignClass(ctx, service.AssignStudentClassInput{Actor: f.actor, StudentID: output.Student.ID, AcademicYearID: f.otherYear.ID, ClassID: f.otherClass.ID, EffectiveDate: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		year     uuid.UUID
		expected domain.Class
	}{{uuid.Nil, f.class}, {f.year.ID, f.class}} {
		directory, err := f.students.List(ctx, service.ListStudentsInput{Actor: f.actor, IncludePlacement: true, AcademicYearID: tc.year, Options: service.ListOptions{Limit: 10}})
		if err != nil || len(directory.Students) != 1 {
			t.Fatalf("directory=%+v err=%v", directory, err)
		}
		placement := directory.Placements[output.Student.ID]
		if placement.ClassID != tc.expected.ID || placement.AcademicYearID != tc.expected.AcademicYearID || placement.Class != "VII A" {
			t.Fatalf("directory placement=%+v", placement)
		}
	}
	wrong := f.createInput("1002")
	wrong.ClassID = f.otherClass.ID
	if _, err := f.students.Create(ctx, wrong); !errors.Is(err, domain.ErrInvalidValue) {
		t.Fatalf("cross-year error=%v", err)
	}
	if _, err := f.repos.Students.GetByNIS(ctx, "1002"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("invalid placement saved a student: %v", err)
	}
	// EndDate is exclusive for both service and database validation.
	_, err = f.classes.Create(ctx, service.CreateClassInput{Actor: f.actor, Name: "Invalid date", Level: 7, AcademicYearID: f.year.ID, EffectiveDate: f.year.EndDate})
	if !errors.Is(err, domain.ErrInvalidDateRange) {
		t.Fatalf("invalid date error=%v", err)
	}
}

func TestStudentPlacementIntegrationImportValidatesWholeWorkbook(t *testing.T) {
	f := newStudentPlacementFixture(t)
	ctx := context.Background()
	if _, err := f.students.Create(ctx, f.createInput("2001")); err != nil {
		t.Fatal(err)
	}
	rows := []service.StudentImportRow{f.importRow(2, "2002"), f.importRow(5, "2001"), f.importRow(9, "2002"), f.importRow(12, "2003")}
	rows[3].Class = "Kelas tidak tersedia"
	result, err := f.students.Import(ctx, service.ImportStudentsInput{Actor: f.actor, Rows: rows})
	if err != nil {
		t.Fatal(err)
	}
	messages := map[int]string{}
	for _, issue := range result.Issues {
		messages[issue.SpreadsheetRow] += " " + issue.Message
	}
	if !strings.Contains(messages[5], "sudah digunakan") || !strings.Contains(messages[9], "baris 2") || !strings.Contains(messages[12], "Kelas") {
		t.Fatalf("issues=%+v", result.Issues)
	}
	for _, nis := range []string{"2002", "2003"} {
		if _, err := f.repos.Students.GetByNIS(ctx, nis); !errors.Is(err, repository.ErrNotFound) {
			t.Fatalf("invalid import partially saved %s: %v", nis, err)
		}
	}
	valid := []service.StudentImportRow{f.importRow(2, "2002"), f.importRow(3, "2003")}
	valid[1].AcademicYear = f.otherYear.Name
	result, err = f.students.Import(ctx, service.ImportStudentsInput{Actor: f.actor, Rows: valid})
	if err != nil || len(result.Issues) != 0 || len(result.Students) != 2 {
		t.Fatalf("valid import=%+v err=%v", result, err)
	}
	for i, student := range result.Students {
		year, class := f.year, f.class
		if i == 1 {
			year, class = f.otherYear, f.otherClass
		}
		histories, err := f.repos.StudentClassHistories.ListByStudentAndAcademicYear(ctx, student.ID, year.ID, repository.ListOptions{Limit: 10})
		if err != nil || len(histories) != 1 || histories[0].ClassID != class.ID || !histories[0].Period.From.Equal(class.EffectiveDate) {
			t.Fatalf("imported history=%+v err=%v", histories, err)
		}
		if student.UserID != nil {
			t.Fatal("import created a login account")
		}
	}
}

// Failure injection keeps real PostgreSQL student writes and transaction
// boundaries while failing history persistence after an earlier row was saved.
type failingHistoryRepository struct {
	repository.StudentClassHistoryRepository
	failAt, calls int
}

var errPlacementWrite = errors.New("history persistence failed")

func (r *failingHistoryRepository) Create(ctx context.Context, h domain.StudentClassHistory) error {
	r.calls++
	if r.calls == r.failAt {
		return errPlacementWrite
	}
	return r.StudentClassHistoryRepository.Create(ctx, h)
}

type placementFailureManager struct {
	repository.UnitOfWorkManager
	failAt int
}
type placementFailureUOW struct {
	repository.UnitOfWork
	repos repository.RepositorySet
}

func (u placementFailureUOW) Repositories() repository.RepositorySet { return u.repos }
func (m placementFailureManager) Begin(ctx context.Context) (repository.UnitOfWork, error) {
	u, err := m.UnitOfWorkManager.Begin(ctx)
	if err != nil {
		return nil, err
	}
	repos := u.Repositories()
	repos.StudentClassHistories = &failingHistoryRepository{StudentClassHistoryRepository: repos.StudentClassHistories, failAt: m.failAt}
	return placementFailureUOW{UnitOfWork: u, repos: repos}, nil
}
func TestStudentPlacementIntegrationRollsBackHistoryWriteFailures(t *testing.T) {
	f := newStudentPlacementFixture(t)
	ctx := context.Background()
	for _, batch := range []bool{false, true} {
		failAt := 1
		if batch {
			failAt = 2
		}
		svc, err := service.NewStudentService(service.Dependencies{Repositories: f.repos, UOW: placementFailureManager{UnitOfWorkManager: f.manager, failAt: failAt}})
		if err != nil {
			t.Fatal(err)
		}
		if batch {
			_, err = svc.Import(ctx, service.ImportStudentsInput{Actor: f.actor, Rows: []service.StudentImportRow{f.importRow(2, "3001"), f.importRow(3, "3002")}})
		} else {
			_, err = svc.Create(ctx, f.createInput("3001"))
		}
		if !errors.Is(err, errPlacementWrite) {
			t.Fatalf("write error=%v", err)
		}
		for _, table := range []string{"students", "student_class_histories"} {
			var count int
			if err := f.db.QueryRowContext(ctx, "SELECT count(*) FROM "+table).Scan(&count); err != nil {
				t.Fatal(err)
			}
			if count != 0 {
				t.Fatalf("%s contains %d partially saved rows", table, count)
			}
		}
	}
}
