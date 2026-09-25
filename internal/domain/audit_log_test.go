package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewAuditLog(t *testing.T) {
	actorID := uuid.New()
	entityID := uuid.New()
	occurredAt := time.Date(
		2026,
		9,
		25,
		10,
		30,
		0,
		0,
		time.UTC,
	)

	beforeData := json.RawMessage(`{"name":"Budi"}`)
	afterData := json.RawMessage(`{"name":"Budi Santoso"}`)

	log, err := NewAuditLog(
		&actorID,
		"UPDATE",
		AuditEntityStudent,
		entityID,
		beforeData,
		afterData,
		occurredAt,
	)
	if err != nil {
		t.Fatalf("NewAuditLog() error = %v", err)
	}

	if log.ID == uuid.Nil {
		t.Fatal("audit log ID must not be nil")
	}

	if log.ActorUserID == nil {
		t.Fatal("ActorUserID must not be nil")
	}

	if *log.ActorUserID != actorID {
		t.Fatalf("ActorUserID = %v, want %v", *log.ActorUserID, actorID)
	}

	if log.Action != "UPDATE" {
		t.Fatalf("Action = %q, want %q", log.Action, "UPDATE")
	}

	if log.EntityType != AuditEntityStudent {
		t.Fatalf(
			"EntityType = %v, want %v",
			log.EntityType,
			AuditEntityStudent,
		)
	}

	if log.EntityID != entityID {
		t.Fatalf("EntityID = %v, want %v", log.EntityID, entityID)
	}

	if !log.OccurredAt.Equal(occurredAt) {
		t.Fatalf(
			"OccurredAt = %v, want %v",
			log.OccurredAt,
			occurredAt,
		)
	}

	if string(log.BeforeData) != string(beforeData) {
		t.Fatalf(
			"BeforeData = %s, want %s",
			log.BeforeData,
			beforeData,
		)
	}

	if string(log.AfterData) != string(afterData) {
		t.Fatalf(
			"AfterData = %s, want %s",
			log.AfterData,
			afterData,
		)
	}
}

func TestNewAuditLogAllowsNilActor(t *testing.T) {
	entityID := uuid.New()

	log, err := NewAuditLog(
		nil,
		"SYSTEM_ACTION",
		AuditEntitySavingsSettlement,
		entityID,
		nil,
		nil,
		time.Now(),
	)
	if err != nil {
		t.Fatalf("NewAuditLog() error = %v", err)
	}

	if log.ActorUserID != nil {
		t.Fatal("ActorUserID must be nil")
	}
}

func TestNewAuditLogAllowsNilBeforeData(t *testing.T) {
	entityID := uuid.New()
	afterData := json.RawMessage(`{"status":"ACTIVE"}`)

	log, err := NewAuditLog(
		nil,
		"CREATE",
		AuditEntityStudent,
		entityID,
		nil,
		afterData,
		time.Now(),
	)
	if err != nil {
		t.Fatalf("NewAuditLog() error = %v", err)
	}

	if log.BeforeData != nil {
		t.Fatal("BeforeData must be nil")
	}

	if string(log.AfterData) != string(afterData) {
		t.Fatalf(
			"AfterData = %s, want %s",
			log.AfterData,
			afterData,
		)
	}
}

func TestNewAuditLogAllowsNilAfterData(t *testing.T) {
	entityID := uuid.New()
	beforeData := json.RawMessage(`{"status":"ACTIVE"}`)

	log, err := NewAuditLog(
		nil,
		"DELETE",
		AuditEntityStudent,
		entityID,
		beforeData,
		nil,
		time.Now(),
	)
	if err != nil {
		t.Fatalf("NewAuditLog() error = %v", err)
	}

	if string(log.BeforeData) != string(beforeData) {
		t.Fatalf(
			"BeforeData = %s, want %s",
			log.BeforeData,
			beforeData,
		)
	}

	if log.AfterData != nil {
		t.Fatal("AfterData must be nil")
	}
}

func TestNewAuditLogAllowsAllEntityTypes(t *testing.T) {
	entityTypes := []AuditEntityType{
		AuditEntityUser,
		AuditEntityAcademicYear,
		AuditEntityClass,
		AuditEntityStudent,
		AuditEntityStudentClassHistory,
		AuditEntityTeacherClassAssignment,
		AuditEntityClassTransferRequest,
		AuditEntitySavingsAccount,
		AuditEntityTransaction,
		AuditEntitySavingsSettlement,
	}

	for _, entityType := range entityTypes {
		t.Run(string(entityType), func(t *testing.T) {
			_, err := NewAuditLog(
				nil,
				"CREATE",
				entityType,
				uuid.New(),
				nil,
				nil,
				time.Now(),
			)
			if err != nil {
				t.Fatalf(
					"NewAuditLog() error = %v for entity type %v",
					err,
					entityType,
				)
			}
		})
	}
}

func TestNewAuditLogRejectsEmptyAction(t *testing.T) {
	_, err := NewAuditLog(
		nil,
		"",
		AuditEntityStudent,
		uuid.New(),
		nil,
		nil,
		time.Now(),
	)
	if err != ErrEmptyName {
		t.Fatalf(
			"NewAuditLog() error = %v, want %v",
			err,
			ErrEmptyName,
		)
	}
}

func TestNewAuditLogRejectsWhitespaceAction(t *testing.T) {
	_, err := NewAuditLog(
		nil,
		"   ",
		AuditEntityStudent,
		uuid.New(),
		nil,
		nil,
		time.Now(),
	)
	if err != ErrEmptyName {
		t.Fatalf(
			"NewAuditLog() error = %v, want %v",
			err,
			ErrEmptyName,
		)
	}
}

func TestNewAuditLogRejectsInvalidEntityType(t *testing.T) {
	invalidEntityType := AuditEntityType("INVALID")

	_, err := NewAuditLog(
		nil,
		"CREATE",
		invalidEntityType,
		uuid.New(),
		nil,
		nil,
		time.Now(),
	)
	if err != ErrInvalidValue {
		t.Fatalf(
			"NewAuditLog() error = %v, want %v",
			err,
			ErrInvalidValue,
		)
	}
}

func TestNewAuditLogRejectsNilEntityID(t *testing.T) {
	_, err := NewAuditLog(
		nil,
		"CREATE",
		AuditEntityStudent,
		uuid.Nil,
		nil,
		nil,
		time.Now(),
	)
	if err != ErrInvalidID {
		t.Fatalf(
			"NewAuditLog() error = %v, want %v",
			err,
			ErrInvalidID,
		)
	}
}

func TestNewAuditLogAllowsValidJSONValues(t *testing.T) {
	testCases := []struct {
		name string
		data json.RawMessage
	}{
		{
			name: "object",
			data: json.RawMessage(`{"name":"Budi"}`),
		},
		{
			name: "array",
			data: json.RawMessage(`[1,2,3]`),
		},
		{
			name: "string",
			data: json.RawMessage(`"Budi"`),
		},
		{
			name: "number",
			data: json.RawMessage(`123`),
		},
		{
			name: "boolean",
			data: json.RawMessage(`true`),
		},
		{
			name: "null",
			data: json.RawMessage(`null`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			log, err := NewAuditLog(
				nil,
				"CREATE",
				AuditEntityStudent,
				uuid.New(),
				tc.data,
				nil,
				time.Now(),
			)
			if err != nil {
				t.Fatalf("NewAuditLog() error = %v", err)
			}

			if string(log.BeforeData) != string(tc.data) {
				t.Fatalf(
					"BeforeData = %s, want %s",
					log.BeforeData,
					tc.data,
				)
			}
		})
	}
}

func TestNewAuditLogRejectsInvalidBeforeData(t *testing.T) {
	invalidJSON := json.RawMessage(`{"name":`)

	_, err := NewAuditLog(
		nil,
		"UPDATE",
		AuditEntityStudent,
		uuid.New(),
		invalidJSON,
		nil,
		time.Now(),
	)
	if err != ErrInvalidValue {
		t.Fatalf(
			"NewAuditLog() error = %v, want %v",
			err,
			ErrInvalidValue,
		)
	}
}

func TestNewAuditLogRejectsInvalidAfterData(t *testing.T) {
	invalidJSON := json.RawMessage(`{"status":`)

	_, err := NewAuditLog(
		nil,
		"UPDATE",
		AuditEntityStudent,
		uuid.New(),
		nil,
		invalidJSON,
		time.Now(),
	)
	if err != ErrInvalidValue {
		t.Fatalf(
			"NewAuditLog() error = %v, want %v",
			err,
			ErrInvalidValue,
		)
	}
}
