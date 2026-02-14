package healthchecker

import (
	"context"
	"fmt"
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
		client: &http.Client{},
		config: config,
	}
}

func (h *httpHealthChecker) Check() error {
	tries := 0

	for tries < h.config.MaxTries {
		tries++
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.config.URL, nil)
		if err != nil {
			return err
		}

		resp, err := h.client.Do(req)
		if err != nil {
			continue
		}

		_ = resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			continue
		}

		return nil
	}

	return fmt.Errorf("max tries reached, service not healthy")
}

func (h *httpHealthChecker) NotifyFailure() {

}
