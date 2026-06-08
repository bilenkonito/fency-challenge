// User repository and domain model.
package users

import (
	"errors"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

// Returned when user does not exist.
var ErrNotFound = errors.New("user not found")

type User struct {
	ID           string
	Username     string
	PasswordHash []byte // Only bcrypt hash of password is stored.
}

type Repository interface {
	FindByUsername(username string) (User, error)
}

// Thread-safe in-memory user repository.
type MemoryRepository struct {
	mu    sync.RWMutex
	users map[string]User
}

// Creates a new user repository seeded with a predefined test user.
func NewMemoryRepository(seedUsername, seedPassword string) (*MemoryRepository, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(seedPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	repo := &MemoryRepository{users: make(map[string]User)}
	repo.users[seedUsername] = User{
		ID:           "u_1",
		Username:     seedUsername,
		PasswordHash: hash,
	}
	return repo, nil
}

// Finds a user by username.
func (r *MemoryRepository) FindByUsername(username string) (User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	u, ok := r.users[username]
	if !ok {
		return User{}, ErrNotFound
	}
	return u, nil
}
