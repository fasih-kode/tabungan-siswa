package service

import (
	"errors"
	"fmt"

	"github.com/fasih/tabungan-siswa/internal/domain"
)

var (
	ErrInvalidActor                     = errors.New("invalid actor")
	ErrForbidden                        = errors.New("forbidden")
	ErrScopeViolation                   = errors.New("scope violation")
	ErrInvalidDependency                = errors.New("invalid dependency")
	ErrInvalidCredentials               = errors.New("invalid credentials")
	ErrTransactionDateOutOfAcademicYear = errors.New("transaction date outside academic year")
	ErrAcademicYearHasUnsettledAccounts = fmt.Errorf("academic year has unsettled savings accounts: %w", domain.ErrInvalidTransition)
	ErrAcademicYearHasPendingTransfers  = fmt.Errorf("academic year has pending class transfers: %w", domain.ErrInvalidTransition)
)
