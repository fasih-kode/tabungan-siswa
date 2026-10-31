package postgres

import (
	"context"
	"fmt"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/platform/database"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

func (r *SettlementRepository) ListByStudentLeavingRequest(
	ctx context.Context,
	requestID uuid.UUID,
) ([]domain.SavingsSettlement, error) {
	if err := requireStudentLeavingRequest(ctx, r.db, requestID); err != nil {
		return nil, err
	}

	const query = `
		SELECT
			settlement.id,
			settlement.savings_account_id,
			settlement.type,
			settlement.amount::text,
			settlement.executed_by,
			settlement.executed_at,
			settlement.status,
			settlement.replaces_settlement_id
		FROM savings_settlements AS settlement
		JOIN savings_accounts AS account
		  ON account.id = settlement.savings_account_id
		JOIN student_leaving_requests AS leaving_request
		  ON leaving_request.id = settlement.student_leaving_request_id
		 AND leaving_request.student_id = account.student_id
		WHERE leaving_request.id = $1
		  AND settlement.type = 'STUDENT_LEAVING'
		ORDER BY settlement.executed_at DESC, settlement.id DESC
	`

	rows, err := r.db.QueryContext(ctx, query, requestID)
	if err != nil {
		return nil, fmt.Errorf("list settlements for student leaving request: %w", err)
	}
	defer rows.Close()

	settlements := make([]domain.SavingsSettlement, 0)
	for rows.Next() {
		settlement, err := scanSavingsSettlement(rows)
		if err != nil {
			return nil, translateSettlementRepositoryError(err)
		}
		settlements = append(settlements, settlement)
	}
	if err := rows.Err(); err != nil {
		return nil, translateSettlementRepositoryError(err)
	}

	return settlements, nil
}

func (r *LeavingTransactionRequestRepository) ListByStudentLeavingRequest(
	ctx context.Context,
	requestID uuid.UUID,
) ([]domain.LeavingTransactionRequest, error) {
	if err := requireStudentLeavingRequest(ctx, r.db, requestID); err != nil {
		return nil, err
	}

	const query = `
		SELECT
			late_request.id,
			late_request.savings_account_id,
			late_request.academic_year_id,
			late_request.transaction_type,
			late_request.amount::text,
			late_request.transaction_date,
			late_request.reason,
			late_request.evidence_reference,
			late_request.requested_by,
			late_request.requested_at,
			late_request.status,
			late_request.decided_by,
			late_request.decided_at,
			late_request.decision_reason,
			late_request.completed_at
		FROM leaving_transaction_requests AS late_request
		WHERE late_request.student_leaving_request_id = $1
		ORDER BY late_request.requested_at DESC, late_request.id DESC
	`

	rows, err := r.db.QueryContext(ctx, query, requestID)
	if err != nil {
		return nil, fmt.Errorf("list late transaction requests for student leaving request: %w", err)
	}
	defer rows.Close()

	requests := make([]domain.LeavingTransactionRequest, 0)
	for rows.Next() {
		request, err := scanLeavingRequest(rows)
		if err != nil {
			return nil, fmt.Errorf("scan late transaction request linked to student leaving request: %w", err)
		}
		requests = append(requests, request)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read late transaction requests linked to student leaving request: %w", err)
	}

	return requests, nil
}

func requireStudentLeavingRequest(
	ctx context.Context,
	db database.DBTX,
	requestID uuid.UUID,
) error {
	var exists bool
	if err := db.QueryRowContext(
		ctx,
		`SELECT EXISTS (SELECT 1 FROM student_leaving_requests WHERE id = $1)`,
		requestID,
	).Scan(&exists); err != nil {
		return fmt.Errorf("check student leaving request: %w", err)
	}
	if !exists {
		return repository.ErrNotFound
	}

	return nil
}
