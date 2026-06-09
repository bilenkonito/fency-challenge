// Exposes a small HTTP API for credential-based login that issues JWTs and provides a token-validation endpoint.
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

	"fency/auth-service/internal/auth"
	"fency/auth-service/internal/config"
	"fency/auth-service/internal/handlers"
	"fency/auth-service/internal/httputil"
	"fency/auth-service/internal/users"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	repo := users.NewHTTPRepository(cfg.DataServiceURL, cfg.DataServiceAPIKey)

	svc := auth.NewService(repo, cfg.JWTSecret, cfg.TokenTTL, cfg.Issuer)
	h := handlers.New(svc)

	handler := httputil.CORS(cfg.AllowedOrigins)(h.Routes())

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Graceful shutdown on SIGINT/SIGTERM.
	go func() {
		log.Printf("auth-service listening on :%s (users via data-service %s)", cfg.Port, cfg.DataServiceURL)
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
	log.Println("auth-service stopped")
}
