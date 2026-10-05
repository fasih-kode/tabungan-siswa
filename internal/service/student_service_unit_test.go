package service

import (
	"context"
	"errors"
	"testing"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

type studentServiceFakeStudentRepository struct {
	repository.StudentRepository

	getByIDFn            func(context.Context, uuid.UUID) (domain.Student, error)
	getByNISFn           func(context.Context, string) (domain.Student, error)
	getByNISNFn          func(context.Context, string) (domain.Student, error)
	getByUserIDFn        func(context.Context, uuid.UUID) (domain.Student, error)
	listByClassAndYearFn func(context.Context, uuid.UUID, uuid.UUID, repository.ListOptions) ([]domain.Student, error)
	createFn             func(context.Context, domain.Student) error
	updateFn             func(context.Context, domain.Student) error
}

func (f *studentServiceFakeStudentRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (domain.Student, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, id)
	}
	return domain.Student{}, repository.ErrNotFound
}

func (f *studentServiceFakeStudentRepository) GetByNIS(
	ctx context.Context,
	nis string,
) (domain.Student, error) {
	if f.getByNISFn != nil {
		return f.getByNISFn(ctx, nis)
	}
	return domain.Student{}, repository.ErrNotFound
}

func (f *studentServiceFakeStudentRepository) GetByNISN(
	ctx context.Context,
	nisn string,
) (domain.Student, error) {
	if f.getByNISNFn != nil {
		return f.getByNISNFn(ctx, nisn)
	}
	return domain.Student{}, repository.ErrNotFound
}

func (f *studentServiceFakeStudentRepository) GetByUserID(
	ctx context.Context,
	userID uuid.UUID,
) (domain.Student, error) {
	if f.getByUserIDFn != nil {
		return f.getByUserIDFn(ctx, userID)
	}
	return domain.Student{}, repository.ErrNotFound
}

func (f *studentServiceFakeStudentRepository) ListByClassAndAcademicYear(
	ctx context.Context,
	classID uuid.UUID,
	academicYearID uuid.UUID,
	options repository.ListOptions,
) ([]domain.Student, error) {
	if f.listByClassAndYearFn != nil {
		return f.listByClassAndYearFn(
			ctx,
			classID,
			academicYearID,
			options,
		)
	}
	return nil, nil
}

func (f *studentServiceFakeStudentRepository) Create(
	ctx context.Context,
	student domain.Student,
) error {
	if f.createFn != nil {
		return f.createFn(ctx, student)
	}
	return nil
}

func (f *studentServiceFakeStudentRepository) Update(
	ctx context.Context,
	student domain.Student,
) error {
	if f.updateFn != nil {
		return f.updateFn(ctx, student)
	}
	return nil
}

type studentServiceFakeStudentClassHistoryRepository struct {
	repository.StudentClassHistoryRepository

	listFn func(
		context.Context,
		uuid.UUID,
		uuid.UUID,
		repository.ListOptions,
	) ([]domain.StudentClassHistory, error)
}

func (f *studentServiceFakeStudentClassHistoryRepository) ListByStudentAndAcademicYear(
	ctx context.Context,
	studentID uuid.UUID,
	academicYearID uuid.UUID,
	options repository.ListOptions,
) ([]domain.StudentClassHistory, error) {
	if f.listFn != nil {
		return f.listFn(
			ctx,
			studentID,
			academicYearID,
			options,
		)
	}
	return nil, nil
}

type studentServiceFakeTeacherAssignmentRepository struct {
	repository.TeacherClassAssignmentRepository

	listByClassFn func(
		context.Context,
		uuid.UUID,
		uuid.UUID,
		repository.ListOptions,
	) ([]domain.TeacherClassAssignment, error)
}

func (f *studentServiceFakeTeacherAssignmentRepository) ListByClassAndAcademicYear(
	ctx context.Context,
	classID uuid.UUID,
	academicYearID uuid.UUID,
	options repository.ListOptions,
) ([]domain.TeacherClassAssignment, error) {
	if f.listByClassFn != nil {
		return f.listByClassFn(
			ctx,
			classID,
			academicYearID,
			options,
		)
	}
	return nil, nil
}

type studentServiceFakeUnitOfWork struct {
	repos         repository.RepositorySet
	commitErr     error
	rollbackErr   error
	commitCount   int
	rollbackCount int
}

func (f *studentServiceFakeUnitOfWork) Repositories() repository.RepositorySet {
	return f.repos
}

func (f *studentServiceFakeUnitOfWork) Commit() error {
	f.commitCount++
	return f.commitErr
}

func (f *studentServiceFakeUnitOfWork) Rollback() error {
	f.rollbackCount++
	return f.rollbackErr
}

type studentServiceFakeUnitOfWorkManager struct {
	uow        repository.UnitOfWork
	beginErr   error
	beginCount int
}

func (f *studentServiceFakeUnitOfWorkManager) Begin(context.Context) (repository.UnitOfWork, error) {
	f.beginCount++

	if f.beginErr != nil {
		return nil, f.beginErr
	}

	return f.uow, nil
}

func newStudentServiceForTest(
	t *testing.T,
	repos repository.RepositorySet,
	uowManager repository.UnitOfWorkManager,
) *studentService {
	t.Helper()

	svc, err := NewStudentService(Dependencies{
		Repositories: repos,
		UOW:          uowManager,
	})
	if err != nil {
		t.Fatalf("NewStudentService() error = %v", err)
	}

	return svc
}

func newTestServiceStudent(
	t *testing.T,
	id uuid.UUID,
	userID *uuid.UUID,
	nis string,
	nisn string,
	name string,
	status domain.StudentStatus,
) domain.Student {
	t.Helper()

	return domain.Student{
		ID:     id,
		UserID: userID,
		NIS:    &nis,
		NISN:   &nisn,
		Name:   name,
		Status: status,
	}
}

func TestNewStudentService(t *testing.T) {
	uowManager := &studentServiceFakeUnitOfWorkManager{}

	svc, err := NewStudentService(Dependencies{
		UOW: uowManager,
	})
	if err != nil {
		t.Fatalf("NewStudentService() error = %v", err)
	}

	if svc == nil {
		t.Fatal("NewStudentService() service = nil")
	}

	if _, err := NewStudentService(Dependencies{}); !errors.Is(
		err,
		ErrInvalidDependency,
	) {
		t.Fatalf(
			"NewStudentService() error = %v, want %v",
			err,
			ErrInvalidDependency,
		)
	}
}

func TestStudentServiceGetAdminAllowed(t *testing.T) {
	studentID := uuid.New()
	actorID := uuid.New()

	student := newTestServiceStudent(
		t,
		studentID,
		nil,
		"NIS-1",
		"NISN-1",
		"Ahmad",
		domain.StudentActive,
	)

	repos := repository.RepositorySet{
		Students: &studentServiceFakeStudentRepository{
			getByIDFn: func(
				context.Context,
				uuid.UUID,
			) (domain.Student, error) {
				return student, nil
			},
		},
	}

	svc := newStudentServiceForTest(
		t,
		repos,
		&studentServiceFakeUnitOfWorkManager{},
	)

	got, err := svc.Get(
		context.Background(),
		GetStudentInput{
			Actor: Actor{
				UserID: actorID,
				Role:   domain.RoleAdmin,
			},
			StudentID: studentID,
		},
	)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.Student == nil || got.Student.ID != studentID {
		t.Fatal("Get() returned unexpected student")
	}
}

func TestStudentServiceGetSiswaSelfAllowed(t *testing.T) {
	studentID := uuid.New()
	actorID := uuid.New()

	student := newTestServiceStudent(
		t,
		studentID,
		&actorID,
		"NIS-1",
		"NISN-1",
		"Ahmad",
		domain.StudentActive,
	)

	repos := repository.RepositorySet{
		Students: &studentServiceFakeStudentRepository{
			getByIDFn: func(
				context.Context,
				uuid.UUID,
			) (domain.Student, error) {
				return student, nil
			},
		},
	}

	svc := newStudentServiceForTest(
		t,
		repos,
		&studentServiceFakeUnitOfWorkManager{},
	)

	got, err := svc.Get(
		context.Background(),
		GetStudentInput{
			Actor: Actor{
				UserID: actorID,
				Role:   domain.RoleSiswa,
			},
			StudentID: studentID,
		},
	)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.Student == nil {
		t.Fatal("Get() student = nil")
	}
}

func TestStudentServiceGetSiswaOtherRejected(t *testing.T) {
	studentID := uuid.New()
	studentUserID := uuid.New()
	actorID := uuid.New()

	student := newTestServiceStudent(
		t,
		studentID,
		&studentUserID,
		"NIS-1",
		"NISN-1",
		"Ahmad",
		domain.StudentActive,
	)

	repos := repository.RepositorySet{
		Students: &studentServiceFakeStudentRepository{
			getByIDFn: func(
				context.Context,
				uuid.UUID,
			) (domain.Student, error) {
				return student, nil
			},
		},
	}

	svc := newStudentServiceForTest(
		t,
		repos,
		&studentServiceFakeUnitOfWorkManager{},
	)

	_, err := svc.Get(
		context.Background(),
		GetStudentInput{
			Actor: Actor{
				UserID: actorID,
				Role:   domain.RoleSiswa,
			},
			StudentID: studentID,
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

func TestStudentServiceListWaliScope(t *testing.T) {
	actorID := uuid.New()
	classID := uuid.New()
	academicYearID := uuid.New()

	student := newTestServiceStudent(
		t,
		uuid.New(),
		nil,
		"NIS-1",
		"NISN-1",
		"Ahmad",
		domain.StudentActive,
	)

	repos := repository.RepositorySet{
		Students: &studentServiceFakeStudentRepository{
			listByClassAndYearFn: func(
				context.Context,
				uuid.UUID,
				uuid.UUID,
				repository.ListOptions,
			) ([]domain.Student, error) {
				return []domain.Student{student}, nil
			},
		},
		TeacherClassAssignments: &studentServiceFakeTeacherAssignmentRepository{
			listByClassFn: func(
				context.Context,
				uuid.UUID,
				uuid.UUID,
				repository.ListOptions,
			) ([]domain.TeacherClassAssignment, error) {
				return []domain.TeacherClassAssignment{
					{
						UserID:  actorID,
						ClassID: classID,
					},
				}, nil
			},
		},
	}

	svc := newStudentServiceForTest(
		t,
		repos,
		&studentServiceFakeUnitOfWorkManager{},
	)

	got, err := svc.List(
		context.Background(),
		ListStudentsInput{
			Actor: Actor{
				UserID: actorID,
				Role:   domain.RoleWaliKelas,
			},
			AcademicYearID: academicYearID,
			ClassID:        classID,
			Options: ListOptions{
				Limit:  10,
				Offset: 0,
			},
		},
	)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(got.Students) != 1 {
		t.Fatalf(
			"List() returned %d students, want 1",
			len(got.Students),
		)
	}
}

func TestStudentServiceListSiswaRejected(t *testing.T) {
	_, err := newStudentServiceForTest(
		t,
		repository.RepositorySet{},
		&studentServiceFakeUnitOfWorkManager{},
	).List(
		context.Background(),
		ListStudentsInput{
			Actor: Actor{
				UserID: uuid.New(),
				Role:   domain.RoleSiswa,
			},
			AcademicYearID: uuid.New(),
			ClassID:        uuid.New(),
		},
	)

	if !errors.Is(err, ErrForbidden) {
		t.Fatalf(
			"List() error = %v, want %v",
			err,
			ErrForbidden,
		)
	}
}

func TestStudentServiceCreateAdmin(t *testing.T) {
	actorID := uuid.New()
	nis := "12345"
	nisn := "0012345678"

	var created domain.Student

	repos := repository.RepositorySet{
		Students: &studentServiceFakeStudentRepository{
			createFn: func(
				_ context.Context,
				student domain.Student,
			) error {
				created = student
				return nil
			},
		},
	}

	uow := &studentServiceFakeUnitOfWork{
		repos: repos,
	}

	svc := newStudentServiceForTest(
		t,
		repos,
		&studentServiceFakeUnitOfWorkManager{
			uow: uow,
		},
	)

	got, err := svc.Create(
		context.Background(),
		CreateStudentInput{
			Actor: Actor{
				UserID: actorID,
				Role:   domain.RoleAdmin,
			},
			NIS:  &nis,
			NISN: &nisn,
			Name: "Ahmad",
		},
	)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if got.Student == nil {
		t.Fatal("Create() student = nil")
	}

	if created.ID != got.Student.ID {
		t.Fatal("Create() did not persist returned student")
	}

	if uow.commitCount != 1 {
		t.Fatalf(
			"Commit() count = %d, want 1",
			uow.commitCount,
		)
	}
}

func TestStudentServiceLeave(t *testing.T) {
	studentID := uuid.New()
	student := newTestServiceStudent(
		t,
		studentID,
		nil,
		"NIS-1",
		"NISN-1",
		"Ahmad",
		domain.StudentActive,
	)

	var updated domain.Student

	repos := repository.RepositorySet{
		Students: &studentServiceFakeStudentRepository{
			getByIDFn: func(
				context.Context,
				uuid.UUID,
			) (domain.Student, error) {
				return student, nil
			},
			updateFn: func(
				_ context.Context,
				value domain.Student,
			) error {
				updated = value
				return nil
			},
		},
	}

	uow := &studentServiceFakeUnitOfWork{
		repos: repos,
	}

	svc := newStudentServiceForTest(
		t,
		repos,
		&studentServiceFakeUnitOfWorkManager{
			uow: uow,
		},
	)

	got, err := svc.Leave(
		context.Background(),
		LeaveStudentInput{
			Actor: Actor{
				UserID: uuid.New(),
				Role:   domain.RoleAdmin,
			},
			StudentID: studentID,
		},
	)
	if err != nil {
		t.Fatalf("Leave() error = %v", err)
	}

	if got.Student.Status != domain.StudentLeft {
		t.Fatalf(
			"status = %v, want %v",
			got.Student.Status,
			domain.StudentLeft,
		)
	}

	if updated.Status != domain.StudentLeft {
		t.Fatalf("updated status = %v, want LEFT", updated.Status)
	}
}

func TestStudentServiceGetWaliKelasAllowed(t *testing.T) {
	actorID := uuid.New()
	studentID := uuid.New()
	classID := uuid.New()
	academicYearID := uuid.New()

	student := newTestServiceStudent(
		t,
		studentID,
		nil,
		"NIS-GET-WALI",
		"NISN-GET-WALI",
		"Student Wali",
		domain.StudentActive,
	)

	repos := repository.RepositorySet{
		Students: &studentServiceFakeStudentRepository{
			getByIDFn: func(
				_ context.Context,
				id uuid.UUID,
			) (domain.Student, error) {
				if id != studentID {
					t.Fatalf("GetByID() id = %v, want %v", id, studentID)
				}
				return student, nil
			},
		},
		StudentClassHistories: &studentServiceFakeStudentClassHistoryRepository{
			listFn: func(
				_ context.Context,
				gotStudentID uuid.UUID,
				gotAcademicYearID uuid.UUID,
				_ repository.ListOptions,
			) ([]domain.StudentClassHistory, error) {
				if gotStudentID != studentID {
					t.Fatalf(
						"ListByStudentAndAcademicYear() studentID = %v, want %v",
						gotStudentID,
						studentID,
					)
				}
				if gotAcademicYearID != academicYearID {
					t.Fatalf(
						"ListByStudentAndAcademicYear() academicYearID = %v, want %v",
						gotAcademicYearID,
						academicYearID,
					)
				}

				return []domain.StudentClassHistory{
					{
						ID:             uuid.New(),
						StudentID:      studentID,
						ClassID:        classID,
						AcademicYearID: academicYearID,
					},
				}, nil
			},
		},
		TeacherClassAssignments: &studentServiceFakeTeacherAssignmentRepository{
			listByClassFn: func(
				_ context.Context,
				gotClassID uuid.UUID,
				gotAcademicYearID uuid.UUID,
				_ repository.ListOptions,
			) ([]domain.TeacherClassAssignment, error) {
				if gotClassID != classID {
					t.Fatalf(
						"ListByClassAndAcademicYear() classID = %v, want %v",
						gotClassID,
						classID,
					)
				}
				if gotAcademicYearID != academicYearID {
					t.Fatalf(
						"ListByClassAndAcademicYear() academicYearID = %v, want %v",
						gotAcademicYearID,
						academicYearID,
					)
				}

				return []domain.TeacherClassAssignment{
					{
						ID:             uuid.New(),
						UserID:         actorID,
						ClassID:        classID,
						AcademicYearID: academicYearID,
					},
				}, nil
			},
		},
	}

	svc := newStudentServiceForTest(
		t,
		repos,
		&studentServiceFakeUnitOfWorkManager{},
	)

	output, err := svc.Get(context.Background(), GetStudentInput{
		Actor: Actor{
			UserID: actorID,
			Role:   domain.RoleWaliKelas,
		},
		StudentID:      studentID,
		AcademicYearID: academicYearID,
	})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if output.Student == nil || output.Student.ID != studentID {
		t.Fatal("Get() returned unexpected student")
	}
}

func TestStudentServiceGetByNISAdminAllowed(t *testing.T) {
	student := newTestServiceStudent(
		t,
		uuid.New(),
		nil,
		"NIS-BY-NIS-ADMIN",
		"NISN-BY-NIS-ADMIN",
		"Student NIS Admin",
		domain.StudentActive,
	)

	repos := repository.RepositorySet{
		Students: &studentServiceFakeStudentRepository{
			getByNISFn: func(
				_ context.Context,
				nis string,
			) (domain.Student, error) {
				if nis != *student.NIS {
					t.Fatalf("GetByNIS() nis = %q, want %q", nis, *student.NIS)
				}
				return student, nil
			},
		},
	}

	svc := newStudentServiceForTest(
		t,
		repos,
		&studentServiceFakeUnitOfWorkManager{},
	)

	output, err := svc.GetByNIS(context.Background(), GetStudentByNISInput{
		Actor: Actor{
			UserID: uuid.New(),
			Role:   domain.RoleAdmin,
		},
		NIS: *student.NIS,
	})
	if err != nil {
		t.Fatalf("GetByNIS() error = %v", err)
	}

	if output.Student == nil || output.Student.ID != student.ID {
		t.Fatal("GetByNIS() returned unexpected student")
	}
}

func TestStudentServiceGetByNISSiswaSelfAllowed(t *testing.T) {
	userID := uuid.New()

	student := newTestServiceStudent(
		t,
		uuid.New(),
		&userID,
		"NIS-BY-NIS-SISWA",
		"NISN-BY-NIS-SISWA",
		"Student NIS Siswa",
		domain.StudentActive,
	)

	repos := repository.RepositorySet{
		Students: &studentServiceFakeStudentRepository{
			getByNISFn: func(
				_ context.Context,
				_ string,
			) (domain.Student, error) {
				return student, nil
			},
		},
	}

	svc := newStudentServiceForTest(
		t,
		repos,
		&studentServiceFakeUnitOfWorkManager{},
	)

	output, err := svc.GetByNIS(context.Background(), GetStudentByNISInput{
		Actor: Actor{
			UserID: userID,
			Role:   domain.RoleSiswa,
		},
		NIS: *student.NIS,
	})
	if err != nil {
		t.Fatalf("GetByNIS() error = %v", err)
	}

	if output.Student == nil || output.Student.ID != student.ID {
		t.Fatal("GetByNIS() returned unexpected student")
	}
}

func TestStudentServiceGetByNISNWaliScope(t *testing.T) {
	actorID := uuid.New()
	studentID := uuid.New()
	classID := uuid.New()
	academicYearID := uuid.New()

	student := newTestServiceStudent(
		t,
		studentID,
		nil,
		"NIS-BY-NISN-WALI",
		"NISN-BY-NISN-WALI",
		"Student NISN Wali",
		domain.StudentActive,
	)

	repos := repository.RepositorySet{
		Students: &studentServiceFakeStudentRepository{
			getByNISNFn: func(
				_ context.Context,
				_ string,
			) (domain.Student, error) {
				return student, nil
			},
		},
		StudentClassHistories: &studentServiceFakeStudentClassHistoryRepository{
			listFn: func(
				_ context.Context,
				_ uuid.UUID,
				_ uuid.UUID,
				_ repository.ListOptions,
			) ([]domain.StudentClassHistory, error) {
				return []domain.StudentClassHistory{
					{
						ID:             uuid.New(),
						StudentID:      studentID,
						ClassID:        classID,
						AcademicYearID: academicYearID,
					},
				}, nil
			},
		},
		TeacherClassAssignments: &studentServiceFakeTeacherAssignmentRepository{
			listByClassFn: func(
				_ context.Context,
				_ uuid.UUID,
				_ uuid.UUID,
				_ repository.ListOptions,
			) ([]domain.TeacherClassAssignment, error) {
				return []domain.TeacherClassAssignment{
					{
						ID:             uuid.New(),
						UserID:         actorID,
						ClassID:        classID,
						AcademicYearID: academicYearID,
					},
				}, nil
			},
		},
	}

	svc := newStudentServiceForTest(
		t,
		repos,
		&studentServiceFakeUnitOfWorkManager{},
	)

	output, err := svc.GetByNISN(context.Background(), GetStudentByNISNInput{
		Actor: Actor{
			UserID: actorID,
			Role:   domain.RoleWaliKelas,
		},
		NISN:           *student.NISN,
		AcademicYearID: academicYearID,
	})
	if err != nil {
		t.Fatalf("GetByNISN() error = %v", err)
	}

	if output.Student == nil || output.Student.ID != student.ID {
		t.Fatal("GetByNISN() returned unexpected student")
	}
}

func TestStudentServiceGetByUserIDSiswaSelfAllowed(t *testing.T) {
	userID := uuid.New()

	student := newTestServiceStudent(
		t,
		uuid.New(),
		&userID,
		"NIS-BY-USER",
		"NISN-BY-USER",
		"Student User",
		domain.StudentActive,
	)

	repos := repository.RepositorySet{
		Students: &studentServiceFakeStudentRepository{
			getByUserIDFn: func(
				_ context.Context,
				gotUserID uuid.UUID,
			) (domain.Student, error) {
				if gotUserID != userID {
					t.Fatalf("GetByUserID() userID = %v, want %v", gotUserID, userID)
				}
				return student, nil
			},
		},
	}

	svc := newStudentServiceForTest(
		t,
		repos,
		&studentServiceFakeUnitOfWorkManager{},
	)

	output, err := svc.GetByUserID(context.Background(), GetStudentByUserIDInput{
		Actor: Actor{
			UserID: userID,
			Role:   domain.RoleSiswa,
		},
		UserID: userID,
	})
	if err != nil {
		t.Fatalf("GetByUserID() error = %v", err)
	}

	if output.Student == nil || output.Student.ID != student.ID {
		t.Fatal("GetByUserID() returned unexpected student")
	}
}

func TestStudentServiceGetByUserIDWaliScope(t *testing.T) {
	actorID := uuid.New()
	studentID := uuid.New()
	classID := uuid.New()
	academicYearID := uuid.New()

	student := newTestServiceStudent(
		t,
		studentID,
		nil,
		"NIS-BY-USER-WALI",
		"NISN-BY-USER-WALI",
		"Student User Wali",
		domain.StudentActive,
	)

	repos := repository.RepositorySet{
		Students: &studentServiceFakeStudentRepository{
			getByUserIDFn: func(
				_ context.Context,
				_ uuid.UUID,
			) (domain.Student, error) {
				return student, nil
			},
		},
		StudentClassHistories: &studentServiceFakeStudentClassHistoryRepository{
			listFn: func(
				_ context.Context,
				_ uuid.UUID,
				_ uuid.UUID,
				_ repository.ListOptions,
			) ([]domain.StudentClassHistory, error) {
				return []domain.StudentClassHistory{
					{
						ID:             uuid.New(),
						StudentID:      studentID,
						ClassID:        classID,
						AcademicYearID: academicYearID,
					},
				}, nil
			},
		},
		TeacherClassAssignments: &studentServiceFakeTeacherAssignmentRepository{
			listByClassFn: func(
				_ context.Context,
				_ uuid.UUID,
				_ uuid.UUID,
				_ repository.ListOptions,
			) ([]domain.TeacherClassAssignment, error) {
				return []domain.TeacherClassAssignment{
					{
						ID:             uuid.New(),
						UserID:         actorID,
						ClassID:        classID,
						AcademicYearID: academicYearID,
					},
				}, nil
			},
		},
	}

	svc := newStudentServiceForTest(
		t,
		repos,
		&studentServiceFakeUnitOfWorkManager{},
	)

	output, err := svc.GetByUserID(context.Background(), GetStudentByUserIDInput{
		Actor: Actor{
			UserID: actorID,
			Role:   domain.RoleWaliKelas,
		},
		UserID:         uuid.New(),
		AcademicYearID: academicYearID,
	})
	if err != nil {
		t.Fatalf("GetByUserID() error = %v", err)
	}

	if output.Student == nil || output.Student.ID != student.ID {
		t.Fatal("GetByUserID() returned unexpected student")
	}
}

func TestStudentServiceListAdminAllowed(t *testing.T) {
	classID := uuid.New()
	academicYearID := uuid.New()

	students := []domain.Student{
		newTestServiceStudent(
			t,
			uuid.New(),
			nil,
			"NIS-LIST-ADMIN",
			"NISN-LIST-ADMIN",
			"Student List Admin",
			domain.StudentActive,
		),
	}

	repos := repository.RepositorySet{
		Students: &studentServiceFakeStudentRepository{
			listByClassAndYearFn: func(
				_ context.Context,
				gotClassID uuid.UUID,
				gotAcademicYearID uuid.UUID,
				options repository.ListOptions,
			) ([]domain.Student, error) {
				if gotClassID != classID {
					t.Fatalf("classID = %v, want %v", gotClassID, classID)
				}
				if gotAcademicYearID != academicYearID {
					t.Fatalf(
						"academicYearID = %v, want %v",
						gotAcademicYearID,
						academicYearID,
					)
				}
				if options.Limit != 10 || options.Offset != 20 {
					t.Fatalf(
						"options = %+v, want limit=10 offset=20",
						options,
					)
				}
				return students, nil
			},
		},
	}

	svc := newStudentServiceForTest(
		t,
		repos,
		&studentServiceFakeUnitOfWorkManager{},
	)

	output, err := svc.List(context.Background(), ListStudentsInput{
		Actor: Actor{
			UserID: uuid.New(),
			Role:   domain.RoleAdmin,
		},
		AcademicYearID: academicYearID,
		ClassID:        classID,
		Options: ListOptions{
			Limit:  10,
			Offset: 20,
		},
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(output.Students) != 1 || output.Students[0].ID != students[0].ID {
		t.Fatal("List() returned unexpected students")
	}
}

func TestStudentServiceUpdateAdmin(t *testing.T) {
	studentID := uuid.New()
	userID := uuid.New()
	newUserID := uuid.New()
	oldNIS := "NIS-OLD"
	oldNISN := "NISN-OLD"
	newNIS := "NIS-NEW"
	newNISN := "NISN-NEW"

	existing := newTestServiceStudent(
		t,
		studentID,
		&userID,
		oldNIS,
		oldNISN,
		"Old Name",
		domain.StudentActive,
	)

	repos := repository.RepositorySet{
		Students: &studentServiceFakeStudentRepository{
			getByIDFn: func(
				_ context.Context,
				id uuid.UUID,
			) (domain.Student, error) {
				if id != studentID {
					t.Fatalf("GetByID() id = %v, want %v", id, studentID)
				}
				return existing, nil
			},
			updateFn: func(
				_ context.Context,
				student domain.Student,
			) error {
				if student.ID != studentID {
					t.Fatalf("Update() ID = %v, want %v", student.ID, studentID)
				}
				if student.UserID == nil || *student.UserID != newUserID {
					t.Fatalf("Update() UserID = %v, want %v", student.UserID, newUserID)
				}
				if student.NIS == nil || *student.NIS != newNIS {
					t.Fatalf("Update() NIS = %v, want %q", student.NIS, newNIS)
				}
				if student.NISN == nil || *student.NISN != newNISN {
					t.Fatalf("Update() NISN = %v, want %q", student.NISN, newNISN)
				}
				if student.Name != "New Name" {
					t.Fatalf("Update() Name = %q, want %q", student.Name, "New Name")
				}
				if student.UpdatedAt.IsZero() {
					t.Fatal("Update() UpdatedAt must be set")
				}
				return nil
			},
		},
	}

	uow := &studentServiceFakeUnitOfWork{
		repos: repos,
	}

	uowManager := &studentServiceFakeUnitOfWorkManager{
		uow: uow,
	}

	svc := newStudentServiceForTest(t, repos, uowManager)

	output, err := svc.Update(context.Background(), UpdateStudentInput{
		Actor: Actor{
			UserID: uuid.New(),
			Role:   domain.RoleAdmin,
		},
		StudentID: studentID,
		UserID:    &newUserID,
		NIS:       &newNIS,
		NISN:      &newNISN,
		Name:      "New Name",
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if output.Student == nil {
		t.Fatal("Update() student = nil")
	}

	if output.Student.Name != "New Name" {
		t.Fatalf("Update() output name = %q, want %q", output.Student.Name, "New Name")
	}

	if uow.commitCount != 1 {
		t.Fatalf("Commit() count = %d, want 1", uow.commitCount)
	}

	if uow.rollbackCount != 0 {
		t.Fatalf("Rollback() count = %d, want 0", uow.rollbackCount)
	}
}

func TestStudentServiceUpdateNonAdminRejected(t *testing.T) {
	svc := newStudentServiceForTest(
		t,
		repository.RepositorySet{},
		&studentServiceFakeUnitOfWorkManager{},
	)

	_, err := svc.Update(context.Background(), UpdateStudentInput{
		Actor: Actor{
			UserID: uuid.New(),
			Role:   domain.RoleWaliKelas,
		},
		StudentID: uuid.New(),
		Name:      "Rejected",
	})

	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Update() error = %v, want %v", err, ErrForbidden)
	}
}

func TestStudentServiceLeaveNonAdminRejected(t *testing.T) {
	svc := newStudentServiceForTest(
		t,
		repository.RepositorySet{},
		&studentServiceFakeUnitOfWorkManager{},
	)

	_, err := svc.Leave(context.Background(), LeaveStudentInput{
		Actor: Actor{
			UserID: uuid.New(),
			Role:   domain.RoleWaliKelas,
		},
		StudentID: uuid.New(),
	})

	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Leave() error = %v, want %v", err, ErrForbidden)
	}
}

func TestStudentServiceCreateNonAdminRejected(t *testing.T) {
	svc := newStudentServiceForTest(
		t,
		repository.RepositorySet{},
		&studentServiceFakeUnitOfWorkManager{},
	)

	_, err := svc.Create(context.Background(), CreateStudentInput{
		Actor: Actor{
			UserID: uuid.New(),
			Role:   domain.RoleWaliKelas,
		},
		Name: "Rejected",
	})

	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Create() error = %v, want %v", err, ErrForbidden)
	}
}

func TestStudentServiceCreateRepositoryErrorRollsBack(t *testing.T) {
	repositoryErr := errors.New("create student repository error")
	nis := "NIS-CREATE-ERROR"
	nisn := "NISN-CREATE-ERROR"

	repos := repository.RepositorySet{
		Students: &studentServiceFakeStudentRepository{
			createFn: func(context.Context, domain.Student) error {
				return repositoryErr
			},
		},
	}

	uow := &studentServiceFakeUnitOfWork{
		repos: repos,
	}

	svc := newStudentServiceForTest(
		t,
		repos,
		&studentServiceFakeUnitOfWorkManager{uow: uow},
	)

	_, err := svc.Create(
		context.Background(),
		CreateStudentInput{
			Actor: Actor{
				UserID: uuid.New(),
				Role:   domain.RoleAdmin,
			},
			NIS:  &nis,
			NISN: &nisn,
			Name: "Create Error",
		},
	)
	if !errors.Is(err, repositoryErr) {
		t.Fatalf("Create() error = %v, want %v", err, repositoryErr)
	}

	if uow.commitCount != 0 {
		t.Fatalf("Commit() count = %d, want 0", uow.commitCount)
	}

	if uow.rollbackCount != 1 {
		t.Fatalf("Rollback() count = %d, want 1", uow.rollbackCount)
	}
}

func TestStudentServiceUpdateRepositoryErrorRollsBack(t *testing.T) {
	repositoryErr := errors.New("update student repository error")
	studentID := uuid.New()
	userID := uuid.New()
	nis := "NIS-UPDATE-ERROR"
	nisn := "NISN-UPDATE-ERROR"

	existing := newTestServiceStudent(
		t,
		studentID,
		&userID,
		"NIS-OLD",
		"NISN-OLD",
		"Old Name",
		domain.StudentActive,
	)

	repos := repository.RepositorySet{
		Students: &studentServiceFakeStudentRepository{
			getByIDFn: func(context.Context, uuid.UUID) (domain.Student, error) {
				return existing, nil
			},
			updateFn: func(context.Context, domain.Student) error {
				return repositoryErr
			},
		},
	}

	uow := &studentServiceFakeUnitOfWork{
		repos: repos,
	}

	svc := newStudentServiceForTest(
		t,
		repos,
		&studentServiceFakeUnitOfWorkManager{uow: uow},
	)

	_, err := svc.Update(
		context.Background(),
		UpdateStudentInput{
			Actor: Actor{
				UserID: uuid.New(),
				Role:   domain.RoleAdmin,
			},
			StudentID: studentID,
			UserID:    &userID,
			NIS:       &nis,
			NISN:      &nisn,
			Name:      "Update Error",
		},
	)
	if !errors.Is(err, repositoryErr) {
		t.Fatalf("Update() error = %v, want %v", err, repositoryErr)
	}

	if uow.commitCount != 0 {
		t.Fatalf("Commit() count = %d, want 0", uow.commitCount)
	}

	if uow.rollbackCount != 1 {
		t.Fatalf("Rollback() count = %d, want 1", uow.rollbackCount)
	}
}

func TestStudentServiceLeaveRepositoryErrorRollsBack(t *testing.T) {
	repositoryErr := errors.New("leave student repository error")
	studentID := uuid.New()

	student := newTestServiceStudent(
		t,
		studentID,
		nil,
		"NIS-LEAVE-ERROR",
		"NISN-LEAVE-ERROR",
		"Leave Error",
		domain.StudentActive,
	)

	repos := repository.RepositorySet{
		Students: &studentServiceFakeStudentRepository{
			getByIDFn: func(context.Context, uuid.UUID) (domain.Student, error) {
				return student, nil
			},
			updateFn: func(context.Context, domain.Student) error {
				return repositoryErr
			},
		},
	}

	uow := &studentServiceFakeUnitOfWork{
		repos: repos,
	}

	svc := newStudentServiceForTest(
		t,
		repos,
		&studentServiceFakeUnitOfWorkManager{uow: uow},
	)

	_, err := svc.Leave(
		context.Background(),
		LeaveStudentInput{
			Actor: Actor{
				UserID: uuid.New(),
				Role:   domain.RoleAdmin,
			},
			StudentID: studentID,
		},
	)
	if !errors.Is(err, repositoryErr) {
		t.Fatalf("Leave() error = %v, want %v", err, repositoryErr)
	}

	if uow.commitCount != 0 {
		t.Fatalf("Commit() count = %d, want 0", uow.commitCount)
	}

	if uow.rollbackCount != 1 {
		t.Fatalf("Rollback() count = %d, want 1", uow.rollbackCount)
	}
}
