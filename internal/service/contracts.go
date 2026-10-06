package service

import (
	"context"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/google/uuid"
)

type ListOptions struct {
	Limit  int
	Offset int
}

// =========================
// Authentication Service
// =========================

type AuthenticationService interface {
	Authenticate(
		ctx context.Context,
		input AuthenticateInput,
	) (AuthenticateOutput, error)
	Logout(
		ctx context.Context,
		input LogoutInput,
	) error
}

type AuthenticateInput struct {
	Username string
	Password string
}

type AuthenticateOutput struct {
	Actor Actor
}

type LogoutInput struct {
	SessionTokenHash string
}

// =========================
// User Service
// =========================

type UserService interface {
	Get(ctx context.Context, input GetUserInput) (GetUserOutput, error)
	GetByUsername(ctx context.Context, input GetUserByUsernameInput) (GetUserByUsernameOutput, error)
	Create(ctx context.Context, input CreateUserInput) (CreateUserOutput, error)
	Update(ctx context.Context, input UpdateUserInput) (UpdateUserOutput, error)
}

type GetUserInput struct {
	Actor          Actor
	UserID         uuid.UUID
	AcademicYearID uuid.UUID
}

type GetUserOutput struct {
	User *domain.User
}

type GetUserByUsernameInput struct {
	Actor    Actor
	Username string
}

type GetUserByUsernameOutput struct {
	User *domain.User
}

type CreateUserInput struct {
	Actor        Actor
	Username     string
	PasswordHash string
	Role         domain.UserRole
}

type CreateUserOutput struct {
	User *domain.User
}

type UpdateUserInput struct {
	Actor        Actor
	UserID       uuid.UUID
	Username     string
	PasswordHash string
	Role         domain.UserRole
}

type UpdateUserOutput struct {
	User *domain.User
}

// =========================
// Academic Year Service
// =========================

type AcademicYearService interface {
	Create(ctx context.Context, input CreateAcademicYearInput) (CreateAcademicYearOutput, error)
	Get(ctx context.Context, input GetAcademicYearInput) (GetAcademicYearOutput, error)
	List(ctx context.Context, input ListAcademicYearsInput) (ListAcademicYearsOutput, error)
	StartClosing(ctx context.Context, input StartClosingAcademicYearInput) (StartClosingAcademicYearOutput, error)
	Close(ctx context.Context, input CloseAcademicYearInput) (CloseAcademicYearOutput, error)
	Reopen(ctx context.Context, input ReopenAcademicYearInput) (ReopenAcademicYearOutput, error)
}

type CreateAcademicYearInput struct {
	Actor     Actor
	Name      string
	StartDate time.Time
	EndDate   time.Time
}

type CreateAcademicYearOutput struct {
	AcademicYear *domain.AcademicYear
}

type GetAcademicYearInput struct {
	Actor          Actor
	AcademicYearID uuid.UUID
}

type GetAcademicYearOutput struct {
	AcademicYear *domain.AcademicYear
}

type ListAcademicYearsInput struct {
	Actor   Actor
	Options ListOptions
}

type ListAcademicYearsOutput struct {
	AcademicYears []*domain.AcademicYear
}

type StartClosingAcademicYearInput struct {
	Actor          Actor
	AcademicYearID uuid.UUID
}

type StartClosingAcademicYearOutput struct {
	AcademicYear *domain.AcademicYear
}

type CloseAcademicYearInput struct {
	Actor          Actor
	AcademicYearID uuid.UUID
}

type CloseAcademicYearOutput struct {
	AcademicYear *domain.AcademicYear
}

type ReopenAcademicYearInput struct {
	Actor          Actor
	AcademicYearID uuid.UUID
}

type ReopenAcademicYearOutput struct {
	AcademicYear *domain.AcademicYear
}

// =========================
// Class Service
// =========================

type ClassService interface {
	Create(ctx context.Context, input CreateClassInput) (CreateClassOutput, error)
	Get(ctx context.Context, input GetClassInput) (GetClassOutput, error)
	List(ctx context.Context, input ListClassesInput) (ListClassesOutput, error)
	Update(ctx context.Context, input UpdateClassInput) (UpdateClassOutput, error)
}

type CreateClassInput struct {
	Actor Actor
	Name  string
	Level int
}

type CreateClassOutput struct {
	Class *domain.Class
}

type GetClassInput struct {
	Actor   Actor
	ClassID uuid.UUID
}

type GetClassOutput struct {
	Class *domain.Class
}

type ListClassesInput struct {
	Actor          Actor
	AcademicYearID uuid.UUID
	Options        ListOptions
}

type ListClassesOutput struct {
	Classes []*domain.Class
}

type UpdateClassInput struct {
	Actor   Actor
	ClassID uuid.UUID
	Name    string
	Level   int
}

type UpdateClassOutput struct {
	Class *domain.Class
}

// =========================
// Student Service
// =========================

type StudentService interface {
	Create(ctx context.Context, input CreateStudentInput) (CreateStudentOutput, error)
	Get(ctx context.Context, input GetStudentInput) (GetStudentOutput, error)
	GetByNIS(ctx context.Context, input GetStudentByNISInput) (GetStudentByNISOutput, error)
	GetByNISN(ctx context.Context, input GetStudentByNISNInput) (GetStudentByNISNOutput, error)
	GetByUserID(ctx context.Context, input GetStudentByUserIDInput) (GetStudentByUserIDOutput, error)
	List(ctx context.Context, input ListStudentsInput) (ListStudentsOutput, error)
	Update(ctx context.Context, input UpdateStudentInput) (UpdateStudentOutput, error)
	Leave(ctx context.Context, input LeaveStudentInput) (LeaveStudentOutput, error)
}

type CreateStudentInput struct {
	Actor  Actor
	UserID *uuid.UUID
	NIS    *string
	NISN   *string
	Name   string
}

type CreateStudentOutput struct {
	Student *domain.Student
}

type GetStudentInput struct {
	Actor          Actor
	StudentID      uuid.UUID
	AcademicYearID uuid.UUID
}

type GetStudentOutput struct {
	Student *domain.Student
}

type GetStudentByNISInput struct {
	Actor          Actor
	NIS            string
	AcademicYearID uuid.UUID
}

type GetStudentByNISOutput struct {
	Student *domain.Student
}

type GetStudentByNISNInput struct {
	Actor          Actor
	NISN           string
	AcademicYearID uuid.UUID
}

type GetStudentByNISNOutput struct {
	Student *domain.Student
}

type GetStudentByUserIDInput struct {
	Actor          Actor
	UserID         uuid.UUID
	AcademicYearID uuid.UUID
}

type GetStudentByUserIDOutput struct {
	Student *domain.Student
}

type ListStudentsInput struct {
	Actor          Actor
	AcademicYearID uuid.UUID
	ClassID        uuid.UUID
	Options        ListOptions
}

type ListStudentsOutput struct {
	Students []*domain.Student
}

type UpdateStudentInput struct {
	Actor     Actor
	StudentID uuid.UUID
	UserID    *uuid.UUID
	NIS       *string
	NISN      *string
	Name      string
}

type UpdateStudentOutput struct {
	Student *domain.Student
}

type LeaveStudentInput struct {
	Actor     Actor
	StudentID uuid.UUID
}

type LeaveStudentOutput struct {
	Student *domain.Student
}

// =========================
// Class Assignment Service
// =========================

type ClassAssignmentService interface {
	Assign(ctx context.Context, input AssignTeacherInput) (AssignTeacherOutput, error)
	Get(ctx context.Context, input GetTeacherAssignmentInput) (GetTeacherAssignmentOutput, error)
	ListByTeacher(ctx context.Context, input ListTeacherAssignmentsInput) (ListTeacherAssignmentsOutput, error)
	ListByClass(ctx context.Context, input ListClassAssignmentsInput) (ListClassAssignmentsOutput, error)
	GetCurrentByClass(ctx context.Context, input GetCurrentClassAssignmentInput) (GetCurrentClassAssignmentOutput, error)
}

type AssignTeacherInput struct {
	Actor          Actor
	UserID         uuid.UUID
	ClassID        uuid.UUID
	AcademicYearID uuid.UUID
}

type AssignTeacherOutput struct {
	Assignment *domain.TeacherClassAssignment
}

type GetTeacherAssignmentInput struct {
	Actor        Actor
	AssignmentID uuid.UUID
}

type GetTeacherAssignmentOutput struct {
	Assignment *domain.TeacherClassAssignment
}

type ListTeacherAssignmentsInput struct {
	Actor          Actor
	UserID         uuid.UUID
	AcademicYearID uuid.UUID
	Options        ListOptions
}

type ListTeacherAssignmentsOutput struct {
	Assignments []*domain.TeacherClassAssignment
}

type ListClassAssignmentsInput struct {
	Actor          Actor
	ClassID        uuid.UUID
	AcademicYearID uuid.UUID
	Options        ListOptions
}

type ListClassAssignmentsOutput struct {
	Assignments []*domain.TeacherClassAssignment
}

type GetCurrentClassAssignmentInput struct {
	Actor          Actor
	ClassID        uuid.UUID
	AcademicYearID uuid.UUID
	Date           time.Time
}

type GetCurrentClassAssignmentOutput struct {
	Assignment *domain.TeacherClassAssignment
}

// =========================
// Class Transfer Service
// =========================

type ClassTransferService interface {
	Request(ctx context.Context, input RequestTransferInput) (RequestTransferOutput, error)
	Get(ctx context.Context, input GetTransferInput) (GetTransferOutput, error)
	ListByStudent(ctx context.Context, input ListStudentTransfersInput) (ListStudentTransfersOutput, error)
	ListPending(ctx context.Context, input ListPendingTransfersInput) (ListPendingTransfersOutput, error)
	Approve(ctx context.Context, input ApproveTransferInput) (ApproveTransferOutput, error)
	Reject(ctx context.Context, input RejectTransferInput) (RejectTransferOutput, error)
}

type RequestTransferInput struct {
	Actor          Actor
	StudentID      uuid.UUID
	AcademicYearID uuid.UUID
	FromClassID    uuid.UUID
	ToClassID      uuid.UUID
}

type RequestTransferOutput struct {
	TransferRequest *domain.ClassTransferRequest
}

type GetTransferInput struct {
	Actor             Actor
	TransferRequestID uuid.UUID
}

type GetTransferOutput struct {
	TransferRequest *domain.ClassTransferRequest
}

type ListStudentTransfersInput struct {
	Actor          Actor
	StudentID      uuid.UUID
	AcademicYearID uuid.UUID
	Options        ListOptions
}

type ListStudentTransfersOutput struct {
	TransferRequests []*domain.ClassTransferRequest
}

type ListPendingTransfersInput struct {
	Actor          Actor
	AcademicYearID uuid.UUID
	Options        ListOptions
}

type ListPendingTransfersOutput struct {
	TransferRequests []*domain.ClassTransferRequest
}

type ApproveTransferInput struct {
	Actor             Actor
	TransferRequestID uuid.UUID
	EffectiveDate     time.Time
}

type ApproveTransferOutput struct {
	TransferRequest *domain.ClassTransferRequest
	ClassHistory    *domain.StudentClassHistory
}

type RejectTransferInput struct {
	Actor             Actor
	TransferRequestID uuid.UUID
	RejectionReason   string
}

type RejectTransferOutput struct {
	TransferRequest *domain.ClassTransferRequest
}

// =========================
// Savings Account Service
// =========================

type SavingsAccountService interface {
	Create(ctx context.Context, input CreateSavingsAccountInput) (CreateSavingsAccountOutput, error)
	Get(ctx context.Context, input GetSavingsAccountInput) (GetSavingsAccountOutput, error)
	GetByStudentAndAcademicYear(
		ctx context.Context,
		input GetSavingsAccountByStudentAndAcademicYearInput,
	) (GetSavingsAccountByStudentAndAcademicYearOutput, error)
	GetBalance(
		ctx context.Context,
		input GetSavingsAccountBalanceInput,
	) (GetSavingsAccountBalanceOutput, error)
}

type CreateSavingsAccountInput struct {
	Actor          Actor
	StudentID      uuid.UUID
	AcademicYearID uuid.UUID
}

type CreateSavingsAccountOutput struct {
	Account *domain.SavingsAccount
}

type GetSavingsAccountInput struct {
	Actor            Actor
	SavingsAccountID uuid.UUID
}

type GetSavingsAccountOutput struct {
	Account *domain.SavingsAccount
}

type GetSavingsAccountByStudentAndAcademicYearInput struct {
	Actor          Actor
	StudentID      uuid.UUID
	AcademicYearID uuid.UUID
}

type GetSavingsAccountByStudentAndAcademicYearOutput struct {
	Account *domain.SavingsAccount
}

type GetSavingsAccountBalanceInput struct {
	Actor            Actor
	SavingsAccountID uuid.UUID
}

type GetSavingsAccountBalanceOutput struct {
	Balance domain.Money
}

// =========================
// Transaction Service
// =========================

type TransactionService interface {
	Deposit(ctx context.Context, input DepositInput) (DepositOutput, error)
	Withdrawal(ctx context.Context, input WithdrawalInput) (WithdrawalOutput, error)
	Get(ctx context.Context, input GetTransactionInput) (GetTransactionOutput, error)
	List(ctx context.Context, input ListTransactionsInput) (ListTransactionsOutput, error)
	Edit(ctx context.Context, input EditTransactionInput) (EditTransactionOutput, error)
	Cancel(ctx context.Context, input CancelTransactionInput) (CancelTransactionOutput, error)
}

type DepositInput struct {
	Actor            Actor
	SavingsAccountID uuid.UUID
	Amount           domain.Money
	TransactionDate  time.Time
}

type DepositOutput struct {
	Transaction *domain.Transaction
}

type WithdrawalInput struct {
	Actor            Actor
	SavingsAccountID uuid.UUID
	Amount           domain.Money
	TransactionDate  time.Time
}

type WithdrawalOutput struct {
	Transaction *domain.Transaction
}

type GetTransactionInput struct {
	Actor         Actor
	TransactionID uuid.UUID
}

type GetTransactionOutput struct {
	Transaction *domain.Transaction
}

type ListTransactionsInput struct {
	Actor            Actor
	SavingsAccountID uuid.UUID
	Options          ListOptions
}

type ListTransactionsOutput struct {
	Transactions []*domain.Transaction
}

type EditTransactionInput struct {
	Actor           Actor
	TransactionID   uuid.UUID
	Amount          domain.Money
	TransactionDate time.Time
}

type EditTransactionOutput struct {
	Transaction *domain.Transaction
}

type CancelTransactionInput struct {
	Actor         Actor
	TransactionID uuid.UUID
}

type CancelTransactionOutput struct {
	Transaction *domain.Transaction
}

// =========================
// Settlement Service
// =========================

type SettlementService interface {
	SettleYearEnd(ctx context.Context, input SettleYearEndInput) (SettleYearEndOutput, error)
	SettleStudentLeaving(ctx context.Context, input SettleStudentLeavingInput) (SettleStudentLeavingOutput, error)
	Get(ctx context.Context, input GetSettlementInput) (GetSettlementOutput, error)
	List(ctx context.Context, input ListSettlementsInput) (ListSettlementsOutput, error)
	Reopen(ctx context.Context, input ReopenSettlementInput) (ReopenSettlementOutput, error)
}

type SettleYearEndInput struct {
	Actor            Actor
	SavingsAccountID uuid.UUID
}

type SettleYearEndOutput struct {
	Settlement *domain.SavingsSettlement
	Account    *domain.SavingsAccount
}

type SettleStudentLeavingInput struct {
	Actor            Actor
	SavingsAccountID uuid.UUID
}

type SettleStudentLeavingOutput struct {
	Settlement *domain.SavingsSettlement
	Account    *domain.SavingsAccount
	Student    *domain.Student
}

type GetSettlementInput struct {
	Actor        Actor
	SettlementID uuid.UUID
}

type GetSettlementOutput struct {
	Settlement *domain.SavingsSettlement
}

type ListSettlementsInput struct {
	Actor            Actor
	SavingsAccountID uuid.UUID
	Options          ListOptions
}

type ListSettlementsOutput struct {
	Settlements []*domain.SavingsSettlement
}

type ReopenSettlementInput struct {
	Actor        Actor
	SettlementID uuid.UUID
}

type ReopenSettlementOutput struct {
	PreviousSettlement *domain.SavingsSettlement
	Account            *domain.SavingsAccount
}

// =========================
// Audit Service
// =========================

type AuditService interface {
	Record(ctx context.Context, input RecordAuditInput) (RecordAuditOutput, error)
	ListByEntity(ctx context.Context, input ListAuditByEntityInput) (ListAuditByEntityOutput, error)
	ListByActor(ctx context.Context, input ListAuditByActorInput) (ListAuditByActorOutput, error)
}

type RecordAuditInput struct {
	Actor      Actor
	Action     string
	EntityType domain.AuditEntityType
	EntityID   uuid.UUID
	Before     []byte
	After      []byte
}

type RecordAuditOutput struct {
	AuditLog *domain.AuditLog
}

type ListAuditByEntityInput struct {
	Actor      Actor
	EntityType domain.AuditEntityType
	EntityID   uuid.UUID
	Options    ListOptions
}

type ListAuditByEntityOutput struct {
	AuditLogs []*domain.AuditLog
}

type ListAuditByActorInput struct {
	Actor         Actor
	TargetActorID uuid.UUID
	Options       ListOptions
}

type ListAuditByActorOutput struct {
	AuditLogs []*domain.AuditLog
}
