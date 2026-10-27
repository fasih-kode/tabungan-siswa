package service

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

type leavingStartUnitYear struct {
	repository.AcademicYearRepository
	lock func(context.Context, uuid.UUID) (domain.AcademicYear, error)
}

func (r leavingStartUnitYear) GetByIDForUpdate(ctx context.Context, id uuid.UUID) (domain.AcademicYear, error) {
	return r.lock(ctx, id)
}

type leavingStartUnitRequests struct {
	repository.StudentLeavingRequestRepository
	create      func(context.Context, domain.StudentLeavingRequest) error
	active, get func(context.Context, uuid.UUID) (domain.StudentLeavingRequest, error)
}

func (r leavingStartUnitRequests) Create(ctx context.Context, v domain.StudentLeavingRequest) error {
	return r.create(ctx, v)
}
func (r leavingStartUnitRequests) GetInProgressByStudent(ctx context.Context, id uuid.UUID) (domain.StudentLeavingRequest, error) {
	return r.active(ctx, id)
}
func (r leavingStartUnitRequests) GetByID(ctx context.Context, id uuid.UUID) (domain.StudentLeavingRequest, error) {
	return r.get(ctx, id)
}

type leavingStartUnitAudit struct {
	repository.AuditRepository
	create func(context.Context, domain.AuditLog) error
}

func (r leavingStartUnitAudit) Create(ctx context.Context, v domain.AuditLog) error {
	return r.create(ctx, v)
}

type leavingStartUnitUOW struct {
	repos              repository.RepositorySet
	calls              *[]string
	commitErr          error
	commits, rollbacks int
}

func (u *leavingStartUnitUOW) Repositories() repository.RepositorySet { return u.repos }
func (u *leavingStartUnitUOW) Commit() error {
	u.commits++
	*u.calls = append(*u.calls, "commit")
	return u.commitErr
}
func (u *leavingStartUnitUOW) Rollback() error { u.rollbacks++; return nil }

type leavingStartUnitFixture struct {
	svc                                                             *studentLeavingRequestService
	in                                                              StartStudentLeavingRequestInput
	year                                                            domain.AcademicYear
	student                                                         domain.Student
	history                                                         domain.StudentClassHistory
	uow                                                             *leavingStartUnitUOW
	manager                                                         *studentServiceFakeUnitOfWorkManager
	calls                                                           []string
	requests                                                        []domain.StudentLeavingRequest
	audits                                                          []domain.AuditLog
	yearErr, studentErr, historyErr, activeErr, insertErr, auditErr error
	active                                                          bool
}

func newLeavingStartUnitFixture(t *testing.T) *leavingStartUnitFixture {
	t.Helper()
	today := time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)
	f := &leavingStartUnitFixture{}
	f.in = StartStudentLeavingRequestInput{Actor: Actor{UserID: uuid.New(), Role: domain.RoleAdmin}, StudentID: uuid.New(), AcademicYearID: uuid.New(), EffectiveLeavingDate: today}
	f.year = newTestAcademicYear(f.in.AcademicYearID, domain.AcademicYearOpen)
	f.student = domain.Student{ID: f.in.StudentID, Status: domain.StudentActive}
	f.history = newTestStudentClassHistory(t, f.student.ID, f.year.ID, uuid.New(), f.year.StartDate, &f.year.EndDate)
	requests := leavingStartUnitRequests{
		active: func(context.Context, uuid.UUID) (domain.StudentLeavingRequest, error) {
			f.calls = append(f.calls, "active")
			if f.activeErr != nil {
				return domain.StudentLeavingRequest{}, f.activeErr
			}
			if f.active {
				return domain.StudentLeavingRequest{ID: uuid.New()}, nil
			}
			return domain.StudentLeavingRequest{}, repository.ErrNotFound
		},
		create: func(_ context.Context, v domain.StudentLeavingRequest) error {
			f.calls = append(f.calls, "insert")
			if f.insertErr != nil {
				return f.insertErr
			}
			f.requests = append(f.requests, v)
			return nil
		},
		get: func(_ context.Context, id uuid.UUID) (domain.StudentLeavingRequest, error) {
			for _, v := range f.requests {
				if v.ID == id {
					return v, nil
				}
			}
			return domain.StudentLeavingRequest{}, repository.ErrNotFound
		},
	}
	repos := repository.RepositorySet{
		AcademicYears: leavingStartUnitYear{lock: func(context.Context, uuid.UUID) (domain.AcademicYear, error) {
			f.calls = append(f.calls, "year")
			return f.year, f.yearErr
		}},
		Students: &studentServiceFakeStudentRepository{getByIDFn: func(context.Context, uuid.UUID) (domain.Student, error) {
			f.calls = append(f.calls, "student")
			return f.student, f.studentErr
		}},
		StudentClassHistories: &classTransferServiceFakeStudentClassHistoryRepository{getCurrentFn: func(context.Context, uuid.UUID, uuid.UUID, time.Time) (domain.StudentClassHistory, error) {
			f.calls = append(f.calls, "placement")
			return f.history, f.historyErr
		}},
		StudentLeavingRequests: requests,
		AuditLogs: leavingStartUnitAudit{create: func(_ context.Context, v domain.AuditLog) error {
			f.calls = append(f.calls, "audit")
			if f.auditErr != nil {
				return f.auditErr
			}
			f.audits = append(f.audits, v)
			return nil
		}},
	}
	f.uow = &leavingStartUnitUOW{repos: repos, calls: &f.calls}
	f.manager = &studentServiceFakeUnitOfWorkManager{uow: f.uow}
	svc, err := NewStudentLeavingRequestService(Dependencies{Repositories: repos, UOW: f.manager})
	if err != nil {
		t.Fatal(err)
	}
	f.svc = svc.(*studentLeavingRequestService)
	f.svc.now = func() time.Time { return today.Add(12 * time.Hour) }
	return f
}

func TestStudentLeavingRequestServiceStartOrderAndAudit(t *testing.T) {
	f := newLeavingStartUnitFixture(t)
	out, err := f.svc.Start(context.Background(), f.in)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"year", "student", "placement", "active", "insert", "audit", "commit"}
	if !reflect.DeepEqual(f.calls, want) {
		t.Fatalf("calls=%v, want %v", f.calls, want)
	}
	if out.Request == nil || out.Request.Status != domain.StudentLeavingInProgress || len(f.requests) != 1 || len(f.audits) != 1 {
		t.Fatalf("output=%+v requests=%d audits=%d", out, len(f.requests), len(f.audits))
	}
	a := f.audits[0]
	if a.EntityType != domain.AuditEntityStudent || a.EntityID != f.student.ID || a.Action != "STUDENT_LEAVING_STARTED" || a.ActorUserID == nil || *a.ActorUserID != f.in.Actor.UserID {
		t.Fatalf("audit=%+v", a)
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(a.AfterData, &payload); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"student_id", "leaving_request_id", "academic_year_id", "effective_leaving_date", "status", "actor_id", "actor_role", "occurred_at"} {
		if len(payload[key]) == 0 {
			t.Fatalf("missing audit field %s", key)
		}
	}
	if f.uow.commits != 1 || f.uow.rollbacks != 0 {
		t.Fatalf("commit/rollback=%d/%d", f.uow.commits, f.uow.rollbacks)
	}
}

func TestStudentLeavingRequestServiceStartFailures(t *testing.T) {
	failure := errors.New("repository failure")
	cases := []struct {
		name      string
		change    func(*leavingStartUnitFixture)
		want      error
		beforeUOW bool
	}{
		{"wali", func(f *leavingStartUnitFixture) { f.in.Actor.Role = domain.RoleWaliKelas }, ErrForbidden, true},
		{"siswa", func(f *leavingStartUnitFixture) { f.in.Actor.Role = domain.RoleSiswa }, ErrForbidden, true},
		{"actor invalid", func(f *leavingStartUnitFixture) { f.in.Actor.UserID = uuid.Nil }, ErrInvalidActor, true},
		{"student id", func(f *leavingStartUnitFixture) { f.in.StudentID = uuid.Nil }, domain.ErrInvalidID, true},
		{"year id", func(f *leavingStartUnitFixture) { f.in.AcademicYearID = uuid.Nil }, domain.ErrInvalidID, true},
		{"date zero", func(f *leavingStartUnitFixture) { f.in.EffectiveLeavingDate = time.Time{} }, domain.ErrInvalidValue, true},
		{"begin", func(f *leavingStartUnitFixture) { f.manager.beginErr = failure }, failure, false},
		{"year missing", func(f *leavingStartUnitFixture) { f.yearErr = repository.ErrNotFound }, repository.ErrNotFound, false},
		{"student missing", func(f *leavingStartUnitFixture) { f.studentErr = repository.ErrNotFound }, repository.ErrNotFound, false},
		{"closed", func(f *leavingStartUnitFixture) { f.year.Status = domain.AcademicYearClosed }, domain.ErrInvalidTransition, false},
		{"closing", func(f *leavingStartUnitFixture) { f.year.Status = domain.AcademicYearClosing }, domain.ErrInvalidTransition, false},
		{"left", func(f *leavingStartUnitFixture) { f.student.Status = domain.StudentLeft }, domain.ErrInvalidTransition, false},
		{"graduated", func(f *leavingStartUnitFixture) { f.student.Status = domain.StudentGraduated }, domain.ErrInvalidTransition, false},
		{"backdate provisional restriction", func(f *leavingStartUnitFixture) {
			f.in.EffectiveLeavingDate = f.in.EffectiveLeavingDate.AddDate(0, 0, -1)
		}, domain.ErrInvalidDateRange, false},
		{"future provisional restriction", func(f *leavingStartUnitFixture) {
			f.in.EffectiveLeavingDate = f.in.EffectiveLeavingDate.AddDate(0, 0, 1)
		}, domain.ErrInvalidDateRange, false},
		{"placement missing", func(f *leavingStartUnitFixture) { f.historyErr = repository.ErrNotFound }, repository.ErrNotFound, false},
		{"placement wrong student", func(f *leavingStartUnitFixture) { f.history.StudentID = uuid.New() }, domain.ErrInvalidTransition, false},
		{"placement wrong year", func(f *leavingStartUnitFixture) { f.history.AcademicYearID = uuid.New() }, domain.ErrInvalidTransition, false},
		{"placement invalid range", func(f *leavingStartUnitFixture) { f.history.Period.To = nil }, domain.ErrInvalidDateRange, false},
		{"active duplicate", func(f *leavingStartUnitFixture) { f.active = true }, repository.ErrConflict, false},
		{"active read failure", func(f *leavingStartUnitFixture) { f.activeErr = failure }, failure, false},
		{"insert conflict", func(f *leavingStartUnitFixture) { f.insertErr = repository.ErrConflict }, repository.ErrConflict, false},
		{"audit failure", func(f *leavingStartUnitFixture) { f.auditErr = failure }, failure, false},
		{"commit failure", func(f *leavingStartUnitFixture) { f.uow.commitErr = failure }, failure, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newLeavingStartUnitFixture(t)
			tc.change(f)
			if _, err := f.svc.Start(context.Background(), f.in); !errors.Is(err, tc.want) {
				t.Fatalf("error=%v, want %v", err, tc.want)
			}
			if tc.beforeUOW && f.manager.beginCount != 0 {
				t.Fatal("invalid input opened UOW")
			}
			if !tc.beforeUOW && f.manager.beginErr == nil && f.uow.rollbacks != 1 {
				t.Fatalf("rollback count=%d", f.uow.rollbacks)
			}
			if f.uow.commits != 0 && f.uow.commitErr == nil {
				t.Fatal("invalid operation committed")
			}
		})
	}
}

func TestStudentLeavingRequestServiceGetAuthorization(t *testing.T) {
	f := newLeavingStartUnitFixture(t)
	out, err := f.svc.Start(context.Background(), f.in)
	if err != nil {
		t.Fatal(err)
	}
	in := GetStudentLeavingRequestInput{Actor: f.in.Actor, RequestID: out.Request.ID}
	got, err := f.svc.Get(context.Background(), in)
	if err != nil || got.Request == nil || got.Request.ID != in.RequestID {
		t.Fatalf("Get=%+v error=%v", got, err)
	}
	for _, role := range []domain.UserRole{domain.RoleWaliKelas, domain.RoleSiswa} {
		invalid := in
		invalid.Actor.Role = role
		if _, err := f.svc.Get(context.Background(), invalid); !errors.Is(err, ErrForbidden) {
			t.Fatalf("role %s error=%v", role, err)
		}
	}
	in.RequestID = uuid.Nil
	if _, err := f.svc.Get(context.Background(), in); !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("invalid ID error=%v", err)
	}
	in.RequestID = uuid.New()
	if _, err := f.svc.Get(context.Background(), in); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("missing error=%v", err)
	}
	if len(f.audits) != 1 {
		t.Fatal("Get created audit")
	}
	if _, err := NewStudentLeavingRequestService(Dependencies{}); !errors.Is(err, ErrInvalidDependency) {
		t.Fatalf("constructor error=%v", err)
	}
}
