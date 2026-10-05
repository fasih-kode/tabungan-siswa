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

var _ repository.TransactionRepository = (*TransactionRepository)(nil)

type TransactionRepository struct {
	db database.DBTX
}

func NewTransactionRepository(db database.DBTX) *TransactionRepository {
	return &TransactionRepository{
		db: db,
	}
}

func (r *TransactionRepository) Create(
	ctx context.Context,
	transaction domain.Transaction,
) error {
	const query = `
		INSERT INTO transactions (
			id,
			savings_account_id,
			type,
			amount,
			transaction_date,
			status,
			created_by,
			created_at,
			updated_at
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
			$9
		)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		transaction.ID,
		transaction.SavingsAccountID,
		transaction.Type,
		transaction.Amount,
		transaction.TransactionDate,
		transaction.Status,
		transaction.CreatedBy,
		transaction.CreatedAt,
		transaction.UpdatedAt,
	)
	if err != nil {
		return translateTransactionError(
			"create transaction",
			err,
		)
	}

	return nil
}

func (r *TransactionRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (domain.Transaction, error) {
	const query = `
		SELECT
			id,
			savings_account_id,
			type,
			amount,
			transaction_date,
			status,
			created_by,
			created_at,
			updated_at
		FROM transactions
		WHERE id = $1
	`

	return r.getOne(ctx, query, id)
}

func (r *TransactionRepository) ListBySavingsAccount(
	ctx context.Context,
	savingsAccountID uuid.UUID,
	options repository.ListOptions,
) ([]domain.Transaction, error) {
	const query = `
		SELECT
			id,
			savings_account_id,
			type,
			amount,
			transaction_date,
			status,
			created_by,
			created_at,
			updated_at
		FROM transactions
		WHERE savings_account_id = $1
		ORDER BY transaction_date DESC, created_at DESC, id DESC
		LIMIT $2 OFFSET $3
	`

	return r.list(
		ctx,
		query,
		savingsAccountID,
		options.Limit,
		options.Offset,
	)
}

func (r *TransactionRepository) ListActiveBySavingsAccount(
	ctx context.Context,
	savingsAccountID uuid.UUID,
) ([]domain.Transaction, error) {
	const query = `
		SELECT
			id,
			savings_account_id,
			type,
			amount,
			transaction_date,
			status,
			created_by,
			created_at,
			updated_at
		FROM transactions
		WHERE savings_account_id = $1
			AND status = $2
		ORDER BY transaction_date ASC, created_at ASC, id ASC
	`

	return r.listActive(
		ctx,
		query,
		savingsAccountID,
		domain.TransactionActive,
	)
}

func (r *TransactionRepository) Update(
	ctx context.Context,
	transaction domain.Transaction,
) error {
	const query = `
		UPDATE transactions
		SET
			type = $2,
			amount = $3,
			transaction_date = $4,
			status = $5,
			updated_at = $6
		WHERE id = $1
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		transaction.ID,
		transaction.Type,
		transaction.Amount,
		transaction.TransactionDate,
		transaction.Status,
		transaction.UpdatedAt,
	)
	if err != nil {
		return translateTransactionError(
			"update transaction",
			err,
		)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf(
			"get updated transaction rows affected: %w",
			err,
		)
	}

	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}

func (r *TransactionRepository) getOne(
	ctx context.Context,
	query string,
	args ...any,
) (domain.Transaction, error) {
	transaction, err := scanTransaction(
		r.db.QueryRowContext(
			ctx,
			query,
			args...,
		),
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Transaction{}, repository.ErrNotFound
		}

		return domain.Transaction{}, fmt.Errorf(
			"get transaction: %w",
			err,
		)
	}

	return transaction, nil
}

func (r *TransactionRepository) list(
	ctx context.Context,
	query string,
	savingsAccountID uuid.UUID,
	limit int,
	offset int,
) ([]domain.Transaction, error) {
	rows, err := r.db.QueryContext(
		ctx,
		query,
		savingsAccountID,
		limit,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	defer rows.Close()

	transactions := make([]domain.Transaction, 0)

	for rows.Next() {
		transaction, err := scanTransaction(rows)
		if err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}

		transactions = append(transactions, transaction)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate transactions: %w", err)
	}

	return transactions, nil
}

func (r *TransactionRepository) listActive(
	ctx context.Context,
	query string,
	savingsAccountID uuid.UUID,
	status domain.TransactionStatus,
) ([]domain.Transaction, error) {
	rows, err := r.db.QueryContext(
		ctx,
		query,
		savingsAccountID,
		status,
	)
	if err != nil {
		return nil, fmt.Errorf("list active transactions: %w", err)
	}
	defer rows.Close()

	transactions := make([]domain.Transaction, 0)

	for rows.Next() {
		transaction, err := scanTransaction(rows)
		if err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}

		transactions = append(transactions, transaction)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active transactions: %w", err)
	}

	return transactions, nil
}

type transactionScanner interface {
	Scan(dest ...any) error
}

func scanTransaction(scanner transactionScanner) (domain.Transaction, error) {
	var transaction domain.Transaction
	var amount string

	err := scanner.Scan(
		&transaction.ID,
		&transaction.SavingsAccountID,
		&transaction.Type,
		&amount,
		&transaction.TransactionDate,
		&transaction.Status,
		&transaction.CreatedBy,
		&transaction.CreatedAt,
		&transaction.UpdatedAt,
	)
	if err != nil {
		return domain.Transaction{}, err
	}

	money, err := parseTransactionAmount(amount)
	if err != nil {
		return domain.Transaction{}, err
	}

	transaction.Amount = money

	return transaction, nil
}

func parseTransactionAmount(value string) (domain.Money, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("transaction amount is empty")
	}

	if strings.ContainsAny(value, "eE") {
		return 0, fmt.Errorf(
			"transaction amount has unsupported format %q",
			value,
		)
	}

	parts := strings.Split(value, ".")
	if len(parts) > 2 {
		return 0, fmt.Errorf(
			"transaction amount has invalid format %q",
			value,
		)
	}

	whole := parts[0]
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
	}

	if fraction != "" && strings.Trim(fraction, "0") != "" {
		return 0, fmt.Errorf(
			"transaction amount is not an integer value: %q",
			value,
		)
	}

	amount, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, fmt.Errorf(
			"parse transaction amount %q: %w",
			value,
			err,
		)
	}

	return domain.Money(amount), nil
}

func translateTransactionError(
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
