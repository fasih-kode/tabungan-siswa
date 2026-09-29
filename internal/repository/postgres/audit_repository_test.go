package postgres_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/repository/postgres"
	"github.com/google/uuid"
)

func TestAuditRepository_CreateAndListByEntity(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewAuditRepository(db)
	ctx := context.Background()

	fixture := newAuditFixture(t, db)

	beforeData := json.RawMessage(`{"name":"Budi","status":"ACTIVE"}`)
	afterData := json.RawMessage(`{"name":"Budi","status":"LEFT"}`)

	occurredAt := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)

	auditLog := newAuditLogFixture(
		fixture.user.ID,
		fixture.entityID,
		occurredAt,
	)
	auditLog.Action = "UPDATE"
	auditLog.BeforeData = beforeData
	auditLog.AfterData = afterData

	if err := repo.Create(ctx, auditLog); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Cleanup(deleteTestAuditLog(t, db, auditLog.ID))

	got, err := repo.ListByEntity(
		ctx,
		auditLog.EntityType,
		auditLog.EntityID,
		repository.ListOptions{
			Limit:  10,
			Offset: 0,
		},
	)
	if err != nil {
		t.Fatalf("ListByEntity() error = %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}

	assertAuditLogEqual(t, auditLog, got[0])
	assertJSONEqual(t, beforeData, got[0].BeforeData)
	assertJSONEqual(t, afterData, got[0].AfterData)
}

func TestAuditRepository_CreateWithNilActor(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewAuditRepository(db)
	ctx := context.Background()

	fixture := newAuditFixture(t, db)

	auditLog := newAuditLogFixture(
		uuid.Nil,
		fixture.entityID,
		time.Date(2026, 9, 25, 11, 0, 0, 0, time.UTC),
	)
	auditLog.ActorUserID = nil
	auditLog.BeforeData = nil
	auditLog.AfterData = nil

	if err := repo.Create(ctx, auditLog); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Cleanup(deleteTestAuditLog(t, db, auditLog.ID))

	got, err := repo.ListByEntity(
		ctx,
		auditLog.EntityType,
		auditLog.EntityID,
		repository.ListOptions{
			Limit:  10,
			Offset: 0,
		},
	)
	if err != nil {
		t.Fatalf("ListByEntity() error = %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}

	if got[0].ActorUserID != nil {
		t.Fatalf(
			"ActorUserID = %v, want nil",
			got[0].ActorUserID,
		)
	}

	if got[0].BeforeData != nil {
		t.Fatalf(
			"BeforeData = %s, want nil",
			got[0].BeforeData,
		)
	}

	if got[0].AfterData != nil {
		t.Fatalf(
			"AfterData = %s, want nil",
			got[0].AfterData,
		)
	}
}

func TestAuditRepository_ListByEntityOrdersByIDWhenOccurredAtMatches(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewAuditRepository(db)
	ctx := context.Background()

	fixture := newAuditFixture(t, db)
	occurredAt := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)

	first := newAuditLogFixture(
		fixture.user.ID,
		fixture.entityID,
		occurredAt,
	)
	first.ID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

	second := newAuditLogFixture(
		fixture.user.ID,
		fixture.entityID,
		occurredAt,
	)
	second.ID = uuid.MustParse("00000000-0000-0000-0000-000000000002")

	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}
	t.Cleanup(deleteTestAuditLog(t, db, first.ID))

	if err := repo.Create(ctx, second); err != nil {
		t.Fatalf("Create(second) error = %v", err)
	}
	t.Cleanup(deleteTestAuditLog(t, db, second.ID))

	got, err := repo.ListByEntity(
		ctx,
		domain.AuditEntityStudent,
		fixture.entityID,
		repository.ListOptions{
			Limit:  10,
			Offset: 0,
		},
	)
	if err != nil {
		t.Fatalf("ListByEntity() error = %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("ListByEntity() returned %d rows, want 2", len(got))
	}

	if got[0].ID != second.ID {
		t.Fatalf(
			"ListByEntity()[0].ID = %v, want %v",
			got[0].ID,
			second.ID,
		)
	}

	if got[1].ID != first.ID {
		t.Fatalf(
			"ListByEntity()[1].ID = %v, want %v",
			got[1].ID,
			first.ID,
		)
	}
}

func TestAuditRepository_ListByEntityIsolation(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewAuditRepository(db)
	ctx := context.Background()

	fixture := newAuditFixture(t, db)

	entityID := fixture.entityID
	otherEntityID := uuid.New()

	first := newAuditLogFixture(
		fixture.user.ID,
		entityID,
		time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC),
	)
	second := newAuditLogFixture(
		fixture.user.ID,
		otherEntityID,
		time.Date(2026, 9, 25, 11, 0, 0, 0, time.UTC),
	)

	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}
	t.Cleanup(deleteTestAuditLog(t, db, first.ID))

	if err := repo.Create(ctx, second); err != nil {
		t.Fatalf("Create(second) error = %v", err)
	}
	t.Cleanup(deleteTestAuditLog(t, db, second.ID))

	got, err := repo.ListByEntity(
		ctx,
		first.EntityType,
		entityID,
		repository.ListOptions{
			Limit:  10,
			Offset: 0,
		},
	)
	if err != nil {
		t.Fatalf("ListByEntity() error = %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}

	if got[0].ID != first.ID {
		t.Fatalf("ID = %v, want %v", got[0].ID, first.ID)
	}
}

func TestAuditRepository_ListByEntityEmpty(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewAuditRepository(db)
	ctx := context.Background()

	entityID := uuid.New()

	got, err := repo.ListByEntity(
		ctx,
		domain.AuditEntityType("STUDENT"),
		entityID,
		repository.ListOptions{
			Limit:  10,
			Offset: 0,
		},
	)
	if err != nil {
		t.Fatalf("ListByEntity() error = %v", err)
	}

	if got == nil {
		t.Fatal("ListByEntity() returned nil slice")
	}

	if len(got) != 0 {
		t.Fatalf("len(got) = %d, want 0", len(got))
	}
}

func TestAuditRepository_ListByEntityPagination(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewAuditRepository(db)
	ctx := context.Background()

	fixture := newAuditFixture(t, db)

	auditLogs := make([]domain.AuditLog, 0, 4)

	for i := 0; i < 4; i++ {
		auditLog := newAuditLogFixture(
			fixture.user.ID,
			fixture.entityID,
			time.Date(
				2026,
				time.September,
				21+i,
				10,
				0,
				0,
				0,
				time.UTC,
			),
		)

		if err := repo.Create(ctx, auditLog); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		t.Cleanup(deleteTestAuditLog(t, db, auditLog.ID))
		auditLogs = append(auditLogs, auditLog)
	}

	got, err := repo.ListByEntity(
		ctx,
		domain.AuditEntityType("STUDENT"),
		fixture.entityID,
		repository.ListOptions{
			Limit:  2,
			Offset: 1,
		},
	)
	if err != nil {
		t.Fatalf("ListByEntity() error = %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}

	assertAuditLogEqual(t, auditLogs[2], got[0])
	assertAuditLogEqual(t, auditLogs[1], got[1])
}

func TestAuditRepository_ListByActor(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewAuditRepository(db)
	ctx := context.Background()

	fixture := newAuditFixture(t, db)

	first := newAuditLogFixture(
		fixture.user.ID,
		uuid.New(),
		time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC),
	)
	second := newAuditLogFixture(
		fixture.user.ID,
		uuid.New(),
		time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC),
	)
	otherActor := newTestUser(t)

	userRepo := postgres.NewUserRepository(db)
	if err := userRepo.Create(ctx, otherActor); err != nil {
		t.Fatalf("create other actor: %v", err)
	}
	t.Cleanup(func() {
		deleteTestUser(t, db, otherActor.ID)
	})

	third := newAuditLogFixture(
		otherActor.ID,
		uuid.New(),
		time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC),
	)

	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}
	t.Cleanup(deleteTestAuditLog(t, db, first.ID))

	if err := repo.Create(ctx, second); err != nil {
		t.Fatalf("Create(second) error = %v", err)
	}
	t.Cleanup(deleteTestAuditLog(t, db, second.ID))

	if err := repo.Create(ctx, third); err != nil {
		t.Fatalf("Create(third) error = %v", err)
	}
	t.Cleanup(deleteTestAuditLog(t, db, third.ID))

	got, err := repo.ListByActor(
		ctx,
		fixture.user.ID,
		repository.ListOptions{
			Limit:  10,
			Offset: 0,
		},
	)
	if err != nil {
		t.Fatalf("ListByActor() error = %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}

	assertAuditLogEqual(t, second, got[0])
	assertAuditLogEqual(t, first, got[1])
}

func TestAuditRepository_ListByActorEmpty(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewAuditRepository(db)
	ctx := context.Background()

	actorID := uuid.New()

	got, err := repo.ListByActor(
		ctx,
		actorID,
		repository.ListOptions{
			Limit:  10,
			Offset: 0,
		},
	)
	if err != nil {
		t.Fatalf("ListByActor() error = %v", err)
	}

	if got == nil {
		t.Fatal("ListByActor() returned nil slice")
	}

	if len(got) != 0 {
		t.Fatalf("len(got) = %d, want 0", len(got))
	}
}

func TestAuditRepository_ListByActorPagination(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewAuditRepository(db)
	ctx := context.Background()

	fixture := newAuditFixture(t, db)

	auditLogs := make([]domain.AuditLog, 0, 4)

	for i := 0; i < 4; i++ {
		auditLog := newAuditLogFixture(
			fixture.user.ID,
			uuid.New(),
			time.Date(
				2026,
				time.September,
				21+i,
				10,
				0,
				0,
				0,
				time.UTC,
			),
		)

		if err := repo.Create(ctx, auditLog); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		t.Cleanup(deleteTestAuditLog(t, db, auditLog.ID))
		auditLogs = append(auditLogs, auditLog)
	}

	got, err := repo.ListByActor(
		ctx,
		fixture.user.ID,
		repository.ListOptions{
			Limit:  2,
			Offset: 1,
		},
	)
	if err != nil {
		t.Fatalf("ListByActor() error = %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}

	assertAuditLogEqual(t, auditLogs[2], got[0])
	assertAuditLogEqual(t, auditLogs[1], got[1])
}

type auditFixture struct {
	user     domain.User
	entityID uuid.UUID
}

func newAuditFixture(t *testing.T, db *sql.DB) auditFixture {
	t.Helper()

	ctx := context.Background()

	userRepo := postgres.NewUserRepository(db)

	user := newTestUser(t)
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("create fixture user: %v", err)
	}

	t.Cleanup(func() {
		deleteTestUser(t, db, user.ID)
	})

	return auditFixture{
		user:     user,
		entityID: uuid.New(),
	}
}

func newAuditLogFixture(
	actorUserID uuid.UUID,
	entityID uuid.UUID,
	occurredAt time.Time,
) domain.AuditLog {
	auditLog := domain.AuditLog{
		ID:         uuid.New(),
		Action:     "CREATE",
		EntityType: domain.AuditEntityType("STUDENT"),
		EntityID:   entityID,
		OccurredAt: occurredAt,
	}

	if actorUserID != uuid.Nil {
		auditLog.ActorUserID = &actorUserID
	}

	return auditLog
}

func deleteTestAuditLog(t *testing.T, db *sql.DB, id uuid.UUID) func() {
	t.Helper()

	return func() {
		t.Helper()

		ctx := context.Background()

		if _, err := db.ExecContext(
			ctx,
			`DELETE FROM audit_logs WHERE id = $1`,
			id,
		); err != nil {
			t.Fatalf("delete test audit log %s: %v", id, err)
		}
	}
}

func assertAuditLogEqual(
	t *testing.T,
	want domain.AuditLog,
	got domain.AuditLog,
) {
	t.Helper()

	if got.ID != want.ID {
		t.Fatalf("ID = %v, want %v", got.ID, want.ID)
	}

	switch {
	case want.ActorUserID == nil && got.ActorUserID == nil:
	case want.ActorUserID != nil && got.ActorUserID != nil:
		if *got.ActorUserID != *want.ActorUserID {
			t.Fatalf(
				"ActorUserID = %v, want %v",
				*got.ActorUserID,
				*want.ActorUserID,
			)
		}
	default:
		t.Fatalf(
			"ActorUserID = %v, want %v",
			got.ActorUserID,
			want.ActorUserID,
		)
	}

	if got.Action != want.Action {
		t.Fatalf("Action = %q, want %q", got.Action, want.Action)
	}

	if got.EntityType != want.EntityType {
		t.Fatalf(
			"EntityType = %q, want %q",
			got.EntityType,
			want.EntityType,
		)
	}

	if got.EntityID != want.EntityID {
		t.Fatalf("EntityID = %v, want %v", got.EntityID, want.EntityID)
	}

	if !got.OccurredAt.Equal(want.OccurredAt) {
		t.Fatalf(
			"OccurredAt = %v, want %v",
			got.OccurredAt,
			want.OccurredAt,
		)
	}

	assertJSONEqual(t, want.BeforeData, got.BeforeData)
	assertJSONEqual(t, want.AfterData, got.AfterData)
}

func assertJSONEqual(
	t *testing.T,
	want json.RawMessage,
	got json.RawMessage,
) {
	t.Helper()

	if want == nil && got == nil {
		return
	}

	if (want == nil) != (got == nil) {
		t.Fatalf(
			"JSON mismatch: got %s, want %s",
			got,
			want,
		)
	}

	var wantValue any
	var gotValue any

	if err := json.Unmarshal(want, &wantValue); err != nil {
		t.Fatalf("unmarshal want JSON: %v", err)
	}

	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatalf("unmarshal got JSON: %v", err)
	}

	wantBytes, err := json.Marshal(wantValue)
	if err != nil {
		t.Fatalf("marshal want JSON: %v", err)
	}

	gotBytes, err := json.Marshal(gotValue)
	if err != nil {
		t.Fatalf("marshal got JSON: %v", err)
	}

	if !bytes.Equal(wantBytes, gotBytes) {
		t.Fatalf(
			"JSON mismatch: got %s, want %s",
			got,
			want,
		)
	}
}
