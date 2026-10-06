package middleware

import (
	"errors"
	"net/http"
	"time"

	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/security"
	"github.com/fasih/tabungan-siswa/internal/service"
)

type AuthenticationMiddleware struct {
	sessions repository.SessionRepository
	users    repository.UserRepository
	cookie   security.SessionCookie
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
		sessions: sessions,
		users:    users,
		cookie:   cookie,
	}, nil
}

func (m *AuthenticationMiddleware) RequireAuthentication(
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := m.cookie.Read(r)
		if err != nil {
			writeUnauthorized(w)
			return
		}

		session, err := m.sessions.GetByTokenHash(
			r.Context(),
			security.HashSessionToken(token),
		)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				writeUnauthorized(w)
				return
			}

			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if !session.IsActive(time.Now()) {
			writeUnauthorized(w)
			return
		}

		user, err := m.users.GetByID(r.Context(), session.UserID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				writeUnauthorized(w)
				return
			}

			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		actor := service.Actor{
			UserID: user.ID,
			Role:   user.Role,
		}
		if err := actor.Validate(); err != nil {
			writeUnauthorized(w)
			return
		}

		next.ServeHTTP(w, r.WithContext(service.WithActor(r.Context(), actor)))
	})
}

func writeUnauthorized(w http.ResponseWriter) {
	w.WriteHeader(http.StatusUnauthorized)
}
