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

// TestStudentLeavingFinalizeLegacyGate mempertahankan regresi bypass yang ditemukan sebelum Finalize.
func TestStudentLeavingFinalizeLegacyGate(t *testing.T) {
	for _, name := range []string{"tanpa proses dan rekening", "proses tanpa rekening", "rekening OPEN saldo positif", "settlement dan handover lengkap", "histori LEFT", "Wali", "Siswa"} {
		t.Run(name, func(t *testing.T) {
			f := newLeavingCancelFixture(t)
			actor := f.in.Actor
			switch name {
			case "rekening OPEN saldo positif", "settlement dan handover lengkap":
				account := createCancelAccount(t, f, f.year.ID)
				createReadinessTransaction(t, f, account, domain.TransactionDeposit, 5000, false)
				request := startLeavingRequestForCancel(t, f)
				if name == "settlement dan handover lengkap" {
					settlement := createLinkedCancelSettlement(t, f, request, account, 5000)
					insertCancelMovement(t, f, settlement, domain.SettlementCashHandover, 5000)
				}
			case "proses tanpa rekening":
				startLeavingRequestForCancel(t, f)
			case "histori LEFT":
				seedHistoricalLeftStudent(t, context.Background(), f.db, f.in.StudentID, f.year.ID, f.in.EffectiveLeavingDate)
			case "Wali":
				actor.Role = domain.RoleWaliKelas
			case "Siswa":
				actor.Role = domain.RoleSiswa
			}
			students, err := service.NewStudentService(service.Dependencies{Repositories: f.repos, UOW: f.manager})
			if err != nil {
				t.Fatal(err)
			}
			before := readinessSnapshot(t, f.db)
			out, err := students.Leave(context.Background(), service.LeaveStudentInput{Actor: actor, StudentID: f.in.StudentID, AcademicYearID: f.year.ID, LeavingDate: f.in.EffectiveLeavingDate})
			want := domain.ErrInvalidTransition
			if actor.Role != domain.RoleAdmin {
				want = service.ErrForbidden
			}
			if !errors.Is(err, want) || out.Student != nil {
				t.Fatalf("Leave()=%+v,%v want %v", out, err, want)
			}
			if !reflect.DeepEqual(before, readinessSnapshot(t, f.db)) {
				t.Fatal("Leave mengubah data")
			}
		})
	}
}

func TestStudentLeavingLegacyConcurrentRejection(t *testing.T) {
	for _, withStart := range []bool{false, true} {
		t.Run(map[bool]string{false: "dua Leave", true: "Leave dan Start"}[withStart], func(t *testing.T) {
			f := newLeavingCancelFixture(t)
			svc, err := service.NewStudentService(service.Dependencies{Repositories: f.repos, UOW: f.manager})
			if err != nil {
				t.Fatal(err)
			}
			before := readinessSnapshot(t, f.db)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			barrier := make(chan struct{})
			results := make(chan error, 2)
			for i := 0; i < 2; i++ {
				go func(i int) {
					<-barrier
					if withStart && i == 1 {
						_, err := f.svc.Start(ctx, f.in)
						results <- err
						return
					}
					_, err := svc.Leave(ctx, service.LeaveStudentInput{Actor: f.in.Actor, StudentID: f.in.StudentID, AcademicYearID: f.year.ID, LeavingDate: f.in.EffectiveLeavingDate})
					results <- err
				}(i)
			}
			close(barrier)
			denied, started := 0, 0
			for i := 0; i < 2; i++ {
				select {
				case err := <-results:
					if errors.Is(err, domain.ErrInvalidTransition) {
						denied++
					} else if err == nil {
						started++
					} else {
						t.Fatal(err)
					}
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				}
			}
			if withStart {
				if denied != 1 || started != 1 {
					t.Fatalf("denied/start=%d/%d", denied, started)
				}
			} else if denied != 2 || !reflect.DeepEqual(before, readinessSnapshot(t, f.db)) {
				t.Fatal("dua Leave harus ditolak tanpa mutasi")
			}
			student, err := f.repos.Students.GetByID(ctx, f.in.StudentID)
			if err != nil || !student.IsActive() {
				t.Fatalf("student=%+v err=%v", student, err)
			}
			if withStart {
				request, err := f.repos.StudentLeavingRequests.GetInProgressByStudent(ctx, f.in.StudentID)
				if err != nil || request.ID == uuid.Nil {
					t.Fatalf("Start gagal: %v", err)
				}
				after := readinessSnapshot(t, f.db)
				for table, value := range before {
					if table != "student_leaving_requests" && table != "audit_logs" && after[table] != value {
						t.Fatalf("tabel %s berubah", table)
					}
				}
			}
		})
	}
}

// TestStudentLeavingProfileUpdateMustNotRestoreLeft mempertahankan reproduksi lost update dengan lock nyata.
func TestStudentLeavingProfileUpdateMustNotRestoreLeft(t *testing.T) {
	testStudentProfileSerialOrder(t, "reactivate", "profile")
}

func TestStudentProfileLifecycleSerialOrder(t *testing.T) {
	for _, pair := range [][2]string{{"profile", "reactivate"}, {"profile", "profile"}, {"profile", "start"}, {"start", "profile"}} {
		t.Run(pair[0]+" lalu "+pair[1], func(t *testing.T) { testStudentProfileSerialOrder(t, pair[0], pair[1]) })
	}
}

func testStudentProfileSerialOrder(t *testing.T, first, second string) {
	t.Helper()
	f := newLeavingCancelFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if first == "reactivate" || second == "reactivate" {
		seedHistoricalLeftStudent(t, ctx, f.db, f.in.StudentID, f.year.ID, f.in.EffectiveLeavingDate.AddDate(0, 0, -1))
	}
	original, err := f.repos.Students.GetByID(ctx, f.in.StudentID)
	if err != nil {
		t.Fatal(err)
	}
	before := readinessSnapshot(t, f.db)
	reached, release := make(chan struct{}), make(chan struct{})
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	run := func(operation, name string, manager repository.UnitOfWorkManager) error {
		if operation == "start" {
			svc, err := service.NewStudentLeavingRequestService(service.Dependencies{Repositories: f.repos, UOW: manager})
			if err != nil {
				return err
			}
			_, err = svc.Start(ctx, f.in)
			return err
		}
		svc, err := service.NewStudentService(service.Dependencies{Repositories: f.repos, UOW: manager})
		if err != nil {
			return err
		}
		if operation == "reactivate" {
			_, err = svc.Reactivate(ctx, service.ReactivateStudentInput{Actor: f.in.Actor, StudentID: original.ID, AcademicYearID: f.year.ID, ClassID: f.class.ID, EffectiveDate: f.in.EffectiveLeavingDate})
			return err
		}
		_, err = svc.Update(ctx, service.UpdateStudentInput{Actor: f.in.Actor, StudentID: original.ID, Name: name, NIS: original.NIS, NISN: original.NISN})
		return err
	}
	firstResult, secondResult := make(chan error, 1), make(chan error, 1)
	go func() {
		firstResult <- run(first, "Profil pertama", legacyProfileBarrierManager{UnitOfWorkManager: f.manager, read: reached, release: release})
	}()
	select {
	case <-reached:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	pid := cancelIdleTransactionPID(t, f.db)
	go func() { secondResult <- run(second, "Profil kedua", f.manager) }()
	waitForCancelBlocker(t, f.db, pid)
	close(release)
	for _, result := range []chan error{firstResult, secondResult} {
		select {
		case err := <-result:
			if err != nil {
				t.Fatal(err)
			}
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	got, err := f.repos.Students.GetByID(ctx, original.ID)
	if err != nil {
		t.Fatal(err)
	}
	wantName := "Profil pertama"
	if second == "profile" {
		wantName = "Profil kedua"
	}
	if !got.IsActive() || got.Name != wantName || !reflect.DeepEqual(original.LeavingDate, got.LeavingDate) {
		t.Fatalf("hasil siswa tidak sesuai: %+v", got)
	}
	after := readinessSnapshot(t, f.db)
	for _, table := range []string{"savings_accounts", "transactions", "savings_settlements", "settlement_cash_movements", "leaving_transaction_requests"} {
		if before[table] != after[table] {
			t.Fatalf("keuangan %s berubah", table)
		}
	}
	histories, err := f.repos.StudentClassHistories.ListByStudentAndAcademicYear(ctx, original.ID, f.year.ID, repository.ListOptions{Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	wantHistory := 1
	if first == "reactivate" || second == "reactivate" {
		wantHistory = 2
	}
	if len(histories) != wantHistory {
		t.Fatalf("histories=%d want %d", len(histories), wantHistory)
	}
	current, err := f.repos.StudentClassHistories.GetCurrentByStudentAndAcademicYear(ctx, original.ID, f.year.ID, f.in.EffectiveLeavingDate)
	if err != nil || current.ClassID != f.class.ID {
		t.Fatalf("placement=%+v err=%v", current, err)
	}
	if first == "start" || second == "start" {
		if _, err := f.repos.StudentLeavingRequests.GetInProgressByStudent(ctx, original.ID); err != nil {
			t.Fatal(err)
		}
	}
}

func TestStudentProfilePreservesLifecycleAndRollsBack(t *testing.T) {
	for _, state := range []string{"ACTIVE", "LEFT"} {
		for _, failure := range []string{"none", "update", "commit"} {
			t.Run(state+"/"+failure, func(t *testing.T) {
				f := newLeavingCancelFixture(t)
				ctx := context.Background()
				if state == "LEFT" {
					seedHistoricalLeftStudent(t, ctx, f.db, f.in.StudentID, f.year.ID, f.in.EffectiveLeavingDate)
				}
				original, err := f.repos.Students.GetByID(ctx, f.in.StudentID)
				if err != nil {
					t.Fatal(err)
				}
				before := readinessSnapshot(t, f.db)
				var manager repository.UnitOfWorkManager = f.manager
				injected := errors.New("kegagalan profil untuk tes rollback")
				if failure == "update" {
					manager = failingStudentUpdateManager{UnitOfWorkManager: f.manager, failure: injected}
				}
				if failure == "commit" {
					manager = cancelFailureManager{UnitOfWorkManager: f.manager, stage: "commit", failure: injected}
				}
				svc, err := service.NewStudentService(service.Dependencies{Repositories: f.repos, UOW: manager})
				if err != nil {
					t.Fatal(err)
				}
				_, err = svc.Update(ctx, service.UpdateStudentInput{Actor: f.in.Actor, StudentID: original.ID, Name: "Profil diperbarui", NIS: original.NIS, NISN: original.NISN})
				if failure != "none" {
					if !errors.Is(err, injected) || !reflect.DeepEqual(before, readinessSnapshot(t, f.db)) {
						t.Fatalf("rollback tidak utuh: %v", err)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				got, err := f.repos.Students.GetByID(ctx, original.ID)
				if err != nil || got.Status != original.Status || !reflect.DeepEqual(got.LeavingDate, original.LeavingDate) || got.Name != "Profil diperbarui" {
					t.Fatalf("profil/lifecycle berubah: %+v %v", got, err)
				}
				after := readinessSnapshot(t, f.db)
				for table, value := range before {
					if table != "students" && after[table] != value {
						t.Fatalf("%s berubah", table)
					}
				}
			})
		}
	}
}

type legacyProfileBarrierManager struct {
	repository.UnitOfWorkManager
	read    chan struct{}
	release chan struct{}
}

func (m legacyProfileBarrierManager) Begin(ctx context.Context) (repository.UnitOfWork, error) {
	uow, err := m.UnitOfWorkManager.Begin(ctx)
	if err != nil {
		return nil, err
	}
	repos := uow.Repositories()
	repos.Students = legacyProfileBarrierStudent{StudentRepository: repos.Students, read: m.read, release: m.release}
	return legacyProfileBarrierUOW{UnitOfWork: uow, repos: repos}, nil
}

type legacyProfileBarrierUOW struct {
	repository.UnitOfWork
	repos repository.RepositorySet
}

func (u legacyProfileBarrierUOW) Repositories() repository.RepositorySet { return u.repos }

type legacyProfileBarrierStudent struct {
	repository.StudentRepository
	read    chan struct{}
	release chan struct{}
}

func (r legacyProfileBarrierStudent) GetByIDForUpdate(ctx context.Context, id uuid.UUID) (domain.Student, error) {
	student, err := r.StudentRepository.GetByIDForUpdate(ctx, id)
	if err != nil {
		return student, err
	}
	close(r.read)
	select {
	case <-r.release:
		return student, nil
	case <-ctx.Done():
		return domain.Student{}, ctx.Err()
	}
}
