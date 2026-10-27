package postgres_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/repository/postgres"
	"github.com/google/uuid"
)

func TestLeavingFinancialLinksRepositoryV16(t *testing.T) {
	db := openTestDatabase(t)
	requireSchemaVersion16(t, db)
	ctx := context.Background()

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

	var settlementIDs, requestIDs, cashMovementIDs, auditIDs, transactionIDs []uuid.UUID
	t.Cleanup(func() {
		for _, id := range cashMovementIDs {
			if _, err := db.ExecContext(ctx, `DELETE FROM settlement_cash_movements WHERE id = $1`, id); err != nil {
				t.Errorf("delete cash movement fixture %s: %v", id, err)
			}
		}
		for _, id := range auditIDs {
			if _, err := db.ExecContext(ctx, `DELETE FROM audit_logs WHERE id = $1`, id); err != nil {
				t.Errorf("delete audit fixture %s: %v", id, err)
			}
		}
		for _, id := range transactionIDs {
			if _, err := db.ExecContext(ctx, `DELETE FROM transactions WHERE id = $1`, id); err != nil {
				t.Errorf("delete transaction fixture %s: %v", id, err)
			}
		}
		for _, id := range requestIDs {
			if _, err := db.ExecContext(ctx, `DELETE FROM leaving_transaction_requests WHERE id = $1`, id); err != nil {
				t.Errorf("delete late request fixture %s: %v", id, err)
			}
		}
		for i := len(settlementIDs) - 1; i >= 0; i-- {
			id := settlementIDs[i]
			if _, err := db.ExecContext(ctx, `DELETE FROM savings_settlements WHERE id = $1`, id); err != nil {
				t.Errorf("delete settlement fixture %s: %v", id, err)
			}
		}
		for _, id := range []uuid.UUID{firstLeaving.ID, secondLeaving.ID, emptyLeaving.ID} {
			if _, err := db.ExecContext(ctx, `DELETE FROM student_leaving_requests WHERE id = $1`, id); err != nil {
				t.Errorf("delete leaving process fixture %s: %v", id, err)
			}
		}
	})

	settlementRepo := postgres.NewSettlementRepository(db)
	legacySettlement := newFinancialLinksSettlement(t, first, domain.SettlementStudentLeaving, 1_000, time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC), nil, true)
	if err := settlementRepo.Create(ctx, legacySettlement); err != nil {
		t.Fatalf("create legacy settlement: %v", err)
	}
	settlementIDs = append(settlementIDs, legacySettlement.ID)

	linkedOld := newFinancialLinksSettlement(t, first, domain.SettlementStudentLeaving, 2_000, time.Date(2026, 9, 2, 9, 0, 0, 0, time.UTC), nil, true)
	if err := settlementRepo.Create(ctx, linkedOld); err != nil {
		t.Fatalf("create earlier linked settlement: %v", err)
	}
	settlementIDs = append(settlementIDs, linkedOld.ID)
	linkFinancialRow(t, db, "savings_settlements", linkedOld.ID, firstLeaving.ID)

	linkedNew := newFinancialLinksSettlement(t, first, domain.SettlementStudentLeaving, 3_000, time.Date(2026, 9, 3, 9, 0, 0, 0, time.UTC), &linkedOld.ID, false)
	if err := settlementRepo.Create(ctx, linkedNew); err != nil {
		t.Fatalf("create latest linked settlement: %v", err)
	}
	settlementIDs = append(settlementIDs, linkedNew.ID)
	linkFinancialRow(t, db, "savings_settlements", linkedNew.ID, firstLeaving.ID)

	regularSettlement := newFinancialLinksSettlement(t, first, domain.SettlementRegularYearEnd, 4_000, time.Date(2026, 9, 4, 9, 0, 0, 0, time.UTC), nil, true)
	if err := settlementRepo.Create(ctx, regularSettlement); err != nil {
		t.Fatalf("create year-end settlement fixture: %v", err)
	}
	settlementIDs = append(settlementIDs, regularSettlement.ID)
	linkFinancialRow(t, db, "savings_settlements", regularSettlement.ID, firstLeaving.ID)

	foreignSettlement := newFinancialLinksSettlement(t, settlementFixture{user: thirdUser, account: thirdAccount}, domain.SettlementStudentLeaving, 4_500, time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC), nil, true)
	if err := settlementRepo.Create(ctx, foreignSettlement); err != nil {
		t.Fatalf("create other student's leaving settlement fixture: %v", err)
	}
	settlementIDs = append(settlementIDs, foreignSettlement.ID)
	linkFinancialRow(t, db, "savings_settlements", foreignSettlement.ID, firstLeaving.ID)

	secondLinkedSettlement := newFinancialLinksSettlement(t, settlementFixture{user: secondUser, account: secondAccount}, domain.SettlementStudentLeaving, 5_000, time.Date(2026, 9, 5, 9, 0, 0, 0, time.UTC), &regularSettlement.ID, false)
	if err := settlementRepo.Create(ctx, secondLinkedSettlement); err != nil {
		t.Fatalf("create second process settlement fixture: %v", err)
	}
	settlementIDs = append(settlementIDs, secondLinkedSettlement.ID)
	linkFinancialRow(t, db, "savings_settlements", secondLinkedSettlement.ID, secondLeaving.ID)

	transaction, err := domain.NewDeposit(first.account.ID, 7_000, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), first.user.ID)
	if err != nil {
		t.Fatalf("NewDeposit() error: %v", err)
	}
	if err := postgres.NewTransactionRepository(db).Create(ctx, transaction); err != nil {
		t.Fatalf("create ledger fixture: %v", err)
	}
	transactionIDs = append(transactionIDs, transaction.ID)

	lateRequestRepo := postgres.NewLeavingTransactionRequestRepository(db)
	lateOld := newFinancialLinksLateRequest(t, first, time.Date(2026, 9, 6, 10, 0, 0, 0, time.UTC), "historis-null")
	lateOld.ID = uuid.New()
	if err := lateRequestRepo.CreateLeavingTransactionRequest(ctx, lateOld); err != nil {
		t.Fatalf("create historical late request: %v", err)
	}
	requestIDs = append(requestIDs, lateOld.ID)

	lateFirst := newFinancialLinksLateRequest(t, first, time.Date(2026, 9, 7, 10, 0, 0, 0, time.UTC), "proses-1-lama")
	if err := lateRequestRepo.CreateLeavingTransactionRequest(ctx, lateFirst); err != nil {
		t.Fatalf("create first linked late request: %v", err)
	}
	requestIDs = append(requestIDs, lateFirst.ID)
	linkFinancialRow(t, db, "leaving_transaction_requests", lateFirst.ID, firstLeaving.ID)

	lateLatest := newFinancialLinksLateRequest(t, first, time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC), "proses-1-baru")
	if err := lateRequestRepo.CreateLeavingTransactionRequest(ctx, lateLatest); err != nil {
		t.Fatalf("create latest linked late request: %v", err)
	}
	requestIDs = append(requestIDs, lateLatest.ID)
	linkFinancialRow(t, db, "leaving_transaction_requests", lateLatest.ID, firstLeaving.ID)

	foreignLateRequest := newFinancialLinksLateRequest(
		t,
		settlementFixture{user: secondUser, year: first.year, account: secondAccount},
		time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC),
		"proses-1-milik-siswa-lain",
	)
	if err := lateRequestRepo.CreateLeavingTransactionRequest(ctx, foreignLateRequest); err != nil {
		t.Fatalf("create other student's late request fixture: %v", err)
	}
	requestIDs = append(requestIDs, foreignLateRequest.ID)
	linkFinancialRow(t, db, "leaving_transaction_requests", foreignLateRequest.ID, firstLeaving.ID)

	lateSecond := newFinancialLinksLateRequest(t, settlementFixture{user: secondUser, year: first.year, account: secondAccount}, time.Date(2026, 9, 9, 10, 0, 0, 0, time.UTC), "proses-2")
	if err := lateSecond.Decide(secondUser.ID, true, "Disetujui untuk uji repository"); err != nil {
		t.Fatalf("Decide(second late request): %v", err)
	}
	if err := lateSecond.Complete(); err != nil {
		t.Fatalf("Complete(second late request): %v", err)
	}
	if err := lateRequestRepo.CreateLeavingTransactionRequest(ctx, lateSecond); err != nil {
		t.Fatalf("create second process late request: %v", err)
	}
	requestIDs = append(requestIDs, lateSecond.ID)
	linkFinancialRow(t, db, "leaving_transaction_requests", lateSecond.ID, secondLeaving.ID)

	movement, err := domain.NewSettlementCashMovement(
		secondLinkedSettlement.ID,
		domain.SettlementCashHandover,
		5_000,
		secondUser.ID,
		time.Date(2026, 9, 10, 10, 0, 0, 0, time.UTC),
		nil,
	)
	if err != nil {
		t.Fatalf("NewSettlementCashMovement() error: %v", err)
	}
	if err := settlementRepo.CreateCashMovement(ctx, movement); err != nil {
		t.Fatalf("create cash movement fixture: %v", err)
	}
	cashMovementIDs = append(cashMovementIDs, movement.ID)

	auditID := uuid.New()
	if _, err := db.ExecContext(ctx, `
		INSERT INTO audit_logs (id, actor_user_id, action, entity_type, entity_id, after_data)
		VALUES ($1, $2, 'MQ08_FINANCIAL_LINK_FIXTURE', 'SAVINGS_SETTLEMENT', $3, '{}'::jsonb)
	`, auditID, first.user.ID, linkedNew.ID); err != nil {
		t.Fatalf("create audit fixture: %v", err)
	}
	auditIDs = append(auditIDs, auditID)

	before := financialLinksSnapshot(t, db)

	gotFirstSettlements, err := settlementRepo.ListByStudentLeavingRequest(ctx, firstLeaving.ID)
	if err != nil {
		t.Fatalf("ListByStudentLeavingRequest(first) error: %v", err)
	}
	if got, want := settlementIDsOf(gotFirstSettlements), []uuid.UUID{linkedNew.ID, linkedOld.ID}; !equalUUIDs(got, want) {
		t.Fatalf("first process settlements = %v, want linked STUDENT_LEAVING rows in newest-first order %v", got, want)
	}
	if gotFirstSettlements[1].ReplacesSettlementID != nil {
		t.Fatalf("nullable replaces_settlement_id = %v, want nil", gotFirstSettlements[1].ReplacesSettlementID)
	}
	for _, settlement := range gotFirstSettlements {
		if settlement.Type != domain.SettlementStudentLeaving {
			t.Fatalf("first process returned settlement type %q; REGULAR_YEAR_END must not be classified as leaving", settlement.Type)
		}
	}

	gotSecondSettlements, err := settlementRepo.ListByStudentLeavingRequest(ctx, secondLeaving.ID)
	if err != nil || len(gotSecondSettlements) != 1 || gotSecondSettlements[0].ID != secondLinkedSettlement.ID {
		t.Fatalf("second process settlements = %#v, %v; want only its student's settlement", gotSecondSettlements, err)
	}
	gotEmptySettlements, err := settlementRepo.ListByStudentLeavingRequest(ctx, emptyLeaving.ID)
	if err != nil || gotEmptySettlements == nil || len(gotEmptySettlements) != 0 {
		t.Fatalf("empty process settlements = %#v, %v; want non-nil empty result", gotEmptySettlements, err)
	}
	if _, err := settlementRepo.ListByStudentLeavingRequest(ctx, uuid.New()); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("missing process settlements error = %v, want ErrNotFound", err)
	}

	gotFirstRequests, err := lateRequestRepo.ListByStudentLeavingRequest(ctx, firstLeaving.ID)
	if err != nil {
		t.Fatalf("ListByStudentLeavingRequest(first late requests) error: %v", err)
	}
	if got, want := lateRequestIDsOf(gotFirstRequests), []uuid.UUID{lateLatest.ID, lateFirst.ID}; !equalUUIDs(got, want) {
		t.Fatalf("first process late requests = %v, want linked rows in newest-first order %v", got, want)
	}
	for _, request := range gotFirstRequests {
		if request.DecidedBy != nil || request.DecidedAt != nil || request.CompletedAt != nil {
			t.Fatalf("pending linked request nullable decision metadata = %#v; want nil", request)
		}
	}
	gotSecondRequests, err := lateRequestRepo.ListByStudentLeavingRequest(ctx, secondLeaving.ID)
	if err != nil || len(gotSecondRequests) != 1 || gotSecondRequests[0].ID != lateSecond.ID {
		t.Fatalf("second process late requests = %#v, %v; want only its student's request", gotSecondRequests, err)
	}
	if gotSecondRequests[0].Status != domain.LeavingTransactionCompleted || gotSecondRequests[0].DecidedBy == nil || gotSecondRequests[0].DecidedAt == nil || gotSecondRequests[0].CompletedAt == nil {
		t.Fatalf("completed late request nullable metadata not mapped: %#v", gotSecondRequests[0])
	}
	gotEmptyRequests, err := lateRequestRepo.ListByStudentLeavingRequest(ctx, emptyLeaving.ID)
	if err != nil || gotEmptyRequests == nil || len(gotEmptyRequests) != 0 {
		t.Fatalf("empty process late requests = %#v, %v; want non-nil empty result", gotEmptyRequests, err)
	}
	if _, err := lateRequestRepo.ListByStudentLeavingRequest(ctx, uuid.New()); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("missing process late requests error = %v, want ErrNotFound", err)
	}

	legacySettlements, err := settlementRepo.ListBySavingsAccount(ctx, first.account.ID, repository.ListOptions{Limit: 20})
	if err != nil {
		t.Fatalf("ListBySavingsAccount() error: %v", err)
	}
	if !containsSettlement(legacySettlements, legacySettlement.ID) {
		t.Fatalf("historical settlement with NULL process link is missing from ListBySavingsAccount()")
	}
	legacyRequests, err := lateRequestRepo.ListLeavingTransactionRequests(ctx, first.year.ID, "")
	if err != nil {
		t.Fatalf("ListLeavingTransactionRequests() error: %v", err)
	}
	if !containsLateRequest(legacyRequests, lateOld.ID) {
		t.Fatalf("historical late request with NULL process link is missing from ListLeavingTransactionRequests()")
	}

	if after := financialLinksSnapshot(t, db); before != after {
		t.Fatalf("repository reads mutated financial or audit data\nbefore: %s\nafter:  %s", before, after)
	}
}

func TestLeavingFinancialLinksRepositoryEmptyAndMissingProcessV16(t *testing.T) {
	db := openTestDatabase(t)
	requireSchemaVersion16(t, db)
	ctx := context.Background()
	fixture := newSettlementFixture(t, db)
	leaving := newFinancialLinksLeavingRequest(
		t,
		postgres.NewStudentLeavingRequestRepository(db),
		fixture.student.ID,
		fixture.year.ID,
		fixture.user.ID,
	)
	t.Cleanup(func() {
		if _, err := db.ExecContext(ctx, `DELETE FROM student_leaving_requests WHERE id = $1`, leaving.ID); err != nil {
			t.Errorf("delete empty leaving process fixture: %v", err)
		}
	})

	settlements, err := postgres.NewSettlementRepository(db).ListByStudentLeavingRequest(ctx, leaving.ID)
	if err != nil || settlements == nil || len(settlements) != 0 {
		t.Fatalf("ListByStudentLeavingRequest(empty) = %#v, %v; want non-nil empty slice", settlements, err)
	}
	requests, err := postgres.NewLeavingTransactionRequestRepository(db).ListByStudentLeavingRequest(ctx, leaving.ID)
	if err != nil || requests == nil || len(requests) != 0 {
		t.Fatalf("ListByStudentLeavingRequest(empty late requests) = %#v, %v; want non-nil empty slice", requests, err)
	}
}

func requireSchemaVersion16(t *testing.T, db *sql.DB) {
	t.Helper()

	var version int
	var dirty bool
	if err := db.QueryRowContext(context.Background(), `SELECT version, dirty FROM schema_migrations LIMIT 1`).Scan(&version, &dirty); err != nil {
		t.Fatalf("read migration version: %v", err)
	}
	if version != 16 || dirty {
		t.Skipf("requires clean schema v16; found version=%d dirty=%t", version, dirty)
	}
}

func newFinancialLinksLeavingRequest(
	t *testing.T,
	repo repository.StudentLeavingRequestRepository,
	studentID, yearID, actorID uuid.UUID,
) domain.StudentLeavingRequest {
	t.Helper()

	request, err := domain.NewStudentLeavingRequest(
		studentID,
		yearID,
		time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		actorID,
	)
	if err != nil {
		t.Fatalf("NewStudentLeavingRequest() error: %v", err)
	}
	if err := repo.Create(context.Background(), request); err != nil {
		t.Fatalf("create student leaving request fixture: %v", err)
	}
	return request
}

func newFinancialLinksSettlement(
	t *testing.T,
	fixture settlementFixture,
	settlementType domain.SettlementType,
	amount domain.Money,
	executedAt time.Time,
	replacesID *uuid.UUID,
	superseded bool,
) domain.SavingsSettlement {
	t.Helper()

	settlement, err := domain.NewSavingsSettlement(
		fixture.account.ID,
		settlementType,
		amount,
		fixture.user.ID,
		executedAt,
		replacesID,
	)
	if err != nil {
		t.Fatalf("NewSavingsSettlement() error: %v", err)
	}
	if superseded {
		if err := settlement.Supersede(); err != nil {
			t.Fatalf("Supersede() error: %v", err)
		}
	}
	return settlement
}

func newFinancialLinksLateRequest(
	t *testing.T,
	fixture settlementFixture,
	requestedAt time.Time,
	evidence string,
) domain.LeavingTransactionRequest {
	t.Helper()

	request, err := domain.NewLeavingTransactionRequest(
		fixture.account.ID,
		fixture.year.ID,
		domain.TransactionDeposit,
		1_000,
		requestedAt,
		"Permintaan uji relasi",
		"MQ08-03B-"+evidence,
		fixture.user.ID,
	)
	if err != nil {
		t.Fatalf("NewLeavingTransactionRequest() error: %v", err)
	}
	request.RequestedAt = requestedAt
	return request
}

func linkFinancialRow(t *testing.T, db *sql.DB, table string, rowID, leavingRequestID uuid.UUID) {
	t.Helper()
	if table != "savings_settlements" && table != "leaving_transaction_requests" {
		t.Fatalf("unexpected fixture table %q", table)
	}
	query := fmt.Sprintf("UPDATE %s SET student_leaving_request_id = $1 WHERE id = $2", table)
	if _, err := db.ExecContext(context.Background(), query, leavingRequestID, rowID); err != nil {
		t.Fatalf("link fixture in %s: %v", table, err)
	}
}

func financialLinksSnapshot(t *testing.T, db *sql.DB) string {
	t.Helper()

	const query = `
		SELECT jsonb_build_object(
			'leaving_requests', COALESCE((SELECT jsonb_agg(to_jsonb(lr) ORDER BY lr.id) FROM student_leaving_requests lr), '[]'::jsonb),
			'savings_accounts', COALESCE((SELECT jsonb_agg(to_jsonb(sa) ORDER BY sa.id) FROM savings_accounts sa), '[]'::jsonb),
			'transactions', COALESCE((SELECT jsonb_agg(to_jsonb(t) ORDER BY t.id) FROM transactions t), '[]'::jsonb),
			'settlements', COALESCE((SELECT jsonb_agg(to_jsonb(s) ORDER BY s.id) FROM savings_settlements s), '[]'::jsonb),
			'cash_movements', COALESCE((SELECT jsonb_agg(to_jsonb(m) ORDER BY m.id) FROM settlement_cash_movements m), '[]'::jsonb),
			'late_requests', COALESCE((SELECT jsonb_agg(to_jsonb(l) ORDER BY l.id) FROM leaving_transaction_requests l), '[]'::jsonb),
			'audit_logs', COALESCE((SELECT jsonb_agg(to_jsonb(a) ORDER BY a.id) FROM audit_logs a), '[]'::jsonb)
		)::text
	`
	var snapshot string
	if err := db.QueryRowContext(context.Background(), query).Scan(&snapshot); err != nil {
		t.Fatalf("snapshot finance-related tables: %v", err)
	}
	return snapshot
}

func settlementIDsOf(items []domain.SavingsSettlement) []uuid.UUID {
	ids := make([]uuid.UUID, len(items))
	for i := range items {
		ids[i] = items[i].ID
	}
	return ids
}

func lateRequestIDsOf(items []domain.LeavingTransactionRequest) []uuid.UUID {
	ids := make([]uuid.UUID, len(items))
	for i := range items {
		ids[i] = items[i].ID
	}
	return ids
}

func equalUUIDs(got, want []uuid.UUID) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func containsSettlement(items []domain.SavingsSettlement, id uuid.UUID) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}

func containsLateRequest(items []domain.LeavingTransactionRequest, id uuid.UUID) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}
