package domain_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/google/uuid"
)

func TestNewStudentLeavingRequest(t *testing.T) {
	location := time.FixedZone("UTC+7", 7*60*60)
	effectiveDate := time.Date(2026, 10, 27, 18, 45, 0, 0, location)

	request, err := domain.NewStudentLeavingRequest(
		uuid.New(),
		uuid.New(),
		effectiveDate,
		uuid.New(),
	)
	if err != nil {
		t.Fatalf("NewStudentLeavingRequest() error = %v", err)
	}
	if request.ID == uuid.Nil {
		t.Fatal("NewStudentLeavingRequest() returned a nil ID")
	}
	if request.Status != domain.StudentLeavingInProgress {
		t.Fatalf("Status = %q, want %q", request.Status, domain.StudentLeavingInProgress)
	}
	if got := request.EffectiveLeavingDate; !got.Equal(time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)) || got.Location() != time.UTC {
		t.Fatalf("EffectiveLeavingDate = %v, want UTC date 2026-10-27", got)
	}
	if request.CreatedAt.IsZero() || request.CreatedAt.Location() != time.UTC {
		t.Fatalf("CreatedAt = %v, want non-zero UTC timestamp", request.CreatedAt)
	}
	if request.FinalizedBy != nil || request.FinalizedAt != nil || request.CancelledBy != nil || request.CancelledAt != nil || request.CancellationReason != nil {
		t.Fatal("new process must not contain finalization or cancellation metadata")
	}
}

func TestNewStudentLeavingRequestRejectsInvalidInput(t *testing.T) {
	validStudentID := uuid.New()
	validYearID := uuid.New()
	validActorID := uuid.New()
	validDate := time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		studentID uuid.UUID
		yearID    uuid.UUID
		date      time.Time
		actorID   uuid.UUID
		wantErr   error
	}{
		{name: "missing student", studentID: uuid.Nil, yearID: validYearID, date: validDate, actorID: validActorID, wantErr: domain.ErrInvalidID},
		{name: "missing year", studentID: validStudentID, yearID: uuid.Nil, date: validDate, actorID: validActorID, wantErr: domain.ErrInvalidID},
		{name: "missing initiator", studentID: validStudentID, yearID: validYearID, date: validDate, actorID: uuid.Nil, wantErr: domain.ErrInvalidID},
		{name: "missing effective date", studentID: validStudentID, yearID: validYearID, date: time.Time{}, actorID: validActorID, wantErr: domain.ErrInvalidValue},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.NewStudentLeavingRequest(tt.studentID, tt.yearID, tt.date, tt.actorID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewStudentLeavingRequest() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestStudentLeavingRequestFinalize(t *testing.T) {
	request := newStudentLeavingRequestForTest(t)
	actorID := uuid.New()
	at := time.Date(2026, 10, 27, 15, 30, 0, 0, time.FixedZone("UTC+7", 7*60*60))

	if err := request.Finalize(actorID, at); err != nil {
		t.Fatalf("Finalize() error = %v", err)
	}
	if request.Status != domain.StudentLeavingFinalized {
		t.Fatalf("Status = %q, want %q", request.Status, domain.StudentLeavingFinalized)
	}
	if request.FinalizedBy == nil || *request.FinalizedBy != actorID {
		t.Fatalf("FinalizedBy = %v, want %s", request.FinalizedBy, actorID)
	}
	if request.FinalizedAt == nil || !request.FinalizedAt.Equal(at) || request.FinalizedAt.Location() != time.UTC {
		t.Fatalf("FinalizedAt = %v, want %v normalized to UTC", request.FinalizedAt, at)
	}
	if request.CancelledBy != nil || request.CancelledAt != nil || request.CancellationReason != nil {
		t.Fatal("finalization must not set cancellation metadata")
	}
}

func TestStudentLeavingRequestFinalizeRejectsInvalidTransitionAndMetadata(t *testing.T) {
	actorID := uuid.New()
	at := time.Now().UTC()

	tests := []struct {
		name    string
		setup   func(*domain.StudentLeavingRequest)
		actor   uuid.UUID
		at      time.Time
		wantErr error
	}{
		{name: "nil actor", actor: uuid.Nil, at: at, wantErr: domain.ErrInvalidID},
		{name: "zero timestamp", actor: actorID, at: time.Time{}, wantErr: domain.ErrInvalidValue},
		{name: "already finalized", setup: func(r *domain.StudentLeavingRequest) { r.Status = domain.StudentLeavingFinalized }, actor: actorID, at: at, wantErr: domain.ErrInvalidTransition},
		{name: "already cancelled", setup: func(r *domain.StudentLeavingRequest) { r.Status = domain.StudentLeavingCancelled }, actor: actorID, at: at, wantErr: domain.ErrInvalidTransition},
		{name: "mixed decision metadata", setup: func(r *domain.StudentLeavingRequest) { id := uuid.New(); r.CancelledBy = &id }, actor: actorID, at: at, wantErr: domain.ErrInvalidTransition},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := newStudentLeavingRequestForTest(t)
			if tt.setup != nil {
				tt.setup(&request)
			}
			if err := request.Finalize(tt.actor, tt.at); !errors.Is(err, tt.wantErr) {
				t.Fatalf("Finalize() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestStudentLeavingRequestCancel(t *testing.T) {
	request := newStudentLeavingRequestForTest(t)
	actorID := uuid.New()
	at := time.Date(2026, 10, 27, 15, 30, 0, 0, time.FixedZone("UTC+7", 7*60*60))
	reason := strings.Repeat("é", 500)

	if err := request.Cancel(actorID, at, "  "+reason+"  "); err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}
	if request.Status != domain.StudentLeavingCancelled {
		t.Fatalf("Status = %q, want %q", request.Status, domain.StudentLeavingCancelled)
	}
	if request.CancelledBy == nil || *request.CancelledBy != actorID {
		t.Fatalf("CancelledBy = %v, want %s", request.CancelledBy, actorID)
	}
	if request.CancelledAt == nil || !request.CancelledAt.Equal(at) || request.CancelledAt.Location() != time.UTC {
		t.Fatalf("CancelledAt = %v, want %v normalized to UTC", request.CancelledAt, at)
	}
	if request.CancellationReason == nil || *request.CancellationReason != reason {
		t.Fatalf("CancellationReason = %v, want trimmed reason", request.CancellationReason)
	}
	if request.FinalizedBy != nil || request.FinalizedAt != nil {
		t.Fatal("cancellation must not set finalization metadata")
	}
}

func TestStudentLeavingRequestCancelRejectsInvalidInputAndTransition(t *testing.T) {
	actorID := uuid.New()
	at := time.Now().UTC()

	tests := []struct {
		name    string
		setup   func(*domain.StudentLeavingRequest)
		actor   uuid.UUID
		at      time.Time
		reason  string
		wantErr error
	}{
		{name: "nil actor", actor: uuid.Nil, at: at, reason: "Alasan", wantErr: domain.ErrInvalidID},
		{name: "zero timestamp", actor: actorID, at: time.Time{}, reason: "Alasan", wantErr: domain.ErrInvalidValue},
		{name: "empty reason", actor: actorID, at: at, reason: "  ", wantErr: domain.ErrInvalidValue},
		{name: "reason too long", actor: actorID, at: at, reason: strings.Repeat("a", 501), wantErr: domain.ErrInvalidValue},
		{name: "already finalized", setup: func(r *domain.StudentLeavingRequest) { r.Status = domain.StudentLeavingFinalized }, actor: actorID, at: at, reason: "Alasan", wantErr: domain.ErrInvalidTransition},
		{name: "already cancelled", setup: func(r *domain.StudentLeavingRequest) { r.Status = domain.StudentLeavingCancelled }, actor: actorID, at: at, reason: "Alasan", wantErr: domain.ErrInvalidTransition},
		{name: "mixed decision metadata", setup: func(r *domain.StudentLeavingRequest) { id := uuid.New(); r.FinalizedBy = &id }, actor: actorID, at: at, reason: "Alasan", wantErr: domain.ErrInvalidTransition},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := newStudentLeavingRequestForTest(t)
			if tt.setup != nil {
				tt.setup(&request)
			}
			if err := request.Cancel(tt.actor, tt.at, tt.reason); !errors.Is(err, tt.wantErr) {
				t.Fatalf("Cancel() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func newStudentLeavingRequestForTest(t *testing.T) domain.StudentLeavingRequest {
	t.Helper()

	request, err := domain.NewStudentLeavingRequest(
		uuid.New(),
		uuid.New(),
		time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC),
		uuid.New(),
	)
	if err != nil {
		t.Fatalf("NewStudentLeavingRequest() error = %v", err)
	}
	return request
}
