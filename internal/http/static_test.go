package http

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func newTestStaticAssetHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
}

func TestNewStaticAssetHandlerServesAsset(t *testing.T) {
	root := t.TempDir()

	cssPath := filepath.Join(root, "css", "app.css")
	if err := os.MkdirAll(filepath.Dir(cssPath), 0o755); err != nil {
		t.Fatal(err)
	}

	const want = "body { margin: 0; }\n"

	if err := os.WriteFile(cssPath, []byte(want), 0o644); err != nil {
		t.Fatal(err)
	}

	handler := NewStaticAssetHandler(root)

	req := httptest.NewRequest(
		http.MethodGet,
		"/static/css/app.css",
		nil,
	)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	if got := recorder.Body.String(); got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestNewStaticAssetHandlerReturnsNotFoundForMissingAsset(t *testing.T) {
	handler := NewStaticAssetHandler(t.TempDir())

	req := httptest.NewRequest(
		http.MethodGet,
		"/static/css/missing.css",
		nil,
	)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}
