package notifier

import (
	"bytes"
	"context"
	"encoding/json"
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

func (n *httpNotifier) Notify(result *worker.TaskResult) error {

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	return nil
}
