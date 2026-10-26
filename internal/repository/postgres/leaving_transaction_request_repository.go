package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/platform/database"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

var _ repository.LeavingTransactionRequestRepository = (*LeavingTransactionRequestRepository)(nil)

type LeavingTransactionRequestRepository struct {
	db database.DBTX
}

func NewLeavingTransactionRequestRepository(db database.DBTX) *LeavingTransactionRequestRepository {
	return &LeavingTransactionRequestRepository{
		db: db,
	}
}

const leavingRequestColumns = `id,
	savings_account_id,
	academic_year_id,
	transaction_type,
	amount::text,
	transaction_date,
	reason,
	evidence_reference,
	requested_by,
	requested_at,
	status,
	decided_by,
	decided_at,
	decision_reason,
	completed_at`

const leavingRequestInsertColumns = `id,
	savings_account_id,
	academic_year_id,
	transaction_type,
	amount,
	transaction_date,
	reason,
	evidence_reference,
	requested_by,
	requested_at,
	status,
	decided_by,
	decided_at,
	decision_reason,
	completed_at`

func nullableText(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func (r *LeavingTransactionRequestRepository) CreateLeavingTransactionRequest(
	ctx context.Context,
	x domain.LeavingTransactionRequest,
) error {
	const query = `
		INSERT INTO leaving_transaction_requests (
			` + leavingRequestInsertColumns + `
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14, $15
		)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		x.ID,
		x.SavingsAccountID,
		x.AcademicYearID,
		x.Type,
		x.Amount,
		x.TransactionDate,
		x.Reason,
		x.EvidenceReference,
		x.RequestedBy,
		x.RequestedAt,
		x.Status,
		x.DecidedBy,
		x.DecidedAt,
		nullableText(x.DecisionReason),
		x.CompletedAt,
	)

	return translateSettlementRepositoryError(err)
}

func (r *LeavingTransactionRequestRepository) GetLeavingTransactionRequest(
	ctx context.Context,
	id uuid.UUID,
) (domain.LeavingTransactionRequest, error) {
	return r.get(
		ctx,
		`SELECT `+leavingRequestColumns+`
		FROM leaving_transaction_requests
		WHERE id = $1`,
		id,
	)
}

func (r *LeavingTransactionRequestRepository) GetLeavingTransactionRequestForUpdate(
	ctx context.Context,
	id uuid.UUID,
) (domain.LeavingTransactionRequest, error) {
	return r.get(
		ctx,
		`SELECT `+leavingRequestColumns+`
		FROM leaving_transaction_requests
		WHERE id = $1
		FOR UPDATE`,
		id,
	)
}

func (r *LeavingTransactionRequestRepository) ListLeavingTransactionRequests(
	ctx context.Context,
	yearID uuid.UUID,
	status domain.LeavingTransactionRequestStatus,
) ([]domain.LeavingTransactionRequest, error) {
	const query = `
		SELECT ` + leavingRequestColumns + `
		FROM leaving_transaction_requests
		WHERE (
			NULLIF($1::text, '00000000-0000-0000-0000-000000000000') IS NULL
			OR academic_year_id = $1::uuid
		)
		AND ($2 = '' OR status = $2)
		ORDER BY requested_at DESC, id
	`

	rows, err := r.db.QueryContext(ctx, query, yearID, status)
	if err != nil {
		return nil, fmt.Errorf("list leaving transaction requests: %w", err)
	}
	defer rows.Close()

	out := []domain.LeavingTransactionRequest{}
	for rows.Next() {
		x, e := scanLeavingRequest(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (r *LeavingTransactionRequestRepository) UpdateLeavingTransactionRequest(
	ctx context.Context,
	x domain.LeavingTransactionRequest,
) error {
	const query = `
		UPDATE leaving_transaction_requests
		SET
			status = $1,
			decided_by = $2,
			decided_at = $3,
			decision_reason = $4,
			completed_at = $5
		WHERE id = $6
	`

	res, err := r.db.ExecContext(
		ctx,
		query,
		x.Status,
		x.DecidedBy,
		x.DecidedAt,
		nullableText(x.DecisionReason),
		x.CompletedAt,
		x.ID,
	)
	if err != nil {
		return translateSettlementRepositoryError(err)
	}

	n, e := res.RowsAffected()
	if e != nil {
		return e
	}
	if n == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *LeavingTransactionRequestRepository) HasPendingLeavingTransactionRequests(
	ctx context.Context,
	yearID uuid.UUID,
) (bool, error) {
	var b bool
	err := r.db.QueryRowContext(
		ctx,
		`SELECT EXISTS (
			SELECT 1
			FROM leaving_transaction_requests
			WHERE academic_year_id = $1
				AND status IN ('PENDING', 'APPROVED')
		)`,
		yearID,
	).Scan(&b)
	return b, err
}

func (r *LeavingTransactionRequestRepository) HasPendingForSavingsAccount(
	ctx context.Context,
	accountID uuid.UUID,
) (bool, error) {
	var b bool
	err := r.db.QueryRowContext(
		ctx,
		`SELECT EXISTS (
			SELECT 1
			FROM leaving_transaction_requests
			WHERE savings_account_id = $1
				AND status IN ('PENDING', 'APPROVED')
		)`,
		accountID,
	).Scan(&b)
	return b, err
}

func (r *LeavingTransactionRequestRepository) HasEvidenceReference(
	ctx context.Context,
	accountID uuid.UUID,
	reference string,
) (bool, error) {
	var b bool
	err := r.db.QueryRowContext(
		ctx,
		`SELECT EXISTS (
			SELECT 1
			FROM leaving_transaction_requests
			WHERE savings_account_id = $1
				AND lower(evidence_reference) = lower($2)
				AND status IN ('PENDING', 'APPROVED', 'COMPLETED')
		)`,
		accountID,
		reference,
	).Scan(&b)
	return b, err
}

func (r *LeavingTransactionRequestRepository) get(
	ctx context.Context,
	q string,
	id uuid.UUID,
) (domain.LeavingTransactionRequest, error) {
	x, err := scanLeavingRequest(
		r.db.QueryRowContext(ctx, q, id),
	)
	if err == sql.ErrNoRows {
		return x, repository.ErrNotFound
	}
	return x, err
}

type leavingRequestScanner interface {
	Scan(...any) error
}

func scanLeavingRequest(row leavingRequestScanner) (domain.LeavingTransactionRequest, error) {
	var x domain.LeavingTransactionRequest
	var typ, status, amount string
	var decidedBy uuid.NullUUID
	var decidedAt, completedAt sql.NullTime
	var decisionReason sql.NullString
	err := row.Scan(
		&x.ID,
		&x.SavingsAccountID,
		&x.AcademicYearID,
		&typ,
		&amount,
		&x.TransactionDate,
		&x.Reason,
		&x.EvidenceReference,
		&x.RequestedBy,
		&x.RequestedAt,
		&status,
		&decidedBy,
		&decidedAt,
		&decisionReason,
		&completedAt,
	)
	if err != nil {
		return x, err
	}
	x.Type = domain.TransactionType(typ)
	x.Status = domain.LeavingTransactionRequestStatus(status)
	x.Amount, err = parseSettlementAmount(amount)
	if err != nil {
		return x, err
	}

	if decidedBy.Valid {
		x.DecidedBy = &decidedBy.UUID
	}
	if decidedAt.Valid {
		x.DecidedAt = &decidedAt.Time
	}
	if completedAt.Valid {
		x.CompletedAt = &completedAt.Time
	}
	x.DecisionReason = decisionReason.String
	return x, nil
}
