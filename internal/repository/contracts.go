package repository

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

type UserRepository interface {
	Create(ctx context.Context, user domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (domain.User, error)
	GetByUsername(ctx context.Context, username string) (domain.User, error)
	Update(ctx context.Context, user domain.User) error
	ExistsByUsername(ctx context.Context, username string) (bool, error)
}

type SessionRepository interface {
	Create(ctx context.Context, session domain.Session) error
	GetByTokenHash(ctx context.Context, tokenHash string) (domain.Session, error)
	Revoke(ctx context.Context, sessionID uuid.UUID, revokedAt time.Time) error
}

type AcademicYearRepository interface {
	Create(ctx context.Context, academicYear domain.AcademicYear) error
	GetByID(ctx context.Context, id uuid.UUID) (domain.AcademicYear, error)
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
	) (bool, error)
}

type StudentRepository interface {
	Create(ctx context.Context, student domain.Student) error
	GetByID(ctx context.Context, id uuid.UUID) (domain.Student, error)
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
	Update(
		ctx context.Context,
		transaction domain.Transaction,
	) error
}

type SettlementRepository interface {
	Create(
		ctx context.Context,
		settlement domain.SavingsSettlement,
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
	Update(
		ctx context.Context,
		settlement domain.SavingsSettlement,
	) error
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
type RepositorySet struct {
	Users                   UserRepository
	Sessions                SessionRepository
	AcademicYears           AcademicYearRepository
	Classes                 ClassRepository
	Students                StudentRepository
	StudentClassHistories   StudentClassHistoryRepository
	TeacherClassAssignments TeacherClassAssignmentRepository
	ClassTransferRequests   ClassTransferRequestRepository
	SavingsAccounts         SavingsAccountRepository
	Transactions            TransactionRepository
	Settlements             SettlementRepository
	AuditLogs               AuditRepository
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
