package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/platform/database"
	"github.com/fasih/tabungan-siswa/internal/repository"
)

type AuditRepository struct {
	db database.DBTX
}

func NewAuditRepository(db database.DBTX) *AuditRepository {
	return &AuditRepository{db: db}
}

func (r *AuditRepository) Create(ctx context.Context, auditLog domain.AuditLog) error {
	const query = `
		INSERT INTO audit_logs (
			id,
			actor_user_id,
			action,
			entity_type,
			entity_id,
			occurred_at,
			before_data,
			after_data
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	var actorUserID any
	if auditLog.ActorUserID != nil {
		actorUserID = *auditLog.ActorUserID
	}

	var beforeData any
	if auditLog.BeforeData != nil {
		beforeData = []byte(auditLog.BeforeData)
	}

	var afterData any
	if auditLog.AfterData != nil {
		afterData = []byte(auditLog.AfterData)
	}

	_, err := r.db.ExecContext(
		ctx,
		query,
		auditLog.ID,
		actorUserID,
		auditLog.Action,
		string(auditLog.EntityType),
		auditLog.EntityID,
		auditLog.OccurredAt,
		beforeData,
		afterData,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf(
				"%w: audit log %s",
				repository.ErrConflict,
				auditLog.ID,
			)
		}

		return err
	}

	return nil
}

func (r *AuditRepository) ListByEntity(
	ctx context.Context,
	entityType domain.AuditEntityType,
	entityID uuid.UUID,
	options repository.ListOptions,
) ([]domain.AuditLog, error) {
	const query = `
		SELECT
			id,
			actor_user_id,
			action,
			entity_type,
			entity_id,
			occurred_at,
			before_data,
			after_data
		FROM audit_logs
		WHERE entity_type = $1
		  AND entity_id = $2
		ORDER BY occurred_at DESC, id DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.db.QueryContext(
		ctx,
		query,
		string(entityType),
		entityID,
		options.Limit,
		options.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	auditLogs, err := scanAuditLogs(rows)
	if err != nil {
		return nil, err
	}

	return auditLogs, nil
}

func (r *AuditRepository) ListByActor(
	ctx context.Context,
	actorUserID uuid.UUID,
	options repository.ListOptions,
) ([]domain.AuditLog, error) {
	const query = `
		SELECT
			id,
			actor_user_id,
			action,
			entity_type,
			entity_id,
			occurred_at,
			before_data,
			after_data
		FROM audit_logs
		WHERE actor_user_id = $1
		ORDER BY occurred_at DESC, id DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(
		ctx,
		query,
		actorUserID,
		options.Limit,
		options.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	auditLogs, err := scanAuditLogs(rows)
	if err != nil {
		return nil, err
	}

	return auditLogs, nil
}

type auditRows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}

func scanAuditLogs(rows auditRows) ([]domain.AuditLog, error) {
	auditLogs := make([]domain.AuditLog, 0)

	for rows.Next() {
		var (
			auditLog   domain.AuditLog
			actor      uuid.NullUUID
			entityType string
			beforeData []byte
			afterData  []byte
		)

		if err := rows.Scan(
			&auditLog.ID,
			&actor,
			&auditLog.Action,
			&entityType,
			&auditLog.EntityID,
			&auditLog.OccurredAt,
			&beforeData,
			&afterData,
		); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, repository.ErrNotFound
			}

			return nil, err
		}

		auditLog.ActorUserID = nullableUUID(actor)
		auditLog.EntityType = domain.AuditEntityType(entityType)

		if beforeData != nil {
			auditLog.BeforeData = json.RawMessage(beforeData)
		}

		if afterData != nil {
			auditLog.AfterData = json.RawMessage(afterData)
		}

		auditLogs = append(auditLogs, auditLog)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return auditLogs, nil
}

func nullableUUID(value uuid.NullUUID) *uuid.UUID {
	if !value.Valid {
		return nil
	}

	id := value.UUID
	return &id
}
