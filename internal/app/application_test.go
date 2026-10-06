package app

import (
	"net/http"
	"testing"
	"time"
)

func TestNewApplication(t *testing.T) {
	handler := http.HandlerFunc(func(
		http.ResponseWriter,
		*http.Request,
	) {
	})

	application, err := NewApplication(
		nil,
		":8080",
		24*time.Hour,
		handler,
	)
	if err != nil {
		t.Fatalf("NewApplication() error = %v", err)
	}

	if application.Server == nil {
		t.Fatal("application.Server = nil, want non-nil")
	}

	if application.Server.Addr != ":8080" {
		t.Fatalf(
			"application.Server.Addr = %q, want %q",
			application.Server.Addr,
			":8080",
		)
	}

	if application.Server.Handler == nil {
		t.Fatal("application.Server.Handler = nil, want non-nil")
	}
}

func TestNewApplicationRejectsInvalidSessionLifetime(t *testing.T) {
	handler := http.HandlerFunc(func(
		http.ResponseWriter,
		*http.Request,
	) {
	})

	application, err := NewApplication(
		nil,
		":8080",
		0,
		handler,
	)

	if err == nil {
		t.Fatal("NewApplication() error = nil, want error")
	}

	if application.Server != nil {
		t.Fatal("application.Server != nil, want nil")
	}
}

func TestNewApplicationRejectsNilProtectedHandler(t *testing.T) {
	application, err := NewApplication(
		nil,
		":8080",
		24*time.Hour,
		nil,
	)

	if err == nil {
		t.Fatal("NewApplication() error = nil, want error")
	}

	if application.Server != nil {
		t.Fatal("application.Server != nil, want nil")
	}
}
