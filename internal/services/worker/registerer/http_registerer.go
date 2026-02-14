package registerer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type HTTPRegistererConfig struct {
	RegisterURL string `yaml:"register_url"`
	SelfPort    int    `yaml:"self_port"` // Port where this worker listens
}

type HTTPRegisterResponse struct {
	ID uuid.UUID `json:"id"`
}

type httpRegisterer struct {
	config *HTTPRegistererConfig
	client *http.Client
}

func NewHTTPRegisterer(config *HTTPRegistererConfig) *httpRegisterer {
	return &httpRegisterer{
		config: config,
		client: &http.Client{},
	}
}

func (r *httpRegisterer) Register() (uuid.UUID, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.config.RegisterURL, nil)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Send worker's listening port in header so manager can construct full address
	req.Header.Set("X-Worker-Port", strconv.Itoa(r.config.SelfPort))

	slog.Info("registering worker",
		slog.String("register_url", r.config.RegisterURL),
		slog.Int("self_port", r.config.SelfPort),
	)

	resp, err := r.client.Do(req)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return uuid.Nil, fmt.Errorf("failed to register: status=%s, body=%s", resp.Status, string(bodyBytes))
	}

	var registerResponse HTTPRegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&registerResponse); err != nil {
		return uuid.Nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if registerResponse.ID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("invalid response: empty id")
	}

	return registerResponse.ID, nil
}
