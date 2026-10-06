package security

import "time"

// SecuritySet berisi seluruh security component yang dibutuhkan oleh
// application composition root.
type SecuritySet struct {
	PasswordHasher    PasswordHasher
	SessionCookie     SessionCookie
	CSRFCookie        CSRFCookie
	SessionExpiration SessionExpirationPolicy
}

// NewSecuritySet membangun seluruh security component.
//
// Session lifetime diberikan oleh composition root agar kebijakan expiration
// tidak tersembunyi sebagai default di dalam package security.
func NewSecuritySet(sessionLifetime time.Duration) (SecuritySet, error) {
	passwordHasher, err := NewDefaultArgon2idHasher()
	if err != nil {
		return SecuritySet{}, err
	}

	sessionExpiration, err := NewSessionExpirationPolicy(sessionLifetime)
	if err != nil {
		return SecuritySet{}, err
	}

	return SecuritySet{
		PasswordHasher:    passwordHasher,
		SessionCookie:     NewDefaultSessionCookie(),
		CSRFCookie:        NewDefaultCSRFCookie(),
		SessionExpiration: sessionExpiration,
	}, nil
}
