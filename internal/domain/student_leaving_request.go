package domain

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

type StudentLeavingRequestStatus string

const (
	StudentLeavingInProgress StudentLeavingRequestStatus = "IN_PROGRESS"
	StudentLeavingFinalized  StudentLeavingRequestStatus = "FINALIZED"
	StudentLeavingCancelled  StudentLeavingRequestStatus = "CANCELLED"
)

type StudentLeavingRequest struct {
	ID                   uuid.UUID
	StudentID            uuid.UUID
	AcademicYearID       uuid.UUID
	EffectiveLeavingDate time.Time
	Status               StudentLeavingRequestStatus
	InitiatedBy          uuid.UUID
	CreatedAt            time.Time
	FinalizedBy          *uuid.UUID
	FinalizedAt          *time.Time
	CancelledBy          *uuid.UUID
	CancelledAt          *time.Time
	CancellationReason   *string
}

func NewStudentLeavingRequest(
	studentID uuid.UUID,
	academicYearID uuid.UUID,
	effectiveLeavingDate time.Time,
	initiatedBy uuid.UUID,
) (StudentLeavingRequest, error) {
	if studentID == uuid.Nil || academicYearID == uuid.Nil || initiatedBy == uuid.Nil {
		return StudentLeavingRequest{}, ErrInvalidID
	}
	if effectiveLeavingDate.IsZero() {
		return StudentLeavingRequest{}, ErrInvalidValue
	}

	return StudentLeavingRequest{
		ID:                   uuid.New(),
		StudentID:            studentID,
		AcademicYearID:       academicYearID,
		EffectiveLeavingDate: studentLeavingDateOnly(effectiveLeavingDate),
		Status:               StudentLeavingInProgress,
		InitiatedBy:          initiatedBy,
		CreatedAt:            time.Now().UTC(),
	}, nil
}

func (r *StudentLeavingRequest) Finalize(actorID uuid.UUID, finalizedAt time.Time) error {
	if r.Status != StudentLeavingInProgress || !r.hasEmptyDecisionMetadata() {
		return ErrInvalidTransition
	}
	if actorID == uuid.Nil {
		return ErrInvalidID
	}
	if finalizedAt.IsZero() {
		return ErrInvalidValue
	}

	at := finalizedAt.UTC()
	r.Status = StudentLeavingFinalized
	r.FinalizedBy = &actorID
	r.FinalizedAt = &at
	return nil
}

func (r *StudentLeavingRequest) Cancel(
	actorID uuid.UUID,
	cancelledAt time.Time,
	reason string,
) error {
	if r.Status != StudentLeavingInProgress || !r.hasEmptyDecisionMetadata() {
		return ErrInvalidTransition
	}
	if actorID == uuid.Nil {
		return ErrInvalidID
	}
	if cancelledAt.IsZero() {
		return ErrInvalidValue
	}

	reason = strings.TrimSpace(reason)
	if reason == "" || utf8.RuneCountInString(reason) > 500 {
		return ErrInvalidValue
	}

	at := cancelledAt.UTC()
	r.Status = StudentLeavingCancelled
	r.CancelledBy = &actorID
	r.CancelledAt = &at
	r.CancellationReason = &reason
	return nil
}

func (r StudentLeavingRequest) hasEmptyDecisionMetadata() bool {
	return r.FinalizedBy == nil &&
		r.FinalizedAt == nil &&
		r.CancelledBy == nil &&
		r.CancelledAt == nil &&
		r.CancellationReason == nil
}

func studentLeavingDateOnly(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
