package autherror

import (
	"errors"
	"net/http"

	"github.com/fasih/tabungan-siswa/internal/service"
)

func publicMessage(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "Permintaan tidak valid. Periksa kembali data yang dikirim."
	case http.StatusUnauthorized:
		return "Silakan masuk kembali untuk melanjutkan."
	case http.StatusForbidden:
		return "Anda tidak memiliki izin untuk mengakses halaman ini."
	case http.StatusNotFound:
		return "Halaman atau data tidak ditemukan."
	case http.StatusConflict:
		return "Data berubah atau bertentangan dengan kondisi saat ini. Muat ulang halaman lalu coba lagi."
	default:
		return "Terjadi kesalahan pada server. Silakan coba lagi."
	}
}

func Write(w http.ResponseWriter, err error) {
	status := Status(err)
	w.Header().Set("Cache-Control", "no-store")
	http.Error(w, publicMessage(status), status)
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
