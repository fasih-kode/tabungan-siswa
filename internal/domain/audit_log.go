package domain

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
)

type AuditEntityType string

const (
	AuditEntityUser                   AuditEntityType = "USER"
	AuditEntityAcademicYear           AuditEntityType = "ACADEMIC_YEAR"
	AuditEntityClass                  AuditEntityType = "CLASS"
	AuditEntityStudent                AuditEntityType = "STUDENT"
	AuditEntityStudentClassHistory    AuditEntityType = "STUDENT_CLASS_HISTORY"
	AuditEntityTeacherClassAssignment AuditEntityType = "TEACHER_CLASS_ASSIGNMENT"
	AuditEntityClassTransferRequest   AuditEntityType = "CLASS_TRANSFER_REQUEST"
	AuditEntitySavingsAccount         AuditEntityType = "SAVINGS_ACCOUNT"
	AuditEntityTransaction            AuditEntityType = "TRANSACTION"
	AuditEntitySavingsSettlement      AuditEntityType = "SAVINGS_SETTLEMENT"
)

func (e AuditEntityType) IsValid() bool {
	switch e {
	case
		AuditEntityUser,
		AuditEntityAcademicYear,
		AuditEntityClass,
		AuditEntityStudent,
		AuditEntityStudentClassHistory,
		AuditEntityTeacherClassAssignment,
		AuditEntityClassTransferRequest,
		AuditEntitySavingsAccount,
		AuditEntityTransaction,
		AuditEntitySavingsSettlement:
		return true
	default:
		return false
	}
}

type AuditLog struct {
	ID          uuid.UUID
	ActorUserID *uuid.UUID
	Action      string
	EntityType  AuditEntityType
	EntityID    uuid.UUID
	OccurredAt  time.Time
	BeforeData  json.RawMessage
	AfterData   json.RawMessage
}

func NewAuditLog(
	actorUserID *uuid.UUID,
	action string,
	entityType AuditEntityType,
	entityID uuid.UUID,
	beforeData json.RawMessage,
	afterData json.RawMessage,
	occurredAt time.Time,
) (AuditLog, error) {
	if strings.TrimSpace(action) == "" {
		return AuditLog{}, ErrEmptyName
	}

	if !entityType.IsValid() {
		return AuditLog{}, ErrInvalidValue
	}

	if entityID == uuid.Nil {
		return AuditLog{}, ErrInvalidID
	}

	if beforeData != nil && !json.Valid(beforeData) {
		return AuditLog{}, ErrInvalidValue
	}

	if afterData != nil && !json.Valid(afterData) {
		return AuditLog{}, ErrInvalidValue
	}

	return AuditLog{
		ID:          uuid.New(),
		ActorUserID: actorUserID,
		Action:      action,
		EntityType:  entityType,
		EntityID:    entityID,
		OccurredAt:  occurredAt,
		BeforeData:  beforeData,
		AfterData:   afterData,
	}, nil
}
