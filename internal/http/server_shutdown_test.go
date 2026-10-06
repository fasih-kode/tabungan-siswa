package http

import (
	"context"
	stdhttp "net/http"
	"testing"
	"time"
)

func TestShutdown(t *testing.T) {
	server := &stdhttp.Server{
		Addr: ":0",
		Handler: stdhttp.HandlerFunc(func(
			stdhttp.ResponseWriter,
			*stdhttp.Request,
		) {
		}),
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := Shutdown(ctx, server); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}

func TestShutdownRejectsNilServer(t *testing.T) {
	ctx := context.Background()

	if err := Shutdown(ctx, nil); err != ErrInvalidServerDependency {
		t.Fatalf(
			"Shutdown() error = %v, want %v",
			err,
			ErrInvalidServerDependency,
		)
	}
}
