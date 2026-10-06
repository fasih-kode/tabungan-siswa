package http

import (
	"mime"
	"net/http"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/security"
	"github.com/fasih/tabungan-siswa/internal/service"
)

type AuthenticationHandler struct {
	authentication service.AuthenticationService
	sessions       repository.SessionRepository
	expiration     security.SessionExpirationPolicy
	cookie         security.SessionCookie
}

func NewAuthenticationHandler(
	authentication service.AuthenticationService,
	sessions repository.SessionRepository,
	expiration security.SessionExpirationPolicy,
	cookie security.SessionCookie,
) (*AuthenticationHandler, error) {
	if authentication == nil || sessions == nil {
		return nil, service.ErrInvalidDependency
	}

	return &AuthenticationHandler{
		authentication: authentication,
		sessions:       sessions,
		expiration:     expiration,
		cookie:         cookie,
	}, nil
}

func (h *AuthenticationHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Cache-Control", "no-store")

	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/x-www-form-urlencoded" {
		http.Error(
			w,
			http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest,
		)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(
			w,
			http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest,
		)
		return
	}

	output, err := h.authentication.Authenticate(
		r.Context(),
		service.AuthenticateInput{
			Username: r.PostForm.Get("username"),
			Password: r.PostForm.Get("password"),
		},
	)
	if err != nil {
		WriteAuthenticationError(w, err)
		return
	}

	token, err := security.GenerateSessionToken()
	if err != nil {
		WriteAuthenticationError(w, err)
		return
	}

	expiresAt := h.expiration.ExpiresAt(time.Now())
	session, err := domain.NewSession(
		output.Actor.UserID,
		security.HashSessionToken(token),
		expiresAt,
	)
	if err != nil {
		WriteAuthenticationError(w, err)
		return
	}

	if err := h.sessions.Create(r.Context(), session); err != nil {
		WriteAuthenticationError(w, err)
		return
	}

	if err := h.cookie.Set(w, token, expiresAt); err != nil {
		WriteAuthenticationError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthenticationHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Cache-Control", "no-store")

	token, err := h.cookie.Read(r)
	if err != nil {
		h.cookie.Clear(w)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err := h.authentication.Logout(
		r.Context(),
		service.LogoutInput{
			SessionTokenHash: security.HashSessionToken(token),
		},
	); err != nil {
		WriteAuthenticationError(w, err)
		return
	}

	h.cookie.Clear(w)
	w.WriteHeader(http.StatusNoContent)
}
