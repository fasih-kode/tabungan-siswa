package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/security"
	"github.com/google/uuid"
)

type authenticationServiceFakeUserRepository struct {
	repository.UserRepository

	getByUsernameFn func(context.Context, string) (domain.User, error)
}

func (f *authenticationServiceFakeUserRepository) GetByUsername(
	ctx context.Context,
	username string,
) (domain.User, error) {
	if f.getByUsernameFn != nil {
		return f.getByUsernameFn(ctx, username)
	}

	return domain.User{}, repository.ErrNotFound
}

type authenticationServiceFakeSessionRepository struct {
	repository.SessionRepository
	session       domain.Session
	getByHashErr  error
	revokeCalls   int
	revokeSession uuid.UUID
	revokeAt      time.Time
	revokeErr     error
}

func (f *authenticationServiceFakeSessionRepository) GetByTokenHash(
	context.Context,
	string,
) (domain.Session, error) {
	if f.getByHashErr != nil {
		return domain.Session{}, f.getByHashErr
	}
	return f.session, nil
}

func (f *authenticationServiceFakeSessionRepository) Revoke(
	_ context.Context,
	sessionID uuid.UUID,
	revokedAt time.Time,
) error {
	f.revokeCalls++
	f.revokeSession = sessionID
	f.revokeAt = revokedAt
	return f.revokeErr
}

type authenticationServiceFakePasswordHasher struct {
	compareFn func(string, string) error
}

func (f *authenticationServiceFakePasswordHasher) Hash(string) (string, error) {
	return "", nil
}

func (f *authenticationServiceFakePasswordHasher) Compare(
	hash string,
	password string,
) error {
	if f.compareFn != nil {
		return f.compareFn(hash, password)
	}

	return nil
}

func TestNewAuthenticationService(t *testing.T) {
	uow := foundationFakeUOWManager{}
	hasher := &authenticationServiceFakePasswordHasher{}
	users := &authenticationServiceFakeUserRepository{}

	t.Run("valid dependencies", func(t *testing.T) {
		svc, err := NewAuthenticationService(Dependencies{
			Repositories:   repository.RepositorySet{Users: users, Sessions: &authenticationServiceFakeSessionRepository{}},
			UOW:            uow,
			PasswordHasher: hasher,
		})
		if err != nil {
			t.Fatalf("NewAuthenticationService() error = %v, want nil", err)
		}

		if svc == nil {
			t.Fatal("NewAuthenticationService() service = nil, want non-nil")
		}
	})

	t.Run("invalid base dependencies", func(t *testing.T) {
		svc, err := NewAuthenticationService(Dependencies{
			PasswordHasher: hasher,
		})
		if !errors.Is(err, ErrInvalidDependency) {
			t.Fatalf(
				"NewAuthenticationService() error = %v, want ErrInvalidDependency",
				err,
			)
		}

		if svc != nil {
			t.Fatal("NewAuthenticationService() service != nil on invalid dependencies")
		}
	})

	t.Run("missing user repository", func(t *testing.T) {
		svc, err := NewAuthenticationService(Dependencies{
			UOW:            uow,
			PasswordHasher: hasher,
		})
		if !errors.Is(err, ErrInvalidDependency) {
			t.Fatalf(
				"NewAuthenticationService() error = %v, want ErrInvalidDependency",
				err,
			)
		}

		if svc != nil {
			t.Fatal("NewAuthenticationService() service != nil without user repository")
		}
	})

	t.Run("missing session repository", func(t *testing.T) {
		svc, err := NewAuthenticationService(Dependencies{
			Repositories:   repository.RepositorySet{Users: users},
			UOW:            uow,
			PasswordHasher: hasher,
		})
		if !errors.Is(err, ErrInvalidDependency) {
			t.Fatalf("NewAuthenticationService() error = %v, want %v", err, ErrInvalidDependency)
		}
		if svc != nil {
			t.Fatal("NewAuthenticationService() service != nil without session repository")
		}
	})

	t.Run("missing password hasher", func(t *testing.T) {
		svc, err := NewAuthenticationService(Dependencies{
			UOW: uow,
		})
		if !errors.Is(err, ErrInvalidDependency) {
			t.Fatalf(
				"NewAuthenticationService() error = %v, want ErrInvalidDependency",
				err,
			)
		}

		if svc != nil {
			t.Fatal("NewAuthenticationService() service != nil without password hasher")
		}
	})
}

func TestAuthenticationService_Authenticate(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	user := domain.User{
		ID:           userID,
		Username:     "wali-01",
		PasswordHash: "stored-password-hash",
		Role:         domain.RoleWaliKelas,
	}

	repositoryErr := errors.New("database unavailable")

	tests := []struct {
		name      string
		input     AuthenticateInput
		userRepo  repository.UserRepository
		hasher    *authenticationServiceFakePasswordHasher
		wantActor Actor
		wantErr   error
	}{
		{
			name: "success",
			input: AuthenticateInput{
				Username: "wali-01",
				Password: "correct-password",
			},
			userRepo: &authenticationServiceFakeUserRepository{
				getByUsernameFn: func(
					context.Context,
					string,
				) (domain.User, error) {
					return user, nil
				},
			},
			hasher: &authenticationServiceFakePasswordHasher{
				compareFn: func(hash, password string) error {
					if hash != user.PasswordHash {
						t.Fatalf("Compare() hash = %q, want %q", hash, user.PasswordHash)
					}
					if password != "correct-password" {
						t.Fatalf(
							"Compare() password = %q, want correct-password",
							password,
						)
					}
					return nil
				},
			},
			wantActor: Actor{
				UserID: user.ID,
				Role:   user.Role,
			},
		},
		{
			name: "unknown username returns generic credentials error",
			input: AuthenticateInput{
				Username: "unknown",
				Password: "correct-password",
			},
			userRepo: &authenticationServiceFakeUserRepository{
				getByUsernameFn: func(
					context.Context,
					string,
				) (domain.User, error) {
					return domain.User{}, repository.ErrNotFound
				},
			},
			hasher: &authenticationServiceFakePasswordHasher{
				compareFn: func(string, string) error {
					t.Fatal("Compare() called for unknown username")
					return nil
				},
			},
			wantErr: ErrInvalidCredentials,
		},
		{
			name: "password mismatch returns generic credentials error",
			input: AuthenticateInput{
				Username: "wali-01",
				Password: "wrong-password",
			},
			userRepo: &authenticationServiceFakeUserRepository{
				getByUsernameFn: func(
					context.Context,
					string,
				) (domain.User, error) {
					return user, nil
				},
			},
			hasher: &authenticationServiceFakePasswordHasher{
				compareFn: func(string, string) error {
					return security.ErrPasswordMismatch
				},
			},
			wantErr: ErrInvalidCredentials,
		},
		{
			name: "malformed stored hash returns generic credentials error",
			input: AuthenticateInput{
				Username: "wali-01",
				Password: "correct-password",
			},
			userRepo: &authenticationServiceFakeUserRepository{
				getByUsernameFn: func(
					context.Context,
					string,
				) (domain.User, error) {
					return user, nil
				},
			},
			hasher: &authenticationServiceFakePasswordHasher{
				compareFn: func(string, string) error {
					return domain.ErrInvalidPasswordHash
				},
			},
			wantErr: ErrInvalidCredentials,
		},
		{
			name: "empty username returns generic credentials error",
			input: AuthenticateInput{
				Username: "",
				Password: "correct-password",
			},
			userRepo: &authenticationServiceFakeUserRepository{
				getByUsernameFn: func(context.Context, string) (domain.User, error) {
					t.Fatal("GetByUsername() called with empty username")
					return domain.User{}, nil
				},
			},
			hasher: &authenticationServiceFakePasswordHasher{
				compareFn: func(string, string) error {
					t.Fatal("Compare() called with empty username")
					return nil
				},
			},
			wantErr: ErrInvalidCredentials,
		},
		{
			name: "empty password returns generic credentials error",
			input: AuthenticateInput{
				Username: "wali-01",
				Password: "",
			},
			userRepo: &authenticationServiceFakeUserRepository{
				getByUsernameFn: func(context.Context, string) (domain.User, error) {
					t.Fatal("GetByUsername() called with empty password")
					return domain.User{}, nil
				},
			},
			hasher: &authenticationServiceFakePasswordHasher{
				compareFn: func(string, string) error {
					t.Fatal("Compare() called with empty password")
					return nil
				},
			},
			wantErr: ErrInvalidCredentials,
		},
		{
			name: "repository error is propagated",
			input: AuthenticateInput{
				Username: "wali-01",
				Password: "correct-password",
			},
			userRepo: &authenticationServiceFakeUserRepository{
				getByUsernameFn: func(
					context.Context,
					string,
				) (domain.User, error) {
					return domain.User{}, repositoryErr
				},
			},
			hasher: &authenticationServiceFakePasswordHasher{
				compareFn: func(string, string) error {
					t.Fatal("Compare() called after repository error")
					return nil
				},
			},
			wantErr: repositoryErr,
		},
		{
			name: "invalid stored user role returns generic credentials error",
			input: AuthenticateInput{
				Username: "wali-01",
				Password: "correct-password",
			},
			userRepo: &authenticationServiceFakeUserRepository{
				getByUsernameFn: func(
					context.Context,
					string,
				) (domain.User, error) {
					invalidUser := user
					invalidUser.Role = domain.UserRole("INVALID")
					return invalidUser, nil
				},
			},
			hasher: &authenticationServiceFakePasswordHasher{
				compareFn: func(string, string) error {
					return nil
				},
			},
			wantErr: ErrInvalidCredentials,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := NewAuthenticationService(Dependencies{
				Repositories: repository.RepositorySet{
					Users:    tt.userRepo,
					Sessions: &authenticationServiceFakeSessionRepository{},
				},
				UOW:            foundationFakeUOWManager{},
				PasswordHasher: tt.hasher,
			})
			if err != nil {
				t.Fatalf("NewAuthenticationService() error = %v", err)
			}

			got, err := svc.Authenticate(ctx, tt.input)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf(
						"Authenticate() error = %v, want %v",
						err,
						tt.wantErr,
					)
				}
				return
			}

			if err != nil {
				t.Fatalf("Authenticate() error = %v, want nil", err)
			}

			if got.Actor != tt.wantActor {
				t.Fatalf(
					"Authenticate() Actor = %+v, want %+v",
					got.Actor,
					tt.wantActor,
				)
			}
		})
	}
}

func newAuthenticationServiceForLogoutTest(
	t *testing.T,
	sessions repository.SessionRepository,
) *authenticationService {
	t.Helper()

	svc, err := NewAuthenticationService(Dependencies{
		Repositories: repository.RepositorySet{
			Users:    &authenticationServiceFakeUserRepository{},
			Sessions: sessions,
		},
		UOW:            foundationFakeUOWManager{},
		PasswordHasher: &authenticationServiceFakePasswordHasher{},
	})
	if err != nil {
		t.Fatalf("NewAuthenticationService() error = %v", err)
	}

	return svc
}

func TestAuthenticationService_Logout(t *testing.T) {
	session := domain.Session{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		TokenHash: "session-token-hash",
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}
	sessions := &authenticationServiceFakeSessionRepository{session: session}
	svc := newAuthenticationServiceForLogoutTest(t, sessions)

	if err := svc.Logout(context.Background(), LogoutInput{
		SessionTokenHash: session.TokenHash,
	}); err != nil {
		t.Fatalf("Logout() error = %v, want nil", err)
	}
	if sessions.revokeCalls != 1 {
		t.Fatalf("Revoke() calls = %d, want 1", sessions.revokeCalls)
	}
	if sessions.revokeSession != session.ID {
		t.Fatalf("Revoke() session ID = %v, want %v", sessions.revokeSession, session.ID)
	}
	if sessions.revokeAt.IsZero() {
		t.Fatal("Revoke() revokedAt is zero, want non-zero timestamp")
	}
}

func TestAuthenticationService_LogoutIsIdempotentForUnknownSession(t *testing.T) {
	sessions := &authenticationServiceFakeSessionRepository{
		getByHashErr: repository.ErrNotFound,
	}
	svc := newAuthenticationServiceForLogoutTest(t, sessions)

	if err := svc.Logout(context.Background(), LogoutInput{
		SessionTokenHash: "missing",
	}); err != nil {
		t.Fatalf("Logout() error = %v, want nil", err)
	}
	if sessions.revokeCalls != 0 {
		t.Fatalf("Revoke() calls = %d, want 0", sessions.revokeCalls)
	}
}

func TestAuthenticationService_LogoutIgnoresExpiredSession(t *testing.T) {
	sessions := &authenticationServiceFakeSessionRepository{
		session: domain.Session{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			TokenHash: "expired-session",
			ExpiresAt: time.Now().Add(-time.Minute),
			CreatedAt: time.Now().Add(-time.Hour),
		},
	}
	svc := newAuthenticationServiceForLogoutTest(t, sessions)

	if err := svc.Logout(context.Background(), LogoutInput{
		SessionTokenHash: sessions.session.TokenHash,
	}); err != nil {
		t.Fatalf("Logout() error = %v, want nil", err)
	}
	if sessions.revokeCalls != 0 {
		t.Fatalf("Revoke() calls = %d, want 0 for expired session", sessions.revokeCalls)
	}
}

func TestAuthenticationService_LogoutIgnoresRevokedSession(t *testing.T) {
	revokedAt := time.Now().Add(-time.Minute)
	sessions := &authenticationServiceFakeSessionRepository{
		session: domain.Session{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			TokenHash: "revoked-session",
			ExpiresAt: time.Now().Add(time.Hour),
			RevokedAt: &revokedAt,
			CreatedAt: time.Now().Add(-time.Hour),
		},
	}
	svc := newAuthenticationServiceForLogoutTest(t, sessions)

	if err := svc.Logout(context.Background(), LogoutInput{
		SessionTokenHash: sessions.session.TokenHash,
	}); err != nil {
		t.Fatalf("Logout() error = %v, want nil", err)
	}
	if sessions.revokeCalls != 0 {
		t.Fatalf("Revoke() calls = %d, want 0 for revoked session", sessions.revokeCalls)
	}
}

func TestAuthenticationService_LogoutPropagatesSessionLookupError(t *testing.T) {
	repositoryErr := errors.New("database unavailable")
	sessions := &authenticationServiceFakeSessionRepository{
		getByHashErr: repositoryErr,
	}
	svc := newAuthenticationServiceForLogoutTest(t, sessions)

	err := svc.Logout(context.Background(), LogoutInput{
		SessionTokenHash: "session-token-hash",
	})
	if !errors.Is(err, repositoryErr) {
		t.Fatalf("Logout() error = %v, want %v", err, repositoryErr)
	}
	if sessions.revokeCalls != 0 {
		t.Fatalf("Revoke() calls = %d, want 0 after lookup error", sessions.revokeCalls)
	}
}

func TestAuthenticationService_LogoutPropagatesRevokeError(t *testing.T) {
	repositoryErr := errors.New("database unavailable")
	session := domain.Session{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		TokenHash: "session-token-hash",
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}
	sessions := &authenticationServiceFakeSessionRepository{
		session:   session,
		revokeErr: repositoryErr,
	}
	svc := newAuthenticationServiceForLogoutTest(t, sessions)

	err := svc.Logout(context.Background(), LogoutInput{
		SessionTokenHash: session.TokenHash,
	})
	if !errors.Is(err, repositoryErr) {
		t.Fatalf("Logout() error = %v, want %v", err, repositoryErr)
	}
	if sessions.revokeCalls != 1 {
		t.Fatalf("Revoke() calls = %d, want 1", sessions.revokeCalls)
	}
}

func TestAuthenticationService_LogoutIsIdempotentForConcurrentRevocation(t *testing.T) {
	session := domain.Session{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		TokenHash: "session-token-hash",
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}
	sessions := &authenticationServiceFakeSessionRepository{
		session:   session,
		revokeErr: repository.ErrNotFound,
	}
	svc := newAuthenticationServiceForLogoutTest(t, sessions)

	if err := svc.Logout(context.Background(), LogoutInput{
		SessionTokenHash: session.TokenHash,
	}); err != nil {
		t.Fatalf("Logout() error = %v, want nil", err)
	}
	if sessions.revokeCalls != 1 {
		t.Fatalf("Revoke() calls = %d, want 1", sessions.revokeCalls)
	}
}

func TestAuthenticationService_LogoutRejectsEmptyTokenHash(t *testing.T) {
	sessions := &authenticationServiceFakeSessionRepository{}
	svc := newAuthenticationServiceForLogoutTest(t, sessions)

	if err := svc.Logout(context.Background(), LogoutInput{}); !errors.Is(err, domain.ErrInvalidSessionTokenHash) {
		t.Fatalf("Logout() error = %v, want %v", err, domain.ErrInvalidSessionTokenHash)
	}
	if sessions.revokeCalls != 0 {
		t.Fatalf("Revoke() calls = %d, want 0 for invalid token hash", sessions.revokeCalls)
	}
}
