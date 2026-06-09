// Owns all storage for the other backend services behind a small HTTP API.
// Storage backend is configured through an ODBC DSN.
// A local SQLite database is used by default.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/crypto/bcrypt"

	"fency/data-service/internal/config"
	"fency/data-service/internal/handlers"
	"fency/data-service/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	st, err := store.Open(cfg.ODBCDSN)
	if err != nil {
		log.Fatalf("failed to open store: %v", err)
	}
	defer st.Close()

	if err := st.Migrate(); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.SeedPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("failed to hash seed password: %v", err)
	}
	if err := st.Seed(store.User{ID: "u_1", Username: cfg.SeedUser, PasswordHash: string(hash)}, cfg.SeedDomains); err != nil {
		log.Fatalf("failed to seed store: %v", err)
	}

	h := handlers.New(st, cfg.APIKey)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           h.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("data-service listening on :%s (DSN backend configured via ODBC)", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
	log.Println("data-service stopped")
}
