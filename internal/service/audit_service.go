package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

var _ AuditService = (*auditService)(nil)

func (s *auditService) Record(
	ctx context.Context,
	input RecordAuditInput,
) (RecordAuditOutput, error) {
	auditLog, err := recordAudit(ctx, s.deps.Repositories, input)
	if err != nil {
		return RecordAuditOutput{}, err
	}

	return RecordAuditOutput{
		AuditLog: &auditLog,
	}, nil
}

func (s *auditService) ListByEntity(
	ctx context.Context,
	input ListAuditByEntityInput,
) (ListAuditByEntityOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleAdmin); err != nil {
		return ListAuditByEntityOutput{}, err
	}

	if !input.EntityType.IsValid() {
		return ListAuditByEntityOutput{}, domain.ErrInvalidValue
	}

	if input.EntityID == uuid.Nil {
		return ListAuditByEntityOutput{}, domain.ErrInvalidID
	}

	auditLogs, err := s.deps.Repositories.AuditLogs.ListByEntity(
		ctx,
		input.EntityType,
		input.EntityID,
		repository.ListOptions{
			Limit:  input.Options.Limit,
			Offset: input.Options.Offset,
		},
	)
	if err != nil {
		return ListAuditByEntityOutput{}, err
	}

	return ListAuditByEntityOutput{
		AuditLogs: auditLogPointers(auditLogs),
	}, nil
}

func (s *auditService) ListByActor(
	ctx context.Context,
	input ListAuditByActorInput,
) (ListAuditByActorOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleAdmin); err != nil {
		return ListAuditByActorOutput{}, err
	}

	if input.TargetActorID == uuid.Nil {
		return ListAuditByActorOutput{}, domain.ErrInvalidID
	}

	auditLogs, err := s.deps.Repositories.AuditLogs.ListByActor(
		ctx,
		input.TargetActorID,
		repository.ListOptions{
			Limit:  input.Options.Limit,
			Offset: input.Options.Offset,
		},
	)
	if err != nil {
		return ListAuditByActorOutput{}, err
	}

	return ListAuditByActorOutput{
		AuditLogs: auditLogPointers(auditLogs),
	}, nil
}

// recordAudit creates and persists an audit log using the supplied repository set.
// The caller owns the transaction boundary. Mutation services must pass the
// RepositorySet returned by UnitOfWork.Repositories() so the audit write is
// atomic with the business mutation.
func recordAudit(
	ctx context.Context,
	repos repository.RepositorySet,
	input RecordAuditInput,
) (domain.AuditLog, error) {
	if err := input.Actor.Validate(); err != nil {
		return domain.AuditLog{}, err
	}

	if input.EntityID == uuid.Nil {
		return domain.AuditLog{}, domain.ErrInvalidID
	}

	now := time.Now()

	auditLog, err := domain.NewAuditLog(
		&input.Actor.UserID,
		input.Action,
		input.EntityType,
		input.EntityID,
		json.RawMessage(input.Before),
		json.RawMessage(input.After),
		now,
	)
	if err != nil {
		return domain.AuditLog{}, err
	}

	if repos.AuditLogs == nil {
		return domain.AuditLog{}, fmt.Errorf("%w: audit repository", ErrInvalidDependency)
	}

	if err := repos.AuditLogs.Create(ctx, auditLog); err != nil {
		return domain.AuditLog{}, err
	}

	return auditLog, nil
}

func auditLogPointers(auditLogs []domain.AuditLog) []*domain.AuditLog {
	result := make([]*domain.AuditLog, len(auditLogs))
	for i := range auditLogs {
		auditLog := auditLogs[i]
		result[i] = &auditLog
	}
	return result
}
