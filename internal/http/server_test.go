package http

import (
	stdhttp "net/http"
	"testing"
)

func TestNewServer(t *testing.T) {
	handler := stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {})

	server, err := NewServer(":8080", handler)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	if server == nil {
		t.Fatal("NewServer() returned nil server")
	}

	if server.Addr != ":8080" {
		t.Fatalf("server.Addr = %q, want %q", server.Addr, ":8080")
	}

	if server.Handler == nil {
		t.Fatal("server.Handler = nil, want non-nil")
	}
}

func TestNewServerRejectsNilHandler(t *testing.T) {
	server, err := NewServer(":8080", nil)
	if err != ErrInvalidServerDependency {
		t.Fatalf(
			"NewServer() error = %v, want %v",
			err,
			ErrInvalidServerDependency,
		)
	}

	if server != nil {
		t.Fatalf("NewServer() server = %v, want nil", server)
	}
}
