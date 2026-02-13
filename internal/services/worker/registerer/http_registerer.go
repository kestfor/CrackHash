package registerer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type HTTPRegistererConfig struct {
	RegisterURL string `yaml:"register_url"`
}

type HTTPRegisterResponse struct {
	ID uuid.UUID `json:"id"`
}

type httpRegisterer struct {
	url    string
	client *http.Client
}

func NewHTTPRegisterer(config *HTTPRegistererConfig) *httpRegisterer {
	return &httpRegisterer{
		url:    config.RegisterURL,
		client: &http.Client{},
	}
}

func (r *httpRegisterer) Register() (uuid.UUID, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.url, nil)
	if err != nil {
		return uuid.Nil, err
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return uuid.Nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {

		// TODO ERROR LOG
		bodyBytes, _ := io.ReadAll(resp.Body)

		return uuid.Nil, fmt.Errorf("failed to register: %s", string(bodyBytes))

	}

	var registerResponse HTTPRegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&registerResponse); err != nil {
		return uuid.Nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if registerResponse.ID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("invalid response: %v", registerResponse)
	}

	return registerResponse.ID, nil
}
