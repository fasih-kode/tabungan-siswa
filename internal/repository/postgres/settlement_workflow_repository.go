package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

func (r *SettlementRepository) ListYearReadiness(ctx context.Context, yearID uuid.UUID) ([]repository.SettlementReadinessAccount, error) {
	const query = `
SELECT a.id, a.student_id, a.academic_year_id, a.status, a.settled_at, a.created_at, a.updated_at,
       s.user_id, s.nis, s.nisn, s.name, s.status, s.created_at, s.updated_at,
       CASE WHEN h.match_count = 1 THEN h.class_id END, COALESCE(c.name, ''),
       CASE WHEN t.match_count = 1 THEN t.user_id END, COALESCE(u.username, ''),
       (h.match_count > 1 OR t.match_count > 1),
       ss.id, ss.savings_account_id, COALESCE(ss.type,''), COALESCE(ss.amount::text,'0'), ss.executed_by, ss.executed_at, COALESCE(ss.status,''), ss.replaces_settlement_id,
       COALESCE(cm.delivered::text, '0'), COALESCE(rr.status, ''), COALESCE(ledger.amount::text,'0')
FROM savings_accounts a
JOIN students s ON s.id = a.student_id
JOIN academic_years ay ON ay.id = a.academic_year_id
LEFT JOIN LATERAL (
    SELECT count(*) AS match_count, (array_agg(x.class_id))[1] AS class_id
    FROM student_class_histories x
    WHERE x.student_id = s.id AND x.academic_year_id = ay.id AND x.validity @> (ay.end_date - 1)
) h ON true
LEFT JOIN classes c ON c.id = h.class_id AND h.match_count = 1
LEFT JOIN LATERAL (
    SELECT count(*) AS match_count, (array_agg(x.user_id))[1] AS user_id
    FROM teacher_class_assignments x
    WHERE h.match_count = 1 AND x.class_id = h.class_id AND x.academic_year_id = ay.id AND x.validity @> (ay.end_date - 1)
) t ON true
LEFT JOIN users u ON u.id = t.user_id
LEFT JOIN LATERAL (
    SELECT x.* FROM savings_settlements x WHERE x.savings_account_id = a.id AND x.status = 'COMPLETED' ORDER BY x.executed_at DESC, x.id DESC LIMIT 1
) ss ON true
LEFT JOIN LATERAL (
    WITH RECURSIVE chain(id, replaces_settlement_id) AS (
        SELECT id, replaces_settlement_id FROM savings_settlements WHERE id = ss.id
        UNION ALL
        SELECT p.id, p.replaces_settlement_id FROM savings_settlements p JOIN chain c ON p.id = c.replaces_settlement_id
    )
    SELECT COALESCE(SUM(CASE WHEN m.kind = 'HANDOVER' THEN m.amount ELSE -m.amount END), 0) AS delivered
    FROM settlement_cash_movements m JOIN chain c ON c.id = m.savings_settlement_id
) cm ON true
LEFT JOIN LATERAL (
    SELECT status FROM settlement_reopen_requests WHERE savings_settlement_id = ss.id AND status IN ('PENDING','APPROVED') ORDER BY requested_at DESC LIMIT 1
) rr ON true
LEFT JOIN LATERAL (
    SELECT COALESCE(SUM(CASE WHEN type='DEPOSIT' THEN amount ELSE -amount END),0) AS amount
    FROM transactions WHERE savings_account_id=a.id AND status='ACTIVE'
) ledger ON true
WHERE a.academic_year_id = $1
ORDER BY COALESCE(c.name, 'ZZZ'), s.name, s.id`
	rows, err := r.db.QueryContext(ctx, query, yearID)
	if err != nil {
		return nil, fmt.Errorf("list settlement readiness: %w", err)
	}
	defer rows.Close()
	result := make([]repository.SettlementReadinessAccount, 0)
	for rows.Next() {
		var row repository.SettlementReadinessAccount
		var accountStatus, studentStatus, settlementType, settlementAmount, settlementStatus string
		var delivered, currentBalance string
		var userID, classID, waliID, settlementID, settlementAccountID, settlementActorID, replacesID uuid.NullUUID
		var nis, nisn sql.NullString
		var settlementAt, settledAt sql.NullTime
		var reopenStatus string
		if err := rows.Scan(&row.Account.ID, &row.Account.StudentID, &row.Account.AcademicYearID, &accountStatus, &settledAt, &row.Account.CreatedAt, &row.Account.UpdatedAt,
			&userID, &nis, &nisn, &row.Student.Name, &studentStatus, &row.Student.CreatedAt, &row.Student.UpdatedAt,
			&classID, &row.ClassName, &waliID, &row.WaliName, &row.AttributionIssue, &settlementID, &settlementAccountID, &settlementType, &settlementAmount, &settlementActorID, &settlementAt, &settlementStatus, &replacesID, &delivered, &reopenStatus, &currentBalance); err != nil {
			return nil, fmt.Errorf("scan settlement readiness: %w", err)
		}
		row.Account.Status = domain.SavingsAccountStatus(accountStatus)
		row.Student.ID = row.Account.StudentID
		row.Account.SettledAt = nullTimePtr(settledAt)
		row.Student.Status = domain.StudentStatus(studentStatus)
		row.Student.UserID = nullUUIDPtr(userID)
		if nis.Valid {
			row.Student.NIS = &nis.String
		}
		if nisn.Valid {
			row.Student.NISN = &nisn.String
		}
		if classID.Valid {
			id := classID.UUID
			row.ClassID = &id
		}
		if waliID.Valid {
			id := waliID.UUID
			row.WaliID = &id
		}
		if settlementID.Valid {
			amount, e := parseSettlementAmount(settlementAmount)
			if e != nil {
				return nil, e
			}
			st := domain.SavingsSettlement{ID: settlementID.UUID, SavingsAccountID: settlementAccountID.UUID, Type: domain.SettlementType(settlementType), Amount: amount, ExecutedBy: settlementActorID.UUID, ExecutedAt: settlementAt.Time, Status: domain.SettlementStatus(settlementStatus)}
			if replacesID.Valid {
				id := replacesID.UUID
				st.ReplacesSettlementID = &id
			}
			row.Settlement = &st
		}
		row.CashDelivered, err = parseSettlementAmount(delivered)
		if err != nil {
			return nil, err
		}
		row.ReopenStatus = domain.SettlementReopenRequestStatus(reopenStatus)
		row.CurrentBalance, err = parseSettlementAmount(currentBalance)
		if err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate settlement readiness: %w", err)
	}
	return result, nil
}

func (r *SettlementRepository) CountPendingTransfersForTeacher(ctx context.Context, yearID, teacherID uuid.UUID) (int, error) {
	const q = `SELECT count(*) FROM class_transfer_requests cr
JOIN academic_years ay ON ay.id=cr.academic_year_id
WHERE cr.academic_year_id=$1 AND cr.status='PENDING'
AND EXISTS (SELECT 1 FROM teacher_class_assignments t
  WHERE t.class_id=cr.from_class_id AND t.academic_year_id=cr.academic_year_id
    AND t.user_id=$2 AND t.validity @> (ay.end_date - 1))`
	var n int
	err := r.db.QueryRowContext(ctx, q, yearID, teacherID).Scan(&n)
	return n, err
}

func nullUUIDPtr(v uuid.NullUUID) *uuid.UUID {
	if !v.Valid {
		return nil
	}
	id := v.UUID
	return &id
}
func nullTimePtr(v sql.NullTime) *time.Time {
	if !v.Valid {
		return nil
	}
	t := v.Time
	return &t
}

func (r *SettlementRepository) HasUnreconciledPositiveSettlements(ctx context.Context, yearID uuid.UUID) (bool, error) {
	const q = `SELECT EXISTS (SELECT 1 FROM savings_accounts a JOIN savings_settlements s ON s.savings_account_id=a.id AND s.status='COMPLETED' AND s.amount>0 LEFT JOIN LATERAL (WITH RECURSIVE chain(id,replaces_settlement_id) AS (SELECT id,replaces_settlement_id FROM savings_settlements WHERE id=s.id UNION ALL SELECT p.id,p.replaces_settlement_id FROM savings_settlements p JOIN chain c ON p.id=c.replaces_settlement_id) SELECT SUM(CASE WHEN m.kind='HANDOVER' THEN m.amount ELSE -m.amount END) AS delivered FROM settlement_cash_movements m JOIN chain c ON c.id=m.savings_settlement_id) m ON true WHERE a.academic_year_id=$1 AND COALESCE(m.delivered,0)<>s.amount)`
	var b bool
	err := r.db.QueryRowContext(ctx, q, yearID).Scan(&b)
	return b, err
}
func (r *SettlementRepository) HasActiveReopenWorkflow(ctx context.Context, yearID uuid.UUID) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM settlement_reopen_requests WHERE academic_year_id=$1 AND status IN ('PENDING','APPROVED'))`
	var b bool
	err := r.db.QueryRowContext(ctx, q, yearID).Scan(&b)
	return b, err
}
func (r *SettlementRepository) CreateCashMovement(ctx context.Context, m domain.SettlementCashMovement) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO settlement_cash_movements(id,savings_settlement_id,kind,amount,confirmed_by,confirmed_at,receipt_reference) VALUES($1,$2,$3,$4,$5,$6,$7)`, m.ID, m.SavingsSettlementID, string(m.Kind), m.Amount, m.ConfirmedBy, m.ConfirmedAt, m.ReceiptReference)
	return translateSettlementRepositoryError(err)
}
func (r *SettlementRepository) ListCashMovements(ctx context.Context, id uuid.UUID) ([]domain.SettlementCashMovement, error) {
	rows, err := r.db.QueryContext(ctx, `WITH RECURSIVE chain(id,replaces_settlement_id) AS (SELECT id,replaces_settlement_id FROM savings_settlements WHERE id=$1 UNION ALL SELECT p.id,p.replaces_settlement_id FROM savings_settlements p JOIN chain c ON p.id=c.replaces_settlement_id) SELECT m.id,m.savings_settlement_id,m.kind,m.amount::text,m.confirmed_by,m.confirmed_at,m.receipt_reference FROM settlement_cash_movements m JOIN chain c ON c.id=m.savings_settlement_id ORDER BY m.confirmed_at,m.id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.SettlementCashMovement{}
	for rows.Next() {
		var m domain.SettlementCashMovement
		var k, a string
		if err := rows.Scan(&m.ID, &m.SavingsSettlementID, &k, &a, &m.ConfirmedBy, &m.ConfirmedAt, &m.ReceiptReference); err != nil {
			return nil, err
		}
		m.Kind = domain.SettlementCashMovementKind(k)
		m.Amount, err = parseSettlementAmount(a)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
func (r *SettlementRepository) CreateReopenRequest(ctx context.Context, x domain.SettlementReopenRequest) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO settlement_reopen_requests(id,savings_settlement_id,savings_account_id,academic_year_id,requested_by,reason,requested_at,status,decided_by,decided_at,decision_reason,completed_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, x.ID, x.SavingsSettlementID, x.SavingsAccountID, x.AcademicYearID, x.RequestedBy, x.Reason, x.RequestedAt, string(x.Status), x.DecidedBy, x.DecidedAt, x.DecisionReason, x.CompletedAt)
	return translateSettlementRepositoryError(err)
}
func (r *SettlementRepository) GetReopenRequest(ctx context.Context, id uuid.UUID) (domain.SettlementReopenRequest, error) {
	return r.getRequest(ctx, `SELECT id,savings_settlement_id,savings_account_id,academic_year_id,requested_by,reason,requested_at,status,decided_by,decided_at,decision_reason,completed_at FROM settlement_reopen_requests WHERE id=$1`, id)
}
func (r *SettlementRepository) GetActiveReopenRequest(ctx context.Context, id uuid.UUID) (domain.SettlementReopenRequest, error) {
	return r.getRequest(ctx, `SELECT id,savings_settlement_id,savings_account_id,academic_year_id,requested_by,reason,requested_at,status,decided_by,decided_at,decision_reason,completed_at FROM settlement_reopen_requests WHERE savings_settlement_id=$1 AND status IN ('PENDING','APPROVED') ORDER BY requested_at DESC LIMIT 1`, id)
}

func (r *SettlementRepository) GetLatestReopenRequest(ctx context.Context, id uuid.UUID) (domain.SettlementReopenRequest, error) {
	return r.getRequest(ctx, `SELECT id,savings_settlement_id,savings_account_id,academic_year_id,requested_by,reason,requested_at,status,decided_by,decided_at,decision_reason,completed_at FROM settlement_reopen_requests WHERE savings_settlement_id=$1 ORDER BY requested_at DESC, id DESC LIMIT 1`, id)
}

func (r *SettlementRepository) GetApprovedReopenRequestByAccount(ctx context.Context, id uuid.UUID) (domain.SettlementReopenRequest, error) {
	return r.getRequest(ctx, `SELECT id,savings_settlement_id,savings_account_id,academic_year_id,requested_by,reason,requested_at,status,decided_by,decided_at,decision_reason,completed_at FROM settlement_reopen_requests WHERE savings_account_id=$1 AND status='APPROVED' ORDER BY requested_at DESC LIMIT 1`, id)
}
func (r *SettlementRepository) GetReopenRequestForUpdate(ctx context.Context, id uuid.UUID) (domain.SettlementReopenRequest, error) {
	return r.getRequest(ctx, `SELECT id,savings_settlement_id,savings_account_id,academic_year_id,requested_by,reason,requested_at,status,decided_by,decided_at,decision_reason,completed_at FROM settlement_reopen_requests WHERE id=$1 FOR UPDATE`, id)
}
func (r *SettlementRepository) getRequest(ctx context.Context, q string, id uuid.UUID) (domain.SettlementReopenRequest, error) {
	var x domain.SettlementReopenRequest
	var st string
	var decided uuid.NullUUID
	var decidedAt, completed sql.NullTime
	var why sql.NullString
	err := r.db.QueryRowContext(ctx, q, id).Scan(&x.ID, &x.SavingsSettlementID, &x.SavingsAccountID, &x.AcademicYearID, &x.RequestedBy, &x.Reason, &x.RequestedAt, &st, &decided, &decidedAt, &why, &completed)
	if err != nil {
		if err == sql.ErrNoRows {
			return x, repository.ErrNotFound
		}
		return x, err
	}
	x.Status = domain.SettlementReopenRequestStatus(st)
	x.DecidedBy = nullUUIDPtr(decided)
	x.DecidedAt = nullTimePtr(decidedAt)
	if why.Valid {
		x.DecisionReason = &why.String
	}
	x.CompletedAt = nullTimePtr(completed)
	return x, nil
}
func (r *SettlementRepository) UpdateReopenRequest(ctx context.Context, x domain.SettlementReopenRequest) error {
	res, err := r.db.ExecContext(ctx, `UPDATE settlement_reopen_requests SET status=$2,decided_by=$3,decided_at=$4,decision_reason=$5,completed_at=$6 WHERE id=$1`, x.ID, string(x.Status), x.DecidedBy, x.DecidedAt, x.DecisionReason, x.CompletedAt)
	if err != nil {
		return translateSettlementRepositoryError(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return repository.ErrNotFound
	}
	return nil
}
