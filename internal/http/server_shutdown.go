package http

import (
	"context"
	stdhttp "net/http"
)

// Shutdown menghentikan HTTP server secara graceful menggunakan context
// yang diberikan untuk mengendalikan batas waktu shutdown.
func Shutdown(ctx context.Context, server *stdhttp.Server) error {
	if server == nil {
		return ErrInvalidServerDependency
	}

	return server.Shutdown(ctx)
}
