// HTTP routing for decoding requests, calling auth service, and encoding responses.
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"fency/auth-service/internal/auth"
	"fency/auth-service/internal/httputil"
)

// Cap request bodies to protect against oversized payloads.
const maxBodyBytes = 4 * 1024

type Handler struct {
	svc *auth.Service
}

// Creates a new handler.
func New(svc *auth.Service) *Handler {
	return &Handler{svc: svc}
}

// Registers all endpoints on a new mux.
func (h *Handler) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("POST /login", h.login)
	mux.HandleFunc("GET /validate", h.validate)
	return mux
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	Username  string    `json:"username"`
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	var req loginRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		httputil.WriteError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	token, exp, err := h.svc.Login(req.Username, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			httputil.WriteError(w, http.StatusUnauthorized, "invalid username or password")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, loginResponse{
		Token:     token,
		ExpiresAt: exp,
		Username:  req.Username,
	})
}

// Verifies a token and returns its claims when valid.
func (h *Handler) validate(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r)
	if token == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "missing bearer token")
		return
	}
	claims, err := h.svc.Validate(token)
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "invalid or expired token")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"valid":    true,
		"username": claims.Username,
		"subject":  claims.Subject,
	})
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(h) > len(prefix) && strings.EqualFold(h[:len(prefix)], prefix) {
		return strings.TrimSpace(h[len(prefix):])
	}
	return ""
}
