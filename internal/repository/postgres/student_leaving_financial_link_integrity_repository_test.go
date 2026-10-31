package postgres_test

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/repository/postgres"
	"github.com/google/uuid"
)

func TestStudentLeavingFinancialLinkIntegrityV16(t *testing.T) {
	db := openTestDatabase(t)
	requireSchemaVersion16(t, db)
	ctx := context.Background()

	historicalYear := newTestAcademicYearWithDates(
		t,
		"MQ08-Integrity-Historical-"+uuid.NewString(),
		time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
		domain.AcademicYearClosed,
	)
	if err := postgres.NewAcademicYearRepository(db).Create(ctx, historicalYear); err != nil {
		t.Fatalf("create historical academic year fixture: %v", err)
	}
	t.Cleanup(func() { deleteTestAcademicYear(t, db, historicalYear.ID) })

	first := newSettlementFixture(t, db)
	secondAccount, secondUser := newSettlementAccountFixture(t, db, first.year.ID)
	thirdAccount, thirdUser := newSettlementAccountFixture(t, db, first.year.ID)
	secondStudent, err := postgres.NewStudentRepository(db).GetByID(ctx, secondAccount.StudentID)
	if err != nil {
		t.Fatalf("load second fixture student: %v", err)
	}
	thirdStudent, err := postgres.NewStudentRepository(db).GetByID(ctx, thirdAccount.StudentID)
	if err != nil {
		t.Fatalf("load third fixture student: %v", err)
	}

	leavingRepo := postgres.NewStudentLeavingRequestRepository(db)
	firstLeaving := newFinancialLinksLeavingRequest(t, leavingRepo, first.student.ID, first.year.ID, first.user.ID)
	secondLeaving := newFinancialLinksLeavingRequest(t, leavingRepo, secondStudent.ID, first.year.ID, secondUser.ID)
	emptyLeaving := newFinancialLinksLeavingRequest(t, leavingRepo, thirdStudent.ID, first.year.ID, thirdUser.ID)

	var settlementIDs, requestIDs []uuid.UUID
	t.Cleanup(func() {
		for _, id := range requestIDs {
			if _, err := db.ExecContext(ctx, `DELETE FROM leaving_transaction_requests WHERE id = $1`, id); err != nil {
				t.Errorf("delete late request fixture %s: %v", id, err)
			}
		}
		for i := len(settlementIDs) - 1; i >= 0; i-- {
			if _, err := db.ExecContext(ctx, `DELETE FROM savings_settlements WHERE id = $1`, settlementIDs[i]); err != nil {
				t.Errorf("delete settlement fixture %s: %v", settlementIDs[i], err)
			}
		}
		for _, id := range []uuid.UUID{firstLeaving.ID, secondLeaving.ID, emptyLeaving.ID} {
			if _, err := db.ExecContext(ctx, `DELETE FROM student_leaving_requests WHERE id = $1`, id); err != nil {
				t.Errorf("delete leaving process fixture %s: %v", id, err)
			}
		}
	})

	settlementRepo := postgres.NewSettlementRepository(db)
	validSettlement := newFinancialLinksSettlement(t, first, domain.SettlementStudentLeaving, 1_000, time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC), nil, false)
	foreignSettlement := newFinancialLinksSettlement(
		t,
		settlementFixture{user: secondUser, account: secondAccount},
		domain.SettlementStudentLeaving,
		2_000,
		time.Date(2026, 9, 2, 9, 0, 0, 0, time.UTC),
		nil,
		true,
	)
	wrongTypeSettlement := newFinancialLinksSettlement(t, first, domain.SettlementRegularYearEnd, 3_000, time.Date(2026, 9, 3, 9, 0, 0, 0, time.UTC), nil, true)
	secondProcessSettlement := newFinancialLinksSettlement(
		t,
		settlementFixture{user: secondUser, account: secondAccount},
		domain.SettlementStudentLeaving,
		4_000,
		time.Date(2026, 9, 4, 9, 0, 0, 0, time.UTC),
		nil,
		false,
	)
	historicalSettlement := newFinancialLinksSettlement(t, first, domain.SettlementStudentLeaving, 5_000, time.Date(2026, 9, 5, 9, 0, 0, 0, time.UTC), nil, true)
	for _, settlement := range []domain.SavingsSettlement{validSettlement, foreignSettlement, wrongTypeSettlement, secondProcessSettlement, historicalSettlement} {
		if err := settlementRepo.Create(ctx, settlement); err != nil {
			t.Fatalf("create settlement fixture: %v", err)
		}
		settlementIDs = append(settlementIDs, settlement.ID)
	}
	linkFinancialRow(t, db, "savings_settlements", validSettlement.ID, firstLeaving.ID)
	linkFinancialRow(t, db, "savings_settlements", foreignSettlement.ID, firstLeaving.ID)
	linkFinancialRow(t, db, "savings_settlements", wrongTypeSettlement.ID, firstLeaving.ID)
	linkFinancialRow(t, db, "savings_settlements", secondProcessSettlement.ID, secondLeaving.ID)

	lateRepo := postgres.NewLeavingTransactionRequestRepository(db)
	validLateRequest := newFinancialLinksLateRequest(t, first, time.Date(2026, 9, 6, 10, 0, 0, 0, time.UTC), "integrity-valid")
	foreignLateRequest := newFinancialLinksLateRequest(
		t,
		settlementFixture{user: secondUser, year: first.year, account: secondAccount},
		time.Date(2026, 9, 7, 10, 0, 0, 0, time.UTC),
		"integrity-foreign",
	)
	wrongYearLateRequest := newFinancialLinksLateRequest(t, first, time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC), "integrity-year")
	secondProcessLateRequest := newFinancialLinksLateRequest(
		t,
		settlementFixture{user: secondUser, year: first.year, account: secondAccount},
		time.Date(2026, 9, 9, 10, 0, 0, 0, time.UTC),
		"integrity-process-2",
	)
	historicalLateRequest := newFinancialLinksLateRequest(t, first, time.Date(2026, 9, 10, 10, 0, 0, 0, time.UTC), "integrity-null")
	for _, request := range []domain.LeavingTransactionRequest{validLateRequest, foreignLateRequest, wrongYearLateRequest, secondProcessLateRequest, historicalLateRequest} {
		if err := lateRepo.CreateLeavingTransactionRequest(ctx, request); err != nil {
			t.Fatalf("create late request fixture: %v", err)
		}
		requestIDs = append(requestIDs, request.ID)
	}
	linkFinancialRow(t, db, "leaving_transaction_requests", validLateRequest.ID, firstLeaving.ID)
	linkFinancialRow(t, db, "leaving_transaction_requests", foreignLateRequest.ID, firstLeaving.ID)
	linkFinancialRow(t, db, "leaving_transaction_requests", wrongYearLateRequest.ID, firstLeaving.ID)
	linkFinancialRow(t, db, "leaving_transaction_requests", secondProcessLateRequest.ID, secondLeaving.ID)
	if _, err := db.ExecContext(ctx, `UPDATE leaving_transaction_requests SET academic_year_id = $1 WHERE id = $2`, historicalYear.ID, wrongYearLateRequest.ID); err != nil {
		t.Fatalf("make year-mismatch fixture: %v", err)
	}

	before := financialLinksSnapshot(t, db)
	checker := postgres.NewStudentLeavingRequestRepository(db)
	issues, err := checker.ListFinancialLinkIssues(ctx, firstLeaving.ID)
	if err != nil {
		t.Fatalf("ListFinancialLinkIssues() error: %v", err)
	}
	want := []repository.StudentLeavingFinancialLinkIssue{
		{Type: repository.StudentLeavingSettlementStudentMismatch, RecordID: foreignSettlement.ID},
		{Type: repository.StudentLeavingSettlementTypeMismatch, RecordID: wrongTypeSettlement.ID},
		{Type: repository.StudentLeavingLateRequestStudentMismatch, RecordID: foreignLateRequest.ID},
		{Type: repository.StudentLeavingLateRequestYearMismatch, RecordID: wrongYearLateRequest.ID},
	}
	sort.Slice(want, func(i, j int) bool {
		if want[i].Type == want[j].Type {
			return want[i].RecordID.String() < want[j].RecordID.String()
		}
		return want[i].Type < want[j].Type
	})
	if !reflect.DeepEqual(issues, want) {
		t.Fatalf("integrity issues = %#v, want deterministic violations %#v", issues, want)
	}

	secondIssues, err := checker.ListFinancialLinkIssues(ctx, secondLeaving.ID)
	if err != nil || secondIssues == nil || len(secondIssues) != 0 {
		t.Fatalf("second process issues = %#v, %v; want non-nil empty result isolated from first process", secondIssues, err)
	}
	emptyIssues, err := checker.ListFinancialLinkIssues(ctx, emptyLeaving.ID)
	if err != nil || emptyIssues == nil || len(emptyIssues) != 0 {
		t.Fatalf("process without financial links issues = %#v, %v; want non-nil empty result", emptyIssues, err)
	}
	if _, err := checker.ListFinancialLinkIssues(ctx, uuid.New()); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("missing process error = %v, want ErrNotFound", err)
	}
	if after := financialLinksSnapshot(t, db); before != after {
		t.Fatalf("integrity check mutated process or financial data\nbefore: %s\nafter:  %s", before, after)
	}
}
