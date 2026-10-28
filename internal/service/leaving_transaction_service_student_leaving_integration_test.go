package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

func TestCreateLateRequestForStudentLeaving(t *testing.T) {
	t.Run("membuat tautan audit tanpa mengubah saldo", func(t *testing.T) {
		f := newInternalLeavingCancelFixture(t)
		cleanupLinkedLateRequests(t, f.db, f.account.ID, f.request.ID, f.waliActor.UserID)
		historical, err := domain.NewLeavingTransactionRequest(f.account.ID, f.year.ID, domain.TransactionDeposit, 1_000, f.year.StartDate, "Permintaan historis", "HISTORIS-"+uuid.NewString(), f.waliActor.UserID)
		if err != nil {
			t.Fatal(err)
		}
		if err := historical.Decide(f.adminActor.UserID, false, "Ditolak dalam fixture historis"); err != nil {
			t.Fatal(err)
		}
		if err := f.repos.LeavingTransactionRequests.CreateLeavingTransactionRequest(context.Background(), historical); err != nil {
			t.Fatal(err)
		}
		before := linkedLateFinancialSnapshot(t, f)
		out, err := linkedLateCreator(f, f.manager).createForStudentLeavingRequest(context.Background(), linkedLateInput(f, f.waliActor, f.year.StartDate, "BUKTI-"+uuid.NewString()))
		if err != nil {
			t.Fatalf("createForStudentLeavingRequest(): %v", err)
		}
		if out.Request.Status != domain.LeavingTransactionPending || out.StudentLeavingRequestID != f.request.ID {
			t.Fatalf("output=%+v", out)
		}
		var linkedID uuid.UUID
		var status string
		if err := f.db.QueryRow(`SELECT student_leaving_request_id, status FROM leaving_transaction_requests WHERE id=$1`, out.Request.ID).Scan(&linkedID, &status); err != nil {
			t.Fatal(err)
		}
		if linkedID != f.request.ID || status != string(domain.LeavingTransactionPending) {
			t.Fatalf("link/status=%s/%s", linkedID, status)
		}
		var studentStatus, accountStatus, processStatus string
		if err := f.db.QueryRow(`SELECT status FROM students WHERE id=$1`, f.student.ID).Scan(&studentStatus); err != nil {
			t.Fatal(err)
		}
		if err := f.db.QueryRow(`SELECT status FROM savings_accounts WHERE id=$1`, f.account.ID).Scan(&accountStatus); err != nil {
			t.Fatal(err)
		}
		if err := f.db.QueryRow(`SELECT status FROM student_leaving_requests WHERE id=$1`, f.request.ID).Scan(&processStatus); err != nil {
			t.Fatal(err)
		}
		if studentStatus != string(domain.StudentActive) || accountStatus != string(domain.SavingsAccountOpen) || processStatus != string(domain.StudentLeavingInProgress) {
			t.Fatalf("lifecycle changed: student=%s account=%s process=%s", studentStatus, accountStatus, processStatus)
		}
		if after := linkedLateFinancialSnapshot(t, f); !reflect.DeepEqual(before, after) {
			t.Fatal("Create mengubah student, rekening, ledger, settlement, atau penyerahan")
		}
		var historicalStatus string
		var historicalLink *uuid.UUID
		if err := f.db.QueryRow(`SELECT student_leaving_request_id, status FROM leaving_transaction_requests WHERE id=$1`, historical.ID).Scan(&historicalLink, &historicalStatus); err != nil {
			t.Fatal(err)
		}
		if historicalLink != nil || historicalStatus != string(domain.LeavingTransactionRejected) {
			t.Fatalf("permintaan historis berubah: link=%v status=%s", historicalLink, historicalStatus)
		}
		var payload []byte
		if err := f.db.QueryRow(`SELECT after_data FROM audit_logs WHERE entity_type='LEAVING_TRANSACTION_REQUEST' AND entity_id=$1 AND action='REQUEST_FOR_STUDENT_LEAVING'`, out.Request.ID).Scan(&payload); err != nil {
			t.Fatal(err)
		}
		var audit map[string]any
		if err := json.Unmarshal(payload, &audit); err != nil {
			t.Fatal(err)
		}
		if audit["student_leaving_request_id"] != f.request.ID.String() || audit["student_id"] != f.student.ID.String() || audit["savings_account_id"] != f.account.ID.String() || audit["academic_year_id"] != f.year.ID.String() || audit["type"] != string(domain.TransactionDeposit) || audit["status"] != string(domain.LeavingTransactionPending) || audit["reason"] != "Transaksi terlambat yang terverifikasi" || audit["evidence_reference"] == "" {
			t.Fatalf("payload audit tidak lengkap: %s", payload)
		}
		settlementBefore := linkedLateFinancialSnapshot(t, f)
		if _, err := f.settlementService.settleForStudentLeavingRequest(context.Background(), settleForStudentLeavingRequestInput{
			Actor: f.waliActor, StudentLeavingRequestID: f.request.ID, SavingsAccountID: f.account.ID, ExpectedBalance: 50_000,
		}); !errors.Is(err, domain.ErrInvalidTransition) {
			t.Fatalf("settlement dengan permintaan tertaut PENDING error=%v, want ErrInvalidTransition", err)
		}
		if after := linkedLateFinancialSnapshot(t, f); !reflect.DeepEqual(settlementBefore, after) {
			t.Fatal("settlement yang ditolak mengubah ledger atau status rekening")
		}
		readiness, err := f.cancelService.Readiness(context.Background(), StudentLeavingReadinessInput{Actor: f.adminActor, StudentLeavingRequestID: f.request.ID})
		if err != nil || readiness.Ready || !linkedLateHasReadinessBlocker(readiness, LeavingBlockerLateRequestPending, &out.Request.ID) {
			t.Fatalf("Readiness tidak memblokir PENDING: %+v, %v", readiness, err)
		}
	})

	t.Run("mendukung rekening pada tahun berbeda", func(t *testing.T) {
		f := newInternalLeavingCancelFixture(t)
		cleanupLinkedLateRequests(t, f.db, f.account.ID, f.request.ID, f.waliActor.UserID)
		settlement, err := f.settlementService.settleForStudentLeavingRequest(context.Background(), settleForStudentLeavingRequestInput{
			Actor: f.waliActor, StudentLeavingRequestID: f.request.ID, SavingsAccountID: f.account.ID, ExpectedBalance: 50_000,
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.settlementService.ConfirmHandover(context.Background(), ConfirmSettlementHandoverInput{
			Actor: f.waliActor, SettlementID: settlement.Settlement.ID, Confirm: true, ReceiptReference: stringPointer("MQ08-CROSS-YEAR-" + uuid.NewString()),
		}); err != nil {
			t.Fatal(err)
		}
		yearService := &academicYearService{deps: Dependencies{Repositories: f.repos, UOW: f.manager}}
		if _, err := yearService.StartClosing(context.Background(), StartClosingAcademicYearInput{Actor: f.adminActor, AcademicYearID: f.year.ID}); err != nil {
			t.Fatal(err)
		}
		if _, err := yearService.Close(context.Background(), CloseAcademicYearInput{Actor: f.adminActor, AcademicYearID: f.year.ID}); err != nil {
			t.Fatal(err)
		}
		if _, err := f.cancelService.Cancel(context.Background(), CancelStudentLeavingRequestInput{
			Actor: f.adminActor, StudentLeavingRequestID: f.request.ID, Reason: "Menyiapkan proses lintas tahun dalam pengujian",
		}); err != nil {
			t.Fatal(err)
		}
		priorYear, err := domain.NewAcademicYear("MQ08 linked prior "+uuid.NewString(), f.year.StartDate.AddDate(-1, 0, 0), f.year.StartDate)
		if err != nil {
			t.Fatal(err)
		}
		if err := f.repos.AcademicYears.Create(context.Background(), priorYear); err != nil {
			t.Fatal(err)
		}
		var classID, accountID, assignmentID, historyID uuid.UUID
		t.Cleanup(func() {
			_, _ = f.db.Exec(`DELETE FROM leaving_transaction_requests WHERE student_leaving_request_id=$1 OR savings_account_id=$2`, f.request.ID, accountID)
			_, _ = f.db.Exec(`DELETE FROM audit_logs WHERE actor_user_id=$1 AND action='REQUEST_FOR_STUDENT_LEAVING'`, f.waliActor.UserID)
			_, _ = f.db.Exec(`DELETE FROM savings_accounts WHERE id=$1`, accountID)
			_, _ = f.db.Exec(`DELETE FROM student_class_histories WHERE id=$1`, historyID)
			_, _ = f.db.Exec(`DELETE FROM teacher_class_assignments WHERE id=$1`, assignmentID)
			_, _ = f.db.Exec(`DELETE FROM classes WHERE id=$1`, classID)
			_, _ = f.db.Exec(`DELETE FROM academic_years WHERE id=$1`, priorYear.ID)
		})
		class, err := domain.NewClass("MQ08 linked prior "+uuid.NewString(), 8)
		if err != nil {
			t.Fatal(err)
		}
		class.AcademicYearID, class.EffectiveDate = priorYear.ID, priorYear.StartDate
		if err := f.repos.Classes.Create(context.Background(), class); err != nil {
			t.Fatal(err)
		}
		period, err := domain.NewDateRange(priorYear.StartDate, &priorYear.EndDate)
		if err != nil {
			t.Fatal(err)
		}
		history, err := domain.NewStudentClassHistory(f.student.ID, priorYear.ID, class.ID, period)
		if err != nil {
			t.Fatal(err)
		}
		assignment, err := domain.NewTeacherClassAssignment(f.waliActor.UserID, priorYear.ID, class.ID, period)
		if err != nil {
			t.Fatal(err)
		}
		if err := f.repos.StudentClassHistories.Create(context.Background(), history); err != nil {
			t.Fatal(err)
		}
		historyID = history.ID
		if err := f.repos.TeacherClassAssignments.Create(context.Background(), assignment); err != nil {
			t.Fatal(err)
		}
		assignmentID = assignment.ID
		classID = class.ID
		account, err := domain.NewSavingsAccount(f.student.ID, priorYear.ID)
		if err != nil {
			t.Fatal(err)
		}
		if err := f.repos.SavingsAccounts.Create(context.Background(), account); err != nil {
			t.Fatal(err)
		}
		accountID = account.ID
		priorYearService := &academicYearService{deps: Dependencies{Repositories: f.repos, UOW: f.manager}}
		if _, err := priorYearService.StartClosing(context.Background(), StartClosingAcademicYearInput{Actor: f.adminActor, AcademicYearID: priorYear.ID}); err != nil {
			t.Fatal(err)
		}
		if _, err := (&academicYearService{deps: Dependencies{Repositories: f.repos, UOW: f.manager}}).Reopen(context.Background(), ReopenAcademicYearInput{Actor: f.adminActor, AcademicYearID: f.year.ID}); err != nil {
			t.Fatal(err)
		}
		started, err := f.cancelService.Start(context.Background(), StartStudentLeavingRequestInput{
			Actor: f.adminActor, StudentID: f.student.ID, AcademicYearID: f.year.ID,
			EffectiveLeavingDate: f.request.EffectiveLeavingDate,
		})
		if err != nil {
			t.Fatal(err)
		}
		f.request = *started.Request
		cleanupLinkedLateRequests(t, f.db, account.ID, f.request.ID, f.waliActor.UserID)
		input := createLeavingTransactionForStudentLeavingInput{Actor: f.waliActor, StudentLeavingRequestID: f.request.ID, SavingsAccountID: account.ID, Type: domain.TransactionWithdrawal, Amount: 1_000, TransactionDate: priorYear.EndDate.AddDate(0, 0, -1), Reason: "Penarikan historis", EvidenceReference: "LINTAS-" + uuid.NewString()}
		probe := &linkedLateLockProbe{}
		out, err := linkedLateCreator(f, linkedLateLockProbeManager{UnitOfWorkManager: f.manager, probe: probe}).createForStudentLeavingRequest(context.Background(), input)
		if err != nil {
			t.Fatalf("lintas tahun: %v", err)
		}
		if out.Request.AcademicYearID != priorYear.ID || out.StudentLeavingRequestID != f.request.ID {
			t.Fatalf("output=%+v", out)
		}
		var link uuid.UUID
		if err := f.db.QueryRow(`SELECT student_leaving_request_id FROM leaving_transaction_requests WHERE id=$1`, out.Request.ID).Scan(&link); err != nil || link != f.request.ID {
			t.Fatalf("foreign key lintas tahun=%s err=%v", link, err)
		}
		wantCalls := []string{"year:" + sortedUUIDs([]uuid.UUID{f.request.AcademicYearID, priorYear.ID})[0].String(), "year:" + sortedUUIDs([]uuid.UUID{f.request.AcademicYearID, priorYear.ID})[1].String(), "account", "student", "process"}
		if got := probe.snapshot(); !reflect.DeepEqual(got, wantCalls) {
			t.Fatalf("urutan lock=%v, want %v", got, wantCalls)
		}
	})
}

func TestCreateLateRequestRepositoryRejectsMismatchedProcessAccount(t *testing.T) {
	f := newInternalLeavingCancelFixture(t)
	cleanupLinkedLateRequests(t, f.db, f.account.ID, f.request.ID, f.waliActor.UserID)
	foreign, err := domain.NewStudent("MQ08 foreign repository "+uuid.NewString(), nil, stringPointer("nis-"+uuid.NewString()), stringPointer("nisn-"+uuid.NewString()))
	if err != nil {
		t.Fatal(err)
	}
	if err := f.repos.Students.Create(context.Background(), foreign); err != nil {
		t.Fatal(err)
	}
	account, err := domain.NewSavingsAccount(foreign.ID, f.year.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.repos.SavingsAccounts.Create(context.Background(), account); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = f.db.Exec(`DELETE FROM savings_accounts WHERE id=$1`, account.ID)
		_, _ = f.db.Exec(`DELETE FROM students WHERE id=$1`, foreign.ID)
	})
	request, err := domain.NewLeavingTransactionRequest(account.ID, f.year.ID, domain.TransactionDeposit, 1_000, f.year.StartDate, "Tidak boleh tertaut", "WRONG-"+uuid.NewString(), f.waliActor.UserID)
	if err != nil {
		t.Fatal(err)
	}
	err = f.repos.LeavingTransactionRequests.CreateLeavingTransactionRequestForStudentLeaving(context.Background(), request, f.request.ID)
	if !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("insert relasi lintas siswa error=%v, want ErrConflict", err)
	}
	var count int
	if err := f.db.QueryRow(`SELECT count(*) FROM leaving_transaction_requests WHERE id=$1`, request.ID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("request invalid tersimpan=%d err=%v", count, err)
	}
}

func TestCreateLateRequestForStudentLeavingRejectsInvalidState(t *testing.T) {
	tests := []string{"cancelled", "finalized", "settled", "closed year", "out of scope", "admin", "student", "foreign account", "effective date", "after effective date", "before account year", "after account year", "duplicate evidence", "invalid type", "invalid amount"}
	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			f := newInternalLeavingCancelFixture(t)
			cleanupLinkedLateRequests(t, f.db, f.account.ID, f.request.ID, f.waliActor.UserID)
			input := linkedLateInput(f, f.waliActor, f.year.StartDate, "CASE-"+uuid.NewString())
			want := error(domain.ErrInvalidTransition)
			switch name {
			case "cancelled":
				if _, err := f.cancelService.Cancel(context.Background(), CancelStudentLeavingRequestInput{Actor: f.adminActor, StudentLeavingRequestID: f.request.ID, Reason: "uji pembatalan"}); err != nil {
					t.Fatal(err)
				}
			case "finalized":
				settled, err := f.settlementService.settleForStudentLeavingRequest(context.Background(), settleForStudentLeavingRequestInput{Actor: f.waliActor, StudentLeavingRequestID: f.request.ID, SavingsAccountID: f.account.ID, ExpectedBalance: 50_000})
				if err != nil {
					t.Fatal(err)
				}
				if _, err := f.settlementService.ConfirmHandover(context.Background(), ConfirmSettlementHandoverInput{Actor: f.waliActor, SettlementID: settled.Settlement.ID, Confirm: true, ReceiptReference: stringPointer("MQ08-FINAL-" + uuid.NewString())}); err != nil {
					t.Fatal(err)
				}
				if _, err := f.cancelService.Finalize(context.Background(), FinalizeStudentLeavingRequestInput{Actor: f.adminActor, StudentLeavingRequestID: f.request.ID}); err != nil {
					t.Fatal(err)
				}
			case "settled":
				if _, err := f.settlementService.settleForStudentLeavingRequest(context.Background(), settleForStudentLeavingRequestInput{Actor: f.waliActor, StudentLeavingRequestID: f.request.ID, SavingsAccountID: f.account.ID, ExpectedBalance: 50_000}); err != nil {
					t.Fatal(err)
				}
			case "closed year":
				manager := linkedLateClosedYearManager{UnitOfWorkManager: f.manager}
				before := linkedLateFinancialSnapshot(t, f)
				_, err := linkedLateCreator(f, manager).createForStudentLeavingRequest(context.Background(), input)
				if !errors.Is(err, want) {
					t.Fatalf("error=%v, want errors.Is(_, %v)", err, want)
				}
				if after := linkedLateFinancialSnapshot(t, f); !reflect.DeepEqual(before, after) {
					t.Fatal("penolakan tahun tertutup mengubah data keuangan/lifecycle")
				}
				return
			case "out of scope":
				user := newInternalCancelUser(t, f.repos, domain.RoleWaliKelas)
				t.Cleanup(func() { _, _ = f.db.Exec(`DELETE FROM users WHERE id=$1`, user.ID) })
				input.Actor.UserID = user.ID
				want = ErrScopeViolation
			case "admin":
				input.Actor = f.adminActor
				want = ErrForbidden
			case "student":
				input.Actor.Role = domain.RoleSiswa
				want = ErrForbidden
			case "foreign account":
				foreign, err := domain.NewStudent("MQ08 foreign "+uuid.NewString(), nil, stringPointer("nis-"+uuid.NewString()), stringPointer("nisn-"+uuid.NewString()))
				if err != nil {
					t.Fatal(err)
				}
				if err := f.repos.Students.Create(context.Background(), foreign); err != nil {
					t.Fatal(err)
				}
				account, err := domain.NewSavingsAccount(foreign.ID, f.year.ID)
				if err != nil {
					t.Fatal(err)
				}
				if err := f.repos.SavingsAccounts.Create(context.Background(), account); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					_, _ = f.db.Exec(`DELETE FROM savings_accounts WHERE id=$1`, account.ID)
					_, _ = f.db.Exec(`DELETE FROM students WHERE id=$1`, foreign.ID)
				})
				input.SavingsAccountID = account.ID
			case "effective date":
				input.TransactionDate = f.request.EffectiveLeavingDate
				want = domain.ErrInvalidDateRange
			case "after effective date":
				input.TransactionDate = f.request.EffectiveLeavingDate.AddDate(0, 0, 1)
				want = domain.ErrInvalidDateRange
			case "before account year":
				input.TransactionDate = f.year.StartDate.AddDate(0, 0, -1)
				want = domain.ErrInvalidDateRange
			case "after account year":
				input.TransactionDate = f.year.EndDate
				want = domain.ErrInvalidDateRange
			case "duplicate evidence":
				historical, err := domain.NewLeavingTransactionRequest(f.account.ID, f.year.ID, domain.TransactionDeposit, 1_000, f.year.StartDate, "Permintaan lama", input.EvidenceReference, f.waliActor.UserID)
				if err != nil {
					t.Fatal(err)
				}
				if err := f.repos.LeavingTransactionRequests.CreateLeavingTransactionRequest(context.Background(), historical); err != nil {
					t.Fatal(err)
				}
				defer func() { _, _ = f.db.Exec(`DELETE FROM leaving_transaction_requests WHERE id=$1`, historical.ID) }()
				want = repository.ErrConflict
			case "invalid type":
				input.Type = domain.TransactionType("TRANSFER")
				want = domain.ErrInvalidAmount
			case "invalid amount":
				input.Amount = 0
				want = domain.ErrInvalidAmount
			}
			before := linkedLateFinancialSnapshot(t, f)
			_, err := linkedLateCreator(f, f.manager).createForStudentLeavingRequest(context.Background(), input)
			if !errors.Is(err, want) {
				t.Fatalf("error=%v, want errors.Is(_, %v)", err, want)
			}
			if after := linkedLateFinancialSnapshot(t, f); !reflect.DeepEqual(before, after) {
				t.Fatal("penolakan mengubah data keuangan/lifecycle")
			}
		})
	}
}

func TestCreateLateRequestForStudentLeavingRollback(t *testing.T) {
	for _, stage := range []string{"insert after write", "audit after write", "commit"} {
		t.Run(stage, func(t *testing.T) {
			f := newInternalLeavingCancelFixture(t)
			cleanupLinkedLateRequests(t, f.db, f.account.ID, f.request.ID, f.waliActor.UserID)
			before := linkedLateFullSnapshot(t, f)
			injected := fmt.Errorf("injected %s failure", stage)
			manager := linkedLateFaultManager{UnitOfWorkManager: f.manager, stage: stage, failure: injected}
			_, err := linkedLateCreator(f, manager).createForStudentLeavingRequest(context.Background(), linkedLateInput(f, f.waliActor, f.year.StartDate, "ROLLBACK-"+uuid.NewString()))
			if !errors.Is(err, injected) {
				t.Fatalf("error=%v, want injected error", err)
			}
			if after := linkedLateFullSnapshot(t, f); !reflect.DeepEqual(before, after) {
				t.Fatalf("failure at %s left partial state\nbefore=%v\nafter=%v", stage, before, after)
			}
		})
	}
}

func linkedLateCreator(f internalLeavingCancelFixture, manager repository.UnitOfWorkManager) *leavingTransactionRequestService {
	return &leavingTransactionRequestService{deps: Dependencies{Repositories: f.repos, UOW: manager}}
}

func linkedLateInput(f internalLeavingCancelFixture, actor Actor, date time.Time, evidence string) createLeavingTransactionForStudentLeavingInput {
	return createLeavingTransactionForStudentLeavingInput{
		Actor: actor, StudentLeavingRequestID: f.request.ID, SavingsAccountID: f.account.ID,
		Type: domain.TransactionDeposit, Amount: 2_000, TransactionDate: date,
		Reason: "Transaksi terlambat yang terverifikasi", EvidenceReference: evidence,
	}
}

func cleanupLinkedLateRequests(t *testing.T, db *sql.DB, accountID, processID, actorID uuid.UUID) {
	t.Helper()
	t.Cleanup(func() {
		if _, err := db.Exec(`DELETE FROM audit_logs WHERE actor_user_id=$1 AND action='REQUEST_FOR_STUDENT_LEAVING'`, actorID); err != nil {
			t.Errorf("cleanup linked late request audits: %v", err)
		}
		if _, err := db.Exec(`DELETE FROM leaving_transaction_requests WHERE savings_account_id=$1 OR student_leaving_request_id=$2`, accountID, processID); err != nil {
			t.Errorf("cleanup linked late requests: %v", err)
		}
	})
}

func linkedLateFinancialSnapshot(t *testing.T, f internalLeavingCancelFixture) map[string]string {
	t.Helper()
	return linkedLateSnapshot(t, f, false)
}

func linkedLateFullSnapshot(t *testing.T, f internalLeavingCancelFixture) map[string]string {
	t.Helper()
	return linkedLateSnapshot(t, f, true)
}

func linkedLateSnapshot(t *testing.T, f internalLeavingCancelFixture, includeLateAndAudit bool) map[string]string {
	t.Helper()
	ctx := context.Background()
	out := map[string]string{}
	queries := map[string]string{
		"student":      `SELECT to_jsonb(row)::text FROM (SELECT * FROM students WHERE id=$1) row`,
		"account":      `SELECT to_jsonb(row)::text FROM (SELECT * FROM savings_accounts WHERE id=$1) row`,
		"process":      `SELECT to_jsonb(row)::text FROM (SELECT * FROM student_leaving_requests WHERE id=$1) row`,
		"transactions": `SELECT COALESCE(jsonb_agg(to_jsonb(row) ORDER BY id)::text,'[]') FROM (SELECT * FROM transactions WHERE savings_account_id=$1) row`,
		"settlements":  `SELECT COALESCE(jsonb_agg(to_jsonb(row) ORDER BY id)::text,'[]') FROM (SELECT * FROM savings_settlements WHERE savings_account_id=$1) row`,
		"movements":    `SELECT COALESCE(jsonb_agg(to_jsonb(row) ORDER BY id)::text,'[]') FROM (SELECT movement.* FROM settlement_cash_movements movement JOIN savings_settlements settlement ON settlement.id=movement.savings_settlement_id WHERE settlement.savings_account_id=$1) row`,
	}
	if includeLateAndAudit {
		queries["late_requests"] = `SELECT COALESCE(jsonb_agg(to_jsonb(row) ORDER BY id)::text,'[]') FROM (SELECT * FROM leaving_transaction_requests WHERE savings_account_id=$1 OR student_leaving_request_id=$2) row`
		queries["audit"] = `SELECT COALESCE(jsonb_agg(to_jsonb(row) ORDER BY id)::text,'[]') FROM (SELECT * FROM audit_logs WHERE entity_id IN ($1,$2) OR actor_user_id=$3) row`
	}
	for name, query := range queries {
		var value string
		args := []any{f.account.ID}
		switch name {
		case "student":
			args[0] = f.student.ID
		case "process":
			args[0] = f.request.ID
		case "late_requests":
			args = []any{f.account.ID, f.request.ID}
		case "audit":
			args = []any{f.student.ID, f.request.ID, f.waliActor.UserID}
		}
		if err := f.db.QueryRowContext(ctx, query, args...).Scan(&value); err != nil {
			t.Fatalf("snapshot %s: %v", name, err)
		}
		out[name] = value
	}
	return out
}

func sortedUUIDs(ids []uuid.UUID) []uuid.UUID {
	out := append([]uuid.UUID(nil), ids...)
	sort.Slice(out, func(i, j int) bool { return out[i].String() < out[j].String() })
	return out
}

func linkedLateHasReadinessBlocker(output StudentLeavingReadinessOutput, code StudentLeavingReadinessBlockerCode, recordID *uuid.UUID) bool {
	for _, blocker := range output.Blockers {
		if blocker.Code == code && (recordID == nil || blocker.RecordID != nil && *blocker.RecordID == *recordID) {
			return true
		}
	}
	return false
}

type linkedLateFaultManager struct {
	repository.UnitOfWorkManager
	stage   string
	failure error
}

// linkedLateClosedYearManager meniru hasil pembacaan tahun yang tertutup tanpa
// merusak skema uji melalui UPDATE langsung yang melanggar trigger rekening.
type linkedLateClosedYearManager struct {
	repository.UnitOfWorkManager
}

func (m linkedLateClosedYearManager) Begin(ctx context.Context) (repository.UnitOfWork, error) {
	uow, err := m.UnitOfWorkManager.Begin(ctx)
	if err != nil {
		return nil, err
	}
	repos := uow.Repositories()
	repos.AcademicYears = linkedLateClosedYearRepository{AcademicYearRepository: repos.AcademicYears}
	return leavingSettlementTestUOW{UnitOfWork: uow, repos: repos}, nil
}

type linkedLateClosedYearRepository struct {
	repository.AcademicYearRepository
}

func (r linkedLateClosedYearRepository) GetByIDForShare(ctx context.Context, id uuid.UUID) (domain.AcademicYear, error) {
	year, err := r.AcademicYearRepository.GetByIDForShare(ctx, id)
	if err == nil {
		year.Status = domain.AcademicYearClosed
	}
	return year, err
}

func (m linkedLateFaultManager) Begin(ctx context.Context) (repository.UnitOfWork, error) {
	uow, err := m.UnitOfWorkManager.Begin(ctx)
	if err != nil {
		return nil, err
	}
	repos := uow.Repositories()
	switch m.stage {
	case "insert after write":
		repos.LeavingTransactionRequests = linkedLateInsertFailureRepository{LeavingTransactionRequestRepository: repos.LeavingTransactionRequests, failure: m.failure}
	case "audit after write":
		repos.AuditLogs = linkedLateAuditFailureRepository{AuditRepository: repos.AuditLogs, failure: m.failure}
	}
	return leavingSettlementTestUOW{UnitOfWork: uow, repos: repos, commitError: func() error {
		if m.stage == "commit" {
			return m.failure
		}
		return nil
	}()}, nil
}

type linkedLateInsertFailureRepository struct {
	repository.LeavingTransactionRequestRepository
	failure error
}

func (r linkedLateInsertFailureRepository) CreateLeavingTransactionRequestForStudentLeaving(ctx context.Context, request domain.LeavingTransactionRequest, processID uuid.UUID) error {
	if err := r.LeavingTransactionRequestRepository.CreateLeavingTransactionRequestForStudentLeaving(ctx, request, processID); err != nil {
		return err
	}
	return r.failure
}

type linkedLateAuditFailureRepository struct {
	repository.AuditRepository
	failure error
}

func (r linkedLateAuditFailureRepository) Create(ctx context.Context, audit domain.AuditLog) error {
	if err := r.AuditRepository.Create(ctx, audit); err != nil {
		return err
	}
	return r.failure
}

type linkedLatePauseManager struct {
	repository.UnitOfWorkManager
	pauseAt string
	gate    *internalCancelGate
}

func (m linkedLatePauseManager) Begin(ctx context.Context) (repository.UnitOfWork, error) {
	uow, err := m.UnitOfWorkManager.Begin(ctx)
	if err != nil {
		return nil, err
	}
	repos := uow.Repositories()
	if m.pauseAt == "year" {
		repos.AcademicYears = linkedLatePauseYearRepository{AcademicYearRepository: repos.AcademicYears, gate: m.gate}
	}
	if m.pauseAt == "account" {
		repos.SavingsAccounts = linkedLatePauseAccountRepository{SavingsAccountRepository: repos.SavingsAccounts, gate: m.gate}
	}
	return leavingSettlementTestUOW{UnitOfWork: uow, repos: repos}, nil
}

type linkedLatePauseYearRepository struct {
	repository.AcademicYearRepository
	gate *internalCancelGate
}

func (r linkedLatePauseYearRepository) GetByIDForShare(ctx context.Context, id uuid.UUID) (domain.AcademicYear, error) {
	year, err := r.AcademicYearRepository.GetByIDForShare(ctx, id)
	if err == nil {
		r.gate.pause()
	}
	return year, err
}

func (r linkedLatePauseYearRepository) GetByIDForUpdate(ctx context.Context, id uuid.UUID) (domain.AcademicYear, error) {
	year, err := r.AcademicYearRepository.GetByIDForUpdate(ctx, id)
	if err == nil {
		r.gate.pause()
	}
	return year, err
}

type linkedLatePauseAccountRepository struct {
	repository.SavingsAccountRepository
	gate *internalCancelGate
}

func (r linkedLatePauseAccountRepository) GetByIDForUpdate(ctx context.Context, id uuid.UUID) (domain.SavingsAccount, error) {
	account, err := r.SavingsAccountRepository.GetByIDForUpdate(ctx, id)
	if err == nil {
		r.gate.pause()
	}
	return account, err
}

func (r linkedLatePauseAccountRepository) ListByStudentForUpdate(ctx context.Context, id uuid.UUID) ([]domain.SavingsAccount, error) {
	accounts, err := r.SavingsAccountRepository.ListByStudentForUpdate(ctx, id)
	if err == nil {
		r.gate.pause()
	}
	return accounts, err
}

type linkedLateLockProbe struct {
	mu    sync.Mutex
	calls []string
}

func (p *linkedLateLockProbe) record(value string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls = append(p.calls, value)
}

func (p *linkedLateLockProbe) snapshot() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.calls...)
}

type linkedLateLockProbeManager struct {
	repository.UnitOfWorkManager
	probe *linkedLateLockProbe
}

func (m linkedLateLockProbeManager) Begin(ctx context.Context) (repository.UnitOfWork, error) {
	uow, err := m.UnitOfWorkManager.Begin(ctx)
	if err != nil {
		return nil, err
	}
	repos := uow.Repositories()
	repos.AcademicYears = linkedLateLockProbeYearRepository{AcademicYearRepository: repos.AcademicYears, probe: m.probe}
	repos.SavingsAccounts = linkedLateLockProbeAccountRepository{SavingsAccountRepository: repos.SavingsAccounts, probe: m.probe}
	repos.Students = linkedLateLockProbeStudentRepository{StudentRepository: repos.Students, probe: m.probe}
	repos.StudentLeavingRequests = linkedLateLockProbeRequestRepository{StudentLeavingRequestRepository: repos.StudentLeavingRequests, probe: m.probe}
	return leavingSettlementTestUOW{UnitOfWork: uow, repos: repos}, nil
}

type linkedLateLockProbeYearRepository struct {
	repository.AcademicYearRepository
	probe *linkedLateLockProbe
}

func (r linkedLateLockProbeYearRepository) GetByIDForShare(ctx context.Context, id uuid.UUID) (domain.AcademicYear, error) {
	year, err := r.AcademicYearRepository.GetByIDForShare(ctx, id)
	if err == nil {
		r.probe.record("year:" + id.String())
	}
	return year, err
}

type linkedLateLockProbeAccountRepository struct {
	repository.SavingsAccountRepository
	probe *linkedLateLockProbe
}

func (r linkedLateLockProbeAccountRepository) GetByIDForUpdate(ctx context.Context, id uuid.UUID) (domain.SavingsAccount, error) {
	account, err := r.SavingsAccountRepository.GetByIDForUpdate(ctx, id)
	if err == nil {
		r.probe.record("account")
	}
	return account, err
}

type linkedLateLockProbeStudentRepository struct {
	repository.StudentRepository
	probe *linkedLateLockProbe
}

func (r linkedLateLockProbeStudentRepository) GetByIDForUpdate(ctx context.Context, id uuid.UUID) (domain.Student, error) {
	student, err := r.StudentRepository.GetByIDForUpdate(ctx, id)
	if err == nil {
		r.probe.record("student")
	}
	return student, err
}

type linkedLateLockProbeRequestRepository struct {
	repository.StudentLeavingRequestRepository
	probe *linkedLateLockProbe
}

func (r linkedLateLockProbeRequestRepository) GetByIDForUpdate(ctx context.Context, id uuid.UUID) (domain.StudentLeavingRequest, error) {
	request, err := r.StudentLeavingRequestRepository.GetByIDForUpdate(ctx, id)
	if err == nil {
		r.probe.record("process")
	}
	return request, err
}

func TestCreateLateRequestForStudentLeavingConcurrency(t *testing.T) {
	for _, operation := range []string{"cancel", "finalize", "settlement", "duplicate", "year closing", "deposit"} {
		for _, createFirst := range []bool{true, false} {
			name := "operasi lebih dahulu"
			if createFirst {
				name = "create lebih dahulu"
			}
			t.Run(operation+"/"+name, func(t *testing.T) {
				f := newInternalLeavingCancelFixture(t)
				cleanupLinkedLateRequests(t, f.db, f.account.ID, f.request.ID, f.waliActor.UserID)
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
				defer cancel()
				if operation == "finalize" && !createFirst {
					settlement, err := f.settlementService.settleForStudentLeavingRequest(ctx, settleForStudentLeavingRequestInput{Actor: f.waliActor, StudentLeavingRequestID: f.request.ID, SavingsAccountID: f.account.ID, ExpectedBalance: 50_000})
					if err != nil {
						t.Fatal(err)
					}
					if _, err := f.settlementService.ConfirmHandover(ctx, ConfirmSettlementHandoverInput{Actor: f.waliActor, SettlementID: settlement.Settlement.ID, Confirm: true, ReceiptReference: stringPointer("RACE-" + uuid.NewString())}); err != nil {
						t.Fatal(err)
					}
				}

				gateAt := "account"
				if operation == "finalize" || operation == "year closing" {
					gateAt = "year"
				}
				gate := newInternalCancelGate()
				defer gate.release()
				run := func(which string, manager repository.UnitOfWorkManager) error {
					creator := linkedLateCreator(f, manager)
					switch which {
					case "create":
						_, err := creator.createForStudentLeavingRequest(ctx, linkedLateInput(f, f.waliActor, f.year.StartDate, "RACE-EVIDENCE"))
						return err
					case "cancel":
						_, err := (&studentLeavingRequestService{deps: Dependencies{Repositories: f.repos, UOW: manager}, now: time.Now}).Cancel(ctx, CancelStudentLeavingRequestInput{Actor: f.adminActor, StudentLeavingRequestID: f.request.ID, Reason: "Uji create bersamaan cancel"})
						return err
					case "finalize":
						_, err := (&studentLeavingRequestService{deps: Dependencies{Repositories: f.repos, UOW: manager}, now: time.Now}).Finalize(ctx, FinalizeStudentLeavingRequestInput{Actor: f.adminActor, StudentLeavingRequestID: f.request.ID})
						return err
					case "settlement":
						_, err := (&settlementService{deps: Dependencies{Repositories: f.repos, UOW: manager}}).settleForStudentLeavingRequest(ctx, settleForStudentLeavingRequestInput{Actor: f.waliActor, StudentLeavingRequestID: f.request.ID, SavingsAccountID: f.account.ID, ExpectedBalance: 50_000})
						return err
					case "year closing":
						_, err := (&academicYearService{deps: Dependencies{Repositories: f.repos, UOW: manager}}).StartClosing(ctx, StartClosingAcademicYearInput{Actor: f.adminActor, AcademicYearID: f.year.ID})
						return err
					case "deposit":
						_, err := (&transactionService{deps: Dependencies{Repositories: f.repos, UOW: manager}}).Deposit(ctx, DepositInput{Actor: f.waliActor, SavingsAccountID: f.account.ID, Amount: 1_000, TransactionDate: f.year.StartDate})
						return err
					case "duplicate":
						_, err := creator.createForStudentLeavingRequest(ctx, linkedLateInput(f, f.waliActor, f.year.StartDate, "RACE-EVIDENCE"))
						return err
					}
					return fmt.Errorf("unknown operation %s", which)
				}

				first, second := operation, "create"
				if createFirst {
					first, second = "create", operation
				}
				firstResult := make(chan error, 1)
				secondResult := make(chan error, 1)
				manager := linkedLatePauseManager{UnitOfWorkManager: f.manager, pauseAt: gateAt, gate: gate}
				go func() { firstResult <- run(first, manager) }()
				waitInternalCancelGate(t, gate)
				blockerPID := internalCancelIdleTransactionPID(t, f.db)
				go func() { secondResult <- run(second, f.manager) }()
				waitInternalCancelBlocker(t, f.db, blockerPID)
				gate.release()
				results := map[string]error{}
				for _, result := range []struct {
					name string
					ch   <-chan error
				}{{first, firstResult}, {second, secondResult}} {
					select {
					case err := <-result.ch:
						results[result.name] = err
					case <-ctx.Done():
						t.Fatal(ctx.Err())
					}
				}
				if err := results[first]; err != nil {
					t.Fatalf("first operation %s: %v", first, err)
				}
				if err := results[second]; err != nil && !errors.Is(err, domain.ErrInvalidTransition) && !errors.Is(err, repository.ErrConflict) && !errors.Is(err, ErrAcademicYearHasPendingTransfers) {
					t.Fatalf("second operation %s unexpected error: %v", second, err)
				}
				assertLateCreateRaceState(t, f, operation, createFirst, results)
			})
		}
	}
}

func assertLateCreateRaceState(t *testing.T, f internalLeavingCancelFixture, operation string, createFirst bool, results map[string]error) {
	t.Helper()
	var requestCount, linkedCount, auditCount, transactionCount int
	if err := f.db.QueryRow(`SELECT count(*), count(*) FILTER (WHERE student_leaving_request_id=$2) FROM leaving_transaction_requests WHERE savings_account_id=$1`, f.account.ID, f.request.ID).Scan(&requestCount, &linkedCount); err != nil {
		t.Fatal(err)
	}
	if err := f.db.QueryRow(`SELECT count(*) FROM audit_logs WHERE action='REQUEST_FOR_STUDENT_LEAVING' AND entity_id IN (SELECT id FROM leaving_transaction_requests WHERE savings_account_id=$1)`, f.account.ID).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if err := f.db.QueryRow(`SELECT count(*) FROM transactions WHERE savings_account_id=$1`, f.account.ID).Scan(&transactionCount); err != nil {
		t.Fatal(err)
	}
	switch operation {
	case "cancel":
		if createFirst && (requestCount != 1 || results["cancel"] != nil) || !createFirst && (requestCount != 0 || results["create"] == nil) {
			t.Fatalf("cancel race state requests=%d results=%v", requestCount, results)
		}
	case "finalize":
		if createFirst && (requestCount != 1 || results["finalize"] == nil) || !createFirst && (requestCount != 0 || results["create"] == nil) {
			t.Fatalf("finalize race state requests=%d results=%v", requestCount, results)
		}
	case "settlement":
		if createFirst && (requestCount != 1 || results["settlement"] == nil) || !createFirst && (requestCount != 0 || results["create"] == nil) {
			t.Fatalf("settlement race state requests=%d results=%v", requestCount, results)
		}
	case "duplicate":
		if requestCount != 1 || linkedCount != 1 || auditCount != 1 ||
			(results["create"] == nil) == (results["duplicate"] == nil) {
			t.Fatalf("duplicate race rows=%d linked=%d audits=%d results=%v", requestCount, linkedCount, auditCount, results)
		}
	case "year closing":
		if requestCount != 1 || linkedCount != 1 {
			t.Fatalf("year closing request rows=%d linked=%d", requestCount, linkedCount)
		}
		var status string
		if err := f.db.QueryRow(`SELECT status FROM academic_years WHERE id=$1`, f.year.ID).Scan(&status); err != nil || status != "CLOSING" {
			t.Fatalf("year status=%s err=%v", status, err)
		}
	case "deposit":
		if requestCount != 1 || linkedCount != 1 || transactionCount != 2 {
			t.Fatalf("deposit race requests=%d linked=%d transactions=%d", requestCount, linkedCount, transactionCount)
		}
	}
}
