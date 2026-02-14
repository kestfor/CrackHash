package workerclient

import (
	"context"

	"github.com/google/uuid"
	"github.com/kestfor/CrackHash/internal/services/worker"
)

type WorkerClient interface {
	CreateTask(ctx context.Context, task *worker.Task) error
	DeleteTask(ctx context.Context, taskID uuid.UUID) error
	DoTask(ctx context.Context, taskID uuid.UUID) error
	TaskProgress(ctx context.Context, taskID uuid.UUID) (*worker.TaskProgress, error)

	Address() string
}
