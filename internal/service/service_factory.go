package service

import "fmt"

// NewServiceSet membangun seluruh application service dari dependencies
// yang diberikan oleh composition root.
//
// Factory ini hanya bertanggung jawab melakukan construction dan wiring.
// Business rule, authorization, dan transaction workflow tetap berada
// di masing-masing service.
func NewServiceSet(deps Dependencies) (ServiceSet, error) {
	if err := deps.Validate(); err != nil {
		return ServiceSet{}, err
	}

	authentication, err := NewAuthenticationService(deps)
	if err != nil {
		return ServiceSet{}, fmt.Errorf(
			"construct authentication service: %w",
			err,
		)
	}

	user, err := NewUserService(deps)
	if err != nil {
		return ServiceSet{}, fmt.Errorf(
			"construct user service: %w",
			err,
		)
	}

	academicYear, err := NewAcademicYearService(deps)
	if err != nil {
		return ServiceSet{}, fmt.Errorf(
			"construct academic year service: %w",
			err,
		)
	}

	class, err := NewClassService(deps)
	if err != nil {
		return ServiceSet{}, fmt.Errorf(
			"construct class service: %w",
			err,
		)
	}

	student, err := NewStudentService(deps)
	if err != nil {
		return ServiceSet{}, fmt.Errorf(
			"construct student service: %w",
			err,
		)
	}

	classAssignment, err := NewClassAssignmentService(deps)
	if err != nil {
		return ServiceSet{}, fmt.Errorf(
			"construct class assignment service: %w",
			err,
		)
	}

	classTransfer, err := NewClassTransferService(deps)
	if err != nil {
		return ServiceSet{}, fmt.Errorf(
			"construct class transfer service: %w",
			err,
		)
	}

	savingsAccount, err := NewSavingsAccountService(deps)
	if err != nil {
		return ServiceSet{}, fmt.Errorf(
			"construct savings account service: %w",
			err,
		)
	}

	transaction, err := NewTransactionService(deps)
	if err != nil {
		return ServiceSet{}, fmt.Errorf(
			"construct transaction service: %w",
			err,
		)
	}

	settlement, err := NewSettlementService(deps)
	if err != nil {
		return ServiceSet{}, fmt.Errorf(
			"construct settlement service: %w",
			err,
		)
	}

	audit, err := NewAuditService(deps)
	if err != nil {
		return ServiceSet{}, fmt.Errorf(
			"construct audit service: %w",
			err,
		)
	}

	return ServiceSet{
		Authentication:  authentication,
		User:            user,
		AcademicYear:    academicYear,
		Class:           class,
		Student:         student,
		ClassAssignment: classAssignment,
		ClassTransfer:   classTransfer,
		SavingsAccount:  savingsAccount,
		Transaction:     transaction,
		Settlement:      settlement,
		Audit:           audit,
	}, nil
}
