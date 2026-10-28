package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
)

func TestStudentLeavingFinalizeVsInternalSettlement(t *testing.T) {
	for _, settleFirst := range []bool{true, false} {
		t.Run(map[bool]string{true: "settlement lebih dahulu", false: "Finalize lebih dahulu"}[settleFirst], func(t *testing.T) {
			f := newInternalLeavingCancelFixture(t)
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			gate := newInternalCancelGate()
			defer gate.release()
			mode := "cancel-finance"
			if settleFirst {
				mode = "settlement-account"
			}
			manager := internalCancelBarrierManager{UnitOfWorkManager: f.manager, mode: mode, gate: gate}
			settleResult, finalizeResult := make(chan error, 1), make(chan error, 1)
			settle := func(paused bool) {
				deps := Dependencies{Repositories: f.repos, UOW: f.manager}
				if paused {
					deps.UOW = manager
				}
				svc := &settlementService{deps: deps}
				_, err := svc.settleForStudentLeavingRequest(ctx, settleForStudentLeavingRequestInput{Actor: f.waliActor, StudentLeavingRequestID: f.request.ID, SavingsAccountID: f.account.ID, ExpectedBalance: 50000})
				settleResult <- err
			}
			finalize := func(paused bool) {
				deps := Dependencies{Repositories: f.repos, UOW: f.manager}
				if paused {
					deps.UOW = manager
				}
				svc, err := NewStudentLeavingRequestService(deps)
				if err == nil {
					_, err = svc.Finalize(ctx, FinalizeStudentLeavingRequestInput{Actor: f.adminActor, StudentLeavingRequestID: f.request.ID})
				}
				finalizeResult <- err
			}
			if settleFirst {
				go settle(true)
			} else {
				go finalize(true)
			}
			waitInternalCancelGate(t, gate)
			pid := internalCancelIdleTransactionPID(t, f.db)
			if settleFirst {
				go finalize(false)
			} else {
				go settle(false)
			}
			waitInternalCancelBlocker(t, f.db, pid)
			gate.release()
			select {
			case err := <-settleResult:
				if err != nil {
					t.Fatal(err)
				}
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			select {
			case err := <-finalizeResult:
				if !errors.Is(err, domain.ErrInvalidTransition) {
					t.Fatalf("Finalize error=%v want blocker rekening/handover", err)
				}
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			student, err := f.repos.Students.GetByID(ctx, f.student.ID)
			if err != nil || !student.IsActive() {
				t.Fatalf("student=%+v %v", student, err)
			}
			request, err := f.repos.StudentLeavingRequests.GetByID(ctx, f.request.ID)
			if err != nil || request.Status != domain.StudentLeavingInProgress {
				t.Fatalf("request=%+v %v", request, err)
			}
			account, err := f.repos.SavingsAccounts.GetByID(ctx, f.account.ID)
			if err != nil || !account.IsSettled() {
				t.Fatalf("account=%+v %v", account, err)
			}
			settlements, err := f.repos.Settlements.ListByStudentLeavingRequest(ctx, f.request.ID)
			if err != nil || len(settlements) != 1 || settlements[0].Amount != 50000 {
				t.Fatalf("settlements=%+v %v", settlements, err)
			}
			transactions, err := f.repos.Transactions.ListAllBySavingsAccount(ctx, f.account.ID)
			if err != nil {
				t.Fatal(err)
			}
			balance, err := domain.CalculateBalance(transactions)
			if err != nil || balance.Amount != 50000 {
				t.Fatalf("ledger=%+v %v", balance, err)
			}
			movements, err := f.repos.Settlements.ListCashMovements(ctx, settlements[0].ID)
			if err != nil || len(movements) != 0 {
				t.Fatalf("movements=%+v %v", movements, err)
			}
			var audits int
			if err := f.db.QueryRowContext(ctx, `SELECT count(*) FROM audit_logs WHERE action='STUDENT_LEAVING_FINALIZED' AND entity_id=$1`, f.student.ID).Scan(&audits); err != nil || audits != 0 {
				t.Fatalf("audit=%d %v", audits, err)
			}
		})
	}
}
