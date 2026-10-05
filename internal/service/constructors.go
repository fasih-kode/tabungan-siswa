package service

// =========================
// Class Service
// =========================

type classService struct {
	deps Dependencies
}

func NewClassService(deps Dependencies) (*classService, error) {
	if err := deps.Validate(); err != nil {
		return nil, err
	}

	return &classService{
		deps: deps,
	}, nil
}

// =========================
// Student Service
// =========================

type studentService struct {
	deps Dependencies
}

func NewStudentService(deps Dependencies) (*studentService, error) {
	if err := deps.Validate(); err != nil {
		return nil, err
	}

	return &studentService{
		deps: deps,
	}, nil
}

// =========================
// Class Assignment Service
// =========================

type classAssignmentService struct {
	deps Dependencies
}

func NewClassAssignmentService(deps Dependencies) (*classAssignmentService, error) {
	if err := deps.Validate(); err != nil {
		return nil, err
	}

	return &classAssignmentService{
		deps: deps,
	}, nil
}

// =========================
// Class Transfer Service
// =========================

type classTransferService struct {
	deps Dependencies
}

func NewClassTransferService(deps Dependencies) (*classTransferService, error) {
	if err := deps.Validate(); err != nil {
		return nil, err
	}

	return &classTransferService{
		deps: deps,
	}, nil
}

// =========================
// Savings Account Service
// =========================

type savingsAccountService struct {
	deps Dependencies
}

func NewSavingsAccountService(deps Dependencies) (*savingsAccountService, error) {
	if err := deps.Validate(); err != nil {
		return nil, err
	}

	return &savingsAccountService{
		deps: deps,
	}, nil
}

// =========================
// Transaction Service
// =========================

type transactionService struct {
	deps Dependencies
}

func NewTransactionService(deps Dependencies) (*transactionService, error) {
	if err := deps.Validate(); err != nil {
		return nil, err
	}

	return &transactionService{
		deps: deps,
	}, nil
}

// =========================
// Settlement Service
// =========================

type settlementService struct {
	deps Dependencies
}

func NewSettlementService(deps Dependencies) (*settlementService, error) {
	if err := deps.Validate(); err != nil {
		return nil, err
	}

	return &settlementService{
		deps: deps,
	}, nil
}

// =========================
// Audit Service
// =========================

type auditService struct {
	deps Dependencies
}

func NewAuditService(deps Dependencies) (*auditService, error) {
	if err := deps.Validate(); err != nil {
		return nil, err
	}

	return &auditService{
		deps: deps,
	}, nil
}
