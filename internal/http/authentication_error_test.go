package http

import (
	"errors"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/fasih/tabungan-siswa/internal/service"
)

func TestAuthenticationErrorStatus(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{
			name: "invalid credentials",
			err:  service.ErrInvalidCredentials,
			want: stdhttp.StatusUnauthorized,
		},
		{
			name: "invalid actor",
			err:  service.ErrInvalidActor,
			want: stdhttp.StatusUnauthorized,
		},
		{
			name: "internal error",
			err:  errors.New("database unavailable"),
			want: stdhttp.StatusInternalServerError,
		},
		{
			name: "invalid dependency",
			err:  service.ErrInvalidDependency,
			want: stdhttp.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AuthenticationErrorStatus(tt.err); got != tt.want {
				t.Fatalf("AuthenticationErrorStatus() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestWriteAuthenticationError(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		wantStatus    int
		wantBody      string
		wantCacheCtrl string
	}{
		{
			name:          "invalid credentials",
			err:           service.ErrInvalidCredentials,
			wantStatus:    stdhttp.StatusUnauthorized,
			wantBody:      "Unauthorized\n",
			wantCacheCtrl: "no-store",
		},
		{
			name:          "internal error does not leak detail",
			err:           errors.New("database password leaked"),
			wantStatus:    stdhttp.StatusInternalServerError,
			wantBody:      "Internal Server Error\n",
			wantCacheCtrl: "no-store",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()

			WriteAuthenticationError(recorder, tt.err)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}

			if got := recorder.Body.String(); got != tt.wantBody {
				t.Fatalf("body = %q, want %q", got, tt.wantBody)
			}

			if got := recorder.Header().Get("Cache-Control"); got != tt.wantCacheCtrl {
				t.Fatalf("Cache-Control = %q, want %q", got, tt.wantCacheCtrl)
			}
		})
	}
}
