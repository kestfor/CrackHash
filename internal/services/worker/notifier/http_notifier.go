package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/kestfor/CrackHash/internal/services/worker"
)

type HTTPNotifierConfig struct {
	NotifyURL string `yaml:"notify_url"`
}

type httpNotifier struct {
	url    string
	client *http.Client
}

func NewHTTPNotifier(config *HTTPNotifierConfig) *httpNotifier {
	return &httpNotifier{
		url:    config.NotifyURL,
		client: &http.Client{},
	}
}

func (n *httpNotifier) Notify(result *worker.TaskProgress) error {
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal task progress: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("notification failed with status: %s", resp.Status)
	}

	slog.Debug("progress notification sent",
		slog.String("task_id", result.TaskID.String()),
		slog.String("status", string(result.Status)),
	)

	return nil
}
