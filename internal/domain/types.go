package domain

type UserRole string

const (
	RoleAdmin     UserRole = "ADMIN"
	RoleWaliKelas UserRole = "WALI_KELAS"
	RoleSiswa     UserRole = "SISWA"
)

func (r UserRole) IsValid() bool {
	switch r {
	case RoleAdmin, RoleWaliKelas, RoleSiswa:
		return true
	default:
		return false
	}
}

type AcademicYearStatus string

const (
	AcademicYearOpen    AcademicYearStatus = "OPEN"
	AcademicYearClosing AcademicYearStatus = "CLOSING"
	AcademicYearClosed  AcademicYearStatus = "CLOSED"
)

func (s AcademicYearStatus) IsValid() bool {
	switch s {
	case AcademicYearOpen, AcademicYearClosing, AcademicYearClosed:
		return true
	default:
		return false
	}
}

type StudentStatus string

const (
	StudentActive StudentStatus = "ACTIVE"
	StudentLeft   StudentStatus = "LEFT"
)

func (s StudentStatus) IsValid() bool {
	switch s {
	case StudentActive, StudentLeft:
		return true
	default:
		return false
	}
}

type SavingsAccountStatus string

const (
	SavingsAccountOpen    SavingsAccountStatus = "OPEN"
	SavingsAccountSettled SavingsAccountStatus = "SETTLED"
)

func (s SavingsAccountStatus) IsValid() bool {
	switch s {
	case SavingsAccountOpen, SavingsAccountSettled:
		return true
	default:
		return false
	}
}

type TransactionType string

const (
	TransactionDeposit    TransactionType = "DEPOSIT"
	TransactionWithdrawal TransactionType = "WITHDRAWAL"
)

func (t TransactionType) IsValid() bool {
	switch t {
	case TransactionDeposit, TransactionWithdrawal:
		return true
	default:
		return false
	}
}

type TransactionStatus string

const (
	TransactionActive    TransactionStatus = "ACTIVE"
	TransactionCancelled TransactionStatus = "CANCELLED"
)

func (s TransactionStatus) IsValid() bool {
	switch s {
	case TransactionActive, TransactionCancelled:
		return true
	default:
		return false
	}
}

type SettlementType string

const (
	SettlementRegularYearEnd SettlementType = "REGULAR_YEAR_END"
	SettlementStudentLeaving SettlementType = "STUDENT_LEAVING"
)

func (t SettlementType) IsValid() bool {
	switch t {
	case SettlementRegularYearEnd, SettlementStudentLeaving:
		return true
	default:
		return false
	}
}

type SettlementStatus string

const (
	SettlementCompleted  SettlementStatus = "COMPLETED"
	SettlementSuperseded SettlementStatus = "SUPERSEDED"
)

func (s SettlementStatus) IsValid() bool {
	switch s {
	case SettlementCompleted, SettlementSuperseded:
		return true
	default:
		return false
	}
}

type ClassTransferRequestStatus string

const (
	TransferPending  ClassTransferRequestStatus = "PENDING"
	TransferApproved ClassTransferRequestStatus = "APPROVED"
	TransferRejected ClassTransferRequestStatus = "REJECTED"
)

func (s ClassTransferRequestStatus) IsValid() bool {
	switch s {
	case TransferPending, TransferApproved, TransferRejected:
		return true
	default:
		return false
	}
}
