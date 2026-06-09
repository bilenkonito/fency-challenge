// Decodes requests, calls the store, and encodes responses.
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"fency/data-service/internal/apikey"
	"fency/data-service/internal/httputil"
	"fency/data-service/internal/store"
)

const maxBodyBytes = 8 * 1024

// Exposes the data-service endpoints over the store.
type Handler struct {
	store  *store.Store
	apiKey string
}

// Creates a new handler.
func New(s *store.Store, key string) *Handler {
	return &Handler{store: s, apiKey: key}
}

// Registers all endpoints.
// Health is unauthenticated (used by container health checks); everything else requires the Bearer API key.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.health)

	protected := http.NewServeMux()
	protected.HandleFunc("GET /users/{username}", h.getUser)
	protected.HandleFunc("GET /blacklist", h.listDomains)
	protected.HandleFunc("GET /blacklist/check", h.checkDomain)
	protected.HandleFunc("POST /blacklist", h.addDomain)
	protected.HandleFunc("DELETE /blacklist/{domain}", h.removeDomain)

	guard := apikey.Middleware(h.apiKey)
	mux.Handle("/users/", guard(protected))
	mux.Handle("/blacklist", guard(protected))
	mux.Handle("/blacklist/", guard(protected))
	return mux
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) getUser(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.PathValue("username"))
	if username == "" {
		httputil.WriteError(w, http.StatusBadRequest, "username is required")
		return
	}
	u, err := h.store.FindUser(username)
	if errors.Is(err, store.ErrNotFound) {
		httputil.WriteError(w, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, u)
}

func (h *Handler) listDomains(w http.ResponseWriter, _ *http.Request) {
	domains, err := h.store.ListDomains()
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string][]string{"domains": domains})
}

func (h *Handler) checkDomain(w http.ResponseWriter, r *http.Request) {
	domain, ok := sanitizeDomain(r.URL.Query().Get("domain"))
	if !ok {
		httputil.WriteError(w, http.StatusBadRequest, "valid domain query parameter is required")
		return
	}
	blacklisted, err := h.store.IsBlacklisted(domain)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"domain":      domain,
		"blacklisted": blacklisted,
	})
}

type domainRequest struct {
	Domain string `json:"domain"`
}

func (h *Handler) addDomain(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	var req domainRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	domain, ok := sanitizeDomain(req.Domain)
	if !ok {
		httputil.WriteError(w, http.StatusBadRequest, "valid domain is required")
		return
	}
	added, err := h.store.AddDomain(domain)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	status := http.StatusOK
	if added {
		status = http.StatusCreated
	}
	httputil.WriteJSON(w, status, map[string]interface{}{"domain": domain, "added": added})
}

func (h *Handler) removeDomain(w http.ResponseWriter, r *http.Request) {
	domain, ok := sanitizeDomain(r.PathValue("domain"))
	if !ok {
		httputil.WriteError(w, http.StatusBadRequest, "valid domain is required")
		return
	}
	removed, err := h.store.RemoveDomain(domain)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if !removed {
		httputil.WriteError(w, http.StatusNotFound, "domain not found")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{"domain": domain, "removed": removed})
}

// Soft defensive check.
// Only guard against empty/oversized/whitespaced values.
// Proper normalisation and validation happen at the public edge.
func sanitizeDomain(raw string) (string, bool) {
	d := strings.ToLower(strings.TrimSpace(raw))
	if d == "" || len(d) > 253 || strings.ContainsAny(d, " \t\r\n/") || !strings.Contains(d, ".") {
		return "", false
	}
	return d, true
}
