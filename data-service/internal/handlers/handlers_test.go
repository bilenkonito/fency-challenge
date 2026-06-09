package handlers

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"fency/data-service/internal/store"
)

const testKey = "test-api-key-at-least-16-chars"

func newTestServer(t *testing.T) (*httptest.Server, *store.Store) {
	t.Helper()
	dsn := "Driver=SQLite3;Database=" + filepath.Join(t.TempDir(), "h.db") + ";"
	st, err := store.Open(dsn)
	if err != nil {
		t.Skipf("skipping: SQLite ODBC driver unavailable: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	srv := httptest.NewServer(New(st, testKey).Routes())
	t.Cleanup(srv.Close)
	return srv, st
}

func do(t *testing.T, method, url, key, body string) *http.Response {
	t.Helper()
	var r *strings.Reader
	if body != "" {
		r = strings.NewReader(body)
	} else {
		r = strings.NewReader("")
	}
	req, err := http.NewRequest(method, url, r)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	return res
}

func TestHealthIsUnauthenticated(t *testing.T) {
	srv, _ := newTestServer(t)
	res := do(t, http.MethodGet, srv.URL+"/health", "", "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("health status = %d", res.StatusCode)
	}
}

func TestProtectedEndpointsRequireApiKey(t *testing.T) {
	srv, _ := newTestServer(t)
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/blacklist"},
		{http.MethodGet, "/users/admin"},
		{http.MethodPost, "/blacklist"},
	} {
		// No key.
		if res := do(t, tc.method, srv.URL+tc.path, "", ""); res.StatusCode != http.StatusUnauthorized {
			t.Fatalf("%s %s without key = %d, want 401", tc.method, tc.path, res.StatusCode)
		}
		// Wrong key.
		if res := do(t, tc.method, srv.URL+tc.path, "wrong-key-aaaaaaaaaaaaaaa", ""); res.StatusCode != http.StatusUnauthorized {
			t.Fatalf("%s %s wrong key = %d, want 401", tc.method, tc.path, res.StatusCode)
		}
	}
}

func TestBlacklistApiLifecycle(t *testing.T) {
	srv, _ := newTestServer(t)

	res := do(t, http.MethodPost, srv.URL+"/blacklist", testKey, `{"domain":"evil.com"}`)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("add = %d, want 201", res.StatusCode)
	}

	res = do(t, http.MethodGet, srv.URL+"/blacklist/check?domain=mail.evil.com", testKey, "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("check = %d", res.StatusCode)
	}

	res = do(t, http.MethodDelete, srv.URL+"/blacklist/evil.com", testKey, "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("delete = %d", res.StatusCode)
	}
	res = do(t, http.MethodDelete, srv.URL+"/blacklist/evil.com", testKey, "")
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("second delete = %d, want 404", res.StatusCode)
	}
}

func TestGetUser(t *testing.T) {
	srv, st := newTestServer(t)
	if err := st.Seed(store.User{ID: "u_1", Username: "admin", PasswordHash: "hash"}, nil); err != nil {
		t.Fatalf("seed: %v", err)
	}
	res := do(t, http.MethodGet, srv.URL+"/users/admin", testKey, "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("get user = %d", res.StatusCode)
	}
	res = do(t, http.MethodGet, srv.URL+"/users/nobody", testKey, "")
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("missing user = %d, want 404", res.StatusCode)
	}
}
