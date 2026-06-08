// Runtime configuration.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Runtime configuration for the auth service.
type Config struct {
	// Port the HTTP server listens on.
	Port string
	// HMAC secret used to sign and verify tokens (shared with other services to validate tokens).
	JWTSecret []byte
	// Token validity duration.
	TokenTTL time.Duration
	// Token issuer ("iss" claim).
	Issuer string
	// Origins permitted by CORS.
	AllowedOrigins []string
}

// Reads configuration from environment variables.
// Returns an error when JWT secret is missing.
func Load() (*Config, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, fmt.Errorf("JWT_SECRET environment variable is required and must not be empty")
	}
	if len(secret) < 16 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 16 characters")
	}

	ttl := 60 * time.Minute
	if raw := os.Getenv("TOKEN_TTL"); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid TOKEN_TTL: %w", err)
		}
		ttl = parsed
	}

	origins := []string{}
	if raw := os.Getenv("ALLOWED_ORIGINS"); raw != "" {
		origins = splitAndTrim(raw)
	}

	return &Config{
		Port:           getEnv("PORT", "8080"),
		JWTSecret:      []byte(secret),
		TokenTTL:       ttl,
		Issuer:         getEnv("JWT_ISSUER", "fency-auth"),
		AllowedOrigins: origins,
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitAndTrim(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
