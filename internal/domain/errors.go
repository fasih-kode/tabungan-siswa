package domain

import "errors"

var (
	ErrInvalidAmount        = errors.New("invalid amount")
	ErrInvalidDateRange     = errors.New("invalid date range")
	ErrInvalidValue         = errors.New("invalid value")
	ErrInvalidID            = errors.New("invalid ID")
	ErrInvalidUsername      = errors.New("invalid username")
	ErrInvalidPasswordHash  = errors.New("invalid password hash")
	ErrInvalidAction        = errors.New("invalid audit action")
	ErrEmptyRejectionReason = errors.New("rejection reason cannot be empty")
	ErrInvalidTransition    = errors.New("invalid state transition")
	ErrAlreadySettled       = errors.New("savings account is already settled")
	ErrAlreadyCancelled     = errors.New("transaction is already cancelled")
	ErrNegativeBalance      = errors.New("balance cannot be negative")
	ErrEmptyName            = errors.New("name cannot be empty")
	ErrInvalidClassLevel    = errors.New("class level must be between 1 and 12")
)
