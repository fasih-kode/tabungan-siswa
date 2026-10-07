package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewLoginPageHandlerRejectsNilTemplate(t *testing.T) {
	if _, err := NewLoginPageHandler(nil); err == nil {
		t.Fatal("NewLoginPageHandler() error = nil, want error")
	}
}

func TestLoginPageHandlerRendersViewModel(t *testing.T) {
	handler := newTestLoginPageHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	recorder := httptest.NewRecorder()

	handler.Get(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	if got := recorder.Body.String(); got != "<html><body>Login</body></html>" {
		t.Fatalf("body = %q, want rendered view model", got)
	}
}
