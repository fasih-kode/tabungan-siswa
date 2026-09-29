package service

import "errors"

var (
	ErrInvalidActor   = errors.New("invalid actor")
	ErrForbidden      = errors.New("forbidden")
	ErrScopeViolation = errors.New("scope violation")
)
