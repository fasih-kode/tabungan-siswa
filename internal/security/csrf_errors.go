package security

import "errors"

var (
	ErrInvalidCSRFCookieName = errors.New("invalid csrf cookie name")
	ErrEmptyCSRFToken        = errors.New("empty csrf token")
	ErrCSRFCookieNotFound    = errors.New("csrf cookie not found")
	ErrInvalidCSRFCookie     = errors.New("invalid csrf cookie")
)
