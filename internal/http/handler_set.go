package http

import (
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/security"
	"github.com/fasih/tabungan-siswa/internal/service"
)

// HandlerSet berisi seluruh HTTP handler yang sudah tersedia.
//
// HandlerSet hanya menjadi container hasil construction.
// Ia tidak menjalankan routing, middleware, atau business logic.
type HandlerSet struct {
	Authentication *AuthenticationHandler
}

// NewHandlerSet membangun seluruh HTTP handler dari dependency yang
// disediakan oleh composition root.
func NewHandlerSet(
	services service.ServiceSet,
	repositories repository.RepositorySet,
	securitySet security.SecuritySet,
) (HandlerSet, error) {
	authentication, err := NewAuthenticationHandler(
		services.Authentication,
		repositories.Sessions,
		securitySet.SessionExpiration,
		securitySet.SessionCookie,
	)
	if err != nil {
		return HandlerSet{}, err
	}

	return HandlerSet{
		Authentication: authentication,
	}, nil
}
