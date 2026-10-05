package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/platform/database"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

var _ repository.ClassTransferRequestRepository = (*ClassTransferRequestRepository)(nil)

type ClassTransferRequestRepository struct {
	db database.DBTX
}

func NewClassTransferRequestRepository(db database.DBTX) *ClassTransferRequestRepository {
	return &ClassTransferRequestRepository{
		db: db,
	}
}

func (r *ClassTransferRequestRepository) Create(
	ctx context.Context,
	request domain.ClassTransferRequest,
) error {
	const query = `
		INSERT INTO class_transfer_requests (
			id,
			student_id,
			academic_year_id,
			from_class_id,
			to_class_id,
			requested_by,
			requested_at,
			status,
			reviewed_by,
			reviewed_at,
			updated_at,
			rejection_reason
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9,
			$10,
			$11,
			$12
		)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		request.ID,
		request.StudentID,
		request.AcademicYearID,
		request.FromClassID,
		request.ToClassID,
		request.RequestedBy,
		request.CreatedAt,
		request.Status,
		request.ReviewedBy,
		request.ReviewedAt,
		request.UpdatedAt,
		request.RejectionReason,
	)
	if err != nil {
		return translateClassTransferRequestError(
			"create class transfer request",
			err,
		)
	}

	return nil
}

func (r *ClassTransferRequestRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (domain.ClassTransferRequest, error) {
	const query = `
		SELECT
			id,
			student_id,
			academic_year_id,
			from_class_id,
			to_class_id,
			requested_by,
			status,
			reviewed_by,
			reviewed_at,
			rejection_reason,
			requested_at,
			updated_at
		FROM class_transfer_requests
		WHERE id = $1
	`

	return r.getOne(ctx, query, id)
}

func (r *ClassTransferRequestRepository) GetByIDForUpdate(
	ctx context.Context,
	id uuid.UUID,
) (domain.ClassTransferRequest, error) {
	const query = `
		SELECT
			id,
			student_id,
			academic_year_id,
			from_class_id,
			to_class_id,
			requested_by,
			status,
			reviewed_by,
			reviewed_at,
			rejection_reason,
			requested_at,
			updated_at
		FROM class_transfer_requests
		WHERE id = $1
		FOR UPDATE
	`

	return r.getOne(ctx, query, id)
}

func (r *ClassTransferRequestRepository) ListByStudentAndAcademicYear(
	ctx context.Context,
	studentID uuid.UUID,
	academicYearID uuid.UUID,
	options repository.ListOptions,
) ([]domain.ClassTransferRequest, error) {
	const query = `
		SELECT
			id,
			student_id,
			academic_year_id,
			from_class_id,
			to_class_id,
			requested_by,
			status,
			reviewed_by,
			reviewed_at,
			rejection_reason,
			requested_at,
			updated_at
		FROM class_transfer_requests
		WHERE student_id = $1
			AND academic_year_id = $2
		ORDER BY requested_at ASC, id ASC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.db.QueryContext(
		ctx,
		query,
		studentID,
		academicYearID,
		options.Limit,
		options.Offset,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"list class transfer requests by student and academic year: %w",
			err,
		)
	}
	defer rows.Close()

	requests := make([]domain.ClassTransferRequest, 0)

	for rows.Next() {
		request, err := scanClassTransferRequest(rows)
		if err != nil {
			return nil, fmt.Errorf(
				"scan class transfer request: %w",
				err,
			)
		}

		requests = append(requests, request)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate class transfer requests: %w",
			err,
		)
	}

	return requests, nil
}

func (r *ClassTransferRequestRepository) ListPending(
	ctx context.Context,
	academicYearID uuid.UUID,
	options repository.ListOptions,
) ([]domain.ClassTransferRequest, error) {
	const query = `
		SELECT
			id,
			student_id,
			academic_year_id,
			from_class_id,
			to_class_id,
			requested_by,
			status,
			reviewed_by,
			reviewed_at,
			rejection_reason,
			requested_at,
			updated_at
		FROM class_transfer_requests
		WHERE academic_year_id = $1
			AND status = $2
		ORDER BY requested_at ASC, id ASC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.db.QueryContext(
		ctx,
		query,
		academicYearID,
		domain.TransferPending,
		options.Limit,
		options.Offset,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"list pending class transfer requests: %w",
			err,
		)
	}
	defer rows.Close()

	requests := make([]domain.ClassTransferRequest, 0)

	for rows.Next() {
		request, err := scanClassTransferRequest(rows)
		if err != nil {
			return nil, fmt.Errorf(
				"scan class transfer request: %w",
				err,
			)
		}

		requests = append(requests, request)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate pending class transfer requests: %w",
			err,
		)
	}

	return requests, nil
}

func (r *ClassTransferRequestRepository) Update(
	ctx context.Context,
	request domain.ClassTransferRequest,
) error {
	const query = `
		UPDATE class_transfer_requests
		SET
			student_id = $2,
			academic_year_id = $3,
			from_class_id = $4,
			to_class_id = $5,
			requested_by = $6,
			status = $7,
			reviewed_by = $8,
			reviewed_at = $9,
			rejection_reason = $10,
			updated_at = $11
		WHERE id = $1
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		request.ID,
		request.StudentID,
		request.AcademicYearID,
		request.FromClassID,
		request.ToClassID,
		request.RequestedBy,
		request.Status,
		request.ReviewedBy,
		request.ReviewedAt,
		request.RejectionReason,
		request.UpdatedAt,
	)
	if err != nil {
		return translateClassTransferRequestError(
			"update class transfer request",
			err,
		)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf(
			"get updated class transfer request rows affected: %w",
			err,
		)
	}

	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}

func (r *ClassTransferRequestRepository) getOne(
	ctx context.Context,
	query string,
	args ...any,
) (domain.ClassTransferRequest, error) {
	var request domain.ClassTransferRequest

	err := r.db.QueryRowContext(
		ctx,
		query,
		args...,
	).Scan(
		&request.ID,
		&request.StudentID,
		&request.AcademicYearID,
		&request.FromClassID,
		&request.ToClassID,
		&request.RequestedBy,
		&request.Status,
		&request.ReviewedBy,
		&request.ReviewedAt,
		&request.RejectionReason,
		&request.CreatedAt,
		&request.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ClassTransferRequest{}, repository.ErrNotFound
		}

		return domain.ClassTransferRequest{}, fmt.Errorf(
			"get class transfer request: %w",
			err,
		)
	}

	return request, nil
}

func scanClassTransferRequest(
	rows *sql.Rows,
) (domain.ClassTransferRequest, error) {
	var request domain.ClassTransferRequest

	err := rows.Scan(
		&request.ID,
		&request.StudentID,
		&request.AcademicYearID,
		&request.FromClassID,
		&request.ToClassID,
		&request.RequestedBy,
		&request.Status,
		&request.ReviewedBy,
		&request.ReviewedAt,
		&request.RejectionReason,
		&request.CreatedAt,
		&request.UpdatedAt,
	)
	if err != nil {
		return domain.ClassTransferRequest{}, err
	}

	return request, nil
}

func translateClassTransferRequestError(
	operation string,
	err error,
) error {
	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505", "23P01":
			return fmt.Errorf(
				"%s: %w",
				operation,
				repository.ErrConflict,
			)
		}
	}

	return fmt.Errorf("%s: %w", operation, err)
}
