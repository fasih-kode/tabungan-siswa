package security

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"
)

const DefaultCSRFCookieName = "__Host-tabungan_siswa_csrf"

const DefaultCSRFHeaderName = "X-CSRF-Token"

type CSRFCookie struct {
	name string
	path string
}

func NewCSRFCookie(name string) (CSRFCookie, error) {
	name = strings.TrimSpace(name)
	if !strings.HasPrefix(name, "__Host-") {
		return CSRFCookie{}, ErrInvalidCSRFCookieName
	}

	return CSRFCookie{
		name: name,
		path: "/",
	}, nil
}

func NewDefaultCSRFCookie() CSRFCookie {
	cookie, err := NewCSRFCookie(DefaultCSRFCookieName)
	if err != nil {
		panic(err)
	}

	return cookie
}

func GenerateCSRFToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func (c CSRFCookie) Set(w http.ResponseWriter, token string) error {
	if strings.TrimSpace(token) == "" {
		return ErrEmptyCSRFToken
	}

	http.SetCookie(w, &http.Cookie{
		Name:     c.name,
		Value:    token,
		Path:     c.path,
		HttpOnly: false,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	return nil
}

func (c CSRFCookie) Read(r *http.Request) (string, error) {
	cookie, err := r.Cookie(c.name)
	if err != nil {
		if err == http.ErrNoCookie {
			return "", ErrCSRFCookieNotFound
		}

		return "", err
	}

	if strings.TrimSpace(cookie.Value) == "" {
		return "", ErrInvalidCSRFCookie
	}

	return cookie.Value, nil
}
