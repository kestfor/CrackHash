package healthchecker

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type HTTPHealthCheckerConfig struct {
	URL      string        `yaml:"url"`
	Period   time.Duration `yaml:"period"`
	MaxTries int           `yaml:"max_tries"`
}

type httpHealthChecker struct {
	client *http.Client
	config *HTTPHealthCheckerConfig
}

func NewHTTPHealthChecker(config *HTTPHealthCheckerConfig) *httpHealthChecker {
	return &httpHealthChecker{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		config: config,
	}
}

// Check performs a single health check
func (h *httpHealthChecker) Check() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.config.URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("health check request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status: %s", resp.Status)
	}

	return nil
}

// NotifyFailure blocks until the worker fails health checks MaxTries times in a row
func (h *httpHealthChecker) NotifyFailure() {
	consecutiveFailures := 0
	ticker := time.NewTicker(h.config.Period)
	defer ticker.Stop()

	slog.Info("starting health check monitoring",
		slog.String("url", h.config.URL),
		slog.Duration("period", h.config.Period),
		slog.Int("max_tries", h.config.MaxTries),
	)

	for {
		<-ticker.C

		err := h.Check()
		if err != nil {
			consecutiveFailures++
			slog.Warn("health check failed",
				slog.String("url", h.config.URL),
				slog.Int("consecutive_failures", consecutiveFailures),
				slog.Int("max_tries", h.config.MaxTries),
				slog.Any("error", err),
			)

			if consecutiveFailures >= h.config.MaxTries {
				slog.Error("worker marked as unhealthy after max retries",
					slog.String("url", h.config.URL),
					slog.Int("consecutive_failures", consecutiveFailures),
				)
				return
			}
		} else {
			if consecutiveFailures > 0 {
				slog.Info("health check recovered",
					slog.String("url", h.config.URL),
					slog.Int("previous_failures", consecutiveFailures),
				)
			}
			consecutiveFailures = 0
		}
	}
}
