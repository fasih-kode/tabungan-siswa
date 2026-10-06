package security

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewSessionCookie(t *testing.T) {
	cookie, err := NewSessionCookie("__Host-test_session")
	if err != nil {
		t.Fatalf("NewSessionCookie() error = %v, want nil", err)
	}

	if cookie.name != "__Host-test_session" {
		t.Fatalf("name = %q, want __Host-test_session", cookie.name)
	}

	if cookie.path != "/" {
		t.Fatalf("path = %q, want /", cookie.path)
	}
}

func TestNewSessionCookieRejectsNonHostName(t *testing.T) {
	for _, name := range []string{"session", "my-session"} {
		t.Run(name, func(t *testing.T) {
			_, err := NewSessionCookie(name)
			if !errors.Is(err, ErrInvalidSessionCookieName) {
				t.Fatalf("NewSessionCookie() error = %v, want %v", err, ErrInvalidSessionCookieName)
			}
		})
	}
}

func TestNewSessionCookieRejectsEmptyName(t *testing.T) {
	_, err := NewSessionCookie("   ")
	if !errors.Is(err, ErrInvalidSessionCookieName) {
		t.Fatalf("NewSessionCookie() error = %v, want %v", err, ErrInvalidSessionCookieName)
	}
}

func TestNewDefaultSessionCookie(t *testing.T) {
	cookie := NewDefaultSessionCookie()

	if cookie.name != DefaultSessionCookieName {
		t.Fatalf("name = %q, want %q", cookie.name, DefaultSessionCookieName)
	}

	if cookie.path != "/" {
		t.Fatalf("path = %q, want /", cookie.path)
	}
}

func TestSessionCookieSet(t *testing.T) {
	cookie := NewDefaultSessionCookie()
	expiresAt := time.Date(2026, 10, 7, 10, 0, 0, 0, time.UTC)

	recorder := httptest.NewRecorder()
	err := cookie.Set(recorder, "opaque-session-token", expiresAt)
	if err != nil {
		t.Fatalf("Set() error = %v, want nil", err)
	}

	result := recorder.Result()
	defer result.Body.Close()

	cookies := result.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("Set-Cookie count = %d, want 1", len(cookies))
	}

	got := cookies[0]

	if got.Name != DefaultSessionCookieName {
		t.Fatalf("Name = %q, want %q", got.Name, DefaultSessionCookieName)
	}
	if got.Value != "opaque-session-token" {
		t.Fatalf("Value = %q, want opaque-session-token", got.Value)
	}
	if got.Path != "/" {
		t.Fatalf("Path = %q, want /", got.Path)
	}
	if !got.HttpOnly {
		t.Fatal("HttpOnly = false, want true")
	}
	if !got.Secure {
		t.Fatal("Secure = false, want true")
	}
	if got.SameSite != http.SameSiteLaxMode {
		t.Fatalf("SameSite = %v, want SameSiteLaxMode", got.SameSite)
	}
	if !got.Expires.Equal(expiresAt) {
		t.Fatalf("Expires = %v, want %v", got.Expires, expiresAt)
	}
	if got.Domain != "" {
		t.Fatalf("Domain = %q, want empty", got.Domain)
	}
}

func TestSessionCookieSetRejectsEmptyToken(t *testing.T) {
	cookie := NewDefaultSessionCookie()
	recorder := httptest.NewRecorder()

	err := cookie.Set(recorder, "   ", time.Now().Add(time.Hour))
	if !errors.Is(err, ErrEmptySessionToken) {
		t.Fatalf("Set() error = %v, want %v", err, ErrEmptySessionToken)
	}

	if len(recorder.Header().Values("Set-Cookie")) != 0 {
		t.Fatal("Set-Cookie header written for empty token")
	}
}

func TestSessionCookieSetRejectsExpiredExpiration(t *testing.T) {
	cookie := NewDefaultSessionCookie()
	recorder := httptest.NewRecorder()

	err := cookie.Set(recorder, "opaque-session-token", time.Now())
	if !errors.Is(err, ErrInvalidSessionCookieExpiration) {
		t.Fatalf("Set() error = %v, want %v", err, ErrInvalidSessionCookieExpiration)
	}
}

func TestSessionCookieRead(t *testing.T) {
	cookie := NewDefaultSessionCookie()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  DefaultSessionCookieName,
		Value: "opaque-session-token",
	})

	got, err := cookie.Read(req)
	if err != nil {
		t.Fatalf("Read() error = %v, want nil", err)
	}

	if got != "opaque-session-token" {
		t.Fatalf("Read() = %q, want opaque-session-token", got)
	}
}

func TestSessionCookieReadNotFound(t *testing.T) {
	cookie := NewDefaultSessionCookie()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	_, err := cookie.Read(req)
	if !errors.Is(err, ErrSessionCookieNotFound) {
		t.Fatalf("Read() error = %v, want %v", err, ErrSessionCookieNotFound)
	}
}

func TestSessionCookieReadRejectsEmptyValue(t *testing.T) {
	cookie := NewDefaultSessionCookie()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  DefaultSessionCookieName,
		Value: "",
	})

	_, err := cookie.Read(req)
	if !errors.Is(err, ErrSessionCookieNotFound) &&
		!errors.Is(err, ErrInvalidSessionCookie) {
		t.Fatalf("Read() error = %v, want cookie validation error", err)
	}
}

func TestSessionCookieClear(t *testing.T) {
	cookie := NewDefaultSessionCookie()
	recorder := httptest.NewRecorder()

	cookie.Clear(recorder)

	result := recorder.Result()
	defer result.Body.Close()

	cookies := result.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("Set-Cookie count = %d, want 1", len(cookies))
	}

	got := cookies[0]

	if got.Name != DefaultSessionCookieName {
		t.Fatalf("Name = %q, want %q", got.Name, DefaultSessionCookieName)
	}
	if got.Value != "" {
		t.Fatalf("Value = %q, want empty", got.Value)
	}
	if got.Path != "/" {
		t.Fatalf("Path = %q, want /", got.Path)
	}
	if got.MaxAge != -1 {
		t.Fatalf("MaxAge = %d, want -1", got.MaxAge)
	}
	if !got.HttpOnly {
		t.Fatal("HttpOnly = false, want true")
	}
	if got.Secure != true {
		t.Fatal("Secure = false, want true")
	}
	if got.SameSite != http.SameSiteLaxMode {
		t.Fatalf("SameSite = %v, want SameSiteLaxMode", got.SameSite)
	}
}

func TestSessionCookieSetDoesNotExposeTokenInErrors(t *testing.T) {
	cookie := NewDefaultSessionCookie()
	recorder := httptest.NewRecorder()

	err := cookie.Set(recorder, "   ", time.Now().Add(time.Hour))
	if err == nil {
		t.Fatal("Set() error = nil, want error")
	}

	if strings.Contains(err.Error(), "   ") {
		t.Fatalf("error contains session token: %v", err)
	}
}
