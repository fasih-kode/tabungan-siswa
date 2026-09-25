package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func newTestClassTransferRequest(t *testing.T) ClassTransferRequest {
	t.Helper()

	request, err := NewClassTransferRequest(
		uuid.New(),
		uuid.New(),
		uuid.New(),
		uuid.New(),
		uuid.New(),
	)
	if err != nil {
		t.Fatalf("NewClassTransferRequest() error = %v", err)
	}

	return request
}

func TestNewClassTransferRequest(t *testing.T) {
	studentID := uuid.New()
	academicYearID := uuid.New()
	fromClassID := uuid.New()
	toClassID := uuid.New()
	requestedBy := uuid.New()

	request, err := NewClassTransferRequest(
		studentID,
		academicYearID,
		fromClassID,
		toClassID,
		requestedBy,
	)
	if err != nil {
		t.Fatalf("NewClassTransferRequest() error = %v", err)
	}

	if request.ID == uuid.Nil {
		t.Fatal("request ID must not be nil")
	}

	if request.StudentID != studentID {
		t.Fatalf("StudentID = %v, want %v", request.StudentID, studentID)
	}

	if request.AcademicYearID != academicYearID {
		t.Fatalf(
			"AcademicYearID = %v, want %v",
			request.AcademicYearID,
			academicYearID,
		)
	}

	if request.FromClassID != fromClassID {
		t.Fatalf(
			"FromClassID = %v, want %v",
			request.FromClassID,
			fromClassID,
		)
	}

	if request.ToClassID != toClassID {
		t.Fatalf(
			"ToClassID = %v, want %v",
			request.ToClassID,
			toClassID,
		)
	}

	if request.RequestedBy != requestedBy {
		t.Fatalf(
			"RequestedBy = %v, want %v",
			request.RequestedBy,
			requestedBy,
		)
	}

	if request.Status != TransferPending {
		t.Fatalf(
			"Status = %v, want %v",
			request.Status,
			TransferPending,
		)
	}

	if !request.IsPending() {
		t.Fatal("request should be pending")
	}

	if request.ReviewedBy != nil {
		t.Fatal("ReviewedBy must be nil for a pending request")
	}

	if request.ReviewedAt != nil {
		t.Fatal("ReviewedAt must be nil for a pending request")
	}

	if request.RejectionReason != nil {
		t.Fatal("RejectionReason must be nil for a pending request")
	}
}

func TestNewClassTransferRequestRejectsNilStudentID(t *testing.T) {
	_, err := NewClassTransferRequest(
		uuid.Nil,
		uuid.New(),
		uuid.New(),
		uuid.New(),
		uuid.New(),
	)

	if err != ErrInvalidID {
		t.Fatalf(
			"NewClassTransferRequest() error = %v, want %v",
			err,
			ErrInvalidID,
		)
	}
}

func TestNewClassTransferRequestRejectsNilAcademicYearID(t *testing.T) {
	_, err := NewClassTransferRequest(
		uuid.New(),
		uuid.Nil,
		uuid.New(),
		uuid.New(),
		uuid.New(),
	)

	if err != ErrInvalidID {
		t.Fatalf(
			"NewClassTransferRequest() error = %v, want %v",
			err,
			ErrInvalidID,
		)
	}
}

func TestNewClassTransferRequestRejectsNilFromClassID(t *testing.T) {
	_, err := NewClassTransferRequest(
		uuid.New(),
		uuid.New(),
		uuid.Nil,
		uuid.New(),
		uuid.New(),
	)

	if err != ErrInvalidID {
		t.Fatalf(
			"NewClassTransferRequest() error = %v, want %v",
			err,
			ErrInvalidID,
		)
	}
}

func TestNewClassTransferRequestRejectsNilToClassID(t *testing.T) {
	_, err := NewClassTransferRequest(
		uuid.New(),
		uuid.New(),
		uuid.New(),
		uuid.Nil,
		uuid.New(),
	)

	if err != ErrInvalidID {
		t.Fatalf(
			"NewClassTransferRequest() error = %v, want %v",
			err,
			ErrInvalidID,
		)
	}
}

func TestNewClassTransferRequestRejectsNilRequestedBy(t *testing.T) {
	_, err := NewClassTransferRequest(
		uuid.New(),
		uuid.New(),
		uuid.New(),
		uuid.New(),
		uuid.Nil,
	)

	if err != ErrInvalidID {
		t.Fatalf(
			"NewClassTransferRequest() error = %v, want %v",
			err,
			ErrInvalidID,
		)
	}
}

func TestNewClassTransferRequestRejectsSameClass(t *testing.T) {
	classID := uuid.New()

	_, err := NewClassTransferRequest(
		uuid.New(),
		uuid.New(),
		classID,
		classID,
		uuid.New(),
	)
	if err != ErrInvalidTransition {
		t.Fatalf(
			"NewClassTransferRequest() error = %v, want %v",
			err,
			ErrInvalidTransition,
		)
	}
}

func TestClassTransferRequestApprove(t *testing.T) {
	request := newTestClassTransferRequest(t)

	reviewedBy := uuid.New()
	reviewedAt := time.Date(
		2026,
		9,
		25,
		10,
		30,
		0,
		0,
		time.UTC,
	)

	if err := request.Approve(reviewedBy, reviewedAt); err != nil {
		t.Fatalf("Approve() error = %v", err)
	}

	if request.Status != TransferApproved {
		t.Fatalf(
			"Status = %v, want %v",
			request.Status,
			TransferApproved,
		)
	}

	if !request.IsApproved() {
		t.Fatal("request should be approved")
	}

	if request.IsPending() {
		t.Fatal("request should not be pending")
	}

	if request.ReviewedBy == nil {
		t.Fatal("ReviewedBy must not be nil")
	}

	if *request.ReviewedBy != reviewedBy {
		t.Fatalf(
			"ReviewedBy = %v, want %v",
			*request.ReviewedBy,
			reviewedBy,
		)
	}

	if request.ReviewedAt == nil {
		t.Fatal("ReviewedAt must not be nil")
	}

	if !request.ReviewedAt.Equal(reviewedAt) {
		t.Fatalf(
			"ReviewedAt = %v, want %v",
			*request.ReviewedAt,
			reviewedAt,
		)
	}

	if request.RejectionReason != nil {
		t.Fatal("RejectionReason must be nil after approval")
	}
}

func TestClassTransferRequestApproveRejectsNilReviewedBy(t *testing.T) {
	request := newTestClassTransferRequest(t)

	err := request.Approve(
		uuid.Nil,
		time.Now(),
	)
	if err != ErrInvalidID {
		t.Fatalf(
			"Approve() error = %v, want %v",
			err,
			ErrInvalidID,
		)
	}

	if request.Status != TransferPending {
		t.Fatalf(
			"Status = %v, want %v",
			request.Status,
			TransferPending,
		)
	}
}

func TestClassTransferRequestCannotApproveTwice(t *testing.T) {
	request := newTestClassTransferRequest(t)

	if err := request.Approve(uuid.New(), time.Now()); err != nil {
		t.Fatalf("first Approve() error = %v", err)
	}

	if err := request.Approve(uuid.New(), time.Now()); err != ErrInvalidTransition {
		t.Fatalf(
			"second Approve() error = %v, want %v",
			err,
			ErrInvalidTransition,
		)
	}
}

func TestClassTransferRequestReject(t *testing.T) {
	request := newTestClassTransferRequest(t)

	reviewedBy := uuid.New()
	reviewedAt := time.Date(
		2026,
		9,
		25,
		11,
		0,
		0,
		0,
		time.UTC,
	)
	reason := "Siswa tetap di kelas asal"

	if err := request.Reject(reviewedBy, reviewedAt, reason); err != nil {
		t.Fatalf("Reject() error = %v", err)
	}

	if request.Status != TransferRejected {
		t.Fatalf(
			"Status = %v, want %v",
			request.Status,
			TransferRejected,
		)
	}

	if !request.IsRejected() {
		t.Fatal("request should be rejected")
	}

	if request.IsPending() {
		t.Fatal("request should not be pending")
	}

	if request.ReviewedBy == nil {
		t.Fatal("ReviewedBy must not be nil")
	}

	if *request.ReviewedBy != reviewedBy {
		t.Fatalf(
			"ReviewedBy = %v, want %v",
			*request.ReviewedBy,
			reviewedBy,
		)
	}

	if request.ReviewedAt == nil {
		t.Fatal("ReviewedAt must not be nil")
	}

	if !request.ReviewedAt.Equal(reviewedAt) {
		t.Fatalf(
			"ReviewedAt = %v, want %v",
			*request.ReviewedAt,
			reviewedAt,
		)
	}

	if request.RejectionReason == nil {
		t.Fatal("RejectionReason must not be nil")
	}

	if *request.RejectionReason != reason {
		t.Fatalf(
			"RejectionReason = %q, want %q",
			*request.RejectionReason,
			reason,
		)
	}
}

func TestClassTransferRequestRejectsNilReviewedBy(t *testing.T) {
	request := newTestClassTransferRequest(t)

	err := request.Reject(
		uuid.Nil,
		time.Now(),
		"Alasan penolakan",
	)
	if err != ErrInvalidID {
		t.Fatalf(
			"Reject() error = %v, want %v",
			err,
			ErrInvalidID,
		)
	}

	if request.Status != TransferPending {
		t.Fatalf(
			"Status = %v, want %v",
			request.Status,
			TransferPending,
		)
	}
}

func TestClassTransferRequestRejectsEmptyReason(t *testing.T) {
	request := newTestClassTransferRequest(t)

	err := request.Reject(
		uuid.New(),
		time.Now(),
		"   ",
	)
	if err != ErrEmptyRejectionReason {
		t.Fatalf(
			"Reject() error = %v, want %v",
			err,
			ErrEmptyRejectionReason,
		)
	}

	if request.Status != TransferPending {
		t.Fatalf(
			"Status = %v, want %v",
			request.Status,
			TransferPending,
		)
	}
}

func TestClassTransferRequestCannotRejectTwice(t *testing.T) {
	request := newTestClassTransferRequest(t)

	if err := request.Reject(
		uuid.New(),
		time.Now(),
		"Alasan penolakan",
	); err != nil {
		t.Fatalf("first Reject() error = %v", err)
	}

	if err := request.Reject(
		uuid.New(),
		time.Now(),
		"Alasan lain",
	); err != ErrInvalidTransition {
		t.Fatalf(
			"second Reject() error = %v, want %v",
			err,
			ErrInvalidTransition,
		)
	}
}

func TestClassTransferRequestCannotApproveAfterReject(t *testing.T) {
	request := newTestClassTransferRequest(t)

	if err := request.Reject(
		uuid.New(),
		time.Now(),
		"Alasan penolakan",
	); err != nil {
		t.Fatalf("Reject() error = %v", err)
	}

	if err := request.Approve(uuid.New(), time.Now()); err != ErrInvalidTransition {
		t.Fatalf(
			"Approve() after reject error = %v, want %v",
			err,
			ErrInvalidTransition,
		)
	}
}

func TestClassTransferRequestCannotRejectAfterApprove(t *testing.T) {
	request := newTestClassTransferRequest(t)

	if err := request.Approve(uuid.New(), time.Now()); err != nil {
		t.Fatalf("Approve() error = %v", err)
	}

	if err := request.Reject(
		uuid.New(),
		time.Now(),
		"Alasan penolakan",
	); err != ErrInvalidTransition {
		t.Fatalf(
			"Reject() after approve error = %v, want %v",
			err,
			ErrInvalidTransition,
		)
	}
}
