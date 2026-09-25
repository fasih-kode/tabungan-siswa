package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type ClassTransferRequest struct {
	ID              uuid.UUID
	StudentID       uuid.UUID
	AcademicYearID  uuid.UUID
	FromClassID     uuid.UUID
	ToClassID       uuid.UUID
	RequestedBy     uuid.UUID
	Status          ClassTransferRequestStatus
	ReviewedBy      *uuid.UUID
	ReviewedAt      *time.Time
	RejectionReason *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func NewClassTransferRequest(
	studentID uuid.UUID,
	academicYearID uuid.UUID,
	fromClassID uuid.UUID,
	toClassID uuid.UUID,
	requestedBy uuid.UUID,
) (ClassTransferRequest, error) {
	if studentID == uuid.Nil {
		return ClassTransferRequest{}, ErrInvalidID
	}

	if academicYearID == uuid.Nil {
		return ClassTransferRequest{}, ErrInvalidID
	}

	if fromClassID == uuid.Nil {
		return ClassTransferRequest{}, ErrInvalidID
	}

	if toClassID == uuid.Nil {
		return ClassTransferRequest{}, ErrInvalidID
	}

	if requestedBy == uuid.Nil {
		return ClassTransferRequest{}, ErrInvalidID
	}

	if fromClassID == toClassID {
		return ClassTransferRequest{}, ErrInvalidTransition
	}

	now := time.Now()

	return ClassTransferRequest{
		ID:             uuid.New(),
		StudentID:      studentID,
		AcademicYearID: academicYearID,
		FromClassID:    fromClassID,
		ToClassID:      toClassID,
		RequestedBy:    requestedBy,
		Status:         TransferPending,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func (r ClassTransferRequest) IsPending() bool {
	return r.Status == TransferPending
}

func (r ClassTransferRequest) IsApproved() bool {
	return r.Status == TransferApproved
}

func (r ClassTransferRequest) IsRejected() bool {
	return r.Status == TransferRejected
}

func (r *ClassTransferRequest) Approve(
	reviewedBy uuid.UUID,
	reviewedAt time.Time,
) error {
	if r.Status != TransferPending {
		return ErrInvalidTransition
	}

	if reviewedBy == uuid.Nil {
		return ErrInvalidID
	}

	r.Status = TransferApproved
	r.ReviewedBy = &reviewedBy
	r.ReviewedAt = &reviewedAt
	r.RejectionReason = nil
	r.UpdatedAt = time.Now()

	return nil
}

func (r *ClassTransferRequest) Reject(
	reviewedBy uuid.UUID,
	reviewedAt time.Time,
	reason string,
) error {
	if r.Status != TransferPending {
		return ErrInvalidTransition
	}

	if reviewedBy == uuid.Nil {
		return ErrInvalidID
	}

	if strings.TrimSpace(reason) == "" {
		return ErrEmptyRejectionReason
	}

	r.Status = TransferRejected
	r.ReviewedBy = &reviewedBy
	r.ReviewedAt = &reviewedAt
	r.RejectionReason = &reason
	r.UpdatedAt = time.Now()

	return nil
}
