package store

import (
	"path/filepath"
	"testing"
)

// Opens a fresh SQLite database (via ODBC) in a temp dir.
// Skips the test when the SQLite ODBC driver is not installed on the machine.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	dsn := "Driver=SQLite3;Database=" + dbPath + ";"
	s, err := Open(dsn)
	if err != nil {
		t.Skipf("skipping: cannot open SQLite ODBC store (driver missing?): %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if err := s.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return s
}

func TestUserSeedAndFind(t *testing.T) {
	s := newTestStore(t)
	if err := s.Seed(User{ID: "u_1", Username: "alice", PasswordHash: "hash"}, nil); err != nil {
		t.Fatalf("seed: %v", err)
	}

	u, err := s.FindUser("alice")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if u.ID != "u_1" || u.PasswordHash != "hash" {
		t.Fatalf("unexpected user: %+v", u)
	}

	if _, err := s.FindUser("bob"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestSeedIsIdempotent(t *testing.T) {
	s := newTestStore(t)
	user := User{ID: "u_1", Username: "alice", PasswordHash: "hash"}
	if err := s.Seed(user, []string{"evil.com"}); err != nil {
		t.Fatalf("seed 1: %v", err)
	}
	// Second seed with different data must not overwrite or duplicate.
	if err := s.Seed(User{ID: "u_2", Username: "mallory", PasswordHash: "x"}, []string{"other.com"}); err != nil {
		t.Fatalf("seed 2: %v", err)
	}
	domains, _ := s.ListDomains()
	if len(domains) != 1 || domains[0] != "evil.com" {
		t.Fatalf("seed not idempotent, domains=%v", domains)
	}
	if _, err := s.FindUser("mallory"); err != ErrNotFound {
		t.Fatalf("expected only the original user to exist")
	}
}

func TestBlacklistLifecycle(t *testing.T) {
	s := newTestStore(t)

	added, err := s.AddDomain("evil.com")
	if err != nil || !added {
		t.Fatalf("add: added=%v err=%v", added, err)
	}
	// Adding again is a no-op.
	added, _ = s.AddDomain("evil.com")
	if added {
		t.Fatal("expected duplicate add to report added=false")
	}

	domains, _ := s.ListDomains()
	if len(domains) != 1 || domains[0] != "evil.com" {
		t.Fatalf("unexpected list: %v", domains)
	}

	// Exact and subdomain matches.
	for _, d := range []string{"evil.com", "mail.evil.com"} {
		ok, err := s.IsBlacklisted(d)
		if err != nil || !ok {
			t.Fatalf("expected %q blacklisted, ok=%v err=%v", d, ok, err)
		}
	}
	if ok, _ := s.IsBlacklisted("good.com"); ok {
		t.Fatal("good.com should not be blacklisted")
	}

	removed, err := s.RemoveDomain("evil.com")
	if err != nil || !removed {
		t.Fatalf("remove: removed=%v err=%v", removed, err)
	}
	if removed, _ := s.RemoveDomain("evil.com"); removed {
		t.Fatal("expected second remove to report removed=false")
	}
}
