package middleware

import (
	"errors"
	"net/http"
	"time"

	"github.com/fasih/tabungan-siswa/internal/http/autherror"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/security"
	"github.com/fasih/tabungan-siswa/internal/service"
)

type AuthenticationMiddleware struct {
	cookie   security.SessionCookie
	sessions repository.SessionRepository
	users    repository.UserRepository
}

func NewAuthenticationMiddleware(
	cookie security.SessionCookie,
	sessions repository.SessionRepository,
	users repository.UserRepository,
) (*AuthenticationMiddleware, error) {
	if sessions == nil || users == nil {
		return nil, service.ErrInvalidDependency
	}

	return &AuthenticationMiddleware{
		cookie:   cookie,
		sessions: sessions,
		users:    users,
	}, nil
}

func (m *AuthenticationMiddleware) RequireAuthentication(
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := m.cookie.Read(r)
		if err != nil {
			autherror.Write(w, service.ErrInvalidCredentials)
			return
		}

		session, err := m.sessions.GetByTokenHash(
			r.Context(),
			security.HashSessionToken(token),
		)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				autherror.Write(w, service.ErrInvalidCredentials)
				return
			}

			autherror.Write(w, err)
			return
		}

		if !session.IsActive(time.Now()) {
			autherror.Write(w, service.ErrInvalidCredentials)
			return
		}

		user, err := m.users.GetByID(r.Context(), session.UserID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				autherror.Write(w, service.ErrInvalidCredentials)
				return
			}

			autherror.Write(w, err)
			return
		}

		actor := service.Actor{
			UserID: user.ID,
			Role:   user.Role,
		}
		if err := actor.Validate(); err != nil {
			autherror.Write(w, err)
			return
		}

		next.ServeHTTP(w, r.WithContext(service.WithActor(r.Context(), actor)))
	})
}
