package security

import "errors"

var (
	ErrInvalidSessionCookieName       = errors.New("invalid session cookie name")
	ErrEmptySessionToken              = errors.New("empty session token")
	ErrSessionCookieNotFound          = errors.New("session cookie not found")
	ErrInvalidSessionCookie           = errors.New("invalid session cookie")
	ErrInvalidSessionCookieExpiration = errors.New("invalid session cookie expiration")
)
