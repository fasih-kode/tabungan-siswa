package service

import (
	"context"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/google/uuid"
)

// StudentLeavingRequestService menyediakan lifecycle proses pindah untuk pemanggil internal.
// Service belum dipasang pada ServiceSet atau endpoint aplikasi.
type StudentLeavingRequestService interface {
	Finalize(context.Context, FinalizeStudentLeavingRequestInput) (FinalizeStudentLeavingRequestOutput, error)
	// Start menerima tanggal hari ini atau tanggal lampau dalam tahun OPEN jika placement berlaku.
	// Tanggal kalender memakai WIB; timestamp audit tetap UTC.
	Start(context.Context, StartStudentLeavingRequestInput) (StartStudentLeavingRequestOutput, error)
	Get(context.Context, GetStudentLeavingRequestInput) (GetStudentLeavingRequestOutput, error)
	Cancel(context.Context, CancelStudentLeavingRequestInput) (CancelStudentLeavingRequestOutput, error)
	Readiness(context.Context, StudentLeavingReadinessInput) (StudentLeavingReadinessOutput, error)
}

type StartStudentLeavingRequestInput struct {
	Actor                Actor
	StudentID            uuid.UUID
	AcademicYearID       uuid.UUID
	EffectiveLeavingDate time.Time
}

type StartStudentLeavingRequestOutput struct {
	Request *domain.StudentLeavingRequest
}

type GetStudentLeavingRequestInput struct {
	Actor     Actor
	RequestID uuid.UUID
}

type GetStudentLeavingRequestOutput struct {
	Request *domain.StudentLeavingRequest
}

type CancelStudentLeavingRequestInput struct {
	Actor                   Actor
	StudentLeavingRequestID uuid.UUID
	Reason                  string
}

type CancelStudentLeavingRequestOutput struct {
	Request *domain.StudentLeavingRequest
}

type StudentLeavingReadinessInput struct {
	Actor                   Actor
	StudentLeavingRequestID uuid.UUID
}

type StudentLeavingReadinessBlockerCode string

const (
	LeavingBlockerPlacementInvalid              StudentLeavingReadinessBlockerCode = "PLACEMENT_INVALID"
	LeavingBlockerLaterPlacement                StudentLeavingReadinessBlockerCode = "LATER_PLACEMENT"
	LeavingBlockerPendingTransfer               StudentLeavingReadinessBlockerCode = "PENDING_TRANSFER"
	LeavingBlockerProcessNotInProgress          StudentLeavingReadinessBlockerCode = "PROCESS_NOT_IN_PROGRESS"
	LeavingBlockerReferenceYearInvalid          StudentLeavingReadinessBlockerCode = "REFERENCE_YEAR_INVALID"
	LeavingBlockerStudentNotActive              StudentLeavingReadinessBlockerCode = "STUDENT_NOT_ACTIVE"
	LeavingBlockerFinancialLinkInconsistent     StudentLeavingReadinessBlockerCode = "FINANCIAL_LINK_INCONSISTENT"
	LeavingBlockerSettlementStudentMismatch     StudentLeavingReadinessBlockerCode = "SETTLEMENT_STUDENT_MISMATCH"
	LeavingBlockerSettlementTypeMismatch        StudentLeavingReadinessBlockerCode = "SETTLEMENT_TYPE_MISMATCH"
	LeavingBlockerLateRequestStudentMismatch    StudentLeavingReadinessBlockerCode = "LATE_REQUEST_STUDENT_MISMATCH"
	LeavingBlockerLinkedLateRequestYearMismatch StudentLeavingReadinessBlockerCode = "LINKED_LATE_REQUEST_YEAR_MISMATCH"
	LeavingBlockerAccountOpen                   StudentLeavingReadinessBlockerCode = "ACCOUNT_OPEN"
	LeavingBlockerAccountOpenYearClosed         StudentLeavingReadinessBlockerCode = "ACCOUNT_OPEN_YEAR_CLOSED"
	LeavingBlockerSettlementMissing             StudentLeavingReadinessBlockerCode = "SETTLEMENT_MISSING"
	LeavingBlockerSettlementInvalid             StudentLeavingReadinessBlockerCode = "SETTLEMENT_INVALID"
	LeavingBlockerSettlementLedgerMismatch      StudentLeavingReadinessBlockerCode = "SETTLEMENT_LEDGER_MISMATCH"
	LeavingBlockerLedgerInvalid                 StudentLeavingReadinessBlockerCode = "LEDGER_INVALID"
	LeavingBlockerHandoverIncomplete            StudentLeavingReadinessBlockerCode = "HANDOVER_INCOMPLETE"
	LeavingBlockerHandoverInvalid               StudentLeavingReadinessBlockerCode = "HANDOVER_INVALID"
	LeavingBlockerLateRequestPending            StudentLeavingReadinessBlockerCode = "LATE_REQUEST_PENDING"
	LeavingBlockerLateRequestApproved           StudentLeavingReadinessBlockerCode = "LATE_REQUEST_APPROVED"
	LeavingBlockerLateRequestYearMismatch       StudentLeavingReadinessBlockerCode = "LATE_REQUEST_YEAR_MISMATCH"
	LeavingBlockerLateRequestLedgerMissing      StudentLeavingReadinessBlockerCode = "LATE_REQUEST_LEDGER_MISSING"
	LeavingBlockerLateRequestLedgerInvalid      StudentLeavingReadinessBlockerCode = "LATE_REQUEST_LEDGER_INVALID"
	LeavingBlockerTransactionLateLinkInvalid    StudentLeavingReadinessBlockerCode = "TRANSACTION_LATE_LINK_INVALID"
	LeavingBlockerTransactionAfterLeavingDate   StudentLeavingReadinessBlockerCode = "TRANSACTION_AFTER_LEAVING_DATE"
)

type StudentLeavingReadinessBlocker struct {
	Code      StudentLeavingReadinessBlockerCode
	AccountID *uuid.UUID
	RecordID  *uuid.UUID
	Message   string
}

type StudentLeavingReadinessOutput struct {
	Request              domain.StudentLeavingRequest
	Student              domain.Student
	TotalAccounts        int
	SettledAccounts      int
	OpenAccounts         int
	OutstandingHandovers int
	BlockingLateRequests int
	Blockers             []StudentLeavingReadinessBlocker
	Ready                bool
}

type FinalizeStudentLeavingRequestInput struct {
	Actor                   Actor
	StudentLeavingRequestID uuid.UUID
}
type FinalizeStudentLeavingRequestOutput struct {
	Request *domain.StudentLeavingRequest
	Student *domain.Student
}
