package autherror

import (
	"errors"
	"net/http"

	"github.com/fasih/tabungan-siswa/internal/service"
)

func Write(w http.ResponseWriter, err error) {
	status := Status(err)
	w.Header().Set("Cache-Control", "no-store")
	http.Error(w, http.StatusText(status), status)
}

func Status(err error) int {
	switch {
	case errors.Is(err, service.ErrInvalidCredentials),
		errors.Is(err, service.ErrInvalidActor):
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}
