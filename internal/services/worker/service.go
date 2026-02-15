package worker

import (
	"context"

	"github.com/google/uuid"
)

type Service interface {
	CreateTask(ctx context.Context, task *Task) error
	DeleteTask(ctx context.Context, taskID uuid.UUID) error
	DoTask(ctx context.Context, taskID uuid.UUID) error
}
