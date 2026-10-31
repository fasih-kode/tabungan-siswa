package service_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/repository/postgres"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

type transferRequestFixture struct {
	db         *sql.DB
	repos      repository.RepositorySet
	manager    repository.UnitOfWorkManager
	svc        service.ClassTransferService
	input      service.RequestTransferInput
	year       domain.AcademicYear
	wrongClass domain.Class
	admin      service.Actor
}

func newTransferRequestFixture(t *testing.T) transferRequestFixture {
	t.Helper()
	ctx := context.Background()
	db := openServiceScopeTestDatabase(t)
	cleanupServiceScopeFixtures(t, db)
	t.Cleanup(func() { cleanupServiceScopeFixtures(t, db) })
	f := transferRequestFixture{db: db, repos: postgres.NewRepositorySet(db), manager: postgres.NewUnitOfWorkManager(db)}
	admin, wali := newServiceScopeUser(t, domain.RoleAdmin), newServiceScopeUser(t, domain.RoleWaliKelas)
	for _, user := range []domain.User{admin, wali} {
		if err := f.repos.Users.Create(ctx, user); err != nil {
			t.Fatal(err)
		}
	}
	f.admin = service.Actor{UserID: admin.ID, Role: domain.RoleAdmin}
	previous := newServiceScopeAcademicYearForStartYear(t, "2025/2026", 2025)
	previous.Status = domain.AcademicYearClosed
	f.year = newServiceScopeAcademicYear(t, "2026/2027")
	for _, year := range []domain.AcademicYear{previous, f.year} {
		if err := f.repos.AcademicYears.Create(ctx, year); err != nil {
			t.Fatal(err)
		}
	}
	classes := make([]domain.Class, 3)
	for i, year := range []domain.AcademicYear{f.year, f.year, previous} {
		class := newServiceScopeClass(t, "Kelas "+uuid.NewString(), 7)
		class.AcademicYearID, class.EffectiveDate = year.ID, year.StartDate
		if err := f.repos.Classes.Create(ctx, class); err != nil {
			t.Fatal(err)
		}
		classes[i] = class
	}
	f.wrongClass = classes[2]
	student := newServiceScopeStudent(t, "Siswa uji transfer")
	if err := f.repos.Students.Create(ctx, student); err != nil {
		t.Fatal(err)
	}
	if err := f.repos.StudentClassHistories.Create(ctx, newServiceScopeStudentHistory(t, student.ID, f.year.ID, classes[0].ID)); err != nil {
		t.Fatal(err)
	}
	if err := f.repos.TeacherClassAssignments.Create(ctx, newServiceScopeAssignment(t, wali.ID, f.year.ID, classes[0].ID)); err != nil {
		t.Fatal(err)
	}
	f.input = service.RequestTransferInput{Actor: service.Actor{UserID: wali.ID, Role: domain.RoleWaliKelas}, StudentID: student.ID, AcademicYearID: f.year.ID, FromClassID: classes[0].ID, ToClassID: classes[1].ID}
	var err error
	f.svc, err = service.NewClassTransferService(service.Dependencies{Repositories: f.repos, UOW: f.manager})
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func (f transferRequestFixture) assertRequestCount(t *testing.T, want int) {
	t.Helper()
	var count int
	if err := f.db.QueryRow(`SELECT count(*) FROM class_transfer_requests WHERE student_id=$1`, f.input.StudentID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("request count=%d, want %d", count, want)
	}
	histories, err := f.repos.StudentClassHistories.ListByStudentAndAcademicYear(context.Background(), f.input.StudentID, f.year.ID, repository.ListOptions{Limit: 10})
	if err != nil || len(histories) != 1 || histories[0].ClassID != f.input.FromClassID {
		t.Fatalf("placement changed: %+v, %v", histories, err)
	}
}

func TestClassTransferRequestIntegrationValidation(t *testing.T) {
	cases := []struct {
		name   string
		change func(*testing.T, *transferRequestFixture, *service.RequestTransferInput)
		want   error
	}{
		{name: "sah"},
		{name: "role admin", change: func(_ *testing.T, f *transferRequestFixture, in *service.RequestTransferInput) { in.Actor = f.admin }, want: service.ErrForbidden},
		{name: "role siswa", change: func(_ *testing.T, _ *transferRequestFixture, in *service.RequestTransferInput) {
			in.Actor.Role = domain.RoleSiswa
		}, want: service.ErrForbidden},
		{name: "scope wali", change: func(_ *testing.T, _ *transferRequestFixture, in *service.RequestTransferInput) {
			in.Actor.UserID = uuid.New()
		}, want: service.ErrScopeViolation},
		{name: "siswa nonaktif", change: func(t *testing.T, f *transferRequestFixture, _ *service.RequestTransferInput) {
			student, err := f.repos.Students.GetByID(context.Background(), f.input.StudentID)
			if err != nil {
				t.Fatal(err)
			}
			if err := student.Leave(); err != nil {
				t.Fatal(err)
			}
			if err := f.repos.Students.Update(context.Background(), student); err != nil {
				t.Fatal(err)
			}
		}, want: domain.ErrInvalidTransition},
		{name: "tahun closing", change: func(t *testing.T, f *transferRequestFixture, _ *service.RequestTransferInput) {
			year := f.year
			year.Status = domain.AcademicYearClosing
			if err := f.repos.AcademicYears.Update(context.Background(), year); err != nil {
				t.Fatal(err)
			}
		}, want: domain.ErrInvalidTransition},
		{name: "tahun closed", change: func(t *testing.T, f *transferRequestFixture, _ *service.RequestTransferInput) {
			year := f.year
			year.Status = domain.AcademicYearClosed
			if err := f.repos.AcademicYears.Update(context.Background(), year); err != nil {
				t.Fatal(err)
			}
		}, want: domain.ErrInvalidTransition},
		{name: "kelas sama", change: func(_ *testing.T, _ *transferRequestFixture, in *service.RequestTransferInput) {
			in.ToClassID = in.FromClassID
		}, want: domain.ErrInvalidValue},
		{name: "kelas asal lintas tahun", change: func(_ *testing.T, f *transferRequestFixture, in *service.RequestTransferInput) {
			in.FromClassID = f.wrongClass.ID
		}, want: domain.ErrInvalidValue},
		{name: "kelas tujuan lintas tahun", change: func(_ *testing.T, f *transferRequestFixture, in *service.RequestTransferInput) {
			in.ToClassID = f.wrongClass.ID
		}, want: domain.ErrInvalidValue},
		{name: "placement asal berbeda", change: func(_ *testing.T, _ *transferRequestFixture, in *service.RequestTransferInput) {
			in.FromClassID, in.ToClassID = in.ToClassID, in.FromClassID
		}, want: service.ErrInvalidDependency},
		{name: "konflik pending", change: func(t *testing.T, f *transferRequestFixture, _ *service.RequestTransferInput) {
			if _, err := f.svc.Request(context.Background(), f.input); err != nil {
				t.Fatal(err)
			}
		}, want: repository.ErrConflict},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newTransferRequestFixture(t)
			in := f.input
			if tc.change != nil {
				tc.change(t, &f, &in)
			}
			out, err := f.svc.Request(context.Background(), in)
			if !errors.Is(err, tc.want) {
				t.Fatalf("Request error=%v, want %v", err, tc.want)
			}
			want := 0
			if tc.want == nil || errors.Is(tc.want, repository.ErrConflict) {
				want = 1
			}
			f.assertRequestCount(t, want)
			if tc.want == nil && (out.TransferRequest == nil || out.TransferRequest.Status != domain.TransferPending) {
				t.Fatalf("request=%+v", out.TransferRequest)
			}
		})
	}
}

// Hook berada pada repository UOW nyata; kegagalan sesudah INSERT membuktikan rollback row tersebut.
type transferRequestHookRepository struct {
	repository.ClassTransferRequestRepository
	afterCreate func(context.Context) error
}

func (r transferRequestHookRepository) Create(ctx context.Context, request domain.ClassTransferRequest) error {
	if err := r.ClassTransferRequestRepository.Create(ctx, request); err != nil {
		return err
	}
	return r.afterCreate(ctx)
}

type transferRequestHookUOW struct {
	repository.UnitOfWork
	repos repository.RepositorySet
}

func (u transferRequestHookUOW) Repositories() repository.RepositorySet { return u.repos }

type transferRequestHookManager struct {
	repository.UnitOfWorkManager
	afterCreate func(context.Context) error
	commitErr   error
}

func (m transferRequestHookManager) Begin(ctx context.Context) (repository.UnitOfWork, error) {
	u, err := m.UnitOfWorkManager.Begin(ctx)
	if err != nil {
		return nil, err
	}
	repos := u.Repositories()
	repos.ClassTransferRequests = transferRequestHookRepository{ClassTransferRequestRepository: repos.ClassTransferRequests, afterCreate: m.afterCreate}
	return transferRequestHookUOW{UnitOfWork: &transferRequestCommitHook{UnitOfWork: u, err: m.commitErr}, repos: repos}, nil
}

type transferRequestCommitHook struct {
	repository.UnitOfWork
	err error
}

func (u *transferRequestCommitHook) Commit() error {
	if u.err != nil {
		return u.err
	}
	return u.UnitOfWork.Commit()
}

func TestClassTransferRequestIntegrationRollback(t *testing.T) {
	for _, atCommit := range []bool{false, true} {
		t.Run(map[bool]string{false: "sesudah insert", true: "commit gagal"}[atCommit], func(t *testing.T) {
			f := newTransferRequestFixture(t)
			failure := errors.New("kegagalan penyimpanan uji")
			m := transferRequestHookManager{UnitOfWorkManager: f.manager, afterCreate: func(context.Context) error {
				if !atCommit {
					return failure
				}
				return nil
			}}
			if atCommit {
				m.commitErr = failure
			}
			svc, err := service.NewClassTransferService(service.Dependencies{Repositories: f.repos, UOW: m})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := svc.Request(context.Background(), f.input); !errors.Is(err, failure) {
				t.Fatalf("error=%v", err)
			}
			f.assertRequestCount(t, 0)
		})
	}
}

func TestClassTransferRequestIntegrationYearLockFirst(t *testing.T) {
	for _, change := range []string{"tahun", "siswa", "scope"} {
		t.Run(change+" berubah saat menunggu", func(t *testing.T) {
			f := newTransferRequestFixture(t)
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			tx, err := f.db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback()
			var pid int
			if err := tx.QueryRowContext(ctx, `SELECT pg_backend_pid()`).Scan(&pid); err != nil {
				t.Fatal(err)
			}
			if _, err := postgres.NewAcademicYearRepository(tx).GetByIDForUpdate(ctx, f.year.ID); err != nil {
				t.Fatal(err)
			}
			result := make(chan error, 1)
			go func() { _, err := f.svc.Request(ctx, f.input); result <- err }()
			waitForBlockedBackend(t, f.db, pid, "%FROM academic_years%FOR SHARE%")
			// NOWAIT membuktikan Request yang menunggu tahun belum memegang row siswa.
			var id uuid.UUID
			if err := tx.QueryRowContext(ctx, `SELECT id FROM students WHERE id=$1 FOR UPDATE NOWAIT`, f.input.StudentID).Scan(&id); err != nil {
				t.Fatalf("Request acquired student before year: %v", err)
			}
			wantError := domain.ErrInvalidTransition
			if change == "siswa" {
				if _, err := tx.ExecContext(ctx, `UPDATE students SET status='LEFT' WHERE id=$1`, f.input.StudentID); err != nil {
					t.Fatal(err)
				}
			} else if change == "scope" {
				if _, err := tx.ExecContext(ctx, `DELETE FROM teacher_class_assignments WHERE user_id=$1`, f.input.Actor.UserID); err != nil {
					t.Fatal(err)
				}
				wantError = service.ErrScopeViolation
			} else {
				year := f.year
				year.Status = domain.AcademicYearClosing
				if err := postgres.NewAcademicYearRepository(tx).Update(ctx, year); err != nil {
					t.Fatal(err)
				}
			}
			if err := tx.Commit(); err != nil {
				t.Fatal(err)
			}
			if err := <-result; !errors.Is(err, wantError) {
				t.Fatalf("Request after commit error=%v", err)
			}
			f.assertRequestCount(t, 0)
		})
	}
}

func TestClassTransferRequestIntegrationBlocksStartClosing(t *testing.T) {
	f := newTransferRequestFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	inserted, release := make(chan struct{}), make(chan struct{})
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	manager := transferRequestHookManager{UnitOfWorkManager: f.manager, afterCreate: func(ctx context.Context) error {
		close(inserted)
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}}
	svc, err := service.NewClassTransferService(service.Dependencies{Repositories: f.repos, UOW: manager})
	if err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() { _, err := svc.Request(ctx, f.input); result <- err }()
	select {
	case <-inserted:
	case <-ctx.Done():
		t.Fatal("Request did not reach INSERT")
	}
	// Hanya Request yang idle in transaction pada fixture ini; statement terakhirnya INSERT.
	var requestPID int
	if err := f.db.QueryRowContext(ctx, `SELECT pid FROM pg_stat_activity WHERE datname=current_database() AND state='idle in transaction' AND query ILIKE '%INSERT INTO class_transfer_requests%'`).Scan(&requestPID); err != nil {
		t.Fatal(err)
	}
	shared, err := f.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	var id uuid.UUID
	if err := shared.QueryRowContext(ctx, `SELECT id FROM academic_years WHERE id=$1 FOR SHARE NOWAIT`, f.year.ID).Scan(&id); err != nil {
		_ = shared.Rollback()
		t.Fatalf("Request must use compatible SHARE lock: %v", err)
	}
	if err := shared.Rollback(); err != nil {
		t.Fatal(err)
	}
	yearService, err := service.NewAcademicYearService(service.Dependencies{Repositories: f.repos, UOW: f.manager})
	if err != nil {
		t.Fatal(err)
	}
	closing := make(chan error, 1)
	go func() {
		_, err := yearService.StartClosing(ctx, service.StartClosingAcademicYearInput{Actor: f.admin, AcademicYearID: f.year.ID})
		closing <- err
	}()
	waitForBlockedBackend(t, f.db, requestPID, "%FROM academic_years%FOR UPDATE%")
	close(release)
	if err := <-result; err != nil {
		t.Fatalf("Request error=%v", err)
	}
	if err := <-closing; !errors.Is(err, service.ErrAcademicYearHasPendingTransfers) {
		t.Fatalf("StartClosing error=%v", err)
	}
	year, err := f.repos.AcademicYears.GetByID(ctx, f.year.ID)
	if err != nil || !year.IsOpen() {
		t.Fatalf("year=%+v err=%v", year, err)
	}
	f.assertRequestCount(t, 1)
}
