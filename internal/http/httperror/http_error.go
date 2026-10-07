package httperror

import (
	"errors"
	"net/http"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/service"
)

func Status(err error) int {
	switch {
	case errors.Is(err, service.ErrInvalidCredentials),
		errors.Is(err, service.ErrInvalidActor):
		return http.StatusUnauthorized

	case errors.Is(err, service.ErrForbidden),
		errors.Is(err, service.ErrScopeViolation):
		return http.StatusForbidden

	case errors.Is(err, repository.ErrNotFound):
		return http.StatusNotFound

	case errors.Is(err, repository.ErrConflict),
		errors.Is(err, domain.ErrInvalidTransition),
		errors.Is(err, domain.ErrAlreadySettled),
		errors.Is(err, domain.ErrAlreadyCancelled),
		errors.Is(err, domain.ErrNegativeBalance),
		errors.Is(err, domain.ErrSessionAlreadyRevoked):
		return http.StatusConflict

	case errors.Is(err, domain.ErrInvalidAmount),
		errors.Is(err, domain.ErrInvalidDateRange),
		errors.Is(err, domain.ErrInvalidValue),
		errors.Is(err, domain.ErrInvalidID),
		errors.Is(err, domain.ErrInvalidUsername),
		errors.Is(err, domain.ErrInvalidPasswordHash),
		errors.Is(err, domain.ErrInvalidAction),
		errors.Is(err, domain.ErrEmptyRejectionReason),
		errors.Is(err, domain.ErrEmptyName),
		errors.Is(err, domain.ErrInvalidClassLevel),
		errors.Is(err, domain.ErrInvalidSessionTokenHash),
		errors.Is(err, domain.ErrInvalidSessionExpiration):
		return http.StatusBadRequest

	default:
		return http.StatusInternalServerError
	}
}

func Write(w http.ResponseWriter, err error) {
	status := Status(err)
	http.Error(w, http.StatusText(status), status)
}
