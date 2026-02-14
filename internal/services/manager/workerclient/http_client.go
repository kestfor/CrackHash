package workerclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/kestfor/CrackHash/internal/services/worker"
)

const (
	pathCreateTask   = "%s/tasks"
	pathDeleteTask   = "%s/tasks/%s"
	pathDoTask       = "%s/tasks/%s/do"
	pathTaskProgress = "%s/tasks/%s/progress"
)

type workerClient struct {
	baseURL    string
	httpClient *http.Client
}

func (c *workerClient) Address() string {
	return c.baseURL
}

func NewWorkerClient(baseURL string) *workerClient {
	return &workerClient{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

func (c *workerClient) CreateTask(ctx context.Context, task *worker.Task) error {

	url := fmt.Sprintf(pathCreateTask, c.baseURL)
	data, err := json.Marshal(task)
	if err != nil {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(data))
	if err != nil {
		return nil
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to create task: %s", resp.Status)
	}

	return nil
}

func (c *workerClient) TaskProgress(ctx context.Context, taskID uuid.UUID) (*worker.TaskProgress, error) {

	url := fmt.Sprintf(pathTaskProgress, c.baseURL, taskID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to create task: %s", resp.Status)
	}

	var taskProgress worker.TaskProgress
	if err := json.NewDecoder(resp.Body).Decode(&taskProgress); err != nil {
		return nil, fmt.Errorf("failed to decode task progress: %s", err)
	}

	return &taskProgress, nil
}

func (c *workerClient) DoTask(ctx context.Context, taskID uuid.UUID) error {
	url := fmt.Sprintf(pathDoTask, c.baseURL, taskID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, nil)
	if err != nil {
		return nil
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil
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
		return nil
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete task: %s", resp.Status)
	}

	return nil
}
