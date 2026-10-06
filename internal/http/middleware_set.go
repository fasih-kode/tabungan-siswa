package http

import (
	"github.com/fasih/tabungan-siswa/internal/http/middleware"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/security"
)

// MiddlewareSet berisi seluruh HTTP middleware yang sudah tersedia.
//
// MiddlewareSet hanya menjadi container hasil construction.
// Ia tidak menentukan urutan middleware pada router.
type MiddlewareSet struct {
	Authentication *middleware.AuthenticationMiddleware
	CSRF           *middleware.CSRFMiddleware
}

// NewMiddlewareSet membangun seluruh HTTP middleware dari dependency
// yang disediakan oleh composition root.
func NewMiddlewareSet(
	repositories repository.RepositorySet,
	securitySet security.SecuritySet,
) (MiddlewareSet, error) {
	authentication, err := middleware.NewAuthenticationMiddleware(
		securitySet.SessionCookie,
		repositories.Sessions,
		repositories.Users,
	)
	if err != nil {
		return MiddlewareSet{}, err
	}

	csrf := middleware.NewCSRFMiddleware(
		securitySet.CSRFCookie,
	)

	return MiddlewareSet{
		Authentication: authentication,
		CSRF:           csrf,
	}, nil
}
