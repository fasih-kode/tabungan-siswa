package autherror

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fasih/tabungan-siswa/internal/service"
)

func TestStatus(t *testing.T) {
	if got := Status(service.ErrInvalidCredentials); got != http.StatusUnauthorized {
		t.Fatalf("Status() = %d, want %d", got, http.StatusUnauthorized)
	}
	if got := Status(service.ErrInvalidActor); got != http.StatusUnauthorized {
		t.Fatalf("Status() = %d, want %d", got, http.StatusUnauthorized)
	}
	if got := Status(errors.New("internal")); got != http.StatusInternalServerError {
		t.Fatalf("Status() = %d, want %d", got, http.StatusInternalServerError)
	}
}

func TestWriteDoesNotLeakError(t *testing.T) {
	recorder := httptest.NewRecorder()
	Write(recorder, errors.New("database password leaked"))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	if recorder.Body.String() != "Internal Server Error\n" {
		t.Fatalf("body = %q, want generic error", recorder.Body.String())
	}
	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
}
