// Holds small JSON HTTP helpers for the data-service.
package httputil

import (
	"encoding/json"
	"net/http"
)

// Standard error.
type ErrorResponse struct {
	Error string `json:"error"`
}

// Serialises value as JSON with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

// Writes a JSON error response.
func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, ErrorResponse{Error: msg})
}
