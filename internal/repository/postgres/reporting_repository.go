package postgres

import (
	"context"
	"fmt"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/platform/database"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

type ReportingRepository struct{ db database.DBTX }

var _ repository.ReportingRepository = (*ReportingRepository)(nil)

func NewReportingRepository(db database.DBTX) *ReportingRepository {
	return &ReportingRepository{db: db}
}

// scopedStudentSQL identifies the placement used by reporting and applies the
// authenticated actor's scope in SQL. Wali access is limited to assigned
// classes in the requested year; student access is limited to their own link.
const scopedStudentSQL = `
WITH placement AS (
  SELECT DISTINCT ON (h.student_id) h.student_id, h.class_id, c.name AS class_name
  FROM student_class_histories h
  JOIN classes c ON c.id=h.class_id
  JOIN academic_years ay ON ay.id=h.academic_year_id
  WHERE h.academic_year_id=$1
    AND h.validity @> GREATEST(ay.start_date, LEAST(CURRENT_DATE, ay.end_date - 1))
  ORDER BY h.student_id, lower(h.validity) DESC, h.id
), scoped AS (
  SELECT s.id AS student_id, s.user_id, s.name AS student_name, COALESCE(s.nis,'') AS nis,
         p.class_id, p.class_name, a.id AS account_id, COALESCE(a.status,'') AS account_status
  FROM placement p
  JOIN students s ON s.id=p.student_id
  LEFT JOIN savings_accounts a ON a.student_id=s.id AND a.academic_year_id=$1
  WHERE ($3='ADMIN'
    OR ($3='WALI_KELAS' AND EXISTS (
      SELECT 1 FROM teacher_class_assignments t
      WHERE t.user_id=$2 AND t.academic_year_id=$1 AND t.class_id=p.class_id
        AND t.validity @> GREATEST((SELECT start_date FROM academic_years WHERE id=$1),
                                  LEAST(CURRENT_DATE, (SELECT end_date - 1 FROM academic_years WHERE id=$1)))
    ))
    OR ($3='SISWA' AND s.user_id=$2))
), tx AS (
  SELECT t.savings_account_id,
    SUM(CASE WHEN t.type='DEPOSIT' THEN t.amount ELSE 0 END) AS deposits,
    SUM(CASE WHEN t.type='WITHDRAWAL' THEN t.amount ELSE 0 END) AS withdrawals,
    SUM(CASE WHEN t.type='DEPOSIT' THEN t.amount ELSE -t.amount END) AS balance
  FROM transactions t
  JOIN scoped s ON s.account_id=t.savings_account_id
  WHERE t.status='ACTIVE'
  GROUP BY t.savings_account_id
)`

func (r *ReportingRepository) Dashboard(ctx context.Context, yearID, userID uuid.UUID, role domain.UserRole) (repository.DashboardMetrics, error) {
	query := scopedStudentSQL + `
SELECT COUNT(DISTINCT s.student_id), COUNT(DISTINCT s.class_id), COUNT(s.account_id),
       COUNT(*) FILTER (WHERE s.account_status='SETTLED'),
       COALESCE(SUM(tx.balance),0)::text, COALESCE(SUM(tx.deposits),0)::text,
       COALESCE(SUM(tx.withdrawals),0)::text
FROM scoped s LEFT JOIN tx ON tx.savings_account_id=s.account_id`
	var out repository.DashboardMetrics
	var balance, deposits, withdrawals string
	if err := r.db.QueryRowContext(ctx, query, yearID, userID, role).Scan(&out.Students, &out.Classes, &out.Accounts, &out.SettledAccounts, &balance, &deposits, &withdrawals); err != nil {
		return repository.DashboardMetrics{}, fmt.Errorf("read dashboard metrics: %w", err)
	}
	var err error
	if out.Balance, err = parseSettlementAmount(balance); err != nil {
		return repository.DashboardMetrics{}, err
	}
	if out.Deposits, err = parseSettlementAmount(deposits); err != nil {
		return repository.DashboardMetrics{}, err
	}
	if out.Withdrawals, err = parseSettlementAmount(withdrawals); err != nil {
		return repository.DashboardMetrics{}, err
	}
	return out, nil
}

func (r *ReportingRepository) StudentReport(ctx context.Context, yearID, userID uuid.UUID, role domain.UserRole) ([]repository.StudentReportRow, error) {
	query := scopedStudentSQL + `
SELECT s.student_id, s.class_id, COALESCE(s.account_id,'00000000-0000-0000-0000-000000000000'),
       s.student_name, s.nis, s.class_name, s.account_status,
       COALESCE(tx.balance,0)::text, COALESCE(tx.deposits,0)::text, COALESCE(tx.withdrawals,0)::text
FROM scoped s LEFT JOIN tx ON tx.savings_account_id=s.account_id
ORDER BY s.class_name, s.student_name, s.student_id`
	rows, err := r.db.QueryContext(ctx, query, yearID, userID, role)
	if err != nil {
		return nil, fmt.Errorf("list student report: %w", err)
	}
	defer rows.Close()
	out := make([]repository.StudentReportRow, 0)
	for rows.Next() {
		var row repository.StudentReportRow
		var accountStatus, balance, deposits, withdrawals string
		if err := rows.Scan(&row.StudentID, &row.ClassID, &row.AccountID, &row.StudentName, &row.NIS, &row.ClassName, &accountStatus, &balance, &deposits, &withdrawals); err != nil {
			return nil, fmt.Errorf("scan student report: %w", err)
		}
		row.AccountStatus = domain.SavingsAccountStatus(accountStatus)
		if row.Balance, err = parseSettlementAmount(balance); err != nil {
			return nil, err
		}
		if row.Deposits, err = parseSettlementAmount(deposits); err != nil {
			return nil, err
		}
		if row.Withdrawals, err = parseSettlementAmount(withdrawals); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate student report: %w", err)
	}
	return out, nil
}

func (r *ReportingRepository) ClassReport(ctx context.Context, yearID, userID uuid.UUID, role domain.UserRole) ([]repository.ClassReportRow, error) {
	query := scopedStudentSQL + `
SELECT s.class_id, s.class_name, COUNT(DISTINCT s.student_id), COUNT(DISTINCT s.account_id),
       COUNT(DISTINCT s.account_id) FILTER (WHERE s.account_status='SETTLED'),
       COALESCE(SUM(tx.balance),0)::text, COALESCE(SUM(tx.deposits),0)::text,
       COALESCE(SUM(tx.withdrawals),0)::text
FROM scoped s LEFT JOIN tx ON tx.savings_account_id=s.account_id
GROUP BY s.class_id, s.class_name ORDER BY s.class_name, s.class_id`
	rows, err := r.db.QueryContext(ctx, query, yearID, userID, role)
	if err != nil {
		return nil, fmt.Errorf("list class report: %w", err)
	}
	defer rows.Close()
	out := make([]repository.ClassReportRow, 0)
	for rows.Next() {
		var row repository.ClassReportRow
		var balance, deposits, withdrawals string
		if err := rows.Scan(&row.ClassID, &row.ClassName, &row.Students, &row.Accounts, &row.Settled, &balance, &deposits, &withdrawals); err != nil {
			return nil, fmt.Errorf("scan class report: %w", err)
		}
		if row.Balance, err = parseSettlementAmount(balance); err != nil {
			return nil, err
		}
		if row.Deposits, err = parseSettlementAmount(deposits); err != nil {
			return nil, err
		}
		if row.Withdrawals, err = parseSettlementAmount(withdrawals); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate class report: %w", err)
	}
	return out, nil
}

func (r *ReportingRepository) SettlementReport(ctx context.Context, yearID, userID uuid.UUID, role domain.UserRole) ([]repository.SettlementReportRow, error) {
	query := scopedStudentSQL + `
SELECT s.student_id, s.class_id, s.account_id, s.student_name, s.nis, s.class_name, s.account_status,
       CASE WHEN s.account_status<>'SETTLED' THEN 'OUTSTANDING'
            WHEN COALESCE(settlement.amount,0)=0 OR COALESCE(movement.handed_over,0)=settlement.amount THEN 'COMPLETED'
            ELSE 'AWAITING_HANDOVER' END,
       COALESCE(tx.balance,0)::text, COALESCE(settlement.amount::text,'0'), COALESCE(movement.handed_over::text,'0')
FROM scoped s
JOIN savings_accounts a ON a.id=s.account_id
LEFT JOIN tx ON tx.savings_account_id=a.id
LEFT JOIN LATERAL (
  SELECT ss.id, ss.amount FROM savings_settlements ss
  WHERE ss.savings_account_id=a.id AND ss.status='COMPLETED'
  ORDER BY ss.executed_at DESC, ss.id DESC LIMIT 1
) settlement ON true
LEFT JOIN LATERAL (
  WITH RECURSIVE chain(id, replaces_settlement_id) AS (
    SELECT id, replaces_settlement_id FROM savings_settlements WHERE id=settlement.id
    UNION ALL
    SELECT prior.id, prior.replaces_settlement_id FROM savings_settlements prior JOIN chain c ON prior.id=c.replaces_settlement_id
  )
  SELECT SUM(CASE WHEN cm.kind='HANDOVER' THEN cm.amount ELSE -cm.amount END) AS handed_over
  FROM settlement_cash_movements cm JOIN chain c ON c.id=cm.savings_settlement_id
) movement ON true
ORDER BY s.class_name, s.student_name, s.student_id`
	rows, err := r.db.QueryContext(ctx, query, yearID, userID, role)
	if err != nil {
		return nil, fmt.Errorf("list settlement report: %w", err)
	}
	defer rows.Close()
	out := make([]repository.SettlementReportRow, 0)
	for rows.Next() {
		var row repository.SettlementReportRow
		var status, settlementState, balance, amount, handed string
		if err := rows.Scan(&row.StudentID, &row.ClassID, &row.AccountID, &row.StudentName, &row.NIS, &row.ClassName, &status, &settlementState, &balance, &amount, &handed); err != nil {
			return nil, fmt.Errorf("scan settlement report: %w", err)
		}
		row.AccountStatus = domain.SavingsAccountStatus(status)
		row.SettlementState = settlementState
		if row.Balance, err = parseSettlementAmount(balance); err != nil {
			return nil, err
		}
		if row.SettlementAmount, err = parseSettlementAmount(amount); err != nil {
			return nil, err
		}
		if row.HandedOver, err = parseSettlementAmount(handed); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate settlement report: %w", err)
	}
	return out, nil
}
