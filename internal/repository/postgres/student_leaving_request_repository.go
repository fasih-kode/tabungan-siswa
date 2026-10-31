package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/platform/database"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

var _ repository.StudentLeavingRequestRepository = (*StudentLeavingRequestRepository)(nil)

type StudentLeavingRequestRepository struct {
	db database.DBTX
}

func NewStudentLeavingRequestRepository(db database.DBTX) *StudentLeavingRequestRepository {
	return &StudentLeavingRequestRepository{db: db}
}

const studentLeavingRequestColumns = `id,
	student_id,
	academic_year_id,
	effective_leaving_date,
	status,
	initiated_by,
	created_at,
	finalized_by,
	finalized_at,
	cancelled_by,
	cancelled_at,
	cancellation_reason`

func (r *StudentLeavingRequestRepository) Create(
	ctx context.Context,
	request domain.StudentLeavingRequest,
) error {
	if request.Status != domain.StudentLeavingInProgress || !requestHasEmptyDecisionMetadata(request) {
		return domain.ErrInvalidTransition
	}

	const query = `
		INSERT INTO student_leaving_requests (
			id,
			student_id,
			academic_year_id,
			effective_leaving_date,
			status,
			initiated_by,
			created_at,
			finalized_by,
			finalized_at,
			cancelled_by,
			cancelled_at,
			cancellation_reason
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		request.ID,
		request.StudentID,
		request.AcademicYearID,
		request.EffectiveLeavingDate,
		request.Status,
		request.InitiatedBy,
		request.CreatedAt,
		request.FinalizedBy,
		request.FinalizedAt,
		request.CancelledBy,
		request.CancelledAt,
		cancellationReasonValue(request.CancellationReason),
	)
	return translateSettlementRepositoryError(err)
}

func (r *StudentLeavingRequestRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (domain.StudentLeavingRequest, error) {
	return r.get(ctx, `SELECT `+studentLeavingRequestColumns+`
		FROM student_leaving_requests
		WHERE id = $1`, id)
}

// GetByIDForUpdate harus dipanggil melalui repository yang terikat pada UOW.
func (r *StudentLeavingRequestRepository) GetByIDForUpdate(
	ctx context.Context,
	id uuid.UUID,
) (domain.StudentLeavingRequest, error) {
	return r.get(ctx, `SELECT `+studentLeavingRequestColumns+`
		FROM student_leaving_requests
		WHERE id = $1
		FOR UPDATE`, id)
}

func (r *StudentLeavingRequestRepository) GetInProgressByStudent(
	ctx context.Context,
	studentID uuid.UUID,
) (domain.StudentLeavingRequest, error) {
	return r.getInProgressByStudent(ctx, studentID, false)
}

// GetInProgressByStudentForUpdate mengunci row proses yang ditemukan.
// Ia tidak mengunci siswa atau mencegah INSERT ketika belum ada proses aktif.
func (r *StudentLeavingRequestRepository) GetInProgressByStudentForUpdate(
	ctx context.Context,
	studentID uuid.UUID,
) (domain.StudentLeavingRequest, error) {
	return r.getInProgressByStudent(ctx, studentID, true)
}

func (r *StudentLeavingRequestRepository) ListByStudent(
	ctx context.Context,
	studentID uuid.UUID,
) ([]domain.StudentLeavingRequest, error) {
	const query = `
		SELECT ` + studentLeavingRequestColumns + `
		FROM student_leaving_requests
		WHERE student_id = $1
		ORDER BY created_at DESC, id DESC
	`

	rows, err := r.db.QueryContext(ctx, query, studentID)
	if err != nil {
		return nil, fmt.Errorf("list student leaving requests: %w", err)
	}
	defer rows.Close()

	requests := []domain.StudentLeavingRequest{}
	for rows.Next() {
		request, scanErr := scanStudentLeavingRequest(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan student leaving request: %w", scanErr)
		}
		requests = append(requests, request)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list student leaving requests: %w", err)
	}
	return requests, nil
}

func (r *StudentLeavingRequestRepository) Update(
	ctx context.Context,
	request domain.StudentLeavingRequest,
) error {
	if request.ID == uuid.Nil {
		return domain.ErrInvalidID
	}
	if err := validateStudentLeavingRequestUpdate(request); err != nil {
		return err
	}

	const query = `
		UPDATE student_leaving_requests
		SET
			status = $1,
			finalized_by = $2,
			finalized_at = $3,
			cancelled_by = $4,
			cancelled_at = $5,
			cancellation_reason = $6
		WHERE id = $7
			AND status = 'IN_PROGRESS'
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		request.Status,
		request.FinalizedBy,
		request.FinalizedAt,
		request.CancelledBy,
		request.CancelledAt,
		cancellationReasonValue(request.CancellationReason),
		request.ID,
	)
	if err != nil {
		return translateSettlementRepositoryError(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update student leaving request rows affected: %w", err)
	}
	if rowsAffected != 0 {
		return nil
	}

	var exists bool
	if err := r.db.QueryRowContext(
		ctx,
		`SELECT EXISTS (SELECT 1 FROM student_leaving_requests WHERE id = $1)`,
		request.ID,
	).Scan(&exists); err != nil {
		return fmt.Errorf("check student leaving request after stale update: %w", err)
	}
	if !exists {
		return repository.ErrNotFound
	}
	return repository.ErrConflict
}

func (r *StudentLeavingRequestRepository) get(
	ctx context.Context,
	query string,
	id uuid.UUID,
) (domain.StudentLeavingRequest, error) {
	request, err := scanStudentLeavingRequest(r.db.QueryRowContext(ctx, query, id))
	if err == sql.ErrNoRows {
		return domain.StudentLeavingRequest{}, repository.ErrNotFound
	}
	if err != nil {
		return domain.StudentLeavingRequest{}, fmt.Errorf("get student leaving request: %w", err)
	}
	return request, nil
}

func (r *StudentLeavingRequestRepository) getInProgressByStudent(
	ctx context.Context,
	studentID uuid.UUID,
	forUpdate bool,
) (domain.StudentLeavingRequest, error) {
	query := `SELECT ` + studentLeavingRequestColumns + `
		FROM student_leaving_requests
		WHERE student_id = $1 AND status = 'IN_PROGRESS'`
	if forUpdate {
		query += ` FOR UPDATE`
	}
	request, err := scanStudentLeavingRequest(r.db.QueryRowContext(ctx, query, studentID))
	if err == sql.ErrNoRows {
		return domain.StudentLeavingRequest{}, repository.ErrNotFound
	}
	if err != nil {
		return domain.StudentLeavingRequest{}, fmt.Errorf("get in-progress student leaving request: %w", err)
	}
	return request, nil
}

type studentLeavingRequestScanner interface {
	Scan(...any) error
}

func scanStudentLeavingRequest(row studentLeavingRequestScanner) (domain.StudentLeavingRequest, error) {
	var request domain.StudentLeavingRequest
	var status string
	var finalizedBy, cancelledBy uuid.NullUUID
	var finalizedAt, cancelledAt sql.NullTime
	var cancellationReason sql.NullString

	err := row.Scan(
		&request.ID,
		&request.StudentID,
		&request.AcademicYearID,
		&request.EffectiveLeavingDate,
		&status,
		&request.InitiatedBy,
		&request.CreatedAt,
		&finalizedBy,
		&finalizedAt,
		&cancelledBy,
		&cancelledAt,
		&cancellationReason,
	)
	if err != nil {
		return domain.StudentLeavingRequest{}, err
	}

	request.EffectiveLeavingDate = studentLeavingDateOnly(request.EffectiveLeavingDate)
	request.CreatedAt = request.CreatedAt.UTC()
	request.Status = domain.StudentLeavingRequestStatus(status)
	if finalizedBy.Valid {
		request.FinalizedBy = &finalizedBy.UUID
	}
	if finalizedAt.Valid {
		value := finalizedAt.Time.UTC()
		request.FinalizedAt = &value
	}
	if cancelledBy.Valid {
		request.CancelledBy = &cancelledBy.UUID
	}
	if cancelledAt.Valid {
		value := cancelledAt.Time.UTC()
		request.CancelledAt = &value
	}
	if cancellationReason.Valid {
		value := cancellationReason.String
		request.CancellationReason = &value
	}
	return request, nil
}

func studentLeavingDateOnly(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func cancellationReasonValue(reason *string) any {
	if reason == nil {
		return nil
	}
	return *reason
}

func requestHasEmptyDecisionMetadata(request domain.StudentLeavingRequest) bool {
	return request.FinalizedBy == nil &&
		request.FinalizedAt == nil &&
		request.CancelledBy == nil &&
		request.CancelledAt == nil &&
		request.CancellationReason == nil
}

func validateStudentLeavingRequestUpdate(request domain.StudentLeavingRequest) error {
	switch request.Status {
	case domain.StudentLeavingFinalized:
		if request.FinalizedBy == nil || *request.FinalizedBy == uuid.Nil || request.FinalizedAt == nil || request.FinalizedAt.IsZero() ||
			request.CancelledBy != nil || request.CancelledAt != nil || request.CancellationReason != nil {
			return domain.ErrInvalidValue
		}
	case domain.StudentLeavingCancelled:
		if request.CancelledBy == nil || *request.CancelledBy == uuid.Nil || request.CancelledAt == nil || request.CancelledAt.IsZero() ||
			request.FinalizedBy != nil || request.FinalizedAt != nil || request.CancellationReason == nil {
			return domain.ErrInvalidValue
		}
		reason := strings.TrimSpace(*request.CancellationReason)
		if reason == "" || utf8.RuneCountInString(reason) > 500 {
			return domain.ErrInvalidValue
		}
	default:
		return domain.ErrInvalidTransition
	}
	return nil
}
