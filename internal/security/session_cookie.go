package security

import (
	"net/http"
	"strings"
	"time"
)

const DefaultSessionCookieName = "__Host-tabungan_siswa_session"

type SessionCookie struct {
	name string
	path string
}

func NewSessionCookie(name string) (SessionCookie, error) {
	name = strings.TrimSpace(name)
	if !strings.HasPrefix(name, "__Host-") {
		return SessionCookie{}, ErrInvalidSessionCookieName
	}

	return SessionCookie{
		name: name,
		path: "/",
	}, nil
}

func NewDefaultSessionCookie() SessionCookie {
	cookie, err := NewSessionCookie(DefaultSessionCookieName)
	if err != nil {
		panic(err)
	}

	return cookie
}

func (c SessionCookie) Set(
	w http.ResponseWriter,
	token string,
	expiresAt time.Time,
) error {
	if strings.TrimSpace(token) == "" {
		return ErrEmptySessionToken
	}

	if !expiresAt.After(time.Now()) {
		return ErrInvalidSessionCookieExpiration
	}

	http.SetCookie(w, &http.Cookie{
		Name:     c.name,
		Value:    token,
		Path:     c.path,
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	return nil
}

func (c SessionCookie) Read(r *http.Request) (string, error) {
	cookie, err := r.Cookie(c.name)
	if err != nil {
		if err == http.ErrNoCookie {
			return "", ErrSessionCookieNotFound
		}

		return "", err
	}

	if strings.TrimSpace(cookie.Value) == "" {
		return "", ErrInvalidSessionCookie
	}

	return cookie.Value, nil
}

func (c SessionCookie) Clear(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     c.name,
		Value:    "",
		Path:     c.path,
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}
