// Authentication logic for verifying credentials and issuing/validating signed JWTs.
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"fency/auth-service/internal/users"
)

// Always return ErrInvalidCredentials so callers cannot distinguish between wrong user and wrong password (avoids leaking which usernames exist).
var ErrInvalidCredentials = errors.New("invalid username or password")

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type Service struct {
	repo     users.Repository
	secret   []byte
	tokenTTL time.Duration
	issuer   string
	now      func() time.Time // simplifies testing
}

// Creates a new auth service.
func NewService(repo users.Repository, secret []byte, ttl time.Duration, issuer string) *Service {
	return &Service{
		repo:     repo,
		secret:   secret,
		tokenTTL: ttl,
		issuer:   issuer,
		now:      time.Now,
	}
}

// Verifies the supplied credentials and returns a signed JWT and its expiry time.
func (s *Service) Login(username, password string) (token string, expiresAt time.Time, err error) {
	user, err := s.repo.FindByUsername(username)
	if err != nil {
		// Run a dummy comparison to mitigate user-enumeration timing attacks.
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		return "", time.Time{}, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(password)); err != nil {
		return "", time.Time{}, ErrInvalidCredentials
	}

	now := s.now()
	exp := now.Add(s.tokenTTL)
	claims := Claims{
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			Issuer:    s.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, exp, nil
}

// Verifies a token string and returns its claims when valid.
// Enforces expected signing algorithm to prevent algorithm-confusion attacks (e.g. a token forged with "alg": "none").
func (s *Service) Validate(tokenString string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer(s.issuer))
	if err != nil {
		return nil, err
	}
	return claims, nil
}

// Precomputed bcrypt hash to ensure the time taken to respond is similar regardless of whether the username exists or not.
var dummyHash = []byte("$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy")
