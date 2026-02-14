package workerclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/kestfor/CrackHash/internal/services/worker"
)

const (
	pathCreateTask = "%s/api/v1/tasks/"
	pathDeleteTask = "%s/api/v1/tasks/%s"
	pathDoTask     = "%s/api/v1/tasks/%s/do"
)

type workerClient struct {
	address    string // raw address without scheme (e.g., "localhost:8081")
	baseURL    string // full URL with scheme (e.g., "http://localhost:8081")
	httpClient *http.Client
}

func (c *workerClient) Address() string {
	return c.address
}

func NewWorkerClient(address string) *workerClient {
	baseURL := address
	// Ensure baseURL has http:// scheme
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + address
	}
	return &workerClient{
		address:    address,
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

func (c *workerClient) CreateTask(ctx context.Context, task *worker.Task) error {
	url := fmt.Sprintf(pathCreateTask, c.baseURL)
	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to create task: %s", resp.Status)
	}

	return nil
}

func (c *workerClient) DoTask(ctx context.Context, taskID uuid.UUID) error {
	url := fmt.Sprintf(pathDoTask, c.baseURL, taskID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to start task: %s", resp.Status)
	}

	return nil
}

func (c *workerClient) DeleteTask(ctx context.Context, taskID uuid.UUID) error {
	url := fmt.Sprintf(pathDeleteTask, c.baseURL, taskID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete task: %s", resp.Status)
	}

	return nil
}
