package httperror_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/http/httperror"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/service"
)

func TestStatus(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"invalid credentials", service.ErrInvalidCredentials, http.StatusUnauthorized},
		{"invalid actor", service.ErrInvalidActor, http.StatusUnauthorized},
		{"forbidden", service.ErrForbidden, http.StatusForbidden},
		{"scope violation", service.ErrScopeViolation, http.StatusForbidden},
		{"not found", repository.ErrNotFound, http.StatusNotFound},
		{"conflict", repository.ErrConflict, http.StatusConflict},
		{"invalid transition", domain.ErrInvalidTransition, http.StatusConflict},
		{"already settled", domain.ErrAlreadySettled, http.StatusConflict},
		{"already cancelled", domain.ErrAlreadyCancelled, http.StatusConflict},
		{"negative balance", domain.ErrNegativeBalance, http.StatusConflict},
		{"session already revoked", domain.ErrSessionAlreadyRevoked, http.StatusConflict},
		{"invalid amount", domain.ErrInvalidAmount, http.StatusBadRequest},
		{"invalid date range", domain.ErrInvalidDateRange, http.StatusBadRequest},
		{"invalid value", domain.ErrInvalidValue, http.StatusBadRequest},
		{"invalid ID", domain.ErrInvalidID, http.StatusBadRequest},
		{"invalid username", domain.ErrInvalidUsername, http.StatusBadRequest},
		{"invalid password hash", domain.ErrInvalidPasswordHash, http.StatusBadRequest},
		{"invalid action", domain.ErrInvalidAction, http.StatusBadRequest},
		{"empty rejection reason", domain.ErrEmptyRejectionReason, http.StatusBadRequest},
		{"empty name", domain.ErrEmptyName, http.StatusBadRequest},
		{"invalid class level", domain.ErrInvalidClassLevel, http.StatusBadRequest},
		{"invalid session token hash", domain.ErrInvalidSessionTokenHash, http.StatusBadRequest},
		{"invalid session expiration", domain.ErrInvalidSessionExpiration, http.StatusBadRequest},
		{
			"wrapped not found",
			errors.Join(errors.New("lookup failed"), repository.ErrNotFound),
			http.StatusNotFound,
		},
		{
			"wrapped forbidden",
			errors.Join(errors.New("authorization failed"), service.ErrForbidden),
			http.StatusForbidden,
		},
		{
			"unknown internal error",
			errors.New("database password leaked"),
			http.StatusInternalServerError,
		},
		{
			"nil error",
			nil,
			http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := httperror.Status(tt.err); got != tt.want {
				t.Fatalf("Status() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestWriteUsesPublicStatusText(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "bad request",
			err:        domain.ErrInvalidValue,
			wantStatus: http.StatusBadRequest,
			wantBody:   "Bad Request\n",
		},
		{
			name:       "unauthorized",
			err:        service.ErrInvalidCredentials,
			wantStatus: http.StatusUnauthorized,
			wantBody:   "Unauthorized\n",
		},
		{
			name:       "forbidden",
			err:        service.ErrForbidden,
			wantStatus: http.StatusForbidden,
			wantBody:   "Forbidden\n",
		},
		{
			name:       "not found",
			err:        repository.ErrNotFound,
			wantStatus: http.StatusNotFound,
			wantBody:   "Not Found\n",
		},
		{
			name:       "conflict",
			err:        repository.ErrConflict,
			wantStatus: http.StatusConflict,
			wantBody:   "Conflict\n",
		},
		{
			name:       "internal error does not leak detail",
			err:        errors.New("database password leaked"),
			wantStatus: http.StatusInternalServerError,
			wantBody:   "Internal Server Error\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()

			httperror.Write(recorder, tt.err)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}

			if got := recorder.Body.String(); got != tt.wantBody {
				t.Fatalf("body = %q, want %q", got, tt.wantBody)
			}

			if tt.err != nil && strings.Contains(recorder.Body.String(), tt.err.Error()) {
				t.Fatalf("response leaked internal error: %q", tt.err.Error())
			}
		})
	}
}
