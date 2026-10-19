package http

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"

	"github.com/google/uuid"
)

const credentialRevealLifetime = 5 * time.Minute

type credentialReveal struct {
	ActorID                uuid.UUID
	Username               string
	Password               string
	PasswordChangeRequired bool
	ExpiresAt              time.Time
}

type credentialRevealStore struct {
	mu      sync.Mutex
	entries map[string]credentialReveal
}

func newCredentialRevealStore() *credentialRevealStore {
	return &credentialRevealStore{entries: make(map[string]credentialReveal)}
}

func generateTemporaryPassword() (string, error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func (s *credentialRevealStore) Put(actorID uuid.UUID, username, password string, passwordChangeRequired bool) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()
	for key, entry := range s.entries {
		if !entry.ExpiresAt.After(now) {
			delete(s.entries, key)
		}
	}
	s.entries[token] = credentialReveal{ActorID: actorID, Username: username, Password: password, PasswordChangeRequired: passwordChangeRequired, ExpiresAt: now.Add(credentialRevealLifetime)}
	return token, nil
}

func (s *credentialRevealStore) Take(actorID uuid.UUID, token string) (credentialReveal, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.entries[token]
	if !ok || entry.ActorID != actorID || !entry.ExpiresAt.After(time.Now()) {
		if ok && !entry.ExpiresAt.After(time.Now()) {
			delete(s.entries, token)
		}
		return credentialReveal{}, false
	}
	delete(s.entries, token)
	return entry, true
}
