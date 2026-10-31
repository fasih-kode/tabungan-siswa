package service_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

func TestStudentAssignClassLockConcurrency(t *testing.T) {
	for _, op := range []string{"finalize", "finalize-later", "start", "reactivate", "year", "assign"} {
		for _, assignFirst := range []bool{true, false} {
			t.Run(op+map[bool]string{true: " Assign dahulu", false: " lifecycle dahulu"}[assignFirst], func(t *testing.T) {
				f := newLeavingCancelFixture(t)
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
				defer cancel()
				var request domain.StudentLeavingRequest
				target := f.class
				if op == "finalize" || op == "finalize-later" {
					request = startLeavingRequestForCancel(t, f)
				}
				if op == "finalize-later" {
					date := f.in.EffectiveLeavingDate.AddDate(0, 0, 1)
					if _, err := f.db.Exec(`UPDATE student_class_histories SET validity=daterange(lower(validity),$2::date,'[)') WHERE student_id=$1`, f.in.StudentID, date); err != nil {
						t.Fatal(err)
					}
					target = newServiceScopeClass(t, "Penempatan berikutnya", 8)
					target.AcademicYearID = f.year.ID
					target.EffectiveDate = date
					if err := f.repos.Classes.Create(ctx, target); err != nil {
						t.Fatal(err)
					}
				}
				if op == "reactivate" {
					seedHistoricalLeftStudent(t, ctx, f.db, f.in.StudentID, f.year.ID, f.in.EffectiveLeavingDate.AddDate(0, 0, -1))
				}
				if op == "assign" || op == "year" {
					if _, err := f.db.Exec(`DELETE FROM student_class_histories WHERE student_id=$1`, f.in.StudentID); err != nil {
						t.Fatal(err)
					}
				}
				if op == "year" {
					year := f.year
					year.Status = domain.AcademicYearClosing
					if err := f.repos.AcademicYears.Update(ctx, year); err != nil {
						t.Fatal(err)
					}
				}
				before := readinessSnapshot(t, f.db)
				var financialAuditsBefore string
				if err := f.db.QueryRow(`SELECT COALESCE(jsonb_agg(to_jsonb(a) ORDER BY id),'[]'::jsonb)::text FROM audit_logs a WHERE entity_type IN ('SAVINGS_ACCOUNT','TRANSACTION','SAVINGS_SETTLEMENT','SETTLEMENT_REOPEN_REQUEST')`).Scan(&financialAuditsBefore); err != nil {
					t.Fatal(err)
				}
				run := func(operation string, manager repository.UnitOfWorkManager) error {
					deps := service.Dependencies{Repositories: f.repos, UOW: manager}
					switch operation {
					case "assign":
						svc, err := service.NewStudentService(deps)
						if err != nil {
							return err
						}
						_, err = svc.AssignClass(ctx, service.AssignStudentClassInput{Actor: f.in.Actor, StudentID: f.in.StudentID, AcademicYearID: f.year.ID, ClassID: target.ID})
						return err
					case "reactivate":
						svc, err := service.NewStudentService(deps)
						if err != nil {
							return err
						}
						_, err = svc.Reactivate(ctx, service.ReactivateStudentInput{Actor: f.in.Actor, StudentID: f.in.StudentID, AcademicYearID: f.year.ID, ClassID: f.class.ID, EffectiveDate: f.in.EffectiveLeavingDate})
						return err
					case "start":
						svc, err := service.NewStudentLeavingRequestService(deps)
						if err != nil {
							return err
						}
						_, err = svc.Start(ctx, f.in)
						return err
					case "finalize", "finalize-later":
						svc, err := service.NewStudentLeavingRequestService(deps)
						if err != nil {
							return err
						}
						_, err = svc.Finalize(ctx, service.FinalizeStudentLeavingRequestInput{Actor: f.in.Actor, StudentLeavingRequestID: request.ID})
						return err
					case "year":
						svc, err := service.NewAcademicYearService(deps)
						if err != nil {
							return err
						}
						_, err = svc.Close(ctx, service.CloseAcademicYearInput{Actor: f.in.Actor, AcademicYearID: f.year.ID})
						return err
					}
					return errors.New("operasi tak dikenal")
				}
				gate := newCancelConcurrencyGate()
				defer gate.release()
				first, second := "assign", op
				if !assignFirst {
					first, second = second, first
				}
				firstResult, secondResult := make(chan error, 1), make(chan error, 1)
				go func() { firstResult <- run(first, finalizeBarrierManager{UnitOfWorkManager: f.manager, gate: gate}) }()
				waitCancelGate(t, gate)
				pid := cancelIdleTransactionPID(t, f.db)
				go func() { secondResult <- run(second, f.manager) }()
				waitForCancelBlocker(t, f.db, pid)
				gate.release()
				var errs []error
				for _, ch := range []chan error{firstResult, secondResult} {
					select {
					case err := <-ch:
						errs = append(errs, err)
						if err != nil && !errors.Is(err, domain.ErrInvalidTransition) && !errors.Is(err, repository.ErrConflict) {
							t.Fatal(err)
						}
					case <-ctx.Done():
						t.Fatal(ctx.Err())
					}
				}
				assignErr, otherErr := errs[0], errs[1]
				if !assignFirst {
					assignErr, otherErr = otherErr, assignErr
				}
				student, err := f.repos.Students.GetByID(ctx, f.in.StudentID)
				if err != nil {
					t.Fatal(err)
				}
				histories, err := f.repos.StudentClassHistories.ListByStudentAndAcademicYear(ctx, student.ID, f.year.ID, repository.ListOptions{Limit: 100})
				if err != nil {
					t.Fatal(err)
				}
				wantHistories := 1
				switch op {
				case "finalize":
					if otherErr != nil || assignErr == nil || !student.IsLeft() {
						t.Fatalf("finalize/assign=%v/%v student=%s", otherErr, assignErr, student.Status)
					}
				case "finalize-later":
					if assignFirst {
						wantHistories = 2
						if assignErr != nil || otherErr == nil || !student.IsActive() {
							t.Fatalf("placement baru harus memblokir: %v/%v", assignErr, otherErr)
						}
					} else {
						if otherErr != nil || assignErr == nil || !student.IsLeft() {
							t.Fatalf("Finalize dahulu: %v/%v", otherErr, assignErr)
						}
					}
				case "start":
					if otherErr != nil || assignErr == nil || !student.IsActive() {
						t.Fatalf("Start/assign=%v/%v", otherErr, assignErr)
					}
					if _, err := f.repos.StudentLeavingRequests.GetInProgressByStudent(ctx, student.ID); err != nil {
						t.Fatal(err)
					}
				case "reactivate":
					wantHistories = 2
					if otherErr != nil || assignErr == nil || !student.IsActive() {
						t.Fatalf("Reactivate/assign=%v/%v", otherErr, assignErr)
					}
				case "year":
					if otherErr != nil || assignErr != nil {
						t.Fatalf("Close/assign=%v/%v", otherErr, assignErr)
					}
					year, err := f.repos.AcademicYears.GetByID(ctx, f.year.ID)
					if err != nil || !year.IsClosed() {
						t.Fatalf("year=%+v %v", year, err)
					}
				case "assign":
					success := 0
					for _, err := range errs {
						if err == nil {
							success++
						}
					}
					if success != 1 {
						t.Fatalf("dua Assign: %v", errs)
					}
				}
				if len(histories) != wantHistories {
					t.Fatalf("histories=%d want %d", len(histories), wantHistories)
				}
				for i, h := range histories {
					if h.StudentID != student.ID || h.AcademicYearID != f.year.ID || h.ValidateWithinAcademicYear(f.year.StartDate, f.year.EndDate) != nil {
						t.Fatalf("histori tidak sah: %+v", h)
					}
					for j, k := range histories {
						if i != j && h.Period.To != nil && k.Period.To != nil && h.Period.From.Before(*k.Period.To) && k.Period.From.Before(*h.Period.To) {
							t.Fatal("histori overlap")
						}
					}
				}
				if student.IsLeft() {
					got, err := f.repos.StudentLeavingRequests.GetByID(ctx, request.ID)
					if err != nil || got.Status != domain.StudentLeavingFinalized || student.LeavingDate == nil || !student.LeavingDate.Equal(got.EffectiveLeavingDate) {
						t.Fatalf("final lifecycle=%+v %v", got, err)
					}
					if _, err := f.repos.StudentClassHistories.GetCurrentByStudentAndAcademicYear(ctx, student.ID, f.year.ID, request.EffectiveLeavingDate); !errors.Is(err, repository.ErrNotFound) {
						t.Fatalf("placement masih aktif %v", err)
					}
				}
				var financialAuditsAfter string
				if err := f.db.QueryRow(`SELECT COALESCE(jsonb_agg(to_jsonb(a) ORDER BY id),'[]'::jsonb)::text FROM audit_logs a WHERE entity_type IN ('SAVINGS_ACCOUNT','TRANSACTION','SAVINGS_SETTLEMENT','SETTLEMENT_REOPEN_REQUEST')`).Scan(&financialAuditsAfter); err != nil {
					t.Fatal(err)
				}
				if financialAuditsBefore != financialAuditsAfter {
					t.Fatal("audit keuangan berubah")
				}
				after := readinessSnapshot(t, f.db)
				for _, table := range []string{"savings_accounts", "transactions", "savings_settlements", "settlement_cash_movements", "leaving_transaction_requests"} {
					if before[table] != after[table] {
						t.Fatalf("keuangan %s berubah", table)
					}
				}
			})
		}
	}
}

func TestStudentAssignClassValidationRollback(t *testing.T) {
	for _, mode := range []string{"normal", "nonaktif", "kelas beda tahun", "tanggal tidak sah", "overlap", "year error", "student error", "class error", "history error", "commit error"} {
		t.Run(mode, func(t *testing.T) {
			f := newLeavingCancelFixture(t)
			ctx := context.Background()
			if mode != "overlap" {
				if _, err := f.db.Exec(`DELETE FROM student_class_histories WHERE student_id=$1`, f.in.StudentID); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "nonaktif" {
				if _, err := f.db.Exec(`UPDATE students SET status='LEFT' WHERE id=$1`, f.in.StudentID); err != nil {
					t.Fatal(err)
				}
			}
			events := []string{}
			failure := errors.New("kegagalan AssignClass untuk rollback")
			manager := assignClassProbeManager{UnitOfWorkManager: f.manager, mode: mode, failure: failure, events: &events}
			svc, err := service.NewStudentService(service.Dependencies{Repositories: f.repos, UOW: manager})
			if err != nil {
				t.Fatal(err)
			}
			before := readinessSnapshot(t, f.db)
			out, err := svc.AssignClass(ctx, service.AssignStudentClassInput{Actor: f.in.Actor, StudentID: f.in.StudentID, AcademicYearID: f.year.ID, ClassID: f.class.ID})
			if mode == "normal" {
				if err != nil || out.History == nil || out.History.ClassID != f.class.ID || !out.History.Period.From.Equal(f.class.EffectiveDate) || out.History.Period.To == nil || !out.History.Period.To.Equal(f.year.EndDate) {
					t.Fatalf("Assign=%+v %v", out, err)
				}
				if !reflect.DeepEqual(events, []string{"year", "student", "class", "history"}) {
					t.Fatalf("urutan/relock: %v", events)
				}
				after := readinessSnapshot(t, f.db)
				for table, value := range before {
					if table != "student_class_histories" && after[table] != value {
						t.Fatalf("%s berubah", table)
					}
				}
			} else {
				want := failure
				switch mode {
				case "nonaktif":
					want = domain.ErrInvalidTransition
				case "kelas beda tahun":
					want = domain.ErrInvalidValue
				case "tanggal tidak sah":
					want = domain.ErrInvalidDateRange
				case "overlap":
					want = repository.ErrConflict
				}
				if !errors.Is(err, want) || !reflect.DeepEqual(before, readinessSnapshot(t, f.db)) {
					t.Fatalf("mode %s error=%v snapshot berubah", mode, err)
				}
			}
		})
	}
}

type assignClassProbeManager struct {
	repository.UnitOfWorkManager
	mode    string
	failure error
	events  *[]string
}

func (m assignClassProbeManager) Begin(ctx context.Context) (repository.UnitOfWork, error) {
	uow, err := m.UnitOfWorkManager.Begin(ctx)
	if err != nil {
		return nil, err
	}
	repos := uow.Repositories()
	repos.AcademicYears = assignClassProbeYear{AcademicYearRepository: repos.AcademicYears, m: m}
	repos.Students = assignClassProbeStudent{StudentRepository: repos.Students, m: m}
	repos.Classes = assignClassProbeClass{ClassRepository: repos.Classes, m: m}
	repos.StudentClassHistories = assignClassProbeHistory{StudentClassHistoryRepository: repos.StudentClassHistories, m: m}
	var commitErr error
	if m.mode == "commit error" {
		commitErr = m.failure
	}
	return cancelFailureUOW{UnitOfWork: uow, repos: repos, commitErr: commitErr}, nil
}

type assignClassProbeYear struct {
	repository.AcademicYearRepository
	m assignClassProbeManager
}

func (r assignClassProbeYear) GetByIDForUpdate(ctx context.Context, id uuid.UUID) (domain.AcademicYear, error) {
	*r.m.events = append(*r.m.events, "year")
	if r.m.mode == "year error" {
		return domain.AcademicYear{}, r.m.failure
	}
	return r.AcademicYearRepository.GetByIDForUpdate(ctx, id)
}

type assignClassProbeStudent struct {
	repository.StudentRepository
	m assignClassProbeManager
}

func (r assignClassProbeStudent) GetByIDForUpdate(ctx context.Context, id uuid.UUID) (domain.Student, error) {
	*r.m.events = append(*r.m.events, "student")
	if r.m.mode == "student error" {
		return domain.Student{}, r.m.failure
	}
	return r.StudentRepository.GetByIDForUpdate(ctx, id)
}

type assignClassProbeClass struct {
	repository.ClassRepository
	m assignClassProbeManager
}

func (r assignClassProbeClass) GetByID(ctx context.Context, id uuid.UUID) (domain.Class, error) {
	*r.m.events = append(*r.m.events, "class")
	if r.m.mode == "class error" {
		return domain.Class{}, r.m.failure
	}
	c, err := r.ClassRepository.GetByID(ctx, id)
	if r.m.mode == "kelas beda tahun" {
		c.AcademicYearID = uuid.New()
	}
	if r.m.mode == "tanggal tidak sah" {
		c.EffectiveDate = c.EffectiveDate.AddDate(50, 0, 0)
	}
	return c, err
}

type assignClassProbeHistory struct {
	repository.StudentClassHistoryRepository
	m assignClassProbeManager
}

func (r assignClassProbeHistory) Create(ctx context.Context, h domain.StudentClassHistory) error {
	*r.m.events = append(*r.m.events, "history")
	if err := r.StudentClassHistoryRepository.Create(ctx, h); err != nil {
		return err
	}
	if r.m.mode == "history error" {
		return r.m.failure
	}
	return nil
}
