package service

import (
	"context"
	"errors"
	"testing"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

type classServiceFakeClassRepository struct {
	repository.ClassRepository

	createFn               func(context.Context, domain.Class) error
	getByIDFn              func(context.Context, uuid.UUID) (domain.Class, error)
	listFn                 func(context.Context, repository.ListOptions) ([]domain.Class, error)
	listByTeacherAndYearFn func(context.Context, uuid.UUID, uuid.UUID, repository.ListOptions) ([]domain.Class, error)
	updateFn               func(context.Context, domain.Class) error
	existsByNameAndLevelFn func(context.Context, string, int) (bool, error)

	createCalls int
	updateCalls int
}

func (f *classServiceFakeClassRepository) Create(ctx context.Context, class domain.Class) error {
	f.createCalls++
	if f.createFn != nil {
		return f.createFn(ctx, class)
	}
	return nil
}

func (f *classServiceFakeClassRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.Class, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, id)
	}
	return domain.Class{}, repository.ErrNotFound
}

func (f *classServiceFakeClassRepository) List(ctx context.Context, options repository.ListOptions) ([]domain.Class, error) {
	if f.listFn != nil {
		return f.listFn(ctx, options)
	}
	return nil, nil
}

func (f *classServiceFakeClassRepository) ListByTeacherAndAcademicYear(
	ctx context.Context,
	userID uuid.UUID,
	academicYearID uuid.UUID,
	options repository.ListOptions,
) ([]domain.Class, error) {
	if f.listByTeacherAndYearFn != nil {
		return f.listByTeacherAndYearFn(ctx, userID, academicYearID, options)
	}
	return nil, nil
}

func (f *classServiceFakeClassRepository) Update(ctx context.Context, class domain.Class) error {
	f.updateCalls++
	if f.updateFn != nil {
		return f.updateFn(ctx, class)
	}
	return nil
}

func (f *classServiceFakeClassRepository) ExistsByNameAndLevel(
	ctx context.Context,
	name string,
	level int,
) (bool, error) {
	if f.existsByNameAndLevelFn != nil {
		return f.existsByNameAndLevelFn(ctx, name, level)
	}
	return false, nil
}

type classServiceFakeStudentRepository struct {
	repository.StudentRepository
	getByUserIDFn func(context.Context, uuid.UUID) (domain.Student, error)
}

func (f *classServiceFakeStudentRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (domain.Student, error) {
	if f.getByUserIDFn != nil {
		return f.getByUserIDFn(ctx, userID)
	}
	return domain.Student{}, repository.ErrNotFound
}

type classServiceFakeHistoryRepository struct {
	repository.StudentClassHistoryRepository
	listFn func(context.Context, uuid.UUID, uuid.UUID, repository.ListOptions) ([]domain.StudentClassHistory, error)
}

func (f *classServiceFakeHistoryRepository) ListByStudentAndAcademicYear(
	ctx context.Context,
	studentID uuid.UUID,
	academicYearID uuid.UUID,
	options repository.ListOptions,
) ([]domain.StudentClassHistory, error) {
	if f.listFn != nil {
		return f.listFn(ctx, studentID, academicYearID, options)
	}
	return nil, nil
}

type classServiceFakeAuditRepository struct {
	repository.AuditRepository
	createFn    func(context.Context, domain.AuditLog) error
	createCalls int
	lastAudit   domain.AuditLog
}

func (f *classServiceFakeAuditRepository) Create(ctx context.Context, audit domain.AuditLog) error {
	f.createCalls++
	f.lastAudit = audit
	if f.createFn != nil {
		return f.createFn(ctx, audit)
	}
	return nil
}

type classServiceFakeUOW struct {
	repos         repository.RepositorySet
	commitErr     error
	commitCalls   int
	rollbackCalls int
}

func (f *classServiceFakeUOW) Repositories() repository.RepositorySet { return f.repos }

func (f *classServiceFakeUOW) Commit() error {
	f.commitCalls++
	return f.commitErr
}

func (f *classServiceFakeUOW) Rollback() error {
	f.rollbackCalls++
	return nil
}

type classServiceFakeUOWManager struct {
	uow        repository.UnitOfWork
	beginErr   error
	beginCalls int
}

func (f *classServiceFakeUOWManager) Begin(context.Context) (repository.UnitOfWork, error) {
	f.beginCalls++
	if f.beginErr != nil {
		return nil, f.beginErr
	}
	return f.uow, nil
}

func newClassServiceForTest(t *testing.T, repos repository.RepositorySet, manager repository.UnitOfWorkManager) *classService {
	t.Helper()

	svc, err := NewClassService(Dependencies{
		Repositories: repos,
		UOW:          manager,
	})
	if err != nil {
		t.Fatalf("NewClassService() error = %v", err)
	}
	return svc
}

func testClass(t *testing.T, name string, level int) domain.Class {
	t.Helper()
	class, err := domain.NewClass(name, level)
	if err != nil {
		t.Fatalf("NewClass() error = %v", err)
	}
	return class
}

func TestClassServiceCreateAdmin(t *testing.T) {
	repo := &classServiceFakeClassRepository{}
	audit := &classServiceFakeAuditRepository{}
	uow := &classServiceFakeUOW{
		repos: repository.RepositorySet{Classes: repo, AuditLogs: audit},
	}
	svc := newClassServiceForTest(t, uow.repos, &classServiceFakeUOWManager{uow: uow})

	got, err := svc.Create(context.Background(), CreateClassInput{
		Actor: Actor{UserID: uuid.New(), Role: domain.RoleAdmin},
		Name:  "VII A",
		Level: 7,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if got.Class == nil || got.Class.Name != "VII A" || got.Class.Level != 7 {
		t.Fatalf("Create() class = %#v, want VII A/7", got.Class)
	}
	if repo.createCalls != 1 || audit.createCalls != 1 || uow.commitCalls != 1 || uow.rollbackCalls != 0 {
		t.Fatalf("calls = create %d audit %d commit %d rollback %d; want 1 1 1 0",
			repo.createCalls, audit.createCalls, uow.commitCalls, uow.rollbackCalls)
	}
	if audit.lastAudit.Action != "CREATE" || audit.lastAudit.EntityType != domain.AuditEntityClass {
		t.Fatalf("audit = %#v, want CREATE CLASS", audit.lastAudit)
	}
	if len(audit.lastAudit.AfterData) == 0 || len(audit.lastAudit.BeforeData) != 0 {
		t.Fatalf("audit before/after = %q/%q, want nil/non-empty", audit.lastAudit.BeforeData, audit.lastAudit.AfterData)
	}
}

func TestClassServiceCreateRejectsInvalidLevel(t *testing.T) {
	repo := &classServiceFakeClassRepository{}
	uow := &classServiceFakeUOW{repos: repository.RepositorySet{Classes: repo}}
	svc := newClassServiceForTest(t, uow.repos, &classServiceFakeUOWManager{uow: uow})

	_, err := svc.Create(context.Background(), CreateClassInput{
		Actor: Actor{UserID: uuid.New(), Role: domain.RoleAdmin},
		Name:  "VII A",
		Level: 13,
	})
	if !errors.Is(err, domain.ErrInvalidClassLevel) {
		t.Fatalf("Create() error = %v, want %v", err, domain.ErrInvalidClassLevel)
	}
	if repo.createCalls != 0 {
		t.Fatalf("Create() repository calls = %d, want 0", repo.createCalls)
	}
}

func TestClassServiceCreateRejectsDuplicate(t *testing.T) {
	repo := &classServiceFakeClassRepository{
		existsByNameAndLevelFn: func(context.Context, string, int) (bool, error) { return true, nil },
	}
	uow := &classServiceFakeUOW{repos: repository.RepositorySet{Classes: repo}}
	svc := newClassServiceForTest(t, uow.repos, &classServiceFakeUOWManager{uow: uow})

	_, err := svc.Create(context.Background(), CreateClassInput{
		Actor: Actor{UserID: uuid.New(), Role: domain.RoleAdmin},
		Name:  "VII A",
		Level: 7,
	})
	if !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("Create() error = %v, want %v", err, repository.ErrConflict)
	}
	if uow.commitCalls != 0 || uow.rollbackCalls != 1 {
		t.Fatalf("calls = commit %d rollback %d; want 0 1", uow.commitCalls, uow.rollbackCalls)
	}
}

func TestClassServiceCreateRejectsStudent(t *testing.T) {
	manager := &classServiceFakeUOWManager{}
	svc := newClassServiceForTest(t, repository.RepositorySet{}, manager)

	_, err := svc.Create(context.Background(), CreateClassInput{
		Actor: Actor{UserID: uuid.New(), Role: domain.RoleSiswa},
		Name:  "VII A",
		Level: 7,
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Create() error = %v, want %v", err, ErrForbidden)
	}
	if manager.beginCalls != 0 {
		t.Fatalf("Begin() calls = %d, want 0", manager.beginCalls)
	}
}

func TestClassServiceCreateRollsBackOnAuditFailure(t *testing.T) {
	repo := &classServiceFakeClassRepository{}
	audit := &classServiceFakeAuditRepository{
		createFn: func(context.Context, domain.AuditLog) error {
			return errors.New("audit failed")
		},
	}
	uow := &classServiceFakeUOW{
		repos: repository.RepositorySet{Classes: repo, AuditLogs: audit},
	}
	svc := newClassServiceForTest(t, uow.repos, &classServiceFakeUOWManager{uow: uow})

	_, err := svc.Create(context.Background(), CreateClassInput{
		Actor: Actor{UserID: uuid.New(), Role: domain.RoleAdmin},
		Name:  "VII A",
		Level: 7,
	})
	if err == nil {
		t.Fatal("Create() error = nil, want audit error")
	}
	if repo.createCalls != 1 || audit.createCalls != 1 || uow.commitCalls != 0 || uow.rollbackCalls != 1 {
		t.Fatalf("calls = create %d audit %d commit %d rollback %d; want 1 1 0 1",
			repo.createCalls, audit.createCalls, uow.commitCalls, uow.rollbackCalls)
	}
}

func TestClassServiceGetAdmin(t *testing.T) {
	class := testClass(t, "VII A", 7)
	repo := &classServiceFakeClassRepository{
		getByIDFn: func(context.Context, uuid.UUID) (domain.Class, error) { return class, nil },
	}
	svc := newClassServiceForTest(t, repository.RepositorySet{Classes: repo}, &classServiceFakeUOWManager{})

	got, err := svc.Get(context.Background(), GetClassInput{
		Actor:   Actor{UserID: uuid.New(), Role: domain.RoleAdmin},
		ClassID: class.ID,
	})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Class == nil || got.Class.ID != class.ID {
		t.Fatalf("Get() class = %#v, want %v", got.Class, class.ID)
	}
}

func TestClassServiceGetRejectsNonAdmin(t *testing.T) {
	svc := newClassServiceForTest(t, repository.RepositorySet{}, &classServiceFakeUOWManager{})

	_, err := svc.Get(context.Background(), GetClassInput{
		Actor:   Actor{UserID: uuid.New(), Role: domain.RoleWaliKelas},
		ClassID: uuid.New(),
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Get() error = %v, want %v", err, ErrForbidden)
	}
}

func TestClassServiceListAdmin(t *testing.T) {
	class := testClass(t, "VII A", 7)
	repo := &classServiceFakeClassRepository{
		listFn: func(context.Context, repository.ListOptions) ([]domain.Class, error) {
			return []domain.Class{class}, nil
		},
	}
	svc := newClassServiceForTest(t, repository.RepositorySet{Classes: repo}, &classServiceFakeUOWManager{})

	got, err := svc.List(context.Background(), ListClassesInput{
		Actor:          Actor{UserID: uuid.New(), Role: domain.RoleAdmin},
		AcademicYearID: uuid.New(),
		Options:        ListOptions{Limit: 10, Offset: 0},
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got.Classes) != 1 || got.Classes[0].ID != class.ID {
		t.Fatalf("List() classes = %#v, want class %v", got.Classes, class.ID)
	}
}

func TestClassServiceListWaliUsesTeacherScope(t *testing.T) {
	class := testClass(t, "VII A", 7)
	teacherID := uuid.New()
	yearID := uuid.New()
	repo := &classServiceFakeClassRepository{
		listByTeacherAndYearFn: func(
			context.Context, uuid.UUID, uuid.UUID, repository.ListOptions,
		) ([]domain.Class, error) {
			return []domain.Class{class}, nil
		},
	}
	svc := newClassServiceForTest(t, repository.RepositorySet{Classes: repo}, &classServiceFakeUOWManager{})

	got, err := svc.List(context.Background(), ListClassesInput{
		Actor:          Actor{UserID: teacherID, Role: domain.RoleWaliKelas},
		AcademicYearID: yearID,
		Options:        ListOptions{Limit: 10},
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got.Classes) != 1 || got.Classes[0].ID != class.ID {
		t.Fatalf("List() classes = %#v, want class %v", got.Classes, class.ID)
	}
}

func TestClassServiceListStudentUsesStudentHistory(t *testing.T) {
	studentID := uuid.New()
	userID := uuid.New()
	yearID := uuid.New()
	classA := testClass(t, "VII A", 7)
	classB := testClass(t, "VII B", 7)

	classRepo := &classServiceFakeClassRepository{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (domain.Class, error) {
			switch id {
			case classA.ID:
				return classA, nil
			case classB.ID:
				return classB, nil
			default:
				return domain.Class{}, repository.ErrNotFound
			}
		},
	}
	studentRepo := &classServiceFakeStudentRepository{
		getByUserIDFn: func(context.Context, uuid.UUID) (domain.Student, error) {
			return domain.Student{ID: studentID, UserID: &userID}, nil
		},
	}
	historyRepo := &classServiceFakeHistoryRepository{
		listFn: func(context.Context, uuid.UUID, uuid.UUID, repository.ListOptions) ([]domain.StudentClassHistory, error) {
			return []domain.StudentClassHistory{
				{ID: uuid.New(), StudentID: studentID, AcademicYearID: yearID, ClassID: classA.ID},
				{ID: uuid.New(), StudentID: studentID, AcademicYearID: yearID, ClassID: classB.ID},
			}, nil
		},
	}
	repos := repository.RepositorySet{
		Classes:               classRepo,
		Students:              studentRepo,
		StudentClassHistories: historyRepo,
	}
	svc := newClassServiceForTest(t, repos, &classServiceFakeUOWManager{})

	got, err := svc.List(context.Background(), ListClassesInput{
		Actor:          Actor{UserID: userID, Role: domain.RoleSiswa},
		AcademicYearID: yearID,
		Options:        ListOptions{Limit: 10},
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got.Classes) != 2 {
		t.Fatalf("List() classes count = %d, want 2", len(got.Classes))
	}
}

func TestClassServiceUpdateAdmin(t *testing.T) {
	existing := testClass(t, "VII A", 7)
	repo := &classServiceFakeClassRepository{
		getByIDFn: func(context.Context, uuid.UUID) (domain.Class, error) { return existing, nil },
	}
	audit := &classServiceFakeAuditRepository{}
	uow := &classServiceFakeUOW{
		repos: repository.RepositorySet{Classes: repo, AuditLogs: audit},
	}
	svc := newClassServiceForTest(t, uow.repos, &classServiceFakeUOWManager{uow: uow})

	got, err := svc.Update(context.Background(), UpdateClassInput{
		Actor:   Actor{UserID: uuid.New(), Role: domain.RoleAdmin},
		ClassID: existing.ID,
		Name:    "VII B",
		Level:   7,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if got.Class.Name != "VII B" || got.Class.Level != 7 {
		t.Fatalf("Update() class = %#v, want VII B/7", got.Class)
	}
	if repo.updateCalls != 1 || audit.createCalls != 1 || uow.commitCalls != 1 || uow.rollbackCalls != 0 {
		t.Fatalf("calls = update %d audit %d commit %d rollback %d; want 1 1 1 0",
			repo.updateCalls, audit.createCalls, uow.commitCalls, uow.rollbackCalls)
	}
	if audit.lastAudit.Action != "UPDATE" || len(audit.lastAudit.BeforeData) == 0 || len(audit.lastAudit.AfterData) == 0 {
		t.Fatalf("audit = %#v, want UPDATE with before/after", audit.lastAudit)
	}
}

func TestClassServiceUpdateAllowsUnchangedNameAndLevel(t *testing.T) {
	existing := testClass(t, "VII A", 7)
	repo := &classServiceFakeClassRepository{
		getByIDFn: func(context.Context, uuid.UUID) (domain.Class, error) { return existing, nil },
		existsByNameAndLevelFn: func(context.Context, string, int) (bool, error) {
			t.Fatal("ExistsByNameAndLevel() must not be called when name/level are unchanged")
			return false, nil
		},
	}
	audit := &classServiceFakeAuditRepository{}
	uow := &classServiceFakeUOW{
		repos: repository.RepositorySet{Classes: repo, AuditLogs: audit},
	}
	svc := newClassServiceForTest(t, uow.repos, &classServiceFakeUOWManager{uow: uow})

	if _, err := svc.Update(context.Background(), UpdateClassInput{
		Actor:   Actor{UserID: uuid.New(), Role: domain.RoleAdmin},
		ClassID: existing.ID,
		Name:    existing.Name,
		Level:   existing.Level,
	}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
}

func TestClassServiceUpdateRejectsDuplicate(t *testing.T) {
	existing := testClass(t, "VII A", 7)
	repo := &classServiceFakeClassRepository{
		getByIDFn:              func(context.Context, uuid.UUID) (domain.Class, error) { return existing, nil },
		existsByNameAndLevelFn: func(context.Context, string, int) (bool, error) { return true, nil },
	}
	uow := &classServiceFakeUOW{repos: repository.RepositorySet{Classes: repo}}
	svc := newClassServiceForTest(t, uow.repos, &classServiceFakeUOWManager{uow: uow})

	_, err := svc.Update(context.Background(), UpdateClassInput{
		Actor:   Actor{UserID: uuid.New(), Role: domain.RoleAdmin},
		ClassID: existing.ID,
		Name:    "VII B",
		Level:   7,
	})
	if !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("Update() error = %v, want %v", err, repository.ErrConflict)
	}
	if repo.updateCalls != 0 || uow.commitCalls != 0 || uow.rollbackCalls != 1 {
		t.Fatalf("calls = update %d commit %d rollback %d; want 0 0 1", repo.updateCalls, uow.commitCalls, uow.rollbackCalls)
	}
}

func TestClassServiceUpdateRejectsInvalidLevel(t *testing.T) {
	existing := testClass(t, "VII A", 7)
	repo := &classServiceFakeClassRepository{
		getByIDFn: func(context.Context, uuid.UUID) (domain.Class, error) { return existing, nil },
	}
	uow := &classServiceFakeUOW{repos: repository.RepositorySet{Classes: repo}}
	svc := newClassServiceForTest(t, uow.repos, &classServiceFakeUOWManager{uow: uow})

	_, err := svc.Update(context.Background(), UpdateClassInput{
		Actor:   Actor{UserID: uuid.New(), Role: domain.RoleAdmin},
		ClassID: existing.ID,
		Name:    "VII A",
		Level:   13,
	})
	if !errors.Is(err, domain.ErrInvalidClassLevel) {
		t.Fatalf("Update() error = %v, want %v", err, domain.ErrInvalidClassLevel)
	}
	if repo.updateCalls != 0 {
		t.Fatalf("Update() repository calls = %d, want 0", repo.updateCalls)
	}
}

func TestClassServiceUpdateRejectsStudent(t *testing.T) {
	manager := &classServiceFakeUOWManager{}
	svc := newClassServiceForTest(t, repository.RepositorySet{}, manager)

	_, err := svc.Update(context.Background(), UpdateClassInput{
		Actor:   Actor{UserID: uuid.New(), Role: domain.RoleSiswa},
		ClassID: uuid.New(),
		Name:    "VII A",
		Level:   7,
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Update() error = %v, want %v", err, ErrForbidden)
	}
	if manager.beginCalls != 0 {
		t.Fatalf("Begin() calls = %d, want 0", manager.beginCalls)
	}
}

func TestClassServiceListRejectsInvalidAcademicYear(t *testing.T) {
	svc := newClassServiceForTest(t, repository.RepositorySet{}, &classServiceFakeUOWManager{})

	_, err := svc.List(context.Background(), ListClassesInput{
		Actor: Actor{UserID: uuid.New(), Role: domain.RoleAdmin},
	})
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("List() error = %v, want %v", err, domain.ErrInvalidID)
	}
}

func TestClassServiceListRejectsUnknownRole(t *testing.T) {
	svc := newClassServiceForTest(t, repository.RepositorySet{}, &classServiceFakeUOWManager{})

	_, err := svc.List(context.Background(), ListClassesInput{
		Actor:          Actor{UserID: uuid.New(), Role: domain.UserRole("INVALID")},
		AcademicYearID: uuid.New(),
	})
	if !errors.Is(err, ErrInvalidActor) {
		t.Fatalf("List() error = %v, want %v", err, ErrInvalidActor)
	}
}

func TestClassServiceGetRejectsInvalidID(t *testing.T) {
	svc := newClassServiceForTest(t, repository.RepositorySet{}, &classServiceFakeUOWManager{})

	_, err := svc.Get(context.Background(), GetClassInput{
		Actor:   Actor{UserID: uuid.New(), Role: domain.RoleAdmin},
		ClassID: uuid.Nil,
	})
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("Get() error = %v, want %v", err, domain.ErrInvalidID)
	}
}

func TestClassServiceCreateParticipatesInOuterTransaction(t *testing.T) {
	repo := &classServiceFakeClassRepository{}
	audit := &classServiceFakeAuditRepository{}
	uow := &classServiceFakeUOW{
		repos: repository.RepositorySet{Classes: repo, AuditLogs: audit},
	}
	manager := &classServiceFakeUOWManager{uow: uow}
	svc := newClassServiceForTest(t, uow.repos, manager)

	err := WithinTransaction(context.Background(), manager, func(ctx context.Context, _ repository.RepositorySet) error {
		_, err := svc.Create(ctx, CreateClassInput{
			Actor: Actor{UserID: uuid.New(), Role: domain.RoleAdmin},
			Name:  "VII A",
			Level: 7,
		})
		return err
	})
	if err != nil {
		t.Fatalf("WithinTransaction() error = %v", err)
	}
	if manager.beginCalls != 1 || uow.commitCalls != 1 || uow.rollbackCalls != 0 {
		t.Fatalf("calls = begin %d commit %d rollback %d; want 1 1 0",
			manager.beginCalls, uow.commitCalls, uow.rollbackCalls)
	}
}
