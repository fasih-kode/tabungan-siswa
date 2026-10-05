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

var errClassAssignmentFakeRepository = errors.New("fake repository error")

// ============================================================
// FAKES
// ============================================================

type classAssignmentFakeUserRepository struct {
	repository.UserRepository

	getByIDFn func(context.Context, uuid.UUID) (domain.User, error)
}

func (f *classAssignmentFakeUserRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (domain.User, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, id)
	}

	return domain.User{}, repository.ErrNotFound
}

type classAssignmentFakeAcademicYearRepository struct {
	repository.AcademicYearRepository

	getByIDFn func(context.Context, uuid.UUID) (domain.AcademicYear, error)
}

func (f *classAssignmentFakeAcademicYearRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (domain.AcademicYear, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, id)
	}

	return domain.AcademicYear{}, repository.ErrNotFound
}

type classAssignmentFakeClassRepository struct {
	repository.ClassRepository

	getByIDFn func(context.Context, uuid.UUID) (domain.Class, error)
}

func (f *classAssignmentFakeClassRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (domain.Class, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, id)
	}

	return domain.Class{}, repository.ErrNotFound
}

type classAssignmentFakeStudentRepository struct {
	repository.StudentRepository

	getByUserIDFn func(context.Context, uuid.UUID) (domain.Student, error)
}

func (f *classAssignmentFakeStudentRepository) GetByUserID(
	ctx context.Context,
	userID uuid.UUID,
) (domain.Student, error) {
	if f.getByUserIDFn != nil {
		return f.getByUserIDFn(ctx, userID)
	}

	return domain.Student{}, repository.ErrNotFound
}

type classAssignmentFakeStudentHistoryRepository struct {
	repository.StudentClassHistoryRepository

	listByStudentAndAcademicYearFn func(
		context.Context,
		uuid.UUID,
		uuid.UUID,
		repository.ListOptions,
	) ([]domain.StudentClassHistory, error)

	getCurrentByStudentAndAcademicYearFn func(
		context.Context,
		uuid.UUID,
		uuid.UUID,
		time.Time,
	) (domain.StudentClassHistory, error)
}

func (f *classAssignmentFakeStudentHistoryRepository) ListByStudentAndAcademicYear(
	ctx context.Context,
	studentID uuid.UUID,
	academicYearID uuid.UUID,
	options repository.ListOptions,
) ([]domain.StudentClassHistory, error) {
	if f.listByStudentAndAcademicYearFn != nil {
		return f.listByStudentAndAcademicYearFn(
			ctx,
			studentID,
			academicYearID,
			options,
		)
	}

	return nil, nil
}

func (f *classAssignmentFakeStudentHistoryRepository) GetCurrentByStudentAndAcademicYear(
	ctx context.Context,
	studentID uuid.UUID,
	academicYearID uuid.UUID,
	date time.Time,
) (domain.StudentClassHistory, error) {
	if f.getCurrentByStudentAndAcademicYearFn != nil {
		return f.getCurrentByStudentAndAcademicYearFn(
			ctx,
			studentID,
			academicYearID,
			date,
		)
	}

	return domain.StudentClassHistory{}, repository.ErrNotFound
}

type classAssignmentFakeTeacherAssignmentRepository struct {
	repository.TeacherClassAssignmentRepository

	createFn func(
		context.Context,
		domain.TeacherClassAssignment,
	) error

	getByIDFn func(
		context.Context,
		uuid.UUID,
	) (domain.TeacherClassAssignment, error)

	listByUserAndAcademicYearFn func(
		context.Context,
		uuid.UUID,
		uuid.UUID,
		repository.ListOptions,
	) ([]domain.TeacherClassAssignment, error)

	listByClassAndAcademicYearFn func(
		context.Context,
		uuid.UUID,
		uuid.UUID,
		repository.ListOptions,
	) ([]domain.TeacherClassAssignment, error)

	getCurrentByClassAndAcademicYearFn func(
		context.Context,
		uuid.UUID,
		uuid.UUID,
		time.Time,
	) (domain.TeacherClassAssignment, error)
}

func (f *classAssignmentFakeTeacherAssignmentRepository) Create(
	ctx context.Context,
	assignment domain.TeacherClassAssignment,
) error {
	if f.createFn != nil {
		return f.createFn(ctx, assignment)
	}

	return nil
}

func (f *classAssignmentFakeTeacherAssignmentRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (domain.TeacherClassAssignment, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, id)
	}

	return domain.TeacherClassAssignment{}, repository.ErrNotFound
}

func (f *classAssignmentFakeTeacherAssignmentRepository) ListByUserAndAcademicYear(
	ctx context.Context,
	userID uuid.UUID,
	academicYearID uuid.UUID,
	options repository.ListOptions,
) ([]domain.TeacherClassAssignment, error) {
	if f.listByUserAndAcademicYearFn != nil {
		return f.listByUserAndAcademicYearFn(
			ctx,
			userID,
			academicYearID,
			options,
		)
	}

	return nil, nil
}

func (f *classAssignmentFakeTeacherAssignmentRepository) ListByClassAndAcademicYear(
	ctx context.Context,
	classID uuid.UUID,
	academicYearID uuid.UUID,
	options repository.ListOptions,
) ([]domain.TeacherClassAssignment, error) {
	if f.listByClassAndAcademicYearFn != nil {
		return f.listByClassAndAcademicYearFn(
			ctx,
			classID,
			academicYearID,
			options,
		)
	}

	return nil, nil
}

func (f *classAssignmentFakeTeacherAssignmentRepository) GetCurrentByClassAndAcademicYear(
	ctx context.Context,
	classID uuid.UUID,
	academicYearID uuid.UUID,
	date time.Time,
) (domain.TeacherClassAssignment, error) {
	if f.getCurrentByClassAndAcademicYearFn != nil {
		return f.getCurrentByClassAndAcademicYearFn(
			ctx,
			classID,
			academicYearID,
			date,
		)
	}

	return domain.TeacherClassAssignment{}, repository.ErrNotFound
}

type classAssignmentFakeUnitOfWork struct {
	repos         repository.RepositorySet
	commitErr     error
	rollbackErr   error
	commitCount   int
	rollbackCount int
}

func (f *classAssignmentFakeUnitOfWork) Repositories() repository.RepositorySet {
	return f.repos
}

func (f *classAssignmentFakeUnitOfWork) Commit() error {
	f.commitCount++
	return f.commitErr
}

func (f *classAssignmentFakeUnitOfWork) Rollback() error {
	f.rollbackCount++
	return f.rollbackErr
}

type classAssignmentFakeUnitOfWorkManager struct {
	uow        repository.UnitOfWork
	beginErr   error
	beginCount int
}

func (f *classAssignmentFakeUnitOfWorkManager) Begin(
	context.Context,
) (repository.UnitOfWork, error) {
	f.beginCount++

	if f.beginErr != nil {
		return nil, f.beginErr
	}

	return f.uow, nil
}

func newClassAssignmentServiceForTest(
	t *testing.T,
	repos repository.RepositorySet,
	uowManager repository.UnitOfWorkManager,
) *classAssignmentService {
	t.Helper()

	svc, err := NewClassAssignmentService(Dependencies{
		Repositories: repos,
		UOW:          uowManager,
	})
	if err != nil {
		t.Fatalf("NewClassAssignmentService() error = %v", err)
	}

	return svc
}

func newTestTeacherAssignment(
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

// ============================================================
// CONSTRUCTOR
// ============================================================

func TestNewClassAssignmentService(t *testing.T) {
	uowManager := &classAssignmentFakeUnitOfWorkManager{}

	svc, err := NewClassAssignmentService(Dependencies{
		UOW: uowManager,
	})
	if err != nil {
		t.Fatalf("NewClassAssignmentService() error = %v", err)
	}

	if svc == nil {
		t.Fatal("NewClassAssignmentService() service = nil")
	}

	if _, err := NewClassAssignmentService(Dependencies{}); !errors.Is(
		err,
		ErrInvalidDependency,
	) {
		t.Fatalf(
			"NewClassAssignmentService() error = %v, want %v",
			err,
			ErrInvalidDependency,
		)
	}
}

// ============================================================
// ASSIGN
// ============================================================

func TestClassAssignmentServiceAssignAdmin(t *testing.T) {
	actorID := uuid.New()
	teacherID := uuid.New()
	classID := uuid.New()
	academicYearID := uuid.New()

	teacher := domain.User{
		ID:       teacherID,
		Username: "wali",
		Role:     domain.RoleWaliKelas,
	}

	class := domain.Class{
		ID:    classID,
		Name:  "7A",
		Level: 7,
	}

	academicYear := newTestAcademicYear(academicYearID, domain.AcademicYearOpen)

	var created domain.TeacherClassAssignment

	repos := repository.RepositorySet{
		Users: &classAssignmentFakeUserRepository{
			getByIDFn: func(
				_ context.Context,
				id uuid.UUID,
			) (domain.User, error) {
				if id != teacherID {
					t.Fatalf("GetByID() user ID = %v, want %v", id, teacherID)
				}

				return teacher, nil
			},
		},
		Classes: &classAssignmentFakeClassRepository{
			getByIDFn: func(
				_ context.Context,
				id uuid.UUID,
			) (domain.Class, error) {
				if id != classID {
					t.Fatalf("GetByID() class ID = %v, want %v", id, classID)
				}

				return class, nil
			},
		},
		AcademicYears: &classAssignmentFakeAcademicYearRepository{
			getByIDFn: func(
				_ context.Context,
				id uuid.UUID,
			) (domain.AcademicYear, error) {
				if id != academicYearID {
					t.Fatalf(
						"GetByID() academic year ID = %v, want %v",
						id,
						academicYearID,
					)
				}

				return academicYear, nil
			},
		},
	}

	uowRepos := repos
	uowRepos.TeacherClassAssignments =
		&classAssignmentFakeTeacherAssignmentRepository{
			createFn: func(
				_ context.Context,
				assignment domain.TeacherClassAssignment,
			) error {
				created = assignment
				return nil
			},
		}

	uow := &classAssignmentFakeUnitOfWork{
		repos: uowRepos,
	}

	svc := newClassAssignmentServiceForTest(
		t,
		repos,
		&classAssignmentFakeUnitOfWorkManager{uow: uow},
	)

	got, err := svc.Assign(
		context.Background(),
		AssignTeacherInput{
			Actor: Actor{
				UserID: actorID,
				Role:   domain.RoleAdmin,
			},
			UserID:         teacherID,
			ClassID:        classID,
			AcademicYearID: academicYearID,
		},
	)
	if err != nil {
		t.Fatalf("Assign() error = %v", err)
	}

	if got.Assignment == nil {
		t.Fatal("Assign() assignment = nil")
	}

	if created.ID != got.Assignment.ID {
		t.Fatal("Assign() did not return persisted assignment")
	}

	if created.UserID != teacherID {
		t.Fatalf("created UserID = %v, want %v", created.UserID, teacherID)
	}

	if created.ClassID != classID {
		t.Fatalf("created ClassID = %v, want %v", created.ClassID, classID)
	}

	if !created.Period.From.Equal(academicYear.StartDate) {
		t.Fatalf(
			"Period.From = %v, want %v",
			created.Period.From,
			academicYear.StartDate,
		)
	}

	if created.Period.To == nil ||
		!created.Period.To.Equal(academicYear.EndDate) {
		t.Fatalf("Period.To = %v, want %v", created.Period.To, academicYear.EndDate)
	}

	if uow.commitCount != 1 {
		t.Fatalf("Commit() count = %d, want 1", uow.commitCount)
	}

	if uow.rollbackCount != 0 {
		t.Fatalf("Rollback() count = %d, want 0", uow.rollbackCount)
	}
}

func TestClassAssignmentServiceAssignNonAdminRejected(t *testing.T) {
	repos := repository.RepositorySet{}

	svc := newClassAssignmentServiceForTest(
		t,
		repos,
		&classAssignmentFakeUnitOfWorkManager{},
	)

	_, err := svc.Assign(
		context.Background(),
		AssignTeacherInput{
			Actor: Actor{
				UserID: uuid.New(),
				Role:   domain.RoleWaliKelas,
			},
			UserID:         uuid.New(),
			ClassID:        uuid.New(),
			AcademicYearID: uuid.New(),
		},
	)

	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Assign() error = %v, want ErrForbidden", err)
	}
}

func TestClassAssignmentServiceAssignRejectsNonTeacher(t *testing.T) {
	teacherID := uuid.New()
	classID := uuid.New()
	academicYearID := uuid.New()

	repos := repository.RepositorySet{
		Users: &classAssignmentFakeUserRepository{
			getByIDFn: func(
				_ context.Context,
				_ uuid.UUID,
			) (domain.User, error) {
				return domain.User{
					ID:   teacherID,
					Role: domain.RoleSiswa,
				}, nil
			},
		},
	}

	svc := newClassAssignmentServiceForTest(
		t,
		repos,
		&classAssignmentFakeUnitOfWorkManager{},
	)

	_, err := svc.Assign(
		context.Background(),
		AssignTeacherInput{
			Actor: Actor{
				UserID: uuid.New(),
				Role:   domain.RoleAdmin,
			},
			UserID:         teacherID,
			ClassID:        classID,
			AcademicYearID: academicYearID,
		},
	)

	if !errors.Is(err, ErrInvalidDependency) {
		t.Fatalf(
			"Assign() error = %v, want ErrInvalidDependency",
			err,
		)
	}
}

func TestClassAssignmentServiceAssignCreateErrorRollsBack(t *testing.T) {
	teacherID := uuid.New()
	classID := uuid.New()
	academicYearID := uuid.New()

	academicYear := newTestAcademicYear(academicYearID, domain.AcademicYearOpen)

	repos := repository.RepositorySet{
		Users: &classAssignmentFakeUserRepository{
			getByIDFn: func(
				_ context.Context,
				_ uuid.UUID,
			) (domain.User, error) {
				return domain.User{
					ID:   teacherID,
					Role: domain.RoleWaliKelas,
				}, nil
			},
		},
		Classes: &classAssignmentFakeClassRepository{
			getByIDFn: func(
				_ context.Context,
				_ uuid.UUID,
			) (domain.Class, error) {
				return domain.Class{
					ID:    classID,
					Name:  "7A",
					Level: 7,
				}, nil
			},
		},
		AcademicYears: &classAssignmentFakeAcademicYearRepository{
			getByIDFn: func(
				_ context.Context,
				_ uuid.UUID,
			) (domain.AcademicYear, error) {
				return academicYear, nil
			},
		},
	}

	uowRepos := repos
	uowRepos.TeacherClassAssignments =
		&classAssignmentFakeTeacherAssignmentRepository{
			createFn: func(
				_ context.Context,
				_ domain.TeacherClassAssignment,
			) error {
				return errClassAssignmentFakeRepository
			},
		}

	uow := &classAssignmentFakeUnitOfWork{
		repos: uowRepos,
	}

	svc := newClassAssignmentServiceForTest(
		t,
		repos,
		&classAssignmentFakeUnitOfWorkManager{uow: uow},
	)

	_, err := svc.Assign(
		context.Background(),
		AssignTeacherInput{
			Actor: Actor{
				UserID: uuid.New(),
				Role:   domain.RoleAdmin,
			},
			UserID:         teacherID,
			ClassID:        classID,
			AcademicYearID: academicYearID,
		},
	)

	if !errors.Is(err, errClassAssignmentFakeRepository) {
		t.Fatalf(
			"Assign() error = %v, want %v",
			err,
			errClassAssignmentFakeRepository,
		)
	}

	if uow.commitCount != 0 {
		t.Fatalf("Commit() count = %d, want 0", uow.commitCount)
	}

	if uow.rollbackCount != 1 {
		t.Fatalf("Rollback() count = %d, want 1", uow.rollbackCount)
	}
}

func TestClassAssignmentServiceAssignCommitErrorRollsBack(t *testing.T) {
	teacherID := uuid.New()
	classID := uuid.New()
	academicYearID := uuid.New()

	repos := repository.RepositorySet{
		Users: &classAssignmentFakeUserRepository{
			getByIDFn: func(
				_ context.Context,
				_ uuid.UUID,
			) (domain.User, error) {
				return domain.User{
					ID:   teacherID,
					Role: domain.RoleWaliKelas,
				}, nil
			},
		},
		Classes: &classAssignmentFakeClassRepository{
			getByIDFn: func(
				_ context.Context,
				_ uuid.UUID,
			) (domain.Class, error) {
				return domain.Class{
					ID:    classID,
					Name:  "7A",
					Level: 7,
				}, nil
			},
		},
		AcademicYears: &classAssignmentFakeAcademicYearRepository{
			getByIDFn: func(
				_ context.Context,
				_ uuid.UUID,
			) (domain.AcademicYear, error) {
				return newTestAcademicYear(academicYearID, domain.AcademicYearOpen), nil
			},
		},
	}

	uowRepos := repos
	uowRepos.TeacherClassAssignments =
		&classAssignmentFakeTeacherAssignmentRepository{}

	uow := &classAssignmentFakeUnitOfWork{
		repos:     uowRepos,
		commitErr: errClassAssignmentFakeRepository,
	}

	svc := newClassAssignmentServiceForTest(
		t,
		repos,
		&classAssignmentFakeUnitOfWorkManager{uow: uow},
	)

	_, err := svc.Assign(
		context.Background(),
		AssignTeacherInput{
			Actor: Actor{
				UserID: uuid.New(),
				Role:   domain.RoleAdmin,
			},
			UserID:         teacherID,
			ClassID:        classID,
			AcademicYearID: academicYearID,
		},
	)

	if !errors.Is(err, errClassAssignmentFakeRepository) {
		t.Fatalf(
			"Assign() error = %v, want %v",
			err,
			errClassAssignmentFakeRepository,
		)
	}

	if uow.commitCount != 1 {
		t.Fatalf("Commit() count = %d, want 1", uow.commitCount)
	}

	if uow.rollbackCount != 1 {
		t.Fatalf("Rollback() count = %d, want 1", uow.rollbackCount)
	}
}

// ============================================================
// GET
// ============================================================

func TestClassAssignmentServiceGetAdmin(t *testing.T) {
	assignmentID := uuid.New()
	classID := uuid.New()
	academicYearID := uuid.New()
	teacherID := uuid.New()

	assignment := newTestTeacherAssignment(
		t,
		teacherID,
		academicYearID,
		classID,
	)

	assignment.ID = assignmentID

	repos := repository.RepositorySet{
		TeacherClassAssignments: &classAssignmentFakeTeacherAssignmentRepository{
			getByIDFn: func(
				_ context.Context,
				id uuid.UUID,
			) (domain.TeacherClassAssignment, error) {
				if id != assignmentID {
					t.Fatalf("GetByID() ID = %v, want %v", id, assignmentID)
				}

				return assignment, nil
			},
		},
	}

	svc := newClassAssignmentServiceForTest(
		t,
		repos,
		&classAssignmentFakeUnitOfWorkManager{},
	)

	got, err := svc.Get(
		context.Background(),
		GetTeacherAssignmentInput{
			Actor: Actor{
				UserID: uuid.New(),
				Role:   domain.RoleAdmin,
			},
			AssignmentID: assignmentID,
		},
	)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.Assignment == nil {
		t.Fatal("Get() assignment = nil")
	}

	if got.Assignment.ID != assignmentID {
		t.Fatalf(
			"Get() assignment ID = %v, want %v",
			got.Assignment.ID,
			assignmentID,
		)
	}
}

func TestClassAssignmentServiceGetWaliInScope(t *testing.T) {
	teacherID := uuid.New()
	classID := uuid.New()
	academicYearID := uuid.New()
	assignment := newTestTeacherAssignment(
		t,
		teacherID,
		academicYearID,
		classID,
	)

	repos := repository.RepositorySet{
		TeacherClassAssignments: &classAssignmentFakeTeacherAssignmentRepository{
			getByIDFn: func(
				_ context.Context,
				_ uuid.UUID,
			) (domain.TeacherClassAssignment, error) {
				return assignment, nil
			},
			listByClassAndAcademicYearFn: func(
				_ context.Context,
				gotClassID uuid.UUID,
				gotYearID uuid.UUID,
				_ repository.ListOptions,
			) ([]domain.TeacherClassAssignment, error) {
				if gotClassID != classID {
					t.Fatalf("classID = %v, want %v", gotClassID, classID)
				}

				if gotYearID != academicYearID {
					t.Fatalf(
						"academicYearID = %v, want %v",
						gotYearID,
						academicYearID,
					)
				}

				return []domain.TeacherClassAssignment{
					assignment,
				}, nil
			},
		},
	}

	svc := newClassAssignmentServiceForTest(
		t,
		repos,
		&classAssignmentFakeUnitOfWorkManager{},
	)

	got, err := svc.Get(
		context.Background(),
		GetTeacherAssignmentInput{
			Actor: Actor{
				UserID: teacherID,
				Role:   domain.RoleWaliKelas,
			},
			AssignmentID: assignment.ID,
		},
	)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.Assignment == nil {
		t.Fatal("Get() assignment = nil")
	}
}

func TestClassAssignmentServiceGetWaliOutOfScope(t *testing.T) {
	teacherID := uuid.New()
	otherTeacherID := uuid.New()
	classID := uuid.New()
	academicYearID := uuid.New()

	assignment := newTestTeacherAssignment(
		t,
		otherTeacherID,
		academicYearID,
		classID,
	)

	repos := repository.RepositorySet{
		TeacherClassAssignments: &classAssignmentFakeTeacherAssignmentRepository{
			getByIDFn: func(
				_ context.Context,
				_ uuid.UUID,
			) (domain.TeacherClassAssignment, error) {
				return assignment, nil
			},
			listByClassAndAcademicYearFn: func(
				_ context.Context,
				_ uuid.UUID,
				_ uuid.UUID,
				_ repository.ListOptions,
			) ([]domain.TeacherClassAssignment, error) {
				return []domain.TeacherClassAssignment{
					assignment,
				}, nil
			},
		},
	}

	svc := newClassAssignmentServiceForTest(
		t,
		repos,
		&classAssignmentFakeUnitOfWorkManager{},
	)

	_, err := svc.Get(
		context.Background(),
		GetTeacherAssignmentInput{
			Actor: Actor{
				UserID: teacherID,
				Role:   domain.RoleWaliKelas,
			},
			AssignmentID: assignment.ID,
		},
	)

	if !errors.Is(err, ErrScopeViolation) {
		t.Fatalf(
			"Get() error = %v, want %v",
			err,
			ErrScopeViolation,
		)
	}
}

func TestClassAssignmentServiceGetSiswaInScope(t *testing.T) {
	studentUserID := uuid.New()
	studentID := uuid.New()
	classID := uuid.New()
	academicYearID := uuid.New()

	assignment := newTestTeacherAssignment(
		t,
		uuid.New(),
		academicYearID,
		classID,
	)

	repos := repository.RepositorySet{
		Students: &classAssignmentFakeStudentRepository{
			getByUserIDFn: func(
				_ context.Context,
				userID uuid.UUID,
			) (domain.Student, error) {
				if userID != studentUserID {
					t.Fatalf(
						"GetByUserID() userID = %v, want %v",
						userID,
						studentUserID,
					)
				}

				return domain.Student{
					ID:     studentID,
					UserID: &studentUserID,
					Status: domain.StudentActive,
				}, nil
			},
		},
		StudentClassHistories: &classAssignmentFakeStudentHistoryRepository{
			listByStudentAndAcademicYearFn: func(
				_ context.Context,
				gotStudentID uuid.UUID,
				gotYearID uuid.UUID,
				_ repository.ListOptions,
			) ([]domain.StudentClassHistory, error) {
				if gotStudentID != studentID {
					t.Fatalf(
						"studentID = %v, want %v",
						gotStudentID,
						studentID,
					)
				}

				if gotYearID != academicYearID {
					t.Fatalf(
						"academicYearID = %v, want %v",
						gotYearID,
						academicYearID,
					)
				}

				return []domain.StudentClassHistory{
					{
						StudentID:      studentID,
						AcademicYearID: academicYearID,
						ClassID:        classID,
					},
				}, nil
			},
		},
		TeacherClassAssignments: &classAssignmentFakeTeacherAssignmentRepository{
			getByIDFn: func(
				_ context.Context,
				id uuid.UUID,
			) (domain.TeacherClassAssignment, error) {
				if id != assignment.ID {
					t.Fatalf(
						"GetByID() ID = %v, want %v",
						id,
						assignment.ID,
					)
				}

				return assignment, nil
			},
			listByClassAndAcademicYearFn: func(
				_ context.Context,
				_ uuid.UUID,
				_ uuid.UUID,
				_ repository.ListOptions,
			) ([]domain.TeacherClassAssignment, error) {
				return []domain.TeacherClassAssignment{
					assignment,
				}, nil
			},
		},
	}

	svc := newClassAssignmentServiceForTest(
		t,
		repos,
		&classAssignmentFakeUnitOfWorkManager{},
	)

	got, err := svc.Get(
		context.Background(),
		GetTeacherAssignmentInput{
			Actor: Actor{
				UserID: studentUserID,
				Role:   domain.RoleSiswa,
			},
			AssignmentID: assignment.ID,
		},
	)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.Assignment == nil {
		t.Fatal("Get() assignment = nil")
	}
}

// ============================================================
// LIST BY TEACHER
// ============================================================

func TestClassAssignmentServiceListByTeacherAdmin(t *testing.T) {
	userID := uuid.New()
	academicYearID := uuid.New()

	assignmentA := newTestTeacherAssignment(
		t,
		userID,
		academicYearID,
		uuid.New(),
	)

	assignmentB := newTestTeacherAssignment(
		t,
		userID,
		academicYearID,
		uuid.New(),
	)

	repos := repository.RepositorySet{
		TeacherClassAssignments: &classAssignmentFakeTeacherAssignmentRepository{
			listByUserAndAcademicYearFn: func(
				_ context.Context,
				gotUserID uuid.UUID,
				gotYearID uuid.UUID,
				_ repository.ListOptions,
			) ([]domain.TeacherClassAssignment, error) {
				if gotUserID != userID {
					t.Fatalf("userID = %v, want %v", gotUserID, userID)
				}

				if gotYearID != academicYearID {
					t.Fatalf(
						"academicYearID = %v, want %v",
						gotYearID,
						academicYearID,
					)
				}

				return []domain.TeacherClassAssignment{
					assignmentA,
					assignmentB,
				}, nil
			},
		},
	}

	svc := newClassAssignmentServiceForTest(
		t,
		repos,
		&classAssignmentFakeUnitOfWorkManager{},
	)

	got, err := svc.ListByTeacher(
		context.Background(),
		ListTeacherAssignmentsInput{
			Actor: Actor{
				UserID: uuid.New(),
				Role:   domain.RoleAdmin,
			},
			UserID:         userID,
			AcademicYearID: academicYearID,
			Options:        repositoryListOptionsForTest(),
		},
	)
	if err != nil {
		t.Fatalf("ListByTeacher() error = %v", err)
	}

	if len(got.Assignments) != 2 {
		t.Fatalf(
			"ListByTeacher() returned %d assignments, want 2",
			len(got.Assignments),
		)
	}
}

func TestClassAssignmentServiceListByTeacherWaliScopeFilters(t *testing.T) {
	teacherID := uuid.New()
	academicYearID := uuid.New()
	classA := uuid.New()
	classB := uuid.New()

	allowed := newTestTeacherAssignment(
		t,
		teacherID,
		academicYearID,
		classA,
	)

	rejected := newTestTeacherAssignment(
		t,
		teacherID,
		academicYearID,
		classB,
	)

	repos := repository.RepositorySet{
		TeacherClassAssignments: &classAssignmentFakeTeacherAssignmentRepository{
			listByUserAndAcademicYearFn: func(
				_ context.Context,
				_ uuid.UUID,
				_ uuid.UUID,
				_ repository.ListOptions,
			) ([]domain.TeacherClassAssignment, error) {
				return []domain.TeacherClassAssignment{
					allowed,
					rejected,
				}, nil
			},
			listByClassAndAcademicYearFn: func(
				_ context.Context,
				gotClassID uuid.UUID,
				_ uuid.UUID,
				_ repository.ListOptions,
			) ([]domain.TeacherClassAssignment, error) {
				if gotClassID == classA {
					return []domain.TeacherClassAssignment{allowed}, nil
				}

				return []domain.TeacherClassAssignment{}, nil
			},
		},
	}

	svc := newClassAssignmentServiceForTest(
		t,
		repos,
		&classAssignmentFakeUnitOfWorkManager{},
	)

	got, err := svc.ListByTeacher(
		context.Background(),
		ListTeacherAssignmentsInput{
			Actor: Actor{
				UserID: teacherID,
				Role:   domain.RoleWaliKelas,
			},
			UserID:         teacherID,
			AcademicYearID: academicYearID,
			Options:        repositoryListOptionsForTest(),
		},
	)
	if err != nil {
		t.Fatalf("ListByTeacher() error = %v", err)
	}

	if len(got.Assignments) != 1 {
		t.Fatalf(
			"ListByTeacher() returned %d assignments, want 1",
			len(got.Assignments),
		)
	}

	if got.Assignments[0].ID != allowed.ID {
		t.Fatalf(
			"ListByTeacher() assignment ID = %v, want %v",
			got.Assignments[0].ID,
			allowed.ID,
		)
	}
}

// ============================================================
// LIST BY CLASS
// ============================================================

func TestClassAssignmentServiceListByClassAdmin(t *testing.T) {
	classID := uuid.New()
	academicYearID := uuid.New()

	assignment := newTestTeacherAssignment(
		t,
		uuid.New(),
		academicYearID,
		classID,
	)

	repos := repository.RepositorySet{
		TeacherClassAssignments: &classAssignmentFakeTeacherAssignmentRepository{
			listByClassAndAcademicYearFn: func(
				_ context.Context,
				gotClassID uuid.UUID,
				gotYearID uuid.UUID,
				_ repository.ListOptions,
			) ([]domain.TeacherClassAssignment, error) {
				if gotClassID != classID {
					t.Fatalf("classID = %v, want %v", gotClassID, classID)
				}

				if gotYearID != academicYearID {
					t.Fatalf(
						"academicYearID = %v, want %v",
						gotYearID,
						academicYearID,
					)
				}

				return []domain.TeacherClassAssignment{assignment}, nil
			},
		},
	}

	svc := newClassAssignmentServiceForTest(
		t,
		repos,
		&classAssignmentFakeUnitOfWorkManager{},
	)

	got, err := svc.ListByClass(
		context.Background(),
		ListClassAssignmentsInput{
			Actor: Actor{
				UserID: uuid.New(),
				Role:   domain.RoleAdmin,
			},
			ClassID:        classID,
			AcademicYearID: academicYearID,
			Options:        repositoryListOptionsForTest(),
		},
	)
	if err != nil {
		t.Fatalf("ListByClass() error = %v", err)
	}

	if len(got.Assignments) != 1 {
		t.Fatalf(
			"ListByClass() returned %d assignments, want 1",
			len(got.Assignments),
		)
	}
}

func TestClassAssignmentServiceListByClassWaliOutOfScope(t *testing.T) {
	teacherID := uuid.New()
	otherTeacherID := uuid.New()
	classID := uuid.New()
	academicYearID := uuid.New()

	assignment := newTestTeacherAssignment(
		t,
		otherTeacherID,
		academicYearID,
		classID,
	)

	repos := repository.RepositorySet{
		TeacherClassAssignments: &classAssignmentFakeTeacherAssignmentRepository{
			listByClassAndAcademicYearFn: func(
				_ context.Context,
				_ uuid.UUID,
				_ uuid.UUID,
				_ repository.ListOptions,
			) ([]domain.TeacherClassAssignment, error) {
				return []domain.TeacherClassAssignment{
					assignment,
				}, nil
			},
		},
	}

	svc := newClassAssignmentServiceForTest(
		t,
		repos,
		&classAssignmentFakeUnitOfWorkManager{},
	)

	_, err := svc.ListByClass(
		context.Background(),
		ListClassAssignmentsInput{
			Actor: Actor{
				UserID: teacherID,
				Role:   domain.RoleWaliKelas,
			},
			ClassID:        classID,
			AcademicYearID: academicYearID,
			Options:        repositoryListOptionsForTest(),
		},
	)

	if !errors.Is(err, ErrScopeViolation) {
		t.Fatalf(
			"ListByClass() error = %v, want %v",
			err,
			ErrScopeViolation,
		)
	}
}

// ============================================================
// GET CURRENT BY CLASS
// ============================================================

func TestClassAssignmentServiceGetCurrentByClassAdmin(t *testing.T) {
	classID := uuid.New()
	academicYearID := uuid.New()
	date := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)

	assignment := newTestTeacherAssignment(
		t,
		uuid.New(),
		academicYearID,
		classID,
	)

	repos := repository.RepositorySet{
		TeacherClassAssignments: &classAssignmentFakeTeacherAssignmentRepository{
			getCurrentByClassAndAcademicYearFn: func(
				_ context.Context,
				gotClassID uuid.UUID,
				gotYearID uuid.UUID,
				gotDate time.Time,
			) (domain.TeacherClassAssignment, error) {
				if gotClassID != classID {
					t.Fatalf("classID = %v, want %v", gotClassID, classID)
				}

				if gotYearID != academicYearID {
					t.Fatalf(
						"academicYearID = %v, want %v",
						gotYearID,
						academicYearID,
					)
				}

				if !gotDate.Equal(date) {
					t.Fatalf(
						"date = %v, want %v",
						gotDate,
						date,
					)
				}

				return assignment, nil
			},
		},
	}

	svc := newClassAssignmentServiceForTest(
		t,
		repos,
		&classAssignmentFakeUnitOfWorkManager{},
	)

	got, err := svc.GetCurrentByClass(
		context.Background(),
		GetCurrentClassAssignmentInput{
			Actor: Actor{
				UserID: uuid.New(),
				Role:   domain.RoleAdmin,
			},
			ClassID:        classID,
			AcademicYearID: academicYearID,
			Date:           date,
		},
	)
	if err != nil {
		t.Fatalf("GetCurrentByClass() error = %v", err)
	}

	if got.Assignment == nil {
		t.Fatal("GetCurrentByClass() assignment = nil")
	}
}

func TestClassAssignmentServiceGetCurrentByClassRejectsZeroDate(t *testing.T) {
	svc := newClassAssignmentServiceForTest(
		t,
		repository.RepositorySet{},
		&classAssignmentFakeUnitOfWorkManager{},
	)

	_, err := svc.GetCurrentByClass(
		context.Background(),
		GetCurrentClassAssignmentInput{
			Actor: Actor{
				UserID: uuid.New(),
				Role:   domain.RoleAdmin,
			},
			ClassID:        uuid.New(),
			AcademicYearID: uuid.New(),
		},
	)

	if !errors.Is(err, domain.ErrInvalidDateRange) &&
		!errors.Is(err, domain.ErrInvalidValue) {
		t.Fatalf(
			"GetCurrentByClass() error = %v, want date validation error",
			err,
		)
	}
}

// ============================================================
// ERROR PROPAGATION
// ============================================================

func TestClassAssignmentServiceGetPropagatesRepositoryError(t *testing.T) {
	repos := repository.RepositorySet{
		TeacherClassAssignments: &classAssignmentFakeTeacherAssignmentRepository{
			getByIDFn: func(
				context.Context,
				uuid.UUID,
			) (domain.TeacherClassAssignment, error) {
				return domain.TeacherClassAssignment{}, errClassAssignmentFakeRepository
			},
		},
	}

	svc := newClassAssignmentServiceForTest(
		t,
		repos,
		&classAssignmentFakeUnitOfWorkManager{},
	)

	_, err := svc.Get(
		context.Background(),
		GetTeacherAssignmentInput{
			Actor: Actor{
				UserID: uuid.New(),
				Role:   domain.RoleAdmin,
			},
			AssignmentID: uuid.New(),
		},
	)

	if !errors.Is(err, errClassAssignmentFakeRepository) {
		t.Fatalf(
			"Get() error = %v, want %v",
			err,
			errClassAssignmentFakeRepository,
		)
	}
}

// ============================================================
// TEST HELPERS
// ============================================================

func repositoryListOptionsForTest() ListOptions {
	return ListOptions{
		Limit:  10,
		Offset: 0,
	}
}
