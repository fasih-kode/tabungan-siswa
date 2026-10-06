package http

import (
	stdhttp "net/http"
)

// NewServer membangun HTTP server dari address dan root handler.
//
// Lifecycle shutdown belum menjadi tanggung jawab constructor ini.
func NewServer(addr string, handler stdhttp.Handler) (*stdhttp.Server, error) {
	if handler == nil {
		return nil, ErrInvalidServerDependency
	}

	return &stdhttp.Server{
		Addr:    addr,
		Handler: handler,
	}, nil
}
