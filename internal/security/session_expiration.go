package security

import "time"

// SessionExpirationPolicy defines the lifetime of a server-side session.
type SessionExpirationPolicy struct {
	lifetime time.Duration
}

// NewSessionExpirationPolicy creates a session expiration policy.
//
// The lifetime must be strictly positive. The policy stores only the
// configured lifetime; expiration is calculated from the caller-provided
// current time so the policy remains deterministic and testable.
func NewSessionExpirationPolicy(lifetime time.Duration) (SessionExpirationPolicy, error) {
	if lifetime <= 0 {
		return SessionExpirationPolicy{}, ErrInvalidSessionLifetime
	}

	return SessionExpirationPolicy{
		lifetime: lifetime,
	}, nil
}

// ExpiresAt returns the absolute expiration time for a session created at now.
func (p SessionExpirationPolicy) ExpiresAt(now time.Time) time.Time {
	return now.Add(p.lifetime)
}
