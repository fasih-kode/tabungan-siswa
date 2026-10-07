package app

import (
	"database/sql"
	"embed"
	"html/template"
	"net/http"
	"time"

	httpapp "github.com/fasih/tabungan-siswa/internal/http"
	"github.com/fasih/tabungan-siswa/internal/repository/postgres"
	"github.com/fasih/tabungan-siswa/internal/security"
	"github.com/fasih/tabungan-siswa/internal/service"
)

//go:embed templates/layouts/*.html templates/partials/*.html templates/auth/*.html
var templateFiles embed.FS

// Application berisi HTTP server hasil seluruh application composition.
type Application struct {
	Server *http.Server
}

// NewApplication membangun seluruh dependency graph aplikasi.
//
// Composition root bertanggung jawab melakukan construction dan wiring.
// Business logic tetap berada pada service dan domain layer.
func NewApplication(
	db *sql.DB,
	addr string,
	sessionLifetime time.Duration,
	protected http.Handler,
) (Application, error) {
	repositories := postgres.NewRepositorySet(db)
	uow := postgres.NewUnitOfWorkManager(db)

	securitySet, err := security.NewSecuritySet(sessionLifetime)
	if err != nil {
		return Application{}, err
	}

	serviceSet, err := service.NewServiceSet(service.Dependencies{
		Repositories:   repositories,
		UOW:            uow,
		PasswordHasher: securitySet.PasswordHasher,
	})
	if err != nil {
		return Application{}, err
	}

	templates, err := template.ParseFS(
		templateFiles,
		"templates/layouts/*.html",
		"templates/partials/*.html",
		"templates/auth/*.html",
	)
	if err != nil {
		return Application{}, err
	}

	handlerSet, err := httpapp.NewHandlerSet(
		serviceSet,
		repositories,
		securitySet,
		templates,
	)
	if err != nil {
		return Application{}, err
	}

	staticAssets := httpapp.NewStaticAssetHandler("static")

	middlewareSet, err := httpapp.NewMiddlewareSet(
		repositories,
		securitySet,
	)
	if err != nil {
		return Application{}, err
	}

	router, err := httpapp.NewRouter(
		handlerSet,
		middlewareSet,
		protected,
		staticAssets,
	)
	if err != nil {
		return Application{}, err
	}

	server, err := httpapp.NewServer(addr, router)
	if err != nil {
		return Application{}, err
	}

	return Application{
		Server: server,
	}, nil
}
