package security

import "errors"

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash string, password string) error
}

var (
	ErrInvalidPassword  = errors.New("invalid password")
	ErrPasswordTooLong  = errors.New("password is too long")
	ErrPasswordMismatch = errors.New("password mismatch")
)
