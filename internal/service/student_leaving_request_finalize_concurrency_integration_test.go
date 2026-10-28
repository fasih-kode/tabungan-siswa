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

func TestStudentLeavingFinalizeConcurrency(t *testing.T) {
	for _, operation := range []string{"deposit", "withdrawal", "handover", "cancel", "finalize", "create", "start", "profile", "year", "yearclose", "transfer"} {
		for _, firstFinalize := range []bool{false, true} {
			t.Run(operation+map[bool]string{false: " lebih dahulu", true: " setelah Finalize"}[firstFinalize], func(t *testing.T) {
				f := newLeavingCancelFixture(t)
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
				defer cancel()
				var account domain.SavingsAccount
				var settlement domain.SavingsSettlement
				wali, waliActor := addCancelWali(t, f)
				if operation == "deposit" || operation == "withdrawal" || operation == "handover" {
					account = createCancelAccount(t, f, f.year.ID)
					createReadinessTransaction(t, f, account, domain.TransactionDeposit, 5000, false)
				}
				request := startLeavingRequestForCancel(t, f)
				if operation == "yearclose" {
					year := f.year
					year.Status = domain.AcademicYearClosing
					if err := f.repos.AcademicYears.Update(ctx, year); err != nil {
						t.Fatal(err)
					}
				}
				if operation == "handover" {
					settlement = createLinkedCancelSettlement(t, f, request, account, 5000)
				}
				target := newServiceScopeClass(t, "Tujuan uji finalisasi", 8)
				target.AcademicYearID = f.year.ID
				target.EffectiveDate = f.year.StartDate
				if operation == "transfer" {
					if err := f.repos.Classes.Create(ctx, target); err != nil {
						t.Fatal(err)
					}
				}
				before := readinessSnapshot(t, f.db)
				gate := newCancelConcurrencyGate()
				defer gate.release()
				run := func(op string, manager repository.UnitOfWorkManager) error {
					deps := service.Dependencies{Repositories: f.repos, UOW: manager}
					switch op {
					case "finalize":
						svc, err := service.NewStudentLeavingRequestService(deps)
						if err != nil {
							return err
						}
						_, err = svc.Finalize(ctx, service.FinalizeStudentLeavingRequestInput{Actor: f.in.Actor, StudentLeavingRequestID: request.ID})
						return err
					case "cancel":
						svc, err := service.NewStudentLeavingRequestService(deps)
						if err != nil {
							return err
						}
						_, err = svc.Cancel(ctx, cancelInput(f, request.ID, "uji kompetisi finalisasi"))
						return err
					case "start":
						svc, err := service.NewStudentLeavingRequestService(deps)
						if err != nil {
							return err
						}
						_, err = svc.Start(ctx, f.in)
						return err
					case "profile":
						svc, err := service.NewStudentService(deps)
						if err != nil {
							return err
						}
						student, err := f.repos.Students.GetByID(ctx, f.in.StudentID)
						if err != nil {
							return err
						}
						_, err = svc.Update(ctx, service.UpdateStudentInput{Actor: f.in.Actor, StudentID: student.ID, Name: "Profil finalisasi", NIS: student.NIS, NISN: student.NISN})
						return err
					case "deposit", "withdrawal":
						svc, err := service.NewTransactionService(deps)
						if err != nil {
							return err
						}
						if op == "deposit" {
							_, err = svc.Deposit(ctx, service.DepositInput{Actor: waliActor, SavingsAccountID: account.ID, Amount: 1000, TransactionDate: f.in.EffectiveLeavingDate})
						} else {
							_, err = svc.Withdrawal(ctx, service.WithdrawalInput{Actor: waliActor, SavingsAccountID: account.ID, Amount: 1000, TransactionDate: f.in.EffectiveLeavingDate})
						}
						return err
					case "handover":
						svc, err := service.NewSettlementService(deps)
						if err != nil {
							return err
						}
						_, err = svc.ConfirmHandover(ctx, service.ConfirmSettlementHandoverInput{Actor: waliActor, SettlementID: settlement.ID, Confirm: true})
						return err
					case "year", "yearclose":
						svc, err := service.NewAcademicYearService(deps)
						if err != nil {
							return err
						}
						if op == "yearclose" {
							_, err = svc.Close(ctx, service.CloseAcademicYearInput{Actor: f.in.Actor, AcademicYearID: f.year.ID})
						} else {
							_, err = svc.StartClosing(ctx, service.StartClosingAcademicYearInput{Actor: f.in.Actor, AcademicYearID: f.year.ID})
						}
						return err
					case "transfer":
						svc, err := service.NewClassTransferService(deps)
						if err != nil {
							return err
						}
						_, err = svc.Request(ctx, service.RequestTransferInput{Actor: waliActor, StudentID: f.in.StudentID, AcademicYearID: f.year.ID, FromClassID: f.class.ID, ToClassID: target.ID})
						return err
					case "create":
						uow, err := manager.Begin(ctx)
						if err != nil {
							return err
						}
						defer uow.Rollback()
						// Lock yang sama dengan gerbang tahun trigger INSERT; tidak ada row rekening untuk dijadikan barrier.
						if _, err := uow.Repositories().AcademicYears.GetByIDForShare(ctx, f.year.ID); err != nil {
							return err
						}
						value, err := domain.NewSavingsAccount(f.in.StudentID, f.year.ID)
						if err != nil {
							return err
						}
						if err := uow.Repositories().SavingsAccounts.Create(ctx, value); err != nil {
							return err
						}
						return uow.Commit()
					}
					return errors.New("operasi tes tidak dikenal")
				}
				first, second := operation, "finalize"
				if firstFinalize {
					first, second = second, first
				}
				manager := finalizeBarrierManager{UnitOfWorkManager: f.manager, gate: gate, student: operation == "profile"}
				firstResult, secondResult := make(chan error, 1), make(chan error, 1)
				go func() { firstResult <- run(first, manager) }()
				waitCancelGate(t, gate)
				pid := cancelIdleTransactionPID(t, f.db)
				go func() { secondResult <- run(second, f.manager) }()
				waitForCancelBlocker(t, f.db, pid)
				gate.release()
				results := map[string]error{}
				for _, pair := range []struct {
					name string
					ch   chan error
				}{{first, firstResult}, {second, secondResult}} {
					select {
					case err := <-pair.ch:
						results[pair.name] = err
					case <-ctx.Done():
						t.Fatal(ctx.Err())
					}
				}
				for name, err := range results {
					if err != nil && !errors.Is(err, domain.ErrInvalidTransition) && !errors.Is(err, repository.ErrConflict) {
						t.Fatalf("%s technical error: %v", name, err)
					}
				}
				wantFinalize := true
				if operation == "deposit" || operation == "withdrawal" || operation == "cancel" || operation == "transfer" || operation == "handover" {
					wantFinalize = firstFinalize && (operation == "cancel" || operation == "transfer") || !firstFinalize && operation == "handover"
				}
				// Dua Finalize memakai nama sama; periksa database dan total satu audit sebagai hasil otoritatif.
				if operation != "finalize" && (results["finalize"] == nil) != wantFinalize {
					t.Fatalf("Finalize success=%t want %t; results=%v", results["finalize"] == nil, wantFinalize, results)
				}
				student, err := f.repos.Students.GetByID(ctx, f.in.StudentID)
				if err != nil {
					t.Fatal(err)
				}
				got, err := f.repos.StudentLeavingRequests.GetByID(ctx, request.ID)
				if err != nil {
					t.Fatal(err)
				}
				if student.IsLeft() != (got.Status == domain.StudentLeavingFinalized) || student.IsLeft() != wantFinalize {
					t.Fatalf("student/request=%s/%s want finalized=%t", student.Status, got.Status, wantFinalize)
				}
				var count int
				if err := f.db.QueryRow(`SELECT count(*) FROM audit_logs WHERE action='STUDENT_LEAVING_FINALIZED' AND entity_id=$1`, student.ID).Scan(&count); err != nil {
					t.Fatal(err)
				}
				expected := 0
				if wantFinalize {
					expected = 1
				}
				if count != expected {
					t.Fatalf("audit=%d want %d", count, expected)
				}
				if wantFinalize {
					if student.LeavingDate == nil || !student.LeavingDate.Equal(request.EffectiveLeavingDate) {
						t.Fatal("tanggal keluar salah")
					}
					if _, err := f.repos.StudentClassHistories.GetCurrentByStudentAndAcademicYear(ctx, student.ID, f.year.ID, request.EffectiveLeavingDate); !errors.Is(err, repository.ErrNotFound) {
						t.Fatalf("placement belum tertutup: %v", err)
					}
					accounts, err := f.repos.SavingsAccounts.ListByStudent(ctx, student.ID)
					if err != nil {
						t.Fatal(err)
					}
					for _, a := range accounts {
						if !a.IsSettled() {
							t.Fatal("LEFT dengan rekening OPEN")
						}
					}
				}
				if operation == "profile" {
					if results["profile"] != nil || student.Name != "Profil finalisasi" {
						t.Fatalf("profil hilang: %v", results)
					}
				}
				if operation == "deposit" || operation == "withdrawal" {
					if results[operation] != nil {
						t.Fatalf("transaksi gagal: %v", results)
					}
					transactions, err := f.repos.Transactions.ListAllBySavingsAccount(ctx, account.ID)
					if err != nil {
						t.Fatal(err)
					}
					balance, err := domain.CalculateBalance(transactions)
					want := domain.Money(6000)
					if operation == "withdrawal" {
						want = 4000
					}
					if err != nil || balance.Amount != want {
						t.Fatalf("saldo=%+v %v", balance, err)
					}
				}
				if operation == "handover" {
					if results[operation] != nil {
						t.Fatal(results[operation])
					}
					movements, err := f.repos.Settlements.ListCashMovements(ctx, settlement.ID)
					if err != nil || len(movements) != 1 || movements[0].Amount != 5000 {
						t.Fatalf("handover=%+v %v", movements, err)
					}
				}
				if operation == "create" && results[operation] == nil {
					t.Fatal("rekening tidak boleh dibuat selama pindah/LEFT")
				}
				if operation == "start" && results[operation] == nil {
					t.Fatal("Start kedua tidak boleh berhasil")
				}
				after := readinessSnapshot(t, f.db)
				for _, table := range []string{"savings_accounts", "savings_settlements", "settlement_cash_movements", "transactions"} {
					if operation == "deposit" || operation == "withdrawal" {
						if table == "transactions" {
							continue
						}
					}
					if operation == "handover" && table == "settlement_cash_movements" {
						continue
					}
					if !reflect.DeepEqual(before[table], after[table]) {
						t.Fatalf("%s berubah tak terduga", table)
					}
				}
				_ = wali
			})
		}
	}
}

type finalizeBarrierManager struct {
	repository.UnitOfWorkManager
	gate    *cancelConcurrencyGate
	student bool
}

func (m finalizeBarrierManager) Begin(ctx context.Context) (repository.UnitOfWork, error) {
	uow, err := m.UnitOfWorkManager.Begin(ctx)
	if err != nil {
		return nil, err
	}
	repos := uow.Repositories()
	if m.student {
		repos.Students = finalizeBarrierStudent{StudentRepository: repos.Students, gate: m.gate}
	} else {
		repos.AcademicYears = &finalizeBarrierYear{AcademicYearRepository: repos.AcademicYears, gate: m.gate}
	}
	return cancelPauseUOW{UnitOfWork: uow, repos: repos}, nil
}

type finalizeBarrierStudent struct {
	repository.StudentRepository
	gate *cancelConcurrencyGate
}

func (r finalizeBarrierStudent) GetByIDForUpdate(ctx context.Context, id uuid.UUID) (domain.Student, error) {
	s, err := r.StudentRepository.GetByIDForUpdate(ctx, id)
	if err == nil {
		r.gate.pause()
	}
	return s, err
}

type finalizeBarrierYear struct {
	repository.AcademicYearRepository
	gate   *cancelConcurrencyGate
	paused bool
}

func (r *finalizeBarrierYear) pause() {
	if !r.paused {
		r.paused = true
		r.gate.pause()
	}
}
func (r *finalizeBarrierYear) GetByIDForUpdate(ctx context.Context, id uuid.UUID) (domain.AcademicYear, error) {
	y, err := r.AcademicYearRepository.GetByIDForUpdate(ctx, id)
	if err == nil {
		r.pause()
	}
	return y, err
}
func (r *finalizeBarrierYear) GetByIDForShare(ctx context.Context, id uuid.UUID) (domain.AcademicYear, error) {
	y, err := r.AcademicYearRepository.GetByIDForShare(ctx, id)
	if err == nil {
		r.pause()
	}
	return y, err
}
