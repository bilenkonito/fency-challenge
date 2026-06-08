package auth

import (
	"testing"
	"time"

	"fency/auth-service/internal/users"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	repo, err := users.NewMemoryRepository("alice", "s3cret-password")
	if err != nil {
		t.Fatalf("seed repo: %v", err)
	}
	return NewService(repo, []byte("test-secret-at-least-16-chars"), time.Hour, "fency-auth")
}

func TestLoginSuccessAndValidate(t *testing.T) {
	svc := newTestService(t)

	token, exp, err := svc.Login("alice", "s3cret-password")
	if err != nil {
		t.Fatalf("unexpected login error: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if !exp.After(time.Now()) {
		t.Fatal("expected expiry in the future")
	}

	claims, err := svc.Validate(token)
	if err != nil {
		t.Fatalf("validate failed: %v", err)
	}
	if claims.Username != "alice" {
		t.Fatalf("expected username alice, got %q", claims.Username)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	svc := newTestService(t)
	if _, _, err := svc.Login("alice", "wrong"); err != ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLoginUnknownUser(t *testing.T) {
	svc := newTestService(t)
	if _, _, err := svc.Login("bob", "whatever"); err != ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestValidateRejectsTamperedToken(t *testing.T) {
	svc := newTestService(t)
	token, _, _ := svc.Login("alice", "s3cret-password")
	if _, err := svc.Validate(token + "tamper"); err == nil {
		t.Fatal("expected error for tampered token")
	}
}

func TestValidateRejectsWrongSecret(t *testing.T) {
	svc := newTestService(t)
	token, _, _ := svc.Login("alice", "s3cret-password")

	other := NewService(nil, []byte("a-different-secret-16chars"), time.Hour, "fency-auth")
	if _, err := other.Validate(token); err == nil {
		t.Fatal("expected validation to fail with a different secret")
	}
}
