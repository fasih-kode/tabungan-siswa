package http

import (
	"errors"
	stdhttp "net/http"

	"github.com/fasih/tabungan-siswa/internal/service"
)

func WriteAuthenticationError(w stdhttp.ResponseWriter, err error) {
	status := AuthenticationErrorStatus(err)

	w.Header().Set("Cache-Control", "no-store")
	stdhttp.Error(w, stdhttp.StatusText(status), status)
}

func AuthenticationErrorStatus(err error) int {
	switch {
	case errors.Is(err, service.ErrInvalidCredentials),
		errors.Is(err, service.ErrInvalidActor):
		return stdhttp.StatusUnauthorized
	default:
		return stdhttp.StatusInternalServerError
	}
}
