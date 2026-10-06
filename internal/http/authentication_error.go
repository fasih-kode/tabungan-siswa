package http

import (
	stdhttp "net/http"

	"github.com/fasih/tabungan-siswa/internal/http/autherror"
)

func WriteAuthenticationError(w stdhttp.ResponseWriter, err error) {
	autherror.Write(w, err)
}

func AuthenticationErrorStatus(err error) int {
	return autherror.Status(err)
}
