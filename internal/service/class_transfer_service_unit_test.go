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

type classTransferServiceFakeClassTransferRepository struct {
	repository.ClassTransferRequestRepository

	createFn           func(context.Context, domain.ClassTransferRequest) error
	getByIDFn          func(context.Context, uuid.UUID) (domain.ClassTransferRequest, error)
	getByIDForUpdateFn func(context.Context, uuid.UUID) (domain.ClassTransferRequest, error)
	listByStudentFn    func(
		context.Context,
		uuid.UUID,
		uuid.UUID,
		repository.ListOptions,
	) ([]domain.ClassTransferRequest, error)
	listPendingFn func(
		context.Context,
		uuid.UUID,
		repository.ListOptions,
	) ([]domain.ClassTransferRequest, error)
	updateFn func(context.Context, domain.ClassTransferRequest) error

	getByIDForUpdateCalls int
}

func (f *classTransferServiceFakeClassTransferRepository) Create(
	ctx context.Context,
	request domain.ClassTransferRequest,
) error {
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return nil
}

func (f *classTransferServiceFakeClassTransferRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (domain.ClassTransferRequest, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, id)
	}
	return domain.ClassTransferRequest{}, repository.ErrNotFound
}

func (f *classTransferServiceFakeClassTransferRepository) GetByIDForUpdate(
	ctx context.Context,
	id uuid.UUID,
) (domain.ClassTransferRequest, error) {
	f.getByIDForUpdateCalls++

	if f.getByIDForUpdateFn != nil {
		return f.getByIDForUpdateFn(ctx, id)
	}

	return domain.ClassTransferRequest{}, repository.ErrNotFound
}

func (f *classTransferServiceFakeClassTransferRepository) ListByStudentAndAcademicYear(
	ctx context.Context,
	studentID uuid.UUID,
	academicYearID uuid.UUID,
	options repository.ListOptions,
) ([]domain.ClassTransferRequest, error) {
	if f.listByStudentFn != nil {
		return f.listByStudentFn(ctx, studentID, academicYearID, options)
	}
	return nil, nil
}

func (f *classTransferServiceFakeClassTransferRepository) ListPending(
	ctx context.Context,
	academicYearID uuid.UUID,
	options repository.ListOptions,
) ([]domain.ClassTransferRequest, error) {
	if f.listPendingFn != nil {
		return f.listPendingFn(ctx, academicYearID, options)
	}
	return nil, nil
}

func (f *classTransferServiceFakeClassTransferRepository) Update(
	ctx context.Context,
	request domain.ClassTransferRequest,
) error {
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	return nil
}

func stringPtr(value string) *string {
	return &value
}

type classTransferServiceFakeStudentRepository struct {
	repository.StudentRepository

	getByIDFn     func(context.Context, uuid.UUID) (domain.Student, error)
	getByUserIDFn func(context.Context, uuid.UUID) (domain.Student, error)
}

func (f *classTransferServiceFakeStudentRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (domain.Student, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, id)
	}
	return domain.Student{}, repository.ErrNotFound
}

func (f *classTransferServiceFakeStudentRepository) GetByUserID(
	ctx context.Context,
	userID uuid.UUID,
) (domain.Student, error) {
	if f.getByUserIDFn != nil {
		return f.getByUserIDFn(ctx, userID)
	}
	return domain.Student{}, repository.ErrNotFound
}

type classTransferServiceFakeStudentClassHistoryRepository struct {
	repository.StudentClassHistoryRepository

	listFn func(
		context.Context,
		uuid.UUID,
		uuid.UUID,
		repository.ListOptions,
	) ([]domain.StudentClassHistory, error)

	getCurrentFn func(
		context.Context,
		uuid.UUID,
		uuid.UUID,
		time.Time,
	) (domain.StudentClassHistory, error)

	closeCurrentFn func(
		context.Context,
		uuid.UUID,
		uuid.UUID,
		time.Time,
	) (domain.StudentClassHistory, error)

	createFn func(context.Context, domain.StudentClassHistory) error

	closeCurrentCalls int
	createCalls       int
}

func (f *classTransferServiceFakeStudentClassHistoryRepository) ListByStudentAndAcademicYear(
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

func (f *classTransferServiceFakeStudentClassHistoryRepository) GetCurrentByStudentAndAcademicYear(
	ctx context.Context,
	studentID uuid.UUID,
	academicYearID uuid.UUID,
	date time.Time,
) (domain.StudentClassHistory, error) {
	if f.getCurrentFn != nil {
		return f.getCurrentFn(ctx, studentID, academicYearID, date)
	}
	return domain.StudentClassHistory{}, repository.ErrNotFound
}

func (f *classTransferServiceFakeStudentClassHistoryRepository) CloseCurrentByStudentAndAcademicYear(
	ctx context.Context,
	studentID uuid.UUID,
	academicYearID uuid.UUID,
	endDate time.Time,
) (domain.StudentClassHistory, error) {
	f.closeCurrentCalls++

	if f.closeCurrentFn != nil {
		return f.closeCurrentFn(
			ctx,
			studentID,
			academicYearID,
			endDate,
		)
	}

	return domain.StudentClassHistory{}, repository.ErrNotFound
}

func (f *classTransferServiceFakeStudentClassHistoryRepository) Create(
	ctx context.Context,
	history domain.StudentClassHistory,
) error {
	f.createCalls++

	if f.createFn != nil {
		return f.createFn(ctx, history)
	}

	return nil
}

type classTransferServiceFakeTeacherAssignmentRepository struct {
	repository.TeacherClassAssignmentRepository

	listByClassFn func(
		context.Context,
		uuid.UUID,
		uuid.UUID,
		repository.ListOptions,
	) ([]domain.TeacherClassAssignment, error)
}

func (f *classTransferServiceFakeTeacherAssignmentRepository) ListByClassAndAcademicYear(
	ctx context.Context,
	classID uuid.UUID,
	academicYearID uuid.UUID,
	options repository.ListOptions,
) ([]domain.TeacherClassAssignment, error) {
	if f.listByClassFn != nil {
		return f.listByClassFn(ctx, classID, academicYearID, options)
	}
	return nil, nil
}

type classTransferServiceFakeUnitOfWork struct {
	repos repository.RepositorySet

	commitErr   error
	rollbackErr error

	commitCount   int
	rollbackCount int
}

func (f *classTransferServiceFakeUnitOfWork) Repositories() repository.RepositorySet {
	return f.repos
}

func (f *classTransferServiceFakeUnitOfWork) Commit() error {
	f.commitCount++
	return f.commitErr
}

func (f *classTransferServiceFakeUnitOfWork) Rollback() error {
	f.rollbackCount++
	return f.rollbackErr
}

type classTransferServiceFakeUnitOfWorkManager struct {
	uow repository.UnitOfWork

	beginErr   error
	beginCount int
}

func (f *classTransferServiceFakeUnitOfWorkManager) Begin(
	context.Context,
) (repository.UnitOfWork, error) {
	f.beginCount++

	if f.beginErr != nil {
		return nil, f.beginErr
	}

	return f.uow, nil
}

func newClassTransferServiceForTest(
	t *testing.T,
	repos repository.RepositorySet,
	uowManager repository.UnitOfWorkManager,
) *classTransferService {
	t.Helper()

	svc, err := NewClassTransferService(Dependencies{
		Repositories: repos,
		UOW:          uowManager,
	})
	if err != nil {
		t.Fatalf("NewClassTransferService() error = %v", err)
	}

	return svc
}

func newTestTransferRequest(
	t *testing.T,
	studentID uuid.UUID,
	academicYearID uuid.UUID,
	fromClassID uuid.UUID,
	toClassID uuid.UUID,
	requestedBy uuid.UUID,
) domain.ClassTransferRequest {
	t.Helper()

	request, err := domain.NewClassTransferRequest(
		studentID,
		academicYearID,
		fromClassID,
		toClassID,
		requestedBy,
	)
	if err != nil {
		t.Fatalf("NewClassTransferRequest() error = %v", err)
	}

	return request
}

func newTestStudentClassHistory(
	t *testing.T,
	studentID uuid.UUID,
	academicYearID uuid.UUID,
	classID uuid.UUID,
	from time.Time,
	to *time.Time,
) domain.StudentClassHistory {
	t.Helper()

	period, err := domain.NewDateRange(from, to)
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

func newTestTransferStudent(
	id uuid.UUID,
	userID uuid.UUID,
) domain.Student {
	nis := "NIS-" + id.String()
	nisn := "NISN-" + id.String()

	return domain.Student{
		ID:     id,
		UserID: &userID,
		NIS:    &nis,
		NISN:   &nisn,
		Name:   "Student Transfer",
		Status: domain.StudentActive,
	}
}

func TestNewClassTransferService(t *testing.T) {
	manager := &classTransferServiceFakeUnitOfWorkManager{}

	t.Run("valid dependencies", func(t *testing.T) {
		svc, err := NewClassTransferService(Dependencies{
			UOW: manager,
		})
		if err != nil {
			t.Fatalf(
				"NewClassTransferService() error = %v, want nil",
				err,
			)
		}

		if svc == nil {
			t.Fatal("NewClassTransferService() service = nil")
		}
	})

	t.Run("invalid dependencies", func(t *testing.T) {
		_, err := NewClassTransferService(Dependencies{})
		if !errors.Is(err, ErrInvalidDependency) {
			t.Fatalf(
				"NewClassTransferService() error = %v, want %v",
				err,
				ErrInvalidDependency,
			)
		}
	})
}

func TestClassTransferServiceRequestAdmin(t *testing.T) {
	studentID := uuid.New()
	academicYearID := uuid.New()
	fromClassID := uuid.New()
	toClassID := uuid.New()
	actorID := uuid.New()

	var created domain.ClassTransferRequest

	student, err := domain.NewStudent(
		"Ali",
		nil,
		stringPtr("NIS-"+uuid.NewString()),
		stringPtr("NISN-"+uuid.NewString()),
	)
	if err != nil {
		t.Fatalf("NewStudent() error = %v", err)
	}
	student.ID = studentID

	academicYear := newTestAcademicYear(
		academicYearID,
		domain.AcademicYearOpen,
	)

	fromClass, err := domain.NewClass("Kelas Asal", 7)
	if err != nil {
		t.Fatalf("NewClass(from) error = %v", err)
	}
	fromClass.ID = fromClassID

	toClass, err := domain.NewClass("Kelas Tujuan", 8)
	if err != nil {
		t.Fatalf("NewClass(to) error = %v", err)
	}
	toClass.ID = toClassID

	now := time.Now()
	currentHistory, err := domain.NewStudentClassHistory(
		studentID,
		academicYearID,
		fromClassID,
		domain.DateRange{
			From: academicYear.StartDate,
			To:   &academicYear.EndDate,
		},
	)
	if err != nil {
		t.Fatalf("NewStudentClassHistory() error = %v", err)
	}
	currentHistory.CreatedAt = now
	currentHistory.UpdatedAt = now

	repo := &classTransferServiceFakeClassTransferRepository{
		createFn: func(
			_ context.Context,
			request domain.ClassTransferRequest,
		) error {
			created = request
			return nil
		},
	}

	studentRepo := &classTransferServiceFakeStudentRepository{
		getByIDFn: func(
			_ context.Context,
			id uuid.UUID,
		) (domain.Student, error) {
			if id != studentID {
				t.Fatalf("GetByID() student ID = %v, want %v", id, studentID)
			}
			return student, nil
		},
	}

	academicYearRepo := &classAssignmentFakeAcademicYearRepository{
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
	}

	classRepo := &classAssignmentFakeClassRepository{
		getByIDFn: func(
			_ context.Context,
			id uuid.UUID,
		) (domain.Class, error) {
			switch id {
			case fromClassID:
				return fromClass, nil
			case toClassID:
				return toClass, nil
			default:
				t.Fatalf("GetByID() class ID = %v is unexpected", id)
				return domain.Class{}, nil
			}
		},
	}

	historyRepo := &classTransferServiceFakeStudentClassHistoryRepository{
		getCurrentFn: func(
			_ context.Context,
			studentIDArg uuid.UUID,
			academicYearIDArg uuid.UUID,
			_ time.Time,
		) (domain.StudentClassHistory, error) {
			if studentIDArg != studentID {
				t.Fatalf(
					"GetCurrentByStudentAndAcademicYear() student ID = %v, want %v",
					studentIDArg,
					studentID,
				)
			}
			if academicYearIDArg != academicYearID {
				t.Fatalf(
					"GetCurrentByStudentAndAcademicYear() academic year ID = %v, want %v",
					academicYearIDArg,
					academicYearID,
				)
			}
			return currentHistory, nil
		},
	}

	repos := repository.RepositorySet{
		Students:              studentRepo,
		AcademicYears:         academicYearRepo,
		Classes:               classRepo,
		StudentClassHistories: historyRepo,
		ClassTransferRequests: repo,
	}

	uow := &classTransferServiceFakeUnitOfWork{
		repos: repos,
	}

	svc := newClassTransferServiceForTest(
		t,
		repos,
		&classTransferServiceFakeUnitOfWorkManager{uow: uow},
	)

	out, err := svc.Request(
		context.Background(),
		RequestTransferInput{
			Actor: Actor{
				UserID: actorID,
				Role:   domain.RoleAdmin,
			},
			StudentID:      studentID,
			AcademicYearID: academicYearID,
			FromClassID:    fromClassID,
			ToClassID:      toClassID,
		},
	)
	if err != nil {
		t.Fatalf("Request() error = %v", err)
	}

	if out.TransferRequest == nil {
		t.Fatal("TransferRequest = nil")
	}

	if created.ID != out.TransferRequest.ID {
		t.Fatal("created request does not match output")
	}

	if created.Status != domain.TransferPending {
		t.Fatalf(
			"Status = %v, want %v",
			created.Status,
			domain.TransferPending,
		)
	}

	if uow.commitCount != 1 {
		t.Fatalf("Commit() count = %d, want 1", uow.commitCount)
	}

	if uow.rollbackCount != 0 {
		t.Fatalf("Rollback() count = %d, want 0", uow.rollbackCount)
	}
}

func TestClassTransferServiceRequestSiswaForbidden(t *testing.T) {
	repo := &classTransferServiceFakeClassTransferRepository{}
	manager := &classTransferServiceFakeUnitOfWorkManager{}

	svc := newClassTransferServiceForTest(
		t,
		repository.RepositorySet{
			ClassTransferRequests: repo,
		},
		manager,
	)

	_, err := svc.Request(
		context.Background(),
		RequestTransferInput{
			Actor: Actor{
				UserID: uuid.New(),
				Role:   domain.RoleSiswa,
			},
			StudentID:      uuid.New(),
			AcademicYearID: uuid.New(),
			FromClassID:    uuid.New(),
			ToClassID:      uuid.New(),
		},
	)

	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Request() error = %v, want %v", err, ErrForbidden)
	}

	if manager.beginCount != 0 {
		t.Fatalf("Begin() count = %d, want 0", manager.beginCount)
	}
}

func TestClassTransferServiceGetAdmin(t *testing.T) {
	requestID := uuid.New()
	request := newTestTransferRequest(
		t,
		uuid.New(),
		uuid.New(),
		uuid.New(),
		uuid.New(),
		uuid.New(),
	)

	request.ID = requestID

	repo := &classTransferServiceFakeClassTransferRepository{
		getByIDFn: func(
			_ context.Context,
			id uuid.UUID,
		) (domain.ClassTransferRequest, error) {
			if id != requestID {
				t.Fatalf("GetByID() id = %v, want %v", id, requestID)
			}
			return request, nil
		},
	}

	svc := newClassTransferServiceForTest(
		t,
		repository.RepositorySet{
			ClassTransferRequests: repo,
		},
		&classTransferServiceFakeUnitOfWorkManager{},
	)

	out, err := svc.Get(
		context.Background(),
		GetTransferInput{
			Actor: Actor{
				UserID: uuid.New(),
				Role:   domain.RoleAdmin,
			},
			TransferRequestID: requestID,
		},
	)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if out.TransferRequest == nil {
		t.Fatal("TransferRequest = nil")
	}

	if out.TransferRequest.ID != requestID {
		t.Fatalf(
			"ID = %v, want %v",
			out.TransferRequest.ID,
			requestID,
		)
	}
}

func TestClassTransferServiceListPendingAdmin(t *testing.T) {
	academicYearID := uuid.New()

	request := newTestTransferRequest(
		t,
		uuid.New(),
		academicYearID,
		uuid.New(),
		uuid.New(),
		uuid.New(),
	)

	repo := &classTransferServiceFakeClassTransferRepository{
		listPendingFn: func(
			_ context.Context,
			id uuid.UUID,
			_ repository.ListOptions,
		) ([]domain.ClassTransferRequest, error) {
			if id != academicYearID {
				t.Fatalf(
					"ListPending() academicYearID = %v, want %v",
					id,
					academicYearID,
				)
			}

			return []domain.ClassTransferRequest{request}, nil
		},
	}

	svc := newClassTransferServiceForTest(
		t,
		repository.RepositorySet{
			ClassTransferRequests: repo,
		},
		&classTransferServiceFakeUnitOfWorkManager{},
	)

	out, err := svc.ListPending(
		context.Background(),
		ListPendingTransfersInput{
			Actor: Actor{
				UserID: uuid.New(),
				Role:   domain.RoleAdmin,
			},
			AcademicYearID: academicYearID,
			Options: ListOptions{
				Limit:  10,
				Offset: 0,
			},
		},
	)
	if err != nil {
		t.Fatalf("ListPending() error = %v", err)
	}

	if len(out.TransferRequests) != 1 {
		t.Fatalf(
			"TransferRequests length = %d, want 1",
			len(out.TransferRequests),
		)
	}
}

func TestClassTransferServiceListPendingNonAdminForbidden(t *testing.T) {
	manager := &classTransferServiceFakeUnitOfWorkManager{}

	svc := newClassTransferServiceForTest(
		t,
		repository.RepositorySet{},
		manager,
	)

	_, err := svc.ListPending(
		context.Background(),
		ListPendingTransfersInput{
			Actor: Actor{
				UserID: uuid.New(),
				Role:   domain.RoleWaliKelas,
			},
			AcademicYearID: uuid.New(),
			Options: ListOptions{
				Limit:  10,
				Offset: 0,
			},
		},
	)

	if !errors.Is(err, ErrForbidden) {
		t.Fatalf(
			"ListPending() error = %v, want %v",
			err,
			ErrForbidden,
		)
	}

	if manager.beginCount != 0 {
		t.Fatalf("Begin() count = %d, want 0", manager.beginCount)
	}
}

func TestClassTransferServiceApprove(t *testing.T) {
	studentID := uuid.New()
	academicYearID := uuid.New()
	fromClassID := uuid.New()
	toClassID := uuid.New()
	requestID := uuid.New()
	reviewerID := uuid.New()

	startDate := time.Date(
		2026,
		7,
		1,
		0,
		0,
		0,
		0,
		time.UTC,
	)

	endDate := time.Date(
		2026,
		10,
		15,
		0,
		0,
		0,
		0,
		time.UTC,
	)

	request := newTestTransferRequest(
		t,
		studentID,
		academicYearID,
		fromClassID,
		toClassID,
		uuid.New(),
	)
	request.ID = requestID

	oldHistory := newTestStudentClassHistory(
		t,
		studentID,
		academicYearID,
		fromClassID,
		startDate,
		nil,
	)

	closedHistory := newTestStudentClassHistory(
		t,
		studentID,
		academicYearID,
		fromClassID,
		startDate,
		&endDate,
	)

	var createdHistory domain.StudentClassHistory
	var updatedRequest domain.ClassTransferRequest

	transferRepo := &classTransferServiceFakeClassTransferRepository{
		getByIDForUpdateFn: func(
			_ context.Context,
			id uuid.UUID,
		) (domain.ClassTransferRequest, error) {
			if id != requestID {
				t.Fatalf(
					"GetByIDForUpdate() id = %v, want %v",
					id,
					requestID,
				)
			}
			return request, nil
		},
		updateFn: func(
			_ context.Context,
			value domain.ClassTransferRequest,
		) error {
			updatedRequest = value
			return nil
		},
	}

	historyRepo := &classTransferServiceFakeStudentClassHistoryRepository{
		getCurrentFn: func(
			_ context.Context,
			studentIDArg uuid.UUID,
			academicYearIDArg uuid.UUID,
			date time.Time,
		) (domain.StudentClassHistory, error) {
			if studentIDArg != studentID {
				t.Fatalf(
					"GetCurrent() studentID = %v, want %v",
					studentIDArg,
					studentID,
				)
			}

			if academicYearIDArg != academicYearID {
				t.Fatalf(
					"GetCurrent() academicYearID = %v, want %v",
					academicYearIDArg,
					academicYearID,
				)
			}

			if !date.Equal(endDate) {
				t.Fatalf(
					"GetCurrent() date = %v, want %v",
					date,
					endDate,
				)
			}

			return oldHistory, nil
		},
		closeCurrentFn: func(
			_ context.Context,
			studentIDArg uuid.UUID,
			academicYearIDArg uuid.UUID,
			endDateArg time.Time,
		) (domain.StudentClassHistory, error) {
			if studentIDArg != studentID {
				t.Fatalf(
					"CloseCurrent() studentID = %v, want %v",
					studentIDArg,
					studentID,
				)
			}

			if academicYearIDArg != academicYearID {
				t.Fatalf(
					"CloseCurrent() academicYearID = %v, want %v",
					academicYearIDArg,
					academicYearID,
				)
			}

			if !endDateArg.Equal(endDate) {
				t.Fatalf(
					"CloseCurrent() endDate = %v, want %v",
					endDateArg,
					endDate,
				)
			}

			return closedHistory, nil
		},
		createFn: func(
			_ context.Context,
			value domain.StudentClassHistory,
		) error {
			createdHistory = value
			return nil
		},
	}

	academicYear := newTestAcademicYear(
		academicYearID,
		domain.AcademicYearOpen,
	)

	academicYearRepo := &classAssignmentFakeAcademicYearRepository{
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
	}

	uow := &classTransferServiceFakeUnitOfWork{
		repos: repository.RepositorySet{
			AcademicYears:         academicYearRepo,
			ClassTransferRequests: transferRepo,
			StudentClassHistories: historyRepo,
		},
	}

	svc := newClassTransferServiceForTest(
		t,
		repository.RepositorySet{},
		&classTransferServiceFakeUnitOfWorkManager{uow: uow},
	)

	out, err := svc.Approve(
		context.Background(),
		ApproveTransferInput{
			Actor: Actor{
				UserID: reviewerID,
				Role:   domain.RoleAdmin,
			},
			TransferRequestID: requestID,
			EffectiveDate:     endDate,
		},
	)
	if err != nil {
		t.Fatalf("Approve() error = %v", err)
	}

	if transferRepo.getByIDForUpdateCalls != 1 {
		t.Fatalf(
			"GetByIDForUpdate() count = %d, want 1",
			transferRepo.getByIDForUpdateCalls,
		)
	}

	if historyRepo.closeCurrentCalls != 1 {
		t.Fatalf(
			"CloseCurrent() count = %d, want 1",
			historyRepo.closeCurrentCalls,
		)
	}

	if historyRepo.createCalls != 1 {
		t.Fatalf(
			"Create history count = %d, want 1",
			historyRepo.createCalls,
		)
	}

	if uow.commitCount != 1 {
		t.Fatalf("Commit() count = %d, want 1", uow.commitCount)
	}

	if uow.rollbackCount != 0 {
		t.Fatalf("Rollback() count = %d, want 0", uow.rollbackCount)
	}

	if updatedRequest.Status != domain.TransferApproved {
		t.Fatalf(
			"updated request status = %v, want %v",
			updatedRequest.Status,
			domain.TransferApproved,
		)
	}

	if createdHistory.StudentID != studentID {
		t.Fatalf(
			"new history StudentID = %v, want %v",
			createdHistory.StudentID,
			studentID,
		)
	}

	if createdHistory.AcademicYearID != academicYearID {
		t.Fatalf(
			"new history AcademicYearID = %v, want %v",
			createdHistory.AcademicYearID,
			academicYearID,
		)
	}

	if createdHistory.ClassID != toClassID {
		t.Fatalf(
			"new history ClassID = %v, want %v",
			createdHistory.ClassID,
			toClassID,
		)
	}

	if !createdHistory.Period.From.Equal(endDate) {
		t.Fatalf(
			"new history Period.From = %v, want %v",
			createdHistory.Period.From,
			endDate,
		)
	}

	if createdHistory.Period.To != nil {
		t.Fatal("new history Period.To must be nil")
	}

	if oldHistory.ClassID == createdHistory.ClassID {
		t.Fatal("new history must use destination class")
	}

	if out.TransferRequest == nil {
		t.Fatal("Approve() TransferRequest = nil")
	}

	if out.ClassHistory == nil {
		t.Fatal("Approve() ClassHistory = nil")
	}
}

func TestClassTransferServiceApproveCloseHistoryErrorRollsBack(t *testing.T) {
	expectedErr := errors.New("close history failed")

	request := newTestTransferRequest(
		t,
		uuid.New(),
		uuid.New(),
		uuid.New(),
		uuid.New(),
		uuid.New(),
	)

	effectiveDate := time.Date(
		2026,
		10,
		15,
		0,
		0,
		0,
		0,
		time.UTC,
	)

	academicYear := newTestAcademicYear(
		request.AcademicYearID,
		domain.AcademicYearOpen,
	)

	academicYearRepo := &classAssignmentFakeAcademicYearRepository{
		getByIDFn: func(
			_ context.Context,
			id uuid.UUID,
		) (domain.AcademicYear, error) {
			return academicYear, nil
		},
	}

	currentHistory := newTestStudentClassHistory(
		t,
		request.StudentID,
		request.AcademicYearID,
		request.FromClassID,
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		nil,
	)

	transferRepo := &classTransferServiceFakeClassTransferRepository{
		getByIDForUpdateFn: func(
			context.Context,
			uuid.UUID,
		) (domain.ClassTransferRequest, error) {
			return request, nil
		},
	}

	historyRepo := &classTransferServiceFakeStudentClassHistoryRepository{
		getCurrentFn: func(
			context.Context,
			uuid.UUID,
			uuid.UUID,
			time.Time,
		) (domain.StudentClassHistory, error) {
			return currentHistory, nil
		},
		closeCurrentFn: func(
			context.Context,
			uuid.UUID,
			uuid.UUID,
			time.Time,
		) (domain.StudentClassHistory, error) {
			return domain.StudentClassHistory{}, expectedErr
		},
	}

	uow := &classTransferServiceFakeUnitOfWork{
		repos: repository.RepositorySet{
			AcademicYears:         academicYearRepo,
			ClassTransferRequests: transferRepo,
			StudentClassHistories: historyRepo,
		},
	}

	svc := newClassTransferServiceForTest(
		t,
		repository.RepositorySet{},
		&classTransferServiceFakeUnitOfWorkManager{uow: uow},
	)

	_, err := svc.Approve(
		context.Background(),
		ApproveTransferInput{
			Actor: Actor{
				UserID: uuid.New(),
				Role:   domain.RoleAdmin,
			},
			TransferRequestID: request.ID,
			EffectiveDate:     effectiveDate,
		},
	)

	if !errors.Is(err, expectedErr) {
		t.Fatalf("Approve() error = %v, want %v", err, expectedErr)
	}

	if uow.commitCount != 0 {
		t.Fatalf("Commit() count = %d, want 0", uow.commitCount)
	}

	if uow.rollbackCount != 1 {
		t.Fatalf("Rollback() count = %d, want 1", uow.rollbackCount)
	}
}

func TestClassTransferServiceApproveCreateHistoryErrorRollsBack(t *testing.T) {
	expectedErr := errors.New("create history failed")

	request := newTestTransferRequest(
		t,
		uuid.New(),
		uuid.New(),
		uuid.New(),
		uuid.New(),
		uuid.New(),
	)

	endDate := time.Date(
		2026,
		10,
		15,
		0,
		0,
		0,
		0,
		time.UTC,
	)

	closedHistory := newTestStudentClassHistory(
		t,
		request.StudentID,
		request.AcademicYearID,
		request.FromClassID,
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		&endDate,
	)

	effectiveDate := endDate

	academicYear := newTestAcademicYear(
		request.AcademicYearID,
		domain.AcademicYearOpen,
	)

	transferRepo := &classTransferServiceFakeClassTransferRepository{
		getByIDForUpdateFn: func(
			context.Context,
			uuid.UUID,
		) (domain.ClassTransferRequest, error) {
			return request, nil
		},
	}

	currentHistory := newTestStudentClassHistory(
		t,
		request.StudentID,
		request.AcademicYearID,
		request.FromClassID,
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		nil,
	)

	historyRepo := &classTransferServiceFakeStudentClassHistoryRepository{
		getCurrentFn: func(
			context.Context,
			uuid.UUID,
			uuid.UUID,
			time.Time,
		) (domain.StudentClassHistory, error) {
			return currentHistory, nil
		},
		closeCurrentFn: func(
			context.Context,
			uuid.UUID,
			uuid.UUID,
			time.Time,
		) (domain.StudentClassHistory, error) {
			return closedHistory, nil
		},
		createFn: func(
			context.Context,
			domain.StudentClassHistory,
		) error {
			return expectedErr
		},
	}

	academicYearRepo := &classAssignmentFakeAcademicYearRepository{
		getByIDFn: func(
			_ context.Context,
			id uuid.UUID,
		) (domain.AcademicYear, error) {
			return academicYear, nil
		},
	}

	uow := &classTransferServiceFakeUnitOfWork{
		repos: repository.RepositorySet{
			AcademicYears:         academicYearRepo,
			ClassTransferRequests: transferRepo,
			StudentClassHistories: historyRepo,
		},
	}

	svc := newClassTransferServiceForTest(
		t,
		repository.RepositorySet{},
		&classTransferServiceFakeUnitOfWorkManager{uow: uow},
	)

	_, err := svc.Approve(
		context.Background(),
		ApproveTransferInput{
			Actor: Actor{
				UserID: uuid.New(),
				Role:   domain.RoleAdmin,
			},
			TransferRequestID: request.ID,
			EffectiveDate:     effectiveDate,
		},
	)

	if !errors.Is(err, expectedErr) {
		t.Fatalf("Approve() error = %v, want %v", err, expectedErr)
	}

	if uow.commitCount != 0 {
		t.Fatalf("Commit() count = %d, want 0", uow.commitCount)
	}

	if uow.rollbackCount != 1 {
		t.Fatalf("Rollback() count = %d, want 1", uow.rollbackCount)
	}
}

func TestClassTransferServiceRejectAdmin(t *testing.T) {
	request := newTestTransferRequest(
		t,
		uuid.New(),
		uuid.New(),
		uuid.New(),
		uuid.New(),
		uuid.New(),
	)

	var updated domain.ClassTransferRequest

	repo := &classTransferServiceFakeClassTransferRepository{
		getByIDForUpdateFn: func(
			context.Context,
			uuid.UUID,
		) (domain.ClassTransferRequest, error) {
			return request, nil
		},
		updateFn: func(
			_ context.Context,
			value domain.ClassTransferRequest,
		) error {
			updated = value
			return nil
		},
	}

	uow := &classTransferServiceFakeUnitOfWork{
		repos: repository.RepositorySet{
			ClassTransferRequests: repo,
		},
	}

	svc := newClassTransferServiceForTest(
		t,
		repository.RepositorySet{},
		&classTransferServiceFakeUnitOfWorkManager{uow: uow},
	)

	out, err := svc.Reject(
		context.Background(),
		RejectTransferInput{
			Actor: Actor{
				UserID: uuid.New(),
				Role:   domain.RoleAdmin,
			},
			TransferRequestID: request.ID,
			RejectionReason:   "Kelas tujuan belum tersedia",
		},
	)
	if err != nil {
		t.Fatalf("Reject() error = %v", err)
	}

	if updated.Status != domain.TransferRejected {
		t.Fatalf(
			"status = %v, want %v",
			updated.Status,
			domain.TransferRejected,
		)
	}

	if updated.RejectionReason == nil {
		t.Fatal("RejectionReason = nil")
	}

	if *updated.RejectionReason != "Kelas tujuan belum tersedia" {
		t.Fatalf(
			"RejectionReason = %q, want %q",
			*updated.RejectionReason,
			"Kelas tujuan belum tersedia",
		)
	}

	if out.TransferRequest == nil {
		t.Fatal("Reject() TransferRequest = nil")
	}

	if uow.commitCount != 1 {
		t.Fatalf("Commit() count = %d, want 1", uow.commitCount)
	}
}

func TestClassTransferServiceRejectEmptyReason(t *testing.T) {
	manager := &classTransferServiceFakeUnitOfWorkManager{}

	svc := newClassTransferServiceForTest(
		t,
		repository.RepositorySet{},
		manager,
	)

	_, err := svc.Reject(
		context.Background(),
		RejectTransferInput{
			Actor: Actor{
				UserID: uuid.New(),
				Role:   domain.RoleAdmin,
			},
			TransferRequestID: uuid.New(),
			RejectionReason:   "   ",
		},
	)

	if !errors.Is(err, domain.ErrEmptyRejectionReason) {
		t.Fatalf(
			"Reject() error = %v, want %v",
			err,
			domain.ErrEmptyRejectionReason,
		)
	}

	if manager.beginCount != 0 {
		t.Fatalf("Begin() count = %d, want 0", manager.beginCount)
	}
}

func TestClassTransferServiceRejectNonAdminForbidden(t *testing.T) {
	manager := &classTransferServiceFakeUnitOfWorkManager{}

	svc := newClassTransferServiceForTest(
		t,
		repository.RepositorySet{},
		manager,
	)

	_, err := svc.Reject(
		context.Background(),
		RejectTransferInput{
			Actor: Actor{
				UserID: uuid.New(),
				Role:   domain.RoleWaliKelas,
			},
			TransferRequestID: uuid.New(),
			RejectionReason:   "Tidak disetujui",
		},
	)

	if !errors.Is(err, ErrForbidden) {
		t.Fatalf(
			"Reject() error = %v, want %v",
			err,
			ErrForbidden,
		)
	}

	if manager.beginCount != 0 {
		t.Fatalf("Begin() count = %d, want 0", manager.beginCount)
	}
}
