package manager

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/kestfor/CrackHash/internal/services/worker"
)

var ErrTaskNotFound = errors.New("task not found")
var ErrNoAvailableWorkers = errors.New("no available workers")
var ErrInvalidMaxLength = errors.New("max length must be greater than 0")

type Service interface {
	SubmitTask(ctx context.Context, targetHash string, maxLength int) (uuid.UUID, error)
	AddWorker(ctx context.Context, workerAddress string) uuid.UUID
	AddTaskResult(ctx context.Context, workerAddress string, result *worker.TaskProgress) error
	TaskProgress(ctx context.Context, taskID uuid.UUID) (*TaskStatus, error)
}
