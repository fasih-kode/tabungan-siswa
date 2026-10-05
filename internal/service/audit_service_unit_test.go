package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

type auditServiceFakeAuditRepository struct {
	repository.AuditRepository

	createFn       func(context.Context, domain.AuditLog) error
	listByEntityFn func(context.Context, domain.AuditEntityType, uuid.UUID, repository.ListOptions) ([]domain.AuditLog, error)
	listByActorFn  func(context.Context, uuid.UUID, repository.ListOptions) ([]domain.AuditLog, error)

	createCalls       int
	listByEntityCalls int
	listByActorCalls  int
	lastCreated       domain.AuditLog
	lastEntityType    domain.AuditEntityType
	lastEntityID      uuid.UUID
	lastActorID       uuid.UUID
	lastListOptions   repository.ListOptions
}

func (f *auditServiceFakeAuditRepository) Create(
	ctx context.Context,
	auditLog domain.AuditLog,
) error {
	f.createCalls++
	f.lastCreated = auditLog
	if f.createFn != nil {
		return f.createFn(ctx, auditLog)
	}
	return nil
}

func (f *auditServiceFakeAuditRepository) ListByEntity(
	ctx context.Context,
	entityType domain.AuditEntityType,
	entityID uuid.UUID,
	options repository.ListOptions,
) ([]domain.AuditLog, error) {
	f.listByEntityCalls++
	f.lastEntityType = entityType
	f.lastEntityID = entityID
	f.lastListOptions = options
	if f.listByEntityFn != nil {
		return f.listByEntityFn(ctx, entityType, entityID, options)
	}
	return nil, nil
}

func (f *auditServiceFakeAuditRepository) ListByActor(
	ctx context.Context,
	actorUserID uuid.UUID,
	options repository.ListOptions,
) ([]domain.AuditLog, error) {
	f.listByActorCalls++
	f.lastActorID = actorUserID
	f.lastListOptions = options
	if f.listByActorFn != nil {
		return f.listByActorFn(ctx, actorUserID, options)
	}
	return nil, nil
}

type auditServiceFakeUOWManager struct{}

func (auditServiceFakeUOWManager) Begin(context.Context) (repository.UnitOfWork, error) {
	return nil, errors.New("not used")
}

func newAuditServiceForTest(t *testing.T, auditRepo repository.AuditRepository) *auditService {
	t.Helper()

	svc, err := NewAuditService(Dependencies{
		Repositories: repository.RepositorySet{
			AuditLogs: auditRepo,
		},
		UOW: auditServiceFakeUOWManager{},
	})
	if err != nil {
		t.Fatalf("NewAuditService() error = %v", err)
	}

	return svc
}

func TestAuditServiceRecordCreatesAuditLog(t *testing.T) {
	auditRepo := &auditServiceFakeAuditRepository{}
	svc := newAuditServiceForTest(t, auditRepo)
	actor := Actor{UserID: uuid.New(), Role: domain.RoleAdmin}
	entityID := uuid.New()

	before := []byte(`{"status":"OPEN"}`)
	after := []byte(`{"status":"SETTLED"}`)
	start := time.Now()

	got, err := svc.Record(context.Background(), RecordAuditInput{
		Actor:      actor,
		Action:     "SETTLE",
		EntityType: domain.AuditEntitySavingsAccount,
		EntityID:   entityID,
		Before:     before,
		After:      after,
	})
	if err != nil {
		t.Fatalf("Record() error = %v", err)
	}

	if got.AuditLog == nil {
		t.Fatal("AuditLog is nil")
	}
	if got.AuditLog.ActorUserID == nil || *got.AuditLog.ActorUserID != actor.UserID {
		t.Fatalf("ActorUserID = %v, want %v", got.AuditLog.ActorUserID, actor.UserID)
	}
	if got.AuditLog.Action != "SETTLE" {
		t.Fatalf("Action = %q, want SETTLE", got.AuditLog.Action)
	}
	if got.AuditLog.EntityType != domain.AuditEntitySavingsAccount {
		t.Fatalf("EntityType = %q, want %q", got.AuditLog.EntityType, domain.AuditEntitySavingsAccount)
	}
	if got.AuditLog.EntityID != entityID {
		t.Fatalf("EntityID = %v, want %v", got.AuditLog.EntityID, entityID)
	}
	if string(got.AuditLog.BeforeData) != string(before) || string(got.AuditLog.AfterData) != string(after) {
		t.Fatalf("Before/After = %s/%s, want %s/%s", got.AuditLog.BeforeData, got.AuditLog.AfterData, before, after)
	}
	if got.AuditLog.OccurredAt.Before(start) || got.AuditLog.OccurredAt.After(time.Now()) {
		t.Fatalf("OccurredAt = %v, want timestamp within Record() execution window", got.AuditLog.OccurredAt)
	}
	if auditRepo.createCalls != 1 {
		t.Fatalf("Create calls = %d, want 1", auditRepo.createCalls)
	}
}

func TestAuditServiceRecordRejectsInvalidActor(t *testing.T) {
	auditRepo := &auditServiceFakeAuditRepository{}
	svc := newAuditServiceForTest(t, auditRepo)

	_, err := svc.Record(context.Background(), RecordAuditInput{
		Actor:      Actor{UserID: uuid.Nil, Role: domain.RoleAdmin},
		Action:     "CREATE",
		EntityType: domain.AuditEntityUser,
		EntityID:   uuid.New(),
		After:      []byte(`{}`),
	})
	if !errors.Is(err, ErrInvalidActor) {
		t.Fatalf("Record() error = %v, want ErrInvalidActor", err)
	}
	if auditRepo.createCalls != 0 {
		t.Fatalf("Create calls = %d, want 0", auditRepo.createCalls)
	}
}

func TestAuditServiceRecordRejectsInvalidJSON(t *testing.T) {
	auditRepo := &auditServiceFakeAuditRepository{}
	svc := newAuditServiceForTest(t, auditRepo)

	_, err := svc.Record(context.Background(), RecordAuditInput{
		Actor:      Actor{UserID: uuid.New(), Role: domain.RoleAdmin},
		Action:     "CREATE",
		EntityType: domain.AuditEntityUser,
		EntityID:   uuid.New(),
		After:      []byte(`not-json`),
	})
	if !errors.Is(err, domain.ErrInvalidValue) {
		t.Fatalf("Record() error = %v, want ErrInvalidValue", err)
	}
	if auditRepo.createCalls != 0 {
		t.Fatalf("Create calls = %d, want 0", auditRepo.createCalls)
	}
}

func TestAuditServiceListByEntityAdmin(t *testing.T) {
	entityID := uuid.New()
	log := domain.AuditLog{
		ID:         uuid.New(),
		EntityType: domain.AuditEntityTransaction,
		EntityID:   entityID,
		Action:     "CREATE",
	}
	auditRepo := &auditServiceFakeAuditRepository{
		listByEntityFn: func(
			context.Context,
			domain.AuditEntityType,
			uuid.UUID,
			repository.ListOptions,
		) ([]domain.AuditLog, error) {
			return []domain.AuditLog{log}, nil
		},
	}
	svc := newAuditServiceForTest(t, auditRepo)

	got, err := svc.ListByEntity(context.Background(), ListAuditByEntityInput{
		Actor:      Actor{UserID: uuid.New(), Role: domain.RoleAdmin},
		EntityType: domain.AuditEntityTransaction,
		EntityID:   entityID,
		Options:    ListOptions{Limit: 10, Offset: 20},
	})
	if err != nil {
		t.Fatalf("ListByEntity() error = %v", err)
	}
	if len(got.AuditLogs) != 1 || got.AuditLogs[0].ID != log.ID {
		t.Fatalf("AuditLogs = %#v, want one log %v", got.AuditLogs, log.ID)
	}
	if auditRepo.lastEntityType != domain.AuditEntityTransaction || auditRepo.lastEntityID != entityID {
		t.Fatalf("entity args = %q/%v, want %q/%v", auditRepo.lastEntityType, auditRepo.lastEntityID, domain.AuditEntityTransaction, entityID)
	}
	if auditRepo.lastListOptions.Limit != 10 || auditRepo.lastListOptions.Offset != 20 {
		t.Fatalf("ListOptions = %#v, want 10/20", auditRepo.lastListOptions)
	}
}

func TestAuditServiceListByEntityRejectsInvalidEntityType(t *testing.T) {
	auditRepo := &auditServiceFakeAuditRepository{}
	svc := newAuditServiceForTest(t, auditRepo)

	_, err := svc.ListByEntity(context.Background(), ListAuditByEntityInput{
		Actor:      Actor{UserID: uuid.New(), Role: domain.RoleAdmin},
		EntityType: domain.AuditEntityType("INVALID"),
		EntityID:   uuid.New(),
	})
	if !errors.Is(err, domain.ErrInvalidValue) {
		t.Fatalf("ListByEntity() error = %v, want ErrInvalidValue", err)
	}
	if auditRepo.listByEntityCalls != 0 {
		t.Fatalf("ListByEntity calls = %d, want 0", auditRepo.listByEntityCalls)
	}
}

func TestAuditServiceListByEntityRejectsNonAdmin(t *testing.T) {
	auditRepo := &auditServiceFakeAuditRepository{}
	svc := newAuditServiceForTest(t, auditRepo)

	_, err := svc.ListByEntity(context.Background(), ListAuditByEntityInput{
		Actor:      Actor{UserID: uuid.New(), Role: domain.RoleWaliKelas},
		EntityType: domain.AuditEntityTransaction,
		EntityID:   uuid.New(),
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("ListByEntity() error = %v, want ErrForbidden", err)
	}
	if auditRepo.listByEntityCalls != 0 {
		t.Fatalf("ListByEntity calls = %d, want 0", auditRepo.listByEntityCalls)
	}
}

func TestAuditServiceListByActorAdmin(t *testing.T) {
	actorID := uuid.New()
	log := domain.AuditLog{
		ID:          uuid.New(),
		ActorUserID: &actorID,
		EntityType:  domain.AuditEntityStudent,
		EntityID:    uuid.New(),
		Action:      "UPDATE",
	}
	auditRepo := &auditServiceFakeAuditRepository{
		listByActorFn: func(
			context.Context,
			uuid.UUID,
			repository.ListOptions,
		) ([]domain.AuditLog, error) {
			return []domain.AuditLog{log}, nil
		},
	}
	svc := newAuditServiceForTest(t, auditRepo)

	got, err := svc.ListByActor(context.Background(), ListAuditByActorInput{
		Actor:         Actor{UserID: uuid.New(), Role: domain.RoleAdmin},
		TargetActorID: actorID,
		Options:       ListOptions{Limit: 5, Offset: 15},
	})
	if err != nil {
		t.Fatalf("ListByActor() error = %v", err)
	}
	if len(got.AuditLogs) != 1 || got.AuditLogs[0].ID != log.ID {
		t.Fatalf("AuditLogs = %#v, want one log %v", got.AuditLogs, log.ID)
	}
	if auditRepo.lastActorID != actorID {
		t.Fatalf("TargetActorID = %v, want %v", auditRepo.lastActorID, actorID)
	}
	if auditRepo.lastListOptions.Limit != 5 || auditRepo.lastListOptions.Offset != 15 {
		t.Fatalf("ListOptions = %#v, want 5/15", auditRepo.lastListOptions)
	}
}

func TestAuditServiceListByActorRejectsNonAdmin(t *testing.T) {
	auditRepo := &auditServiceFakeAuditRepository{}
	svc := newAuditServiceForTest(t, auditRepo)

	_, err := svc.ListByActor(context.Background(), ListAuditByActorInput{
		Actor:         Actor{UserID: uuid.New(), Role: domain.RoleSiswa},
		TargetActorID: uuid.New(),
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("ListByActor() error = %v, want ErrForbidden", err)
	}
	if auditRepo.listByActorCalls != 0 {
		t.Fatalf("ListByActor calls = %d, want 0", auditRepo.listByActorCalls)
	}
}

func TestRecordAuditUsesSuppliedRepositorySet(t *testing.T) {
	auditRepo := &auditServiceFakeAuditRepository{}
	repos := repository.RepositorySet{AuditLogs: auditRepo}
	actor := Actor{UserID: uuid.New(), Role: domain.RoleAdmin}

	log, err := recordAudit(context.Background(), repos, RecordAuditInput{
		Actor:      actor,
		Action:     "UPDATE",
		EntityType: domain.AuditEntityStudent,
		EntityID:   uuid.New(),
		After:      []byte(`{"name":"Ahmad"}`),
	})
	if err != nil {
		t.Fatalf("recordAudit() error = %v", err)
	}
	if log.ID == uuid.Nil {
		t.Fatal("AuditLog.ID is nil")
	}
	if auditRepo.createCalls != 1 {
		t.Fatalf("Create calls = %d, want 1", auditRepo.createCalls)
	}
}
