package http

import (
	"html/template"

	"github.com/fasih/tabungan-siswa/internal/security"
	"github.com/fasih/tabungan-siswa/internal/service"
)

// HandlerSet berisi seluruh HTTP handler yang sudah tersedia.
//
// HandlerSet hanya menjadi container hasil construction.
// Ia tidak menjalankan routing, middleware, atau business logic.
type HandlerSet struct {
	Authentication *AuthenticationHandler
	LoginPage      *LoginPageHandler
	Dashboard      *DashboardPageHandler
	AcademicYear   *AcademicYearPageHandler
	Class          *ClassPageHandler
}

// NewHandlerSet membangun seluruh HTTP handler dari dependency yang
// disediakan oleh composition root.
func NewHandlerSet(
	services service.ServiceSet,
	securitySet security.SecuritySet,
	templates *template.Template,
) (HandlerSet, error) {
	authentication, err := NewAuthenticationHandler(
		services.Authentication,
		securitySet.SessionExpiration,
		securitySet.SessionCookie,
	)
	if err != nil {
		return HandlerSet{}, err
	}

	loginPage, err := NewLoginPageHandler(templates)
	if err != nil {
		return HandlerSet{}, err
	}

	dashboard, err := NewDashboardPageHandler(templates)
	if err != nil {
		return HandlerSet{}, err
	}

	academicYear, err := NewAcademicYearPageHandler(services.AcademicYear, templates)
	if err != nil {
		return HandlerSet{}, err
	}

	classHandler, err := NewClassPageHandler(services.Class, services.AcademicYear, templates)
	if err != nil {
		return HandlerSet{}, err
	}

	return HandlerSet{
		Authentication: authentication,
		LoginPage:      loginPage,
		Dashboard:      dashboard,
		AcademicYear:   academicYear,
		Class:          classHandler,
	}, nil
}
