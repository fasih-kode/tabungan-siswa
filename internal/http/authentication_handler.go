package http

import (
	"errors"
	"mime"
	"net/http"
	"time"

	"github.com/fasih/tabungan-siswa/internal/security"
	"github.com/fasih/tabungan-siswa/internal/service"
)

const maxLoginRequestBodyBytes int64 = 16 << 10

type AuthenticationHandler struct {
	authentication service.AuthenticationService
	expiration     security.SessionExpirationPolicy
	cookie         security.SessionCookie
}

func NewAuthenticationHandler(
	authentication service.AuthenticationService,
	expiration security.SessionExpirationPolicy,
	cookie security.SessionCookie,
) (*AuthenticationHandler, error) {
	if authentication == nil || !expiration.IsValid() || !cookie.IsValid() {
		return nil, service.ErrInvalidDependency
	}

	return &AuthenticationHandler{
		authentication: authentication,
		expiration:     expiration,
		cookie:         cookie,
	}, nil
}

func (h *AuthenticationHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, localizedStatusMessage(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Cache-Control", "no-store")

	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/x-www-form-urlencoded" {
		http.Error(
			w,
			localizedStatusMessage(http.StatusBadRequest),
			http.StatusBadRequest,
		)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxLoginRequestBodyBytes)
	if err := r.ParseForm(); err != nil {
		status := http.StatusBadRequest
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			status = http.StatusRequestEntityTooLarge
		}

		http.Error(w, localizedStatusMessage(status), status)
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
	if err := h.authentication.CreateSession(
		r.Context(),
		service.CreateSessionInput{
			UserID:    output.Actor.UserID,
			TokenHash: security.HashSessionToken(token),
			ExpiresAt: expiresAt,
		},
	); err != nil {
		WriteAuthenticationError(w, err)
		return
	}

	if err := h.cookie.Set(w, token, expiresAt); err != nil {
		WriteAuthenticationError(w, err)
		return
	}

	http.Redirect(w, r, "/protected", http.StatusSeeOther)
}

func (h *AuthenticationHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, localizedStatusMessage(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Cache-Control", "no-store")

	token, err := h.cookie.Read(r)
	if err != nil {
		h.cookie.Clear(w)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
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
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
