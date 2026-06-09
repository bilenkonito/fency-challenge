// Bearer API-key authentication middleware.
package apikey

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"fency/data-service/internal/httputil"
)

// Returns a handler wrapper that requires a matching Bearer API key.
func Middleware(expected string) func(http.Handler) http.Handler {
	expectedBytes := []byte(expected)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := bearerToken(r)
			if token == "" || subtle.ConstantTimeCompare([]byte(token), expectedBytes) != 1 {
				httputil.WriteError(w, http.StatusUnauthorized, "invalid or missing API key")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(h) > len(prefix) && strings.EqualFold(h[:len(prefix)], prefix) {
		return strings.TrimSpace(h[len(prefix):])
	}
	return ""
}
