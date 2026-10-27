package service_test

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

func TestStudentLeavingReadinessIntegration(t *testing.T) {
	tests := []struct {
		name string
		run  func(*testing.T, leavingStartFixture, domain.StudentLeavingRequest)
	}{
		{name: "tanpa rekening siap", run: func(t *testing.T, f leavingStartFixture, request domain.StudentLeavingRequest) {
			out := readStudentLeavingReadiness(t, f, request.ID)
			if !out.Ready || out.TotalAccounts != 0 || len(out.Blockers) != 0 {
				t.Fatalf("readiness=%+v", out)
			}
		}},
		{name: "rekening terbuka menghambat", run: func(t *testing.T, f leavingStartFixture, request domain.StudentLeavingRequest) {
			account := mustReadinessAccount(t, f)
			out := readStudentLeavingReadiness(t, f, request.ID)
			if out.Ready || out.OpenAccounts != 1 || !hasReadinessBlocker(out, service.LeavingBlockerAccountOpen, &account.ID) {
				t.Fatalf("readiness=%+v", out)
			}
		}},
		{name: "rekening lintas tahun seluruhnya diperiksa", run: func(t *testing.T, f leavingStartFixture, request domain.StudentLeavingRequest) {
			out := readStudentLeavingReadiness(t, f, request.ID)
			if out.Ready || out.TotalAccounts != 2 || out.OpenAccounts != 2 || !hasReadinessBlocker(out, service.LeavingBlockerAccountOpen, nil) {
				t.Fatalf("readiness=%+v", out)
			}
		}},
		{name: "rekening settled lintas tahun siap setelah handover", run: func(t *testing.T, f leavingStartFixture, request domain.StudentLeavingRequest) {
			accounts, err := f.repos.SavingsAccounts.ListByStudent(context.Background(), f.in.StudentID)
			if err != nil || len(accounts) != 2 {
				t.Fatalf("accounts=%d err=%v", len(accounts), err)
			}
			var current, historical domain.SavingsAccount
			for _, account := range accounts {
				if account.AcademicYearID == f.year.ID {
					current = account
				} else {
					historical = account
				}
			}
			settleReadinessAccount(t, f, historical)
			oldSettlement, err := domain.NewSavingsSettlement(historical.ID, domain.SettlementRegularYearEnd, 0, f.in.Actor.UserID, time.Now().UTC(), nil)
			if err != nil {
				t.Fatal(err)
			}
			if err := f.repos.Settlements.Create(context.Background(), oldSettlement); err != nil {
				t.Fatal(err)
			}
			currentSettlement := createLinkedCancelSettlement(t, f, request, current, 5000)
			insertCancelMovement(t, f, currentSettlement, domain.SettlementCashHandover, 5000)
			out := readStudentLeavingReadiness(t, f, request.ID)
			if !out.Ready || out.TotalAccounts != 2 || out.SettledAccounts != 2 {
				t.Fatalf("readiness=%+v", out)
			}
		}},
		{name: "rekening terbuka tahun tutup blocker khusus", run: func(t *testing.T, f leavingStartFixture, request domain.StudentLeavingRequest) {
			out := readStudentLeavingReadiness(t, f, request.ID)
			if !hasReadinessBlocker(out, service.LeavingBlockerAccountOpenYearClosed, nil) {
				t.Fatalf("readiness=%+v", out)
			}
		}},
		{name: "settlement nol tanpa handover siap", run: func(t *testing.T, f leavingStartFixture, request domain.StudentLeavingRequest) {
			account := mustReadinessAccount(t, f)
			createLinkedCancelSettlement(t, f, request, account, 0)
			out := readStudentLeavingReadiness(t, f, request.ID)
			if !out.Ready || out.SettledAccounts != 1 {
				t.Fatalf("readiness=%+v", out)
			}
		}},
		{name: "settlement historis tanpa relasi proses tetap sah", run: func(t *testing.T, f leavingStartFixture, request domain.StudentLeavingRequest) {
			account := mustReadinessAccount(t, f)
			settleReadinessAccount(t, f, account)
			settlement, err := domain.NewSavingsSettlement(account.ID, domain.SettlementRegularYearEnd, 0, f.in.Actor.UserID, time.Now().UTC(), nil)
			if err != nil {
				t.Fatal(err)
			}
			if err := f.repos.Settlements.Create(context.Background(), settlement); err != nil {
				t.Fatal(err)
			}
			out := readStudentLeavingReadiness(t, f, request.ID)
			if !out.Ready {
				t.Fatalf("settlement historis tidak boleh dianggap bermasalah: %+v", out)
			}
		}},
		{name: "rekening settled tanpa settlement menghambat", run: func(t *testing.T, f leavingStartFixture, request domain.StudentLeavingRequest) {
			account := mustReadinessAccount(t, f)
			settleReadinessAccount(t, f, account)
			out := readStudentLeavingReadiness(t, f, request.ID)
			if out.Ready || !hasReadinessBlocker(out, service.LeavingBlockerSettlementMissing, &account.ID) {
				t.Fatalf("readiness=%+v", out)
			}
		}},
		{name: "settlement positif tanpa handover", run: func(t *testing.T, f leavingStartFixture, request domain.StudentLeavingRequest) {
			account := mustReadinessAccount(t, f)
			settlement := createLinkedCancelSettlement(t, f, request, account, 10000)
			out := readStudentLeavingReadiness(t, f, request.ID)
			if out.Ready || out.OutstandingHandovers != 1 || !hasReadinessBlocker(out, service.LeavingBlockerHandoverIncomplete, &settlement.ID) {
				t.Fatalf("readiness=%+v", out)
			}
		}},
		{name: "handover lengkap siap", run: func(t *testing.T, f leavingStartFixture, request domain.StudentLeavingRequest) {
			account := mustReadinessAccount(t, f)
			settlement := createLinkedCancelSettlement(t, f, request, account, 10000)
			insertCancelMovement(t, f, settlement, domain.SettlementCashHandover, 10000)
			out := readStudentLeavingReadiness(t, f, request.ID)
			if !out.Ready || out.OutstandingHandovers != 0 {
				t.Fatalf("readiness=%+v", out)
			}
		}},
		{name: "handover sebagian menghambat", run: func(t *testing.T, f leavingStartFixture, request domain.StudentLeavingRequest) {
			account := mustReadinessAccount(t, f)
			settlement := createLinkedCancelSettlement(t, f, request, account, 10000)
			insertCancelMovement(t, f, settlement, domain.SettlementCashHandover, 4000)
			out := readStudentLeavingReadiness(t, f, request.ID)
			if out.Ready || out.OutstandingHandovers != 1 {
				t.Fatalf("readiness=%+v", out)
			}
		}},
		{name: "handover berlebih masalah integritas", run: func(t *testing.T, f leavingStartFixture, request domain.StudentLeavingRequest) {
			account := mustReadinessAccount(t, f)
			settlement := createLinkedCancelSettlement(t, f, request, account, 10000)
			insertCancelMovement(t, f, settlement, domain.SettlementCashHandover, 12000)
			out := readStudentLeavingReadiness(t, f, request.ID)
			if out.Ready || !hasReadinessBlocker(out, service.LeavingBlockerHandoverInvalid, &settlement.ID) {
				t.Fatalf("readiness=%+v", out)
			}
		}},
		{name: "beberapa blocker terurut deterministik", run: func(t *testing.T, f leavingStartFixture, request domain.StudentLeavingRequest) {
			account := mustReadinessAccount(t, f)
			late := createReadinessLateRequest(t, f, account, domain.LeavingTransactionPending)
			first := readStudentLeavingReadiness(t, f, request.ID)
			second := readStudentLeavingReadiness(t, f, request.ID)
			if first.Ready || !reflect.DeepEqual(first.Blockers, second.Blockers) || len(first.Blockers) < 2 ||
				!hasReadinessBlocker(first, service.LeavingBlockerAccountOpen, &account.ID) ||
				!hasReadinessBlocker(first, service.LeavingBlockerLateRequestPending, &late.ID) {
				t.Fatalf("blocker tidak lengkap/deterministik: %+v / %+v", first, second)
			}
		}},
		{name: "permintaan terlambat pending", run: func(t *testing.T, f leavingStartFixture, request domain.StudentLeavingRequest) {
			account := mustReadinessAccount(t, f)
			late := createReadinessLateRequest(t, f, account, domain.LeavingTransactionPending)
			out := readStudentLeavingReadiness(t, f, request.ID)
			if out.Ready || out.BlockingLateRequests != 1 || !hasReadinessBlocker(out, service.LeavingBlockerLateRequestPending, &late.ID) {
				t.Fatalf("readiness=%+v", out)
			}
		}},
		{name: "permintaan terlambat approved", run: func(t *testing.T, f leavingStartFixture, request domain.StudentLeavingRequest) {
			account := mustReadinessAccount(t, f)
			late := createReadinessLateRequest(t, f, account, domain.LeavingTransactionApproved)
			out := readStudentLeavingReadiness(t, f, request.ID)
			if out.Ready || out.BlockingLateRequests != 1 || !hasReadinessBlocker(out, service.LeavingBlockerLateRequestApproved, &late.ID) {
				t.Fatalf("readiness=%+v", out)
			}
		}},
		{name: "permintaan completed wajib punya ledger", run: func(t *testing.T, f leavingStartFixture, request domain.StudentLeavingRequest) {
			account := mustReadinessAccount(t, f)
			late := createCompletedReadinessLateRequest(t, f, account)
			out := readStudentLeavingReadiness(t, f, request.ID)
			if !hasReadinessBlocker(out, service.LeavingBlockerLateRequestLedgerMissing, &late.ID) {
				t.Fatalf("readiness=%+v", out)
			}
		}},
		{name: "permintaan completed cocok ledger tidak menjadi blocker", run: func(t *testing.T, f leavingStartFixture, request domain.StudentLeavingRequest) {
			account := mustReadinessAccount(t, f)
			late := createCompletedReadinessLateRequest(t, f, account)
			wali := readinessWali(t, f)
			transaction, err := domain.NewDeposit(account.ID, late.Amount, late.TransactionDate, wali.ID)
			if err != nil {
				t.Fatal(err)
			}
			transaction.LateRequestID = &late.ID
			if err := f.repos.Transactions.Create(context.Background(), transaction); err != nil {
				t.Fatal(err)
			}
			out := readStudentLeavingReadiness(t, f, request.ID)
			if hasReadinessBlocker(out, service.LeavingBlockerLateRequestLedgerMissing, &late.ID) || hasReadinessBlocker(out, service.LeavingBlockerLateRequestLedgerInvalid, &late.ID) {
				t.Fatalf("ledger cocok tetapi menjadi blocker: %+v", out)
			}
		}},
		{name: "relasi late request pada ledger tidak konsisten", run: func(t *testing.T, f leavingStartFixture, request domain.StudentLeavingRequest) {
			account := mustReadinessAccount(t, f)
			late := createCompletedReadinessLateRequest(t, f, account)
			wali := readinessWali(t, f)
			transaction, err := domain.NewDeposit(account.ID, late.Amount+1, late.TransactionDate, wali.ID)
			if err != nil {
				t.Fatal(err)
			}
			transaction.LateRequestID = &late.ID
			if err := f.repos.Transactions.Create(context.Background(), transaction); err != nil {
				t.Fatal(err)
			}
			out := readStudentLeavingReadiness(t, f, request.ID)
			if out.Ready || !hasReadinessBlocker(out, service.LeavingBlockerTransactionLateLinkInvalid, &transaction.ID) {
				t.Fatalf("readiness=%+v", out)
			}
		}},
		{name: "relasi keuangan salah menjadi blocker", run: func(t *testing.T, f leavingStartFixture, request domain.StudentLeavingRequest) {
			account := mustReadinessAccount(t, f)
			settlement := createLinkedCancelSettlement(t, f, request, account, 0)
			if _, err := f.db.Exec(`UPDATE savings_settlements SET type='REGULAR_YEAR_END' WHERE id=$1`, settlement.ID); err != nil {
				t.Fatal(err)
			}
			out := readStudentLeavingReadiness(t, f, request.ID)
			if out.Ready || !hasReadinessBlocker(out, service.LeavingBlockerSettlementTypeMismatch, &settlement.ID) {
				t.Fatalf("readiness=%+v", out)
			}
		}},
		{name: "transaksi setelah tanggal efektif menghambat", run: func(t *testing.T, f leavingStartFixture, request domain.StudentLeavingRequest) {
			account := mustReadinessAccount(t, f)
			wali := readinessWali(t, f)
			transaction, err := domain.NewDeposit(account.ID, 1000, request.EffectiveLeavingDate.AddDate(0, 0, 1), wali.ID)
			if err != nil {
				t.Fatal(err)
			}
			if err := f.repos.Transactions.Create(context.Background(), transaction); err != nil {
				t.Fatal(err)
			}
			out := readStudentLeavingReadiness(t, f, request.ID)
			if out.Ready || !hasReadinessBlocker(out, service.LeavingBlockerTransactionAfterLeavingDate, &transaction.ID) {
				t.Fatalf("readiness=%+v", out)
			}
		}},
		{name: "tanggal efektif tidak menghambat", run: func(t *testing.T, f leavingStartFixture, request domain.StudentLeavingRequest) {
			account := mustReadinessAccount(t, f)
			wali := readinessWali(t, f)
			transaction, err := domain.NewDeposit(account.ID, 1000, request.EffectiveLeavingDate, wali.ID)
			if err != nil {
				t.Fatal(err)
			}
			if err := f.repos.Transactions.Create(context.Background(), transaction); err != nil {
				t.Fatal(err)
			}
			out := readStudentLeavingReadiness(t, f, request.ID)
			if hasReadinessBlocker(out, service.LeavingBlockerTransactionAfterLeavingDate, &transaction.ID) {
				t.Fatalf("transaksi pada tanggal efektif tidak seharusnya menghambat: %+v", out)
			}
		}},
		{name: "transaksi cancelled tidak menghasilkan blocker tanggal", run: func(t *testing.T, f leavingStartFixture, request domain.StudentLeavingRequest) {
			account := mustReadinessAccount(t, f)
			wali := readinessWali(t, f)
			transaction, err := domain.NewDeposit(account.ID, 1000, request.EffectiveLeavingDate.AddDate(0, 0, 1), wali.ID)
			if err != nil {
				t.Fatal(err)
			}
			if err := transaction.Cancel(); err != nil {
				t.Fatal(err)
			}
			if err := f.repos.Transactions.Create(context.Background(), transaction); err != nil {
				t.Fatal(err)
			}
			out := readStudentLeavingReadiness(t, f, request.ID)
			if hasReadinessBlocker(out, service.LeavingBlockerTransactionAfterLeavingDate, &transaction.ID) {
				t.Fatalf("transaksi CANCELLED tidak seharusnya menghambat: %+v", out)
			}
		}},
		{name: "proses selesai tidak pernah ready", run: func(t *testing.T, f leavingStartFixture, request domain.StudentLeavingRequest) {
			if err := request.Cancel(f.in.Actor.UserID, time.Now().UTC(), "uji"); err != nil {
				t.Fatal(err)
			}
			if err := f.repos.StudentLeavingRequests.Update(context.Background(), request); err != nil {
				t.Fatal(err)
			}
			out := readStudentLeavingReadiness(t, f, request.ID)
			if out.Ready || !hasReadinessBlocker(out, service.LeavingBlockerProcessNotInProgress, &request.ID) {
				t.Fatalf("readiness=%+v", out)
			}
		}},
		{name: "proses finalized tidak pernah ready", run: func(t *testing.T, f leavingStartFixture, request domain.StudentLeavingRequest) {
			if err := request.Finalize(f.in.Actor.UserID, time.Now().UTC()); err != nil {
				t.Fatal(err)
			}
			if err := f.repos.StudentLeavingRequests.Update(context.Background(), request); err != nil {
				t.Fatal(err)
			}
			out := readStudentLeavingReadiness(t, f, request.ID)
			if out.Ready || !hasReadinessBlocker(out, service.LeavingBlockerProcessNotInProgress, &request.ID) {
				t.Fatalf("readiness=%+v", out)
			}
		}},
		{name: "siswa left tidak ready", run: func(t *testing.T, f leavingStartFixture, request domain.StudentLeavingRequest) {
			if _, err := f.db.Exec(`UPDATE students SET status='LEFT' WHERE id=$1`, request.StudentID); err != nil {
				t.Fatal(err)
			}
			out := readStudentLeavingReadiness(t, f, request.ID)
			if out.Ready || !hasReadinessBlocker(out, service.LeavingBlockerStudentNotActive, &request.StudentID) {
				t.Fatalf("readiness=%+v", out)
			}
		}},
		{name: "wali dan siswa ditolak", run: func(t *testing.T, f leavingStartFixture, request domain.StudentLeavingRequest) {
			for _, role := range []domain.UserRole{domain.RoleWaliKelas, domain.RoleSiswa} {
				_, err := f.svc.Readiness(context.Background(), service.StudentLeavingReadinessInput{Actor: service.Actor{UserID: f.in.Actor.UserID, Role: role}, StudentLeavingRequestID: request.ID})
				if !errors.Is(err, service.ErrForbidden) {
					t.Fatalf("role %s error=%v", role, err)
				}
			}
		}},
		{name: "proses tidak ditemukan", run: func(t *testing.T, f leavingStartFixture, _ domain.StudentLeavingRequest) {
			_, err := f.svc.Readiness(context.Background(), service.StudentLeavingReadinessInput{Actor: f.in.Actor, StudentLeavingRequestID: uuid.New()})
			if !errors.Is(err, repository.ErrNotFound) {
				t.Fatalf("error=%v, want ErrNotFound", err)
			}
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newLeavingCancelFixture(t)
			var request domain.StudentLeavingRequest
			switch tc.name {
			case "tanpa rekening siap":
				request = startLeavingRequestForCancel(t, f)
			case "rekening lintas tahun seluruhnya diperiksa", "rekening settled lintas tahun siap setelah handover":
				createCancelAccount(t, f, f.year.ID)
				moveReadinessFixtureToNextYear(t, &f)
				createCancelAccount(t, f, f.year.ID)
				request = createReadinessProcess(t, f)
			case "rekening terbuka tahun tutup blocker khusus":
				createCancelAccount(t, f, f.year.ID)
				moveReadinessFixtureToNextYear(t, &f)
				request = createReadinessProcess(t, f)
			case "settlement historis tanpa relasi proses tetap sah", "rekening settled tanpa settlement menghambat":
				createCancelAccount(t, f, f.year.ID)
				request = startLeavingRequestForCancel(t, f)
			default:
				createCancelAccount(t, f, f.year.ID)
				request = startLeavingRequestForCancel(t, f)
			}
			tc.run(t, f, request)
		})
	}
}

func TestStudentLeavingReadinessIsDeterministicAndReadOnly(t *testing.T) {
	f := newLeavingCancelFixture(t)
	account := createCancelAccount(t, f, f.year.ID)
	request := startLeavingRequestForCancel(t, f)
	before := readinessSnapshot(t, f.db)
	first := readStudentLeavingReadiness(t, f, request.ID)
	second := readStudentLeavingReadiness(t, f, request.ID)
	if !reflect.DeepEqual(first.Blockers, second.Blockers) || first.Ready != second.Ready {
		t.Fatalf("readiness differs between identical reads: %+v / %+v", first, second)
	}
	if after := readinessSnapshot(t, f.db); !reflect.DeepEqual(before, after) {
		t.Fatal("Readiness mengubah data proses, siswa, rekening, transaksi, settlement, cash movement, atau audit")
	}
	if account.ID == uuid.Nil {
		t.Fatal("fixture rekening tidak valid")
	}
}

func readStudentLeavingReadiness(t *testing.T, f leavingStartFixture, requestID uuid.UUID) service.StudentLeavingReadinessOutput {
	t.Helper()
	before := readinessSnapshot(t, f.db)
	out, err := f.svc.Readiness(context.Background(), service.StudentLeavingReadinessInput{Actor: f.in.Actor, StudentLeavingRequestID: requestID})
	if err != nil {
		t.Fatal(err)
	}
	if after := readinessSnapshot(t, f.db); !reflect.DeepEqual(before, after) {
		t.Fatal("Readiness mengubah data")
	}
	return out
}

func mustReadinessAccount(t *testing.T, f leavingStartFixture) domain.SavingsAccount {
	t.Helper()
	accounts, err := f.repos.SavingsAccounts.ListByStudent(context.Background(), f.in.StudentID)
	if err != nil || len(accounts) == 0 {
		t.Fatalf("rekening fixture: %v", err)
	}
	return accounts[0]
}

func moveReadinessFixtureToNextYear(t *testing.T, f *leavingStartFixture) {
	t.Helper()
	accounts, err := f.repos.SavingsAccounts.ListByStudent(context.Background(), f.in.StudentID)
	if err != nil || len(accounts) == 0 {
		t.Fatalf("rekening tahun lama: %v", err)
	}
	oldAccount := accounts[0]
	if err := oldAccount.Settle(time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := f.repos.SavingsAccounts.Update(context.Background(), oldAccount); err != nil {
		t.Fatal(err)
	}
	previous := f.year
	if err := previous.StartClosing(); err != nil {
		t.Fatal(err)
	}
	if err := f.repos.AcademicYears.Update(context.Background(), previous); err != nil {
		t.Fatal(err)
	}
	if err := previous.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.repos.AcademicYears.Update(context.Background(), previous); err != nil {
		t.Fatal(err)
	}
	start := previous.EndDate
	value, err := domain.NewAcademicYear("Tahun readiness berikutnya", start, start.AddDate(1, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if err := f.repos.AcademicYears.Create(context.Background(), value); err != nil {
		t.Fatal(err)
	}
	class := newServiceScopeClass(t, "Kelas readiness berikutnya", 7)
	class.AcademicYearID = value.ID
	class.EffectiveDate = value.StartDate
	if err := f.repos.Classes.Create(context.Background(), class); err != nil {
		t.Fatal(err)
	}
	period, err := domain.NewDateRange(value.StartDate, &value.EndDate)
	if err != nil {
		t.Fatal(err)
	}
	history, err := domain.NewStudentClassHistory(f.in.StudentID, value.ID, class.ID, period)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.repos.StudentClassHistories.Create(context.Background(), history); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(`ALTER TABLE savings_accounts DISABLE TRIGGER savings_accounts_no_reopen_after_year_close_trigger`); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(`UPDATE savings_accounts SET status='OPEN', settled_at=NULL WHERE id=$1`, oldAccount.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(`ALTER TABLE savings_accounts ENABLE TRIGGER savings_accounts_no_reopen_after_year_close_trigger`); err != nil {
		t.Fatal(err)
	}
	f.year, f.class = value, class
	f.in.AcademicYearID = value.ID
	f.in.EffectiveLeavingDate = serviceDateOnlyUTC(value.StartDate)
}

func createReadinessProcess(t *testing.T, f leavingStartFixture) domain.StudentLeavingRequest {
	t.Helper()
	request, err := domain.NewStudentLeavingRequest(f.in.StudentID, f.year.ID, f.in.EffectiveLeavingDate, f.in.Actor.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.repos.StudentLeavingRequests.Create(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	return request
}

func readinessWali(t *testing.T, f leavingStartFixture) domain.User {
	t.Helper()
	wali := newServiceScopeUser(t, domain.RoleWaliKelas)
	if err := f.repos.Users.Create(context.Background(), wali); err != nil {
		t.Fatal(err)
	}
	return wali
}

func settleReadinessAccount(t *testing.T, f leavingStartFixture, account domain.SavingsAccount) {
	t.Helper()
	if err := account.Settle(time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := f.repos.SavingsAccounts.Update(context.Background(), account); err != nil {
		t.Fatal(err)
	}
}

func createReadinessLateRequest(t *testing.T, f leavingStartFixture, account domain.SavingsAccount, status domain.LeavingTransactionRequestStatus) domain.LeavingTransactionRequest {
	t.Helper()
	request, err := domain.NewLeavingTransactionRequest(account.ID, account.AcademicYearID, domain.TransactionDeposit, 1000, f.in.EffectiveLeavingDate, "uji kesiapan", "bukti-readiness-"+uuid.NewString(), f.in.Actor.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if status == domain.LeavingTransactionApproved {
		if err := request.Decide(f.in.Actor.UserID, true, "Disetujui untuk uji kesiapan"); err != nil {
			t.Fatal(err)
		}
	} else {
		request.Status = status
	}
	if err := f.repos.LeavingTransactionRequests.CreateLeavingTransactionRequest(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	return request
}

func createCompletedReadinessLateRequest(t *testing.T, f leavingStartFixture, account domain.SavingsAccount) domain.LeavingTransactionRequest {
	t.Helper()
	request := createReadinessLateRequest(t, f, account, domain.LeavingTransactionPending)
	if err := request.Decide(f.in.Actor.UserID, true, "Disetujui untuk uji kesiapan"); err != nil {
		t.Fatal(err)
	}
	if err := f.repos.LeavingTransactionRequests.UpdateLeavingTransactionRequest(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if err := request.Complete(); err != nil {
		t.Fatal(err)
	}
	if err := f.repos.LeavingTransactionRequests.UpdateLeavingTransactionRequest(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	return request
}

func hasReadinessBlocker(out service.StudentLeavingReadinessOutput, code service.StudentLeavingReadinessBlockerCode, recordID *uuid.UUID) bool {
	for _, blocker := range out.Blockers {
		if blocker.Code == code && (recordID == nil || (blocker.RecordID != nil && *blocker.RecordID == *recordID) || (blocker.AccountID != nil && *blocker.AccountID == *recordID)) {
			return true
		}
	}
	return false
}

func readinessSnapshot(t *testing.T, db *sql.DB) map[string]string {
	t.Helper()
	result := make(map[string]string)
	for _, table := range []string{"students", "academic_years", "student_class_histories", "savings_accounts", "transactions", "leaving_transaction_requests", "savings_settlements", "settlement_cash_movements", "student_leaving_requests", "audit_logs"} {
		var rows string
		query := "SELECT COALESCE(jsonb_agg(to_jsonb(t) ORDER BY t.id),'[]'::jsonb)::text FROM " + table + " t"
		if err := db.QueryRow(query).Scan(&rows); err != nil {
			t.Fatal(err)
		}
		result[table] = rows
	}
	return result
}
