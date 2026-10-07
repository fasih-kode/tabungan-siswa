package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/fasih/tabungan-siswa/internal/app"
	"github.com/fasih/tabungan-siswa/internal/platform/database"
)

const (
	serverAddr          = ":8080"
	databaseDSNEnv      = "TABUNGAN_SISWA_DATABASE_DSN"
	sessionLifetime     = 24 * time.Hour
	databasePingTimeout = 5 * time.Second
)

func main() {
	dsn := os.Getenv(databaseDSNEnv)
	if dsn == "" {
		log.Fatalf("%s is required", databaseDSNEnv)
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		databasePingTimeout,
	)
	defer cancel()

	db, err := database.OpenAndPing(ctx, dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := database.Close(db); err != nil {
			log.Printf("close database: %v", err)
		}
	}()

	protected := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.WriteHeader(http.StatusNoContent)
	})

	application, err := app.NewApplication(
		db,
		serverAddr,
		sessionLifetime,
		protected,
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("server listening on %s", application.Server.Addr)

	if err := application.Server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
