// Persists users and the domain blacklist in a SQL database reached over ODBC.
// Backend is configurable purely through the ODBC DSN (SQLite by default).
//
// Only standard SQL and `?` parameter markers are used so the same code works
// against any ODBC-reachable database; timestamps are generated in Go rather
// than via database-specific functions to keep the schema portable.
package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/alexbrainman/odbc" // registers the "odbc" sql driver
)

// Returned when a requested record does not exist.
var ErrNotFound = errors.New("not found")

// Stored application user.
type User struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"passwordHash"` // Only bcrypt hash of password is stored.
}

// Wraps a database/sql handle opened against an ODBC DSN.
type Store struct {
	db *sql.DB
}

// Connects to the database identified by the ODBC DSN and verifies the connection.
func Open(dsn string) (*Store, error) {
	db, err := sql.Open("odbc", dsn)
	if err != nil {
		return nil, fmt.Errorf("open odbc connection: %w", err)
	}
	// SQLite over ODBC does not tolerate many concurrent writers; keep the pool
	// small and predictable.
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &Store{db: db}, nil
}

// Releases the underlying connection pool.
func (s *Store) Close() error { return s.db.Close() }

// Creates the required tables if they do not already exist.
func (s *Store) Migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id            VARCHAR(64) PRIMARY KEY,
			username      VARCHAR(255) NOT NULL UNIQUE,
			password_hash VARCHAR(255) NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS blacklist (
			domain     VARCHAR(253) PRIMARY KEY,
			created_at VARCHAR(32) NOT NULL
		)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}

// Inserts the predefined user and starting domains, but only when the
// respective tables are still empty, so restarts never clobber live data.
func (s *Store) Seed(user User, domains []string) error {
	var userCount int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&userCount); err != nil {
		return err
	}
	if userCount == 0 {
		if _, err := s.db.Exec(
			`INSERT INTO users (id, username, password_hash) VALUES (?, ?, ?)`,
			user.ID, user.Username, user.PasswordHash,
		); err != nil {
			return fmt.Errorf("seed user: %w", err)
		}
	}

	var domainCount int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM blacklist`).Scan(&domainCount); err != nil {
		return err
	}
	if domainCount == 0 {
		for _, d := range domains {
			if _, err := s.AddDomain(d); err != nil {
				return fmt.Errorf("seed domain %q: %w", d, err)
			}
		}
	}
	return nil
}

// Looks up a user by username.
func (s *Store) FindUser(username string) (User, error) {
	var u User
	err := s.db.QueryRow(
		`SELECT id, username, password_hash FROM users WHERE username = ?`,
		username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	return u, nil
}

// Returns the blacklist sorted alphabetically.
func (s *Store) ListDomains() ([]string, error) {
	rows, err := s.db.Query(`SELECT domain FROM blacklist ORDER BY domain ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	domains := []string{}
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		domains = append(domains, d)
	}
	return domains, rows.Err()
}

// Inserts a domain.
// Returns added=false if the domain was already present.
// Caller is responsible for normalising/validating the value.
func (s *Store) AddDomain(domain string) (bool, error) {
	var exists int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM blacklist WHERE domain = ?`, domain).Scan(&exists); err != nil {
		return false, err
	}
	if exists > 0 {
		return false, nil
	}
	if _, err := s.db.Exec(
		`INSERT INTO blacklist (domain, created_at) VALUES (?, ?)`,
		domain, time.Now().UTC().Format(time.RFC3339),
	); err != nil {
		return false, err
	}
	return true, nil
}

// Deletes a domain, returning removed=false if it was absent.
func (s *Store) RemoveDomain(domain string) (bool, error) {
	res, err := s.db.Exec(`DELETE FROM blacklist WHERE domain = ?`, domain)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		// Some ODBC drivers do not report affected rows; fall back to a lookup.
		var exists int
		if qErr := s.db.QueryRow(`SELECT COUNT(*) FROM blacklist WHERE domain = ?`, domain).Scan(&exists); qErr != nil {
			return false, qErr
		}
		return exists == 0, nil
	}
	return affected > 0, nil
}

// Reports whether the domain matches an entry exactly or is a subdomain of one.
func (s *Store) IsBlacklisted(domain string) (bool, error) {
	domains, err := s.ListDomains()
	if err != nil {
		return false, err
	}
	for _, blocked := range domains {
		if domain == blocked || endsWithSuffix(domain, "."+blocked) {
			return true, nil
		}
	}
	return false, nil
}

func endsWithSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
