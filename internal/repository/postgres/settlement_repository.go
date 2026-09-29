package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/platform/database"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

var _ repository.SettlementRepository = (*SettlementRepository)(nil)

type SettlementRepository struct {
	db database.DBTX
}

func NewSettlementRepository(db database.DBTX) *SettlementRepository {
	return &SettlementRepository{db: db}
}

func (r *SettlementRepository) Create(
	ctx context.Context,
	settlement domain.SavingsSettlement,
) error {
	const query = `
		INSERT INTO savings_settlements (
			id,
			savings_account_id,
			type,
			amount,
			executed_by,
			executed_at,
			status,
			replaces_settlement_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		settlement.ID,
		settlement.SavingsAccountID,
		string(settlement.Type),
		settlement.Amount,
		settlement.ExecutedBy,
		settlement.ExecutedAt,
		string(settlement.Status),
		settlement.ReplacesSettlementID,
	)
	if err != nil {
		return translateSettlementRepositoryError(err)
	}

	return nil
}

func (r *SettlementRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (domain.SavingsSettlement, error) {
	const query = `
		SELECT
			id,
			savings_account_id,
			type,
			amount,
			executed_by,
			executed_at,
			status,
			replaces_settlement_id
		FROM savings_settlements
		WHERE id = $1
	`

	return r.getOne(ctx, query, id)
}

func (r *SettlementRepository) GetCompletedBySavingsAccount(
	ctx context.Context,
	savingsAccountID uuid.UUID,
) (domain.SavingsSettlement, error) {
	const query = `
		SELECT
			id,
			savings_account_id,
			type,
			amount,
			executed_by,
			executed_at,
			status,
			replaces_settlement_id
		FROM savings_settlements
		WHERE savings_account_id = $1
		  AND status = 'COMPLETED'
	`

	return r.getOne(ctx, query, savingsAccountID)
}

func (r *SettlementRepository) ListBySavingsAccount(
	ctx context.Context,
	savingsAccountID uuid.UUID,
	options repository.ListOptions,
) ([]domain.SavingsSettlement, error) {
	const query = `
		SELECT
			id,
			savings_account_id,
			type,
			amount,
			executed_by,
			executed_at,
			status,
			replaces_settlement_id
		FROM savings_settlements
		WHERE savings_account_id = $1
		ORDER BY executed_at DESC, id DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(
		ctx,
		query,
		savingsAccountID,
		options.Limit,
		options.Offset,
	)
	if err != nil {
		return nil, translateSettlementRepositoryError(err)
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

func (r *SettlementRepository) Update(
	ctx context.Context,
	settlement domain.SavingsSettlement,
) error {
	const query = `
		UPDATE savings_settlements
		SET
			type = $2,
			amount = $3,
			executed_by = $4,
			executed_at = $5,
			status = $6,
			replaces_settlement_id = $7
		WHERE id = $1
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		settlement.ID,
		string(settlement.Type),
		settlement.Amount,
		settlement.ExecutedBy,
		settlement.ExecutedAt,
		string(settlement.Status),
		settlement.ReplacesSettlementID,
	)
	if err != nil {
		return translateSettlementRepositoryError(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("settlement repository: rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}

type settlementScanner interface {
	Scan(dest ...any) error
}

func (r *SettlementRepository) getOne(
	ctx context.Context,
	query string,
	id uuid.UUID,
) (domain.SavingsSettlement, error) {
	row := r.db.QueryRowContext(ctx, query, id)

	settlement, err := scanSavingsSettlement(row)
	if err != nil {
		return domain.SavingsSettlement{}, translateSettlementRepositoryError(err)
	}

	return settlement, nil
}

func scanSavingsSettlement(scanner settlementScanner) (domain.SavingsSettlement, error) {
	var (
		settlement     domain.SavingsSettlement
		settlementType string
		amount         string
		status         string
		replacesID     uuid.NullUUID
	)

	if err := scanner.Scan(
		&settlement.ID,
		&settlement.SavingsAccountID,
		&settlementType,
		&amount,
		&settlement.ExecutedBy,
		&settlement.ExecutedAt,
		&status,
		&replacesID,
	); err != nil {
		return domain.SavingsSettlement{}, err
	}

	parsedAmount, err := parseSettlementAmount(amount)
	if err != nil {
		return domain.SavingsSettlement{}, err
	}

	settlement.Type = domain.SettlementType(settlementType)
	settlement.Amount = parsedAmount
	settlement.Status = domain.SettlementStatus(status)

	if replacesID.Valid {
		id := replacesID.UUID
		settlement.ReplacesSettlementID = &id
	}

	return settlement, nil
}

func parseSettlementAmount(value string) (domain.Money, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("settlement repository: empty amount")
	}

	if strings.ContainsAny(value, "eE") {
		return 0, fmt.Errorf("settlement repository: invalid amount %q", value)
	}

	parts := strings.Split(value, ".")
	if len(parts) > 2 {
		return 0, fmt.Errorf("settlement repository: invalid amount %q", value)
	}

	integerPart := parts[0]
	if integerPart == "" || integerPart == "+" || integerPart == "-" {
		return 0, fmt.Errorf("settlement repository: invalid amount %q", value)
	}

	if len(parts) == 2 && strings.Trim(parts[1], "0") != "" {
		return 0, fmt.Errorf(
			"settlement repository: non-integer amount %q",
			value,
		)
	}

	parsed, err := strconv.ParseInt(integerPart, 10, 64)
	if err != nil {
		return 0, fmt.Errorf(
			"settlement repository: parse amount %q: %w",
			value,
			err,
		)
	}

	return domain.Money(parsed), nil
}

func translateSettlementRepositoryError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return repository.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505", "23P01":
			return repository.ErrConflict
		}
	}

	return err
}
