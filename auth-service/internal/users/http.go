package users

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Data Service backed repository.
type HTTPRepository struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

// Creates a new data service backed repository.
func NewHTTPRepository(baseURL, apiKey string) *HTTPRepository {
	return &HTTPRepository{
		baseURL: baseURL,
		apiKey:  apiKey,
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

type userResponse struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"passwordHash"`
}

// Fetches a user from the data-service.
// 404 maps to ErrNotFound; any other non-200 is surfaced as an error.
func (r *HTTPRepository) FindByUsername(username string) (User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	endpoint := r.baseURL + "/users/" + url.PathEscape(username)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return User{}, err
	}
	req.Header.Set("Authorization", "Bearer "+r.apiKey)

	resp, err := r.client.Do(req)
	if err != nil {
		return User{}, fmt.Errorf("contact data-service: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var body userResponse
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			return User{}, fmt.Errorf("decode data-service response: %w", err)
		}
		return User{
			ID:           body.ID,
			Username:     body.Username,
			PasswordHash: []byte(body.PasswordHash),
		}, nil
	case http.StatusNotFound:
		return User{}, ErrNotFound
	default:
		return User{}, fmt.Errorf("data-service returned status %d", resp.StatusCode)
	}
}
