package app

import (
	"testing"
	"time"
)

func TestNewApplication(t *testing.T) {
	application, err := NewApplication(
		nil,
		":8080",
		24*time.Hour,
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
	application, err := NewApplication(
		nil,
		":8080",
		0,
	)

	if err == nil {
		t.Fatal("NewApplication() error = nil, want error")
	}

	if application.Server != nil {
		t.Fatal("application.Server != nil, want nil")
	}
}
