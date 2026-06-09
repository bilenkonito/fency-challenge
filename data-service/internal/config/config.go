// Loads runtime configuration.
package config

import (
	"fmt"
	"os"
	"strings"
)

// Holds the data-service runtime configuration.
type Config struct {
	// HTTP server port.
	Port string
	// ODBC connection string identifying the storage backend.
	// Defaults to a local SQLite database for development.
	ODBCDSN string
	// Shared secret callers must present as a Bearer token.
	APIKey string
	// Seed data, applied once on first start when the tables are empty.
	SeedUser     string
	SeedPassword string
	SeedDomains  []string
}

// Reads configuration from environment.
func Load() (*Config, error) {
	apiKey := os.Getenv("DATA_SERVICE_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("DATA_SERVICE_API_KEY is required and must not be empty")
	}
	if len(apiKey) < 16 {
		return nil, fmt.Errorf("DATA_SERVICE_API_KEY must be at least 16 characters")
	}

	return &Config{
		Port:         getEnv("PORT", "9090"),
		ODBCDSN:      getEnv("ODBC_DSN", "Driver=SQLite3;Database=/data/fency.db;"),
		APIKey:       apiKey,
		SeedUser:     getEnv("SEED_USER", "admin"),
		SeedPassword: getEnv("SEED_USER_PASSWORD", "ChangeMe123!"),
		SeedDomains:  splitAndTrim(getEnv("SEED_DOMAINS", "malware-example.com,phishing-example.net")),
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
