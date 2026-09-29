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

func TestClassTransferRequestRepository_CreateAndGetByID(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewClassTransferRequestRepository(db)
	ctx := context.Background()

	fixture := newClassTransferRequestFixture(t, db)

	request := newTestClassTransferRequest(
		t,
		fixture.student.ID,
		fixture.academicYear.ID,
		fixture.fromClass.ID,
		fixture.toClass.ID,
		fixture.requestedBy.ID,
	)

	if err := repo.Create(ctx, request); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestClassTransferRequest(t, db, request.ID)
	})

	got, err := repo.GetByID(ctx, request.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	assertClassTransferRequestEqual(t, got, request)
}

func TestClassTransferRequestRepository_GetByIDNotFound(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewClassTransferRequestRepository(db)

	_, err := repo.GetByID(context.Background(), uuid.New())
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("GetByID() error = %v, want ErrNotFound", err)
	}
}

func TestClassTransferRequestRepository_ListByStudentAndAcademicYear(
	t *testing.T,
) {
	db := openTestDatabase(t)
	repo := postgres.NewClassTransferRequestRepository(db)
	ctx := context.Background()

	fixture := newClassTransferRequestFixture(t, db)

	first := newTestClassTransferRequest(
		t,
		fixture.student.ID,
		fixture.academicYear.ID,
		fixture.fromClass.ID,
		fixture.toClass.ID,
		fixture.requestedBy.ID,
	)
	first.CreatedAt = time.Date(
		2026, 7, 2, 10, 0, 0, 0, time.UTC,
	)
	first.UpdatedAt = first.CreatedAt

	second := newTestClassTransferRequest(
		t,
		fixture.student.ID,
		fixture.academicYear.ID,
		fixture.toClass.ID,
		fixture.thirdClass.ID,
		fixture.requestedBy.ID,
	)
	second.CreatedAt = time.Date(
		2026, 8, 2, 10, 0, 0, 0, time.UTC,
	)
	second.UpdatedAt = second.CreatedAt

	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}

	if err := repo.Create(ctx, second); err != nil {
		t.Fatalf("Create(second) error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestClassTransferRequest(t, db, first.ID)
		deleteTestClassTransferRequest(t, db, second.ID)
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
		t.Fatalf(
			"ListByStudentAndAcademicYear() error = %v",
			err,
		)
	}

	if len(got) != 2 {
		t.Fatalf(
			"ListByStudentAndAcademicYear() length = %d, want 2",
			len(got),
		)
	}

	assertClassTransferRequestEqual(t, got[0], first)
	assertClassTransferRequestEqual(t, got[1], second)
}

func TestClassTransferRequestRepository_ListByStudentAndAcademicYearEmpty(
	t *testing.T,
) {
	db := openTestDatabase(t)
	repo := postgres.NewClassTransferRequestRepository(db)

	fixture := newClassTransferRequestFixture(t, db)

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
		t.Fatalf(
			"ListByStudentAndAcademicYear() error = %v",
			err,
		)
	}

	if got == nil {
		t.Fatal(
			"ListByStudentAndAcademicYear() returned nil slice, want empty slice",
		)
	}

	if len(got) != 0 {
		t.Fatalf(
			"ListByStudentAndAcademicYear() length = %d, want 0",
			len(got),
		)
	}
}

func TestClassTransferRequestRepository_ListPending(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewClassTransferRequestRepository(db)
	ctx := context.Background()

	fixture := newClassTransferRequestFixture(t, db)

	first := newTestClassTransferRequest(
		t,
		fixture.student.ID,
		fixture.academicYear.ID,
		fixture.fromClass.ID,
		fixture.toClass.ID,
		fixture.requestedBy.ID,
	)
	first.CreatedAt = time.Date(
		2026, 7, 2, 10, 0, 0, 0, time.UTC,
	)
	first.UpdatedAt = first.CreatedAt

	second := newTestClassTransferRequest(
		t,
		fixture.student.ID,
		fixture.academicYear.ID,
		fixture.toClass.ID,
		fixture.thirdClass.ID,
		fixture.requestedBy.ID,
	)
	second.CreatedAt = time.Date(
		2026, 8, 2, 10, 0, 0, 0, time.UTC,
	)
	second.UpdatedAt = second.CreatedAt

	if err := second.Approve(
		fixture.reviewedBy.ID,
		time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC),
	); err != nil {
		t.Fatalf("Approve() error = %v", err)
	}

	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}

	if err := repo.Create(ctx, second); err != nil {
		t.Fatalf("Create(second) error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestClassTransferRequest(t, db, first.ID)
		deleteTestClassTransferRequest(t, db, second.ID)
	})

	got, err := repo.ListPending(
		ctx,
		fixture.academicYear.ID,
		repository.ListOptions{
			Limit:  10,
			Offset: 0,
		},
	)
	if err != nil {
		t.Fatalf("ListPending() error = %v", err)
	}

	if len(got) != 1 {
		t.Fatalf(
			"ListPending() length = %d, want 1",
			len(got),
		)
	}

	assertClassTransferRequestEqual(t, got[0], first)
}

func TestClassTransferRequestRepository_ListPendingEmpty(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewClassTransferRequestRepository(db)

	fixture := newClassTransferRequestFixture(t, db)

	got, err := repo.ListPending(
		context.Background(),
		fixture.academicYear.ID,
		repository.ListOptions{
			Limit:  10,
			Offset: 0,
		},
	)
	if err != nil {
		t.Fatalf("ListPending() error = %v", err)
	}

	if got == nil {
		t.Fatal("ListPending() returned nil slice, want empty slice")
	}

	if len(got) != 0 {
		t.Fatalf(
			"ListPending() length = %d, want 0",
			len(got),
		)
	}
}

func TestClassTransferRequestRepository_ListPagination(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewClassTransferRequestRepository(db)
	ctx := context.Background()

	fixture := newClassTransferRequestFixture(t, db)

	requests := make([]domain.ClassTransferRequest, 0, 3)

	classPairs := [][2]uuid.UUID{
		{fixture.fromClass.ID, fixture.toClass.ID},
		{fixture.toClass.ID, fixture.thirdClass.ID},
		{fixture.thirdClass.ID, fixture.fromClass.ID},
	}

	for i, classPair := range classPairs {
		request := newTestClassTransferRequest(
			t,
			fixture.student.ID,
			fixture.academicYear.ID,
			classPair[0],
			classPair[1],
			fixture.requestedBy.ID,
		)
		request.CreatedAt = time.Date(
			2026,
			time.Month(7+i),
			2,
			10,
			0,
			0,
			0,
			time.UTC,
		)
		request.UpdatedAt = request.CreatedAt

		if err := repo.Create(ctx, request); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		requests = append(requests, request)
	}

	t.Cleanup(func() {
		for _, request := range requests {
			deleteTestClassTransferRequest(t, db, request.ID)
		}
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
		t.Fatalf(
			"ListByStudentAndAcademicYear() error = %v",
			err,
		)
	}

	if len(got) != 1 {
		t.Fatalf(
			"ListByStudentAndAcademicYear() length = %d, want 1",
			len(got),
		)
	}

	assertClassTransferRequestEqual(t, got[0], requests[1])
}

func TestClassTransferRequestRepository_UpdatePending(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewClassTransferRequestRepository(db)
	ctx := context.Background()

	fixture := newClassTransferRequestFixture(t, db)

	request := newTestClassTransferRequest(
		t,
		fixture.student.ID,
		fixture.academicYear.ID,
		fixture.fromClass.ID,
		fixture.toClass.ID,
		fixture.requestedBy.ID,
	)

	if err := repo.Create(ctx, request); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestClassTransferRequest(t, db, request.ID)
	})

	request.UpdatedAt = request.UpdatedAt.Add(time.Minute)

	if err := repo.Update(ctx, request); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, err := repo.GetByID(ctx, request.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	assertClassTransferRequestEqual(t, got, request)
}

func TestClassTransferRequestRepository_UpdateApproved(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewClassTransferRequestRepository(db)
	ctx := context.Background()

	fixture := newClassTransferRequestFixture(t, db)

	request := newTestClassTransferRequest(
		t,
		fixture.student.ID,
		fixture.academicYear.ID,
		fixture.fromClass.ID,
		fixture.toClass.ID,
		fixture.requestedBy.ID,
	)

	if err := repo.Create(ctx, request); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestClassTransferRequest(t, db, request.ID)
	})

	if err := request.Approve(
		fixture.reviewedBy.ID,
		time.Date(2026, 8, 10, 9, 0, 0, 0, time.UTC),
	); err != nil {
		t.Fatalf("Approve() error = %v", err)
	}

	if err := repo.Update(ctx, request); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, err := repo.GetByID(ctx, request.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	assertClassTransferRequestEqual(t, got, request)

	if got.ReviewedBy == nil {
		t.Fatal("ReviewedBy = nil, want reviewer")
	}

	if got.ReviewedAt == nil {
		t.Fatal("ReviewedAt = nil, want review time")
	}

	if got.RejectionReason != nil {
		t.Fatalf(
			"RejectionReason = %v, want nil",
			*got.RejectionReason,
		)
	}
}

func TestClassTransferRequestRepository_UpdateRejected(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewClassTransferRequestRepository(db)
	ctx := context.Background()

	fixture := newClassTransferRequestFixture(t, db)

	request := newTestClassTransferRequest(
		t,
		fixture.student.ID,
		fixture.academicYear.ID,
		fixture.fromClass.ID,
		fixture.toClass.ID,
		fixture.requestedBy.ID,
	)

	if err := repo.Create(ctx, request); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestClassTransferRequest(t, db, request.ID)
	})

	const reason = "Permintaan tidak disetujui"

	if err := request.Reject(
		fixture.reviewedBy.ID,
		time.Date(2026, 8, 11, 9, 0, 0, 0, time.UTC),
		reason,
	); err != nil {
		t.Fatalf("Reject() error = %v", err)
	}

	if err := repo.Update(ctx, request); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, err := repo.GetByID(ctx, request.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	assertClassTransferRequestEqual(t, got, request)

	if got.ReviewedBy == nil {
		t.Fatal("ReviewedBy = nil, want reviewer")
	}

	if got.ReviewedAt == nil {
		t.Fatal("ReviewedAt = nil, want review time")
	}

	if got.RejectionReason == nil {
		t.Fatal("RejectionReason = nil, want reason")
	}

	if *got.RejectionReason != reason {
		t.Fatalf(
			"RejectionReason = %q, want %q",
			*got.RejectionReason,
			reason,
		)
	}
}

func TestClassTransferRequestRepository_UpdateNotFound(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewClassTransferRequestRepository(db)

	request, err := domain.NewClassTransferRequest(
		uuid.New(),
		uuid.New(),
		uuid.New(),
		uuid.New(),
		uuid.New(),
	)
	if err != nil {
		t.Fatalf(
			"NewClassTransferRequest() error = %v",
			err,
		)
	}

	err = repo.Update(context.Background(), request)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf(
			"Update() error = %v, want ErrNotFound",
			err,
		)
	}
}

type classTransferRequestFixture struct {
	user         domain.User
	requestedBy  domain.User
	reviewedBy   domain.User
	student      domain.Student
	academicYear domain.AcademicYear
	fromClass    domain.Class
	toClass      domain.Class
	thirdClass   domain.Class
}

func newClassTransferRequestFixture(
	t *testing.T,
	db *sql.DB,
) classTransferRequestFixture {
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

	requestedBy := newTestUser(t)
	if err := userRepo.Create(ctx, requestedBy); err != nil {
		t.Fatalf("create fixture requestedBy user: %v", err)
	}

	t.Cleanup(func() {
		deleteTestUser(t, db, requestedBy.ID)
	})

	reviewedBy := newTestUser(t)
	if err := userRepo.Create(ctx, reviewedBy); err != nil {
		t.Fatalf("create fixture reviewedBy user: %v", err)
	}

	t.Cleanup(func() {
		deleteTestUser(t, db, reviewedBy.ID)
	})

	student := newTestStudentWithUserID(
		t,
		"Ahmad Transfer",
		user.ID,
	)
	if err := studentRepo.Create(ctx, student); err != nil {
		t.Fatalf("create fixture student: %v", err)
	}

	t.Cleanup(func() {
		deleteTestStudent(t, db, student.ID)
	})

	academicYear := newTestAcademicYear(t)
	academicYear.Name = "2026/2027"
	academicYear.StartDate = time.Date(
		2026, 7, 1, 0, 0, 0, 0, time.UTC,
	)
	academicYear.EndDate = time.Date(
		2027, 7, 1, 0, 0, 0, 0, time.UTC,
	)

	if err := academicYearRepo.Create(ctx, academicYear); err != nil {
		t.Fatalf(
			"create fixture academic year: %v",
			err,
		)
	}

	t.Cleanup(func() {
		deleteTestAcademicYear(t, db, academicYear.ID)
	})

	fromClass, err := domain.NewClass("7A Transfer", 7)
	if err != nil {
		t.Fatalf("NewClass(fromClass) error = %v", err)
	}

	if err := classRepo.Create(ctx, fromClass); err != nil {
		t.Fatalf("create fixture from class: %v", err)
	}

	t.Cleanup(func() {
		deleteTestClass(t, db, fromClass.ID)
	})

	toClass, err := domain.NewClass("7B Transfer", 7)
	if err != nil {
		t.Fatalf("NewClass(toClass) error = %v", err)
	}

	if err := classRepo.Create(ctx, toClass); err != nil {
		t.Fatalf("create fixture to class: %v", err)
	}

	t.Cleanup(func() {
		deleteTestClass(t, db, toClass.ID)
	})

	thirdClass, err := domain.NewClass("7C Transfer", 7)
	if err != nil {
		t.Fatalf("NewClass(thirdClass) error = %v", err)
	}

	if err := classRepo.Create(ctx, thirdClass); err != nil {
		t.Fatalf("create fixture third class: %v", err)
	}

	t.Cleanup(func() {
		deleteTestClass(t, db, thirdClass.ID)
	})

	return classTransferRequestFixture{
		user:         user,
		requestedBy:  requestedBy,
		reviewedBy:   reviewedBy,
		student:      student,
		academicYear: academicYear,
		fromClass:    fromClass,
		toClass:      toClass,
		thirdClass:   thirdClass,
	}
}

func newTestClassTransferRequest(
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
		t.Fatalf(
			"NewClassTransferRequest() error = %v",
			err,
		)
	}

	return request
}

func assertClassTransferRequestEqual(
	t *testing.T,
	got domain.ClassTransferRequest,
	want domain.ClassTransferRequest,
) {
	t.Helper()

	if got.ID != want.ID {
		t.Errorf("ID = %v, want %v", got.ID, want.ID)
	}

	if got.StudentID != want.StudentID {
		t.Errorf(
			"StudentID = %v, want %v",
			got.StudentID,
			want.StudentID,
		)
	}

	if got.AcademicYearID != want.AcademicYearID {
		t.Errorf(
			"AcademicYearID = %v, want %v",
			got.AcademicYearID,
			want.AcademicYearID,
		)
	}

	if got.FromClassID != want.FromClassID {
		t.Errorf(
			"FromClassID = %v, want %v",
			got.FromClassID,
			want.FromClassID,
		)
	}

	if got.ToClassID != want.ToClassID {
		t.Errorf(
			"ToClassID = %v, want %v",
			got.ToClassID,
			want.ToClassID,
		)
	}

	if got.RequestedBy != want.RequestedBy {
		t.Errorf(
			"RequestedBy = %v, want %v",
			got.RequestedBy,
			want.RequestedBy,
		)
	}

	if got.Status != want.Status {
		t.Errorf(
			"Status = %v, want %v",
			got.Status,
			want.Status,
		)
	}

	if got.ReviewedBy == nil && want.ReviewedBy == nil {
	} else if got.ReviewedBy == nil || want.ReviewedBy == nil {
		t.Errorf(
			"ReviewedBy = %v, want %v",
			got.ReviewedBy,
			want.ReviewedBy,
		)
	} else if *got.ReviewedBy != *want.ReviewedBy {
		t.Errorf(
			"ReviewedBy = %v, want %v",
			*got.ReviewedBy,
			*want.ReviewedBy,
		)
	}

	if got.ReviewedAt == nil && want.ReviewedAt == nil {
	} else if got.ReviewedAt == nil || want.ReviewedAt == nil {
		t.Errorf(
			"ReviewedAt = %v, want %v",
			got.ReviewedAt,
			want.ReviewedAt,
		)
	} else if !got.ReviewedAt.Equal(*want.ReviewedAt) {
		t.Errorf(
			"ReviewedAt = %v, want %v",
			*got.ReviewedAt,
			*want.ReviewedAt,
		)
	}

	if got.RejectionReason == nil && want.RejectionReason == nil {
	} else if got.RejectionReason == nil || want.RejectionReason == nil {
		t.Errorf(
			"RejectionReason = %v, want %v",
			got.RejectionReason,
			want.RejectionReason,
		)
	} else if *got.RejectionReason != *want.RejectionReason {
		t.Errorf(
			"RejectionReason = %q, want %q",
			*got.RejectionReason,
			*want.RejectionReason,
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

func deleteTestClassTransferRequest(
	t *testing.T,
	db *sql.DB,
	id uuid.UUID,
) {
	t.Helper()

	_, err := db.ExecContext(
		context.Background(),
		"DELETE FROM class_transfer_requests WHERE id = $1",
		id,
	)
	if err != nil {
		t.Errorf(
			"delete test class transfer request: %v",
			err,
		)
	}
}
