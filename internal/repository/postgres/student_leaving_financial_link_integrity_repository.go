package postgres

import (
	"context"
	"fmt"

	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

func (r *StudentLeavingRequestRepository) ListFinancialLinkIssues(
	ctx context.Context,
	requestID uuid.UUID,
) ([]repository.StudentLeavingFinancialLinkIssue, error) {
	if err := requireStudentLeavingRequest(ctx, r.db, requestID); err != nil {
		return nil, err
	}

	const query = `
		SELECT issue_type, record_id
		FROM (
			SELECT
				'SETTLEMENT_STUDENT_MISMATCH'::text AS issue_type,
				settlement.id AS record_id
			FROM savings_settlements AS settlement
			JOIN savings_accounts AS account
			  ON account.id = settlement.savings_account_id
			JOIN student_leaving_requests AS leaving_request
			  ON leaving_request.id = settlement.student_leaving_request_id
			WHERE leaving_request.id = $1
			  AND account.student_id <> leaving_request.student_id

			UNION ALL

			SELECT
				'SETTLEMENT_TYPE_MISMATCH'::text,
				settlement.id
			FROM savings_settlements AS settlement
			WHERE settlement.student_leaving_request_id = $1
			  AND settlement.type <> 'STUDENT_LEAVING'

			UNION ALL

			SELECT
				'LATE_REQUEST_STUDENT_MISMATCH'::text,
				late_request.id
			FROM leaving_transaction_requests AS late_request
			JOIN savings_accounts AS account
			  ON account.id = late_request.savings_account_id
			JOIN student_leaving_requests AS leaving_request
			  ON leaving_request.id = late_request.student_leaving_request_id
			WHERE leaving_request.id = $1
			  AND account.student_id <> leaving_request.student_id

			UNION ALL

			SELECT
				'LATE_REQUEST_YEAR_MISMATCH'::text,
				late_request.id
			FROM leaving_transaction_requests AS late_request
			JOIN savings_accounts AS account
			  ON account.id = late_request.savings_account_id
			WHERE late_request.student_leaving_request_id = $1
			  AND late_request.academic_year_id <> account.academic_year_id
		) AS issues
		ORDER BY issue_type, record_id
	`

	rows, err := r.db.QueryContext(ctx, query, requestID)
	if err != nil {
		return nil, fmt.Errorf("list student leaving financial link issues: %w", err)
	}
	defer rows.Close()

	issues := make([]repository.StudentLeavingFinancialLinkIssue, 0)
	for rows.Next() {
		var issueType string
		var recordID uuid.UUID
		if err := rows.Scan(&issueType, &recordID); err != nil {
			return nil, fmt.Errorf("scan student leaving financial link issue: %w", err)
		}
		issues = append(issues, repository.StudentLeavingFinancialLinkIssue{
			Type:     repository.StudentLeavingFinancialLinkIssueType(issueType),
			RecordID: recordID,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read student leaving financial link issues: %w", err)
	}

	return issues, nil
}
