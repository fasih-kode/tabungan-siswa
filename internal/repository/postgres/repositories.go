package postgres

import (
	"database/sql"

	"github.com/fasih/tabungan-siswa/internal/platform/database"
	"github.com/fasih/tabungan-siswa/internal/repository"
)

// NewRepositorySet membuat seluruh repository PostgreSQL yang menggunakan
// execution context database yang sama.
//
// db dapat berupa *sql.DB untuk operasi normal atau *sql.Tx untuk operasi
// dalam satu transaction boundary.
func NewRepositorySet(db database.DBTX) repository.RepositorySet {
	return repository.RepositorySet{
		Users:                   NewUserRepository(db),
		Sessions:                NewSessionRepository(db),
		AcademicYears:           NewAcademicYearRepository(db),
		Classes:                 NewClassRepository(db),
		Students:                NewStudentRepository(db),
		StudentClassHistories:   NewStudentClassHistoryRepository(db),
		TeacherClassAssignments: NewTeacherClassAssignmentRepository(db),
		ClassTransferRequests:   NewClassTransferRequestRepository(db),
		SavingsAccounts:         NewSavingsAccountRepository(db),
		Transactions:            NewTransactionRepository(db),
		Settlements:             NewSettlementRepository(db),
		AuditLogs:               NewAuditRepository(db),
	}
}

// compile-time assertion memastikan *sql.DB tetap memenuhi DBTX
// yang dibutuhkan oleh repository factory.
var _ database.DBTX = (*sql.DB)(nil)
