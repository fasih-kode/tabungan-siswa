package repository

import (
	"context"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/google/uuid"
)

type ListOptions struct {
	AcademicYearID uuid.UUID
	ClassID        uuid.UUID
	TeacherID      uuid.UUID
	Limit          int
	Offset         int
	Search         string
	StudentStatus  string
	UnassignedOnly bool
}

type UserRepository interface {
	Create(ctx context.Context, user domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (domain.User, error)
	GetByUsername(ctx context.Context, username string) (domain.User, error)
	ListByRole(ctx context.Context, role domain.UserRole, options ListOptions) ([]domain.User, error)
	Update(ctx context.Context, user domain.User) error
	ExistsByUsername(ctx context.Context, username string) (bool, error)
}

type SessionRepository interface {
	Create(ctx context.Context, session domain.Session) error
	GetByTokenHash(ctx context.Context, tokenHash string) (domain.Session, error)
	Revoke(ctx context.Context, sessionID uuid.UUID, revokedAt time.Time) error
}

type UserActivationRepository interface {
	SetActive(ctx context.Context, userID uuid.UUID, active bool, updatedAt time.Time) error
}

type SessionRevocationRepository interface {
	RevokeAllByUserID(ctx context.Context, userID uuid.UUID, revokedAt time.Time) error
}

type AcademicYearRepository interface {
	Create(ctx context.Context, academicYear domain.AcademicYear) error
	GetByID(ctx context.Context, id uuid.UUID) (domain.AcademicYear, error)
	GetByIDForShare(ctx context.Context, id uuid.UUID) (domain.AcademicYear, error)
	GetByIDForUpdate(ctx context.Context, id uuid.UUID) (domain.AcademicYear, error)
	HasSuccessor(ctx context.Context, id uuid.UUID) (bool, error)
	ListForUpdate(ctx context.Context) ([]domain.AcademicYear, error)
	List(ctx context.Context, options ListOptions) ([]domain.AcademicYear, error)
	Update(ctx context.Context, academicYear domain.AcademicYear) error
	GetOpen(ctx context.Context) (domain.AcademicYear, error)
}

type ClassRepository interface {
	Create(ctx context.Context, class domain.Class) error
	GetByID(ctx context.Context, id uuid.UUID) (domain.Class, error)
	List(ctx context.Context, options ListOptions) ([]domain.Class, error)
	ListByTeacherAndAcademicYear(
		ctx context.Context,
		userID uuid.UUID,
		academicYearID uuid.UUID,
		options ListOptions,
	) ([]domain.Class, error)
	Update(ctx context.Context, class domain.Class) error
	ExistsByNameAndLevel(
		ctx context.Context,
		name string,
		level int,
		academicYearID ...uuid.UUID,
	) (bool, error)
}

type StudentDirectoryEntry struct {
	Student        domain.Student
	AcademicYearID uuid.UUID
	ClassID        uuid.UUID
	AcademicYear   string
	Class          string
}

type StudentRepository interface {
	ListDirectory(ctx context.Context, options ListOptions) ([]StudentDirectoryEntry, error)
	Create(ctx context.Context, student domain.Student) error
	GetByID(ctx context.Context, id uuid.UUID) (domain.Student, error)
	GetByIDForUpdate(ctx context.Context, id uuid.UUID) (domain.Student, error)
	GetByNIS(ctx context.Context, nis string) (domain.Student, error)
	GetByNISN(ctx context.Context, nisn string) (domain.Student, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (domain.Student, error)
	List(ctx context.Context, options ListOptions) ([]domain.Student, error)
	ListByClassAndAcademicYear(
		ctx context.Context,
		classID uuid.UUID,
		academicYearID uuid.UUID,
		options ListOptions,
	) ([]domain.Student, error)
	Update(ctx context.Context, student domain.Student) error
}

type StudentClassHistoryRepository interface {
	Create(ctx context.Context, history domain.StudentClassHistory) error
	GetByID(
		ctx context.Context,
		id uuid.UUID,
	) (domain.StudentClassHistory, error)
	ListByStudentAndAcademicYear(
		ctx context.Context,
		studentID uuid.UUID,
		academicYearID uuid.UUID,
		options ListOptions,
	) ([]domain.StudentClassHistory, error)
	GetCurrentByStudentAndAcademicYear(
		ctx context.Context,
		studentID uuid.UUID,
		academicYearID uuid.UUID,
		date time.Time,
	) (domain.StudentClassHistory, error)
	CloseCurrentByStudentAndAcademicYear(
		ctx context.Context,
		studentID uuid.UUID,
		academicYearID uuid.UUID,
		endDate time.Time,
	) (domain.StudentClassHistory, error)
}

type TeacherClassAssignmentRepository interface {
	Create(
		ctx context.Context,
		assignment domain.TeacherClassAssignment,
	) error
	GetByID(
		ctx context.Context,
		id uuid.UUID,
	) (domain.TeacherClassAssignment, error)
	ListByUserAndAcademicYear(
		ctx context.Context,
		userID uuid.UUID,
		academicYearID uuid.UUID,
		options ListOptions,
	) ([]domain.TeacherClassAssignment, error)
	ListByClassAndAcademicYear(
		ctx context.Context,
		classID uuid.UUID,
		academicYearID uuid.UUID,
		options ListOptions,
	) ([]domain.TeacherClassAssignment, error)
	GetCurrentByClassAndAcademicYear(
		ctx context.Context,
		classID uuid.UUID,
		academicYearID uuid.UUID,
		date time.Time,
	) (domain.TeacherClassAssignment, error)
}

type ClassTransferRequestRepository interface {
	Create(
		ctx context.Context,
		request domain.ClassTransferRequest,
	) error
	GetByID(
		ctx context.Context,
		id uuid.UUID,
	) (domain.ClassTransferRequest, error)
	// GetByIDForUpdate harus dijalankan oleh repository
	// yang terikat pada transaction aktif.
	//
	// Implementasi wajib menggunakan row-level locking
	// untuk menjaga konsistensi state transition seperti
	// approval dan rejection transfer kelas.
	GetByIDForUpdate(
		ctx context.Context,
		id uuid.UUID,
	) (domain.ClassTransferRequest, error)
	ListByStudentAndAcademicYear(
		ctx context.Context,
		studentID uuid.UUID,
		academicYearID uuid.UUID,
		options ListOptions,
	) ([]domain.ClassTransferRequest, error)
	ListPending(
		ctx context.Context,
		academicYearID uuid.UUID,
		options ListOptions,
	) ([]domain.ClassTransferRequest, error)
	HasPendingByStudent(ctx context.Context, studentID uuid.UUID) (bool, error)
	Update(
		ctx context.Context,
		request domain.ClassTransferRequest,
	) error
}

type SavingsAccountRepository interface {
	Create(
		ctx context.Context,
		account domain.SavingsAccount,
	) error
	GetByID(
		ctx context.Context,
		id uuid.UUID,
	) (domain.SavingsAccount, error)
	GetByStudentAndAcademicYear(
		ctx context.Context,
		studentID uuid.UUID,
		academicYearID uuid.UUID,
	) (domain.SavingsAccount, error)

	// GetByIDForUpdate harus dijalankan oleh repository
	// yang terikat pada transaction aktif.
	//
	// Implementasi wajib menggunakan row-level locking
	// untuk menjaga konsistensi operasi yang sensitif terhadap
	// konkurensi, seperti withdrawal dan settlement.
	GetByIDForUpdate(
		ctx context.Context,
		id uuid.UUID,
	) (domain.SavingsAccount, error)

	Update(
		ctx context.Context,
		account domain.SavingsAccount,
	) error
	HasOpenByAcademicYear(ctx context.Context, academicYearID uuid.UUID) (bool, error)
}

type TransactionRepository interface {
	Create(
		ctx context.Context,
		transaction domain.Transaction,
	) error
	GetByID(
		ctx context.Context,
		id uuid.UUID,
	) (domain.Transaction, error)
	ListBySavingsAccount(
		ctx context.Context,
		savingsAccountID uuid.UUID,
		options ListOptions,
	) ([]domain.Transaction, error)
	ListActiveBySavingsAccount(
		ctx context.Context,
		savingsAccountID uuid.UUID,
	) ([]domain.Transaction, error)
	ListAllBySavingsAccount(
		ctx context.Context,
		savingsAccountID uuid.UUID,
	) ([]domain.Transaction, error)
	Update(
		ctx context.Context,
		transaction domain.Transaction,
	) error
}

type LeavingTransactionRequestRepository interface {
	CreateLeavingTransactionRequest(context.Context, domain.LeavingTransactionRequest) error
	GetLeavingTransactionRequest(context.Context, uuid.UUID) (domain.LeavingTransactionRequest, error)
	GetLeavingTransactionRequestForUpdate(context.Context, uuid.UUID) (domain.LeavingTransactionRequest, error)
	ListLeavingTransactionRequests(context.Context, uuid.UUID, domain.LeavingTransactionRequestStatus) ([]domain.LeavingTransactionRequest, error)
	// ListByStudentLeavingRequest membaca relasi yang tersedia mulai schema v16.
	// Proses yang tidak ditemukan menghasilkan repository.ErrNotFound.
	ListByStudentLeavingRequest(context.Context, uuid.UUID) ([]domain.LeavingTransactionRequest, error)
	UpdateLeavingTransactionRequest(context.Context, domain.LeavingTransactionRequest) error
	HasPendingLeavingTransactionRequests(context.Context, uuid.UUID) (bool, error)
	HasPendingForSavingsAccount(context.Context, uuid.UUID) (bool, error)
	HasEvidenceReference(context.Context, uuid.UUID, string) (bool, error)
}

// StudentLeavingRequestRepository mengelola proses pindah administratif.
// Pemanggil harus menetapkan urutan lock lintas siswa, tahun ajaran, rekening,
// dan proses sebelum memakai operasi ForUpdate dalam UOW.
type StudentLeavingRequestRepository interface {
	Create(context.Context, domain.StudentLeavingRequest) error
	GetByID(context.Context, uuid.UUID) (domain.StudentLeavingRequest, error)
	GetByIDForUpdate(context.Context, uuid.UUID) (domain.StudentLeavingRequest, error)
	GetInProgressByStudent(context.Context, uuid.UUID) (domain.StudentLeavingRequest, error)
	GetInProgressByStudentForUpdate(context.Context, uuid.UUID) (domain.StudentLeavingRequest, error)
	ListByStudent(context.Context, uuid.UUID) ([]domain.StudentLeavingRequest, error)
	// ListFinancialLinkIssues memeriksa konsistensi relasi keuangan pada schema v16.
	ListFinancialLinkIssues(context.Context, uuid.UUID) ([]StudentLeavingFinancialLinkIssue, error)
	Update(context.Context, domain.StudentLeavingRequest) error
}

type StudentLeavingFinancialLinkIssueType string

const (
	StudentLeavingSettlementStudentMismatch  StudentLeavingFinancialLinkIssueType = "SETTLEMENT_STUDENT_MISMATCH"
	StudentLeavingSettlementTypeMismatch     StudentLeavingFinancialLinkIssueType = "SETTLEMENT_TYPE_MISMATCH"
	StudentLeavingLateRequestStudentMismatch StudentLeavingFinancialLinkIssueType = "LATE_REQUEST_STUDENT_MISMATCH"
	StudentLeavingLateRequestYearMismatch    StudentLeavingFinancialLinkIssueType = "LATE_REQUEST_YEAR_MISMATCH"
)

// StudentLeavingFinancialLinkIssue mengidentifikasi satu relasi keuangan yang tidak konsisten.
type StudentLeavingFinancialLinkIssue struct {
	Type     StudentLeavingFinancialLinkIssueType
	RecordID uuid.UUID
}

type SettlementRepository interface {
	Create(
		ctx context.Context,
		settlement domain.SavingsSettlement,
	) error
	// CreateForStudentLeavingRequest membuat settlement pindah yang tertaut pada schema v16.
	CreateForStudentLeavingRequest(
		ctx context.Context,
		settlement domain.SavingsSettlement,
		studentLeavingRequestID uuid.UUID,
	) error
	GetByID(
		ctx context.Context,
		id uuid.UUID,
	) (domain.SavingsSettlement, error)
	GetCompletedBySavingsAccount(
		ctx context.Context,
		savingsAccountID uuid.UUID,
	) (domain.SavingsSettlement, error)
	ListBySavingsAccount(
		ctx context.Context,
		savingsAccountID uuid.UUID,
		options ListOptions,
	) ([]domain.SavingsSettlement, error)
	// ListByStudentLeavingRequest membaca settlement pindah yang terhubung
	// melalui relasi schema v16; settlement akhir tahun tidak disertakan.
	ListByStudentLeavingRequest(context.Context, uuid.UUID) ([]domain.SavingsSettlement, error)
	Update(
		ctx context.Context,
		settlement domain.SavingsSettlement,
	) error
	ListYearReadiness(ctx context.Context, academicYearID uuid.UUID) ([]SettlementReadinessAccount, error)
	CountPendingTransfersForTeacher(ctx context.Context, academicYearID, teacherID uuid.UUID) (int, error)
	HasUnreconciledPositiveSettlements(ctx context.Context, academicYearID uuid.UUID) (bool, error)
	HasActiveReopenWorkflow(ctx context.Context, academicYearID uuid.UUID) (bool, error)
	CreateCashMovement(ctx context.Context, movement domain.SettlementCashMovement) error
	ListCashMovements(ctx context.Context, settlementID uuid.UUID) ([]domain.SettlementCashMovement, error)
	CreateReopenRequest(ctx context.Context, request domain.SettlementReopenRequest) error
	GetReopenRequest(ctx context.Context, id uuid.UUID) (domain.SettlementReopenRequest, error)
	GetActiveReopenRequest(ctx context.Context, settlementID uuid.UUID) (domain.SettlementReopenRequest, error)
	GetLatestReopenRequest(ctx context.Context, settlementID uuid.UUID) (domain.SettlementReopenRequest, error)
	GetApprovedReopenRequestByAccount(ctx context.Context, accountID uuid.UUID) (domain.SettlementReopenRequest, error)
	GetReopenRequestForUpdate(ctx context.Context, id uuid.UUID) (domain.SettlementReopenRequest, error)
	UpdateReopenRequest(ctx context.Context, request domain.SettlementReopenRequest) error
}

// SettlementReadinessAccount is a server-derived row scoped to one academic
// year; Wali attribution uses the year-end historical placement and assignment.
type SettlementReadinessAccount struct {
	Account          domain.SavingsAccount
	Student          domain.Student
	ClassID          *uuid.UUID
	ClassName        string
	WaliID           *uuid.UUID
	WaliName         string
	Settlement       *domain.SavingsSettlement
	CashDelivered    domain.Money
	CurrentBalance   domain.Money
	ReopenStatus     domain.SettlementReopenRequestStatus
	AttributionIssue bool
}

type AuditRepository interface {
	Create(
		ctx context.Context,
		auditLog domain.AuditLog,
	) error
	ListByEntity(
		ctx context.Context,
		entityType domain.AuditEntityType,
		entityID uuid.UUID,
		options ListOptions,
	) ([]domain.AuditLog, error)
	ListByActor(
		ctx context.Context,
		actorUserID uuid.UUID,
		options ListOptions,
	) ([]domain.AuditLog, error)
}

// RepositorySet berisi seluruh repository yang terikat pada
// satu execution context database yang sama.
//
// Pada penggunaan biasa, repository dapat bekerja menggunakan
// koneksi database normal.
//
// Pada operasi yang membutuhkan atomic transaction, service
// menggunakan RepositorySet yang diperoleh dari UnitOfWork.
// ListTotalsRepository counts the same authorized scopes and filters as list queries.
type ListTotalsRepository interface {
	AcademicYears(context.Context) (int, error)
	Classes(ctx context.Context, yearID, teacherID, studentID uuid.UUID) (int, error)
	Users(ctx context.Context, role domain.UserRole) (int, error)
	StudentDirectory(ctx context.Context, options ListOptions) (int, error)
	StudentClassHistory(ctx context.Context, studentID, yearID uuid.UUID) (int, error)
	Transfers(ctx context.Context, studentID, yearID uuid.UUID, pendingOnly bool) (int, error)
}

// ReportingRepository menyediakan agregasi khusus reporting dalam satu
// academic year. Actor scope wajib diterapkan pada query, bukan oleh pemanggil.
type ReportingRepository interface {
	Dashboard(ctx context.Context, yearID, userID uuid.UUID, role domain.UserRole) (DashboardMetrics, error)
	StudentReport(ctx context.Context, yearID, userID uuid.UUID, role domain.UserRole) ([]StudentReportRow, error)
	ClassReport(ctx context.Context, yearID, userID uuid.UUID, role domain.UserRole) ([]ClassReportRow, error)
	SettlementReport(ctx context.Context, yearID, userID uuid.UUID, role domain.UserRole) ([]SettlementReportRow, error)
}

type DashboardMetrics struct {
	Students, Classes, Accounts, SettledAccounts int
	Balance, Deposits, Withdrawals               domain.Money
}

type StudentReportRow struct {
	StudentID, ClassID, AccountID  uuid.UUID
	StudentName, NIS, ClassName    string
	AccountStatus                  domain.SavingsAccountStatus
	Balance, Deposits, Withdrawals domain.Money
}

type ClassReportRow struct {
	ClassID                        uuid.UUID
	ClassName                      string
	Students, Accounts, Settled    int
	Balance, Deposits, Withdrawals domain.Money
}

type SettlementReportRow struct {
	StudentID, ClassID, AccountID         uuid.UUID
	StudentName, NIS, ClassName           string
	AccountStatus                         domain.SavingsAccountStatus
	SettlementState                       string
	Balance, SettlementAmount, HandedOver domain.Money
}

type RepositorySet struct {
	ListTotals                 ListTotalsRepository
	Reporting                  ReportingRepository
	Users                      UserRepository
	Sessions                   SessionRepository
	AcademicYears              AcademicYearRepository
	Classes                    ClassRepository
	Students                   StudentRepository
	StudentClassHistories      StudentClassHistoryRepository
	TeacherClassAssignments    TeacherClassAssignmentRepository
	ClassTransferRequests      ClassTransferRequestRepository
	SavingsAccounts            SavingsAccountRepository
	Transactions               TransactionRepository
	LeavingTransactionRequests LeavingTransactionRequestRepository
	StudentLeavingRequests     StudentLeavingRequestRepository
	Settlements                SettlementRepository
	AuditLogs                  AuditRepository
}

// UnitOfWork merepresentasikan satu transaction boundary.
//
// Seluruh repository yang dikembalikan oleh Repositories()
// harus menggunakan transaction database yang sama.
//
// Service bertanggung jawab menentukan kapan transaction
// dimulai, kapan commit dilakukan, dan kapan rollback dilakukan.
type UnitOfWork interface {
	Repositories() RepositorySet
	Commit() error
	Rollback() error
}

// UnitOfWorkManager membuat UnitOfWork baru.
//
// Implementasi konkret berada di infrastructure/database layer.
// Service hanya bergantung pada kontrak ini.
type UnitOfWorkManager interface {
	Begin(ctx context.Context) (UnitOfWork, error)
}
