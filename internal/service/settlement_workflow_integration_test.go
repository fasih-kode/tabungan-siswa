package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/repository/postgres"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

func TestYearEndSettlementHandoverReopenCorrectionAndSuccessorGuard(t *testing.T) {
	db := openServiceScopeTestDatabase(t)
	ctx := context.Background()
	cleanupServiceScopeFixtures(t, db)
	repos := postgres.NewRepositorySet(db)
	uow := postgres.NewUnitOfWorkManager(db)
	wali := newServiceScopeUser(t, domain.RoleWaliKelas)
	admin := newServiceScopeUser(t, domain.RoleAdmin)
	if err := repos.Users.Create(ctx, wali); err != nil {
		t.Fatal(err)
	}
	if err := repos.Users.Create(ctx, admin); err != nil {
		t.Fatal(err)
	}
	year := newServiceScopeAcademicYear(t, "Settlement workflow "+uuid.NewString())
	if err := repos.AcademicYears.Create(ctx, year); err != nil {
		t.Fatal(err)
	}
	student := newServiceScopeStudent(t, "Settlement student "+uuid.NewString())
	if err := repos.Students.Create(ctx, student); err != nil {
		t.Fatal(err)
	}
	class := newServiceScopeClass(t, "Settlement class "+uuid.NewString(), 7)
	class.AcademicYearID = year.ID
	class.EffectiveDate = year.StartDate
	if err := repos.Classes.Create(ctx, class); err != nil {
		t.Fatal(err)
	}
	assignment := newServiceScopeAssignment(t, wali.ID, year.ID, class.ID)
	if err := repos.TeacherClassAssignments.Create(ctx, assignment); err != nil {
		t.Fatal(err)
	}
	history := newServiceScopeStudentHistory(t, student.ID, year.ID, class.ID)
	if err := repos.StudentClassHistories.Create(ctx, history); err != nil {
		t.Fatal(err)
	}
	leavingStudent := newServiceScopeStudent(t, "Leaving settlement student "+uuid.NewString())
	if err := repos.Students.Create(ctx, leavingStudent); err != nil {
		t.Fatal(err)
	}
	leavingHistory := newServiceScopeStudentHistory(t, leavingStudent.ID, year.ID, class.ID)
	if err := repos.StudentClassHistories.Create(ctx, leavingHistory); err != nil {
		t.Fatal(err)
	}
	account, err := domain.NewSavingsAccount(student.ID, year.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := repos.SavingsAccounts.Create(ctx, account); err != nil {
		t.Fatal(err)
	}
	leavingAccount, err := domain.NewSavingsAccount(leavingStudent.ID, year.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := repos.SavingsAccounts.Create(ctx, leavingAccount); err != nil {
		t.Fatal(err)
	}
	settlementSvc, err := service.NewSettlementService(service.Dependencies{Repositories: repos, UOW: uow})
	if err != nil {
		t.Fatal(err)
	}
	yearSvc, err := service.NewAcademicYearService(service.Dependencies{Repositories: repos, UOW: uow})
	if err != nil {
		t.Fatal(err)
	}
	txSvc, err := service.NewTransactionService(service.Dependencies{Repositories: repos, UOW: uow})
	if err != nil {
		t.Fatal(err)
	}
	waliActor := service.Actor{UserID: wali.ID, Role: domain.RoleWaliKelas}
	adminActor := service.Actor{UserID: admin.ID, Role: domain.RoleAdmin}
	deposit, err := txSvc.Deposit(ctx, service.DepositInput{Actor: waliActor, SavingsAccountID: account.ID, Amount: 100, TransactionDate: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	leavingDeposit, err := txSvc.Deposit(ctx, service.DepositInput{Actor: waliActor, SavingsAccountID: leavingAccount.ID, Amount: 50, TransactionDate: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	leavingStudent.Status = domain.StudentLeft
	if err := repos.Students.Update(ctx, leavingStudent); err != nil {
		t.Fatal(err)
	}
	var successorYearID uuid.UUID
	t.Cleanup(func() {
		cleanup := func(q string, args ...any) {
			if _, e := db.ExecContext(ctx, q, args...); e != nil {
				t.Errorf("cleanup %s: %v", q, e)
			}
		}
		cleanup(`DELETE FROM audit_logs WHERE actor_user_id IN ($1,$2) OR entity_id IN ($3,$4,$5,$6,$7,$8,$9,$10) OR entity_id IN (SELECT id FROM transactions WHERE savings_account_id IN ($11,$12)) OR entity_id IN (SELECT id FROM savings_settlements WHERE savings_account_id IN ($11,$12)) OR entity_id IN (SELECT id FROM settlement_reopen_requests WHERE academic_year_id=$13)`, wali.ID, admin.ID, deposit.Transaction.ID, leavingDeposit.Transaction.ID, account.ID, leavingAccount.ID, year.ID, class.ID, student.ID, leavingStudent.ID, account.ID, leavingAccount.ID, year.ID)
		cleanup(`DELETE FROM settlement_cash_movements WHERE savings_settlement_id IN (SELECT id FROM savings_settlements WHERE savings_account_id IN ($1,$2))`, account.ID, leavingAccount.ID)
		cleanup(`DELETE FROM settlement_reopen_requests WHERE academic_year_id=$1`, year.ID)
		cleanup(`DELETE FROM transactions WHERE savings_account_id IN ($1,$2)`, account.ID, leavingAccount.ID)
		cleanup(`DELETE FROM savings_settlements WHERE savings_account_id IN ($1,$2)`, account.ID, leavingAccount.ID)
		cleanup(`DELETE FROM savings_accounts WHERE id IN ($1,$2)`, account.ID, leavingAccount.ID)
		cleanup(`DELETE FROM student_class_histories WHERE id IN ($1,$2)`, history.ID, leavingHistory.ID)
		cleanup(`DELETE FROM teacher_class_assignments WHERE id=$1`, assignment.ID)
		cleanup(`DELETE FROM classes WHERE id=$1`, class.ID)
		cleanup(`DELETE FROM students WHERE id IN ($1,$2)`, student.ID, leavingStudent.ID)
		if successorYearID != uuid.Nil {
			cleanup(`DELETE FROM academic_years WHERE id=$1`, successorYearID)
		}
		cleanup(`DELETE FROM academic_years WHERE id=$1`, year.ID)
		cleanup(`DELETE FROM users WHERE id IN ($1,$2)`, wali.ID, admin.ID)
	})
	leavingPreview, err := settlementSvc.PreviewStudentLeaving(ctx, service.PreviewStudentLeavingInput{Actor: waliActor, SavingsAccountID: leavingAccount.ID})
	if err != nil || leavingPreview.Balance != 50 {
		t.Fatalf("student-leaving preview=%v err=%v, want 50", leavingPreview.Balance, err)
	}
	leavingSettlement, err := settlementSvc.SettleStudentLeaving(ctx, service.SettleStudentLeavingInput{Actor: waliActor, SavingsAccountID: leavingAccount.ID, ExpectedBalance: leavingPreview.Balance})
	if err != nil || leavingSettlement.Settlement.Amount != 50 {
		t.Fatalf("student-leaving settlement=%+v err=%v, want positive settlement 50", leavingSettlement.Settlement, err)
	}
	if _, err := settlementSvc.ConfirmHandover(ctx, service.ConfirmSettlementHandoverInput{Actor: waliActor, SettlementID: leavingSettlement.Settlement.ID, Confirm: true}); err != nil {
		t.Fatalf("confirm student-leaving handover while year is OPEN: %v", err)
	}
	if _, err := yearSvc.StartClosing(ctx, service.StartClosingAcademicYearInput{Actor: adminActor, AcademicYearID: year.ID}); err != nil {
		t.Fatal(err)
	}
	preview, err := settlementSvc.PreviewYearEnd(ctx, service.PreviewYearEndInput{Actor: waliActor, SavingsAccountID: account.ID})
	if err != nil || preview.Balance != 100 {
		t.Fatalf("PreviewYearEnd balance=%v err=%v", preview.Balance, err)
	}
	type settleResult struct {
		output service.SettleYearEndOutput
		err    error
	}
	settleStart := make(chan struct{})
	settleResults := make(chan settleResult, 3)
	for i := 0; i < 2; i++ {
		go func() {
			<-settleStart
			output, e := settlementSvc.SettleYearEnd(ctx, service.SettleYearEndInput{Actor: waliActor, SavingsAccountID: account.ID, ExpectedBalance: 100})
			settleResults <- settleResult{output: output, err: e}
		}()
	}
	withdrawalResult := make(chan error, 1)
	go func() {
		<-settleStart
		_, e := txSvc.Withdrawal(ctx, service.WithdrawalInput{Actor: waliActor, SavingsAccountID: account.ID, Amount: 10, TransactionDate: time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)})
		withdrawalResult <- e
	}()
	close(settleStart)
	var initial service.SettleYearEndOutput
	successes := 0
	for i := 0; i < 2; i++ {
		select {
		case result := <-settleResults:
			if result.err == nil {
				successes++
				initial = result.output
			} else if !errors.Is(result.err, domain.ErrInvalidTransition) && !errors.Is(result.err, repository.ErrConflict) {
				t.Fatalf("concurrent settlement/withdrawal: %v", result.err)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("concurrent settlement/withdrawal deadlocked")
		}
	}
	withdrawalErr := <-withdrawalResult
	if withdrawalErr != nil && !errors.Is(withdrawalErr, domain.ErrInvalidTransition) {
		t.Fatalf("concurrent withdrawal error=%v", withdrawalErr)
	}
	if successes > 1 || (successes == 1 && withdrawalErr == nil) || (successes == 0 && withdrawalErr != nil) {
		t.Fatalf("inconsistent settlement/withdrawal outcome: successes=%d withdrawal=%v", successes, withdrawalErr)
	}
	if successes == 0 {
		fresh, e := settlementSvc.PreviewYearEnd(ctx, service.PreviewYearEndInput{Actor: waliActor, SavingsAccountID: account.ID})
		if e != nil || fresh.Balance != 90 {
			t.Fatalf("fresh preview after winning withdrawal balance=%v err=%v, want 90", fresh.Balance, e)
		}
		initial, e = settlementSvc.SettleYearEnd(ctx, service.SettleYearEndInput{Actor: waliActor, SavingsAccountID: account.ID, ExpectedBalance: fresh.Balance})
		if e != nil {
			t.Fatalf("settle after refreshed preview: %v", e)
		}
	}
	if initial.Settlement == nil {
		t.Fatal("concurrent race produced no settlement")
	}
	if _, err := yearSvc.Close(ctx, service.CloseAcademicYearInput{Actor: adminActor, AcademicYearID: year.ID}); !errors.Is(err, service.ErrAcademicYearHasUnconfirmedHandover) {
		t.Fatalf("Close before handover=%v", err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	go func() {
		<-start
		_, e := settlementSvc.ConfirmHandover(ctx, service.ConfirmSettlementHandoverInput{Actor: waliActor, SettlementID: initial.Settlement.ID, Confirm: true})
		results <- e
	}()
	go func() {
		<-start
		_, e := yearSvc.Close(ctx, service.CloseAcademicYearInput{Actor: adminActor, AcademicYearID: year.ID})
		results <- e
	}()
	close(start)
	var closeRaceErr error
	for i := 0; i < 2; i++ {
		select {
		case e := <-results:
			if e != nil && !errors.Is(e, service.ErrAcademicYearHasUnconfirmedHandover) {
				t.Fatalf("settlement/close interleaving: %v", e)
			}
			if errors.Is(e, service.ErrAcademicYearHasUnconfirmedHandover) {
				closeRaceErr = e
			}
		case <-time.After(10 * time.Second):
			t.Fatal("settlement and close deadlocked")
		}
	}
	if closeRaceErr != nil {
		if _, err := yearSvc.Close(ctx, service.CloseAcademicYearInput{Actor: adminActor, AcademicYearID: year.ID}); err != nil {
			t.Fatal(err)
		}
	}
	type raceResult struct {
		name      string
		err       error
		yearID    uuid.UUID
		requestID uuid.UUID
	}
	race := make(chan raceResult, 2)
	raceYear := newServiceScopeAcademicYearForStartYear(t, "Racing successor "+uuid.NewString(), 2027)
	go func() {
		o, e := settlementSvc.RequestReopen(ctx, service.RequestSettlementReopenInput{Actor: waliActor, SettlementID: initial.Settlement.ID, Reason: "Perlu koreksi transaksi yang salah dicatat"})
		r := raceResult{name: "request", err: e}
		if e == nil {
			r.requestID = o.Request.ID
		}
		race <- r
	}()
	go func() {
		o, e := yearSvc.Create(ctx, service.CreateAcademicYearInput{Actor: adminActor, Name: raceYear.Name, StartDate: raceYear.StartDate, EndDate: raceYear.EndDate})
		r := raceResult{name: "create", err: e}
		if e == nil {
			r.yearID = o.AcademicYear.ID
		}
		race <- r
	}()
	var requestResult, createResult raceResult
	for i := 0; i < 2; i++ {
		select {
		case r := <-race:
			if r.name == "request" {
				requestResult = r
			} else {
				createResult = r
			}
		case <-time.After(10 * time.Second):
			t.Fatal("reopen request and successor creation deadlocked")
		}
	}
	if requestResult.err == nil && createResult.err != nil {
		requestID := requestResult.requestID
		if _, e := settlementSvc.DecideReopen(ctx, service.DecideSettlementReopenInput{Actor: service.Actor{UserID: wali.ID, Role: domain.RoleAdmin}, RequestID: requestID, Approve: true, Reason: "Self approval must be rejected"}); !errors.Is(e, repository.ErrConflict) {
			t.Fatalf("requester self-approval error=%v, want conflict", e)
		}
		if _, err := settlementSvc.DecideReopen(ctx, service.DecideSettlementReopenInput{Actor: adminActor, RequestID: requestID, Approve: true, Reason: "Disetujui setelah pemeriksaan"}); err != nil {
			t.Fatal(err)
		}
	} else if requestResult.err != nil && createResult.err == nil {
		// Successor creation won the year lock; the request is rejected and no reopen may follow.
		successorYearID = createResult.yearID
		if _, e := settlementSvc.RequestReopen(ctx, service.RequestSettlementReopenInput{Actor: waliActor, SettlementID: initial.Settlement.ID, Reason: "Coba buka kembali setelah penerus dibuat"}); e == nil {
			t.Fatal("reopen request accepted after concurrent successor creation")
		}
		return
	} else {
		t.Fatalf("invalid reopen/successor serialization request=%v create=%v", requestResult.err, createResult.err)
	}

	if _, err := settlementSvc.Reopen(ctx, service.ReopenSettlementInput{Actor: waliActor, SettlementID: initial.Settlement.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := txSvc.Edit(ctx, service.EditTransactionInput{Actor: adminActor, TransactionID: deposit.Transaction.ID, Amount: 80, TransactionDate: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), Reason: "Admin must not correct finances"}); !errors.Is(err, service.ErrForbidden) {
		t.Fatalf("Admin FASE22 correction error=%v, want forbidden", err)
	}
	if _, err := txSvc.Edit(ctx, service.EditTransactionInput{Actor: waliActor, TransactionID: deposit.Transaction.ID, Amount: 80, TransactionDate: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), Reason: "Koreksi melalui jendela yang disetujui"}); err != nil {
		t.Fatalf("approved FASE22 correction: %v", err)
	}
	replacementBalance := initial.Settlement.Amount - 20
	replacement, err := settlementSvc.SettleYearEnd(ctx, service.SettleYearEndInput{Actor: waliActor, SavingsAccountID: account.ID, ExpectedBalance: replacementBalance})
	if err != nil {
		t.Fatal(err)
	}
	if replacement.Settlement.ReplacesSettlementID == nil || *replacement.Settlement.ReplacesSettlementID != initial.Settlement.ID {
		t.Fatalf("replacement link=%v", replacement.Settlement.ReplacesSettlementID)
	}
	if _, err := settlementSvc.ConfirmHandover(ctx, service.ConfirmSettlementHandoverInput{Actor: waliActor, SettlementID: replacement.Settlement.ID, Confirm: true}); err != nil {
		t.Fatal(err)
	}
	moves, err := repos.Settlements.ListCashMovements(ctx, replacement.Settlement.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(moves) != 2 || moves[1].Kind != domain.SettlementCashReturn || moves[1].Amount != 20 {
		t.Fatalf("cash reconciliation=%#v, want return 20", moves)
	}
	if _, err := yearSvc.Close(ctx, service.CloseAcademicYearInput{Actor: adminActor, AcademicYearID: year.ID}); err != nil {
		t.Fatalf("close reconciled year: %v", err)
	}
	rejected, err := settlementSvc.RequestReopen(ctx, service.RequestSettlementReopenInput{Actor: waliActor, SettlementID: replacement.Settlement.ID, Reason: "Permintaan untuk menguji keputusan penolakan"})
	if err != nil {
		t.Fatalf("request reopen for rejection path: %v", err)
	}
	rejectedDecision, err := settlementSvc.DecideReopen(ctx, service.DecideSettlementReopenInput{Actor: adminActor, RequestID: rejected.Request.ID, Approve: false, Reason: "Bukti koreksi belum memadai"})
	if err != nil {
		t.Fatalf("reject reopen request: %v", err)
	}
	if rejectedDecision.Request.Status != domain.SettlementReopenRejected || rejectedDecision.Request.DecisionReason == nil || *rejectedDecision.Request.DecisionReason != "Bukti koreksi belum memadai" {
		t.Fatalf("reopen decision = %+v, want recorded rejection reason", rejectedDecision.Request)
	}
	successor := newServiceScopeAcademicYearForStartYear(t, "Successor "+uuid.NewString(), 2027)
	created, err := yearSvc.Create(ctx, service.CreateAcademicYearInput{Actor: adminActor, Name: successor.Name, StartDate: successor.StartDate, EndDate: successor.EndDate})
	if err != nil {
		t.Fatalf("create successor: %v", err)
	}
	successorYearID = created.AcademicYear.ID
	if _, err := settlementSvc.RequestReopen(ctx, service.RequestSettlementReopenInput{Actor: waliActor, SettlementID: replacement.Settlement.ID, Reason: "Coba buka kembali setelah penerus dibuat"}); err == nil {
		t.Fatal("reopen request after successor year should be rejected")
	}
}
