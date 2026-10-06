package service

// ServiceSet contains all application service contracts.
//
// It is used as the composition container for the service layer.
// ServiceSet does not create services, manage transactions,
// authorize actors, or implement business rules.
type ServiceSet struct {
	Authentication  AuthenticationService
	User            UserService
	AcademicYear    AcademicYearService
	Class           ClassService
	Student         StudentService
	ClassAssignment ClassAssignmentService
	ClassTransfer   ClassTransferService
	SavingsAccount  SavingsAccountService
	Transaction     TransactionService
	Settlement      SettlementService
	Audit           AuditService
}
