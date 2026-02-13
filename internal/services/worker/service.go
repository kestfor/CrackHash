package worker

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var ErrInvalidTask = errors.New("invalid task")
var ErrTaskNotFound = errors.New("task not found")
var ErrTaskAlreadyExists = errors.New("task already exists")

type Service interface {
	SubmitTask(ctx context.Context, task *Task) error
	TaskProgress(ctx context.Context, taskID uuid.UUID) (*TaskProgress, error)
}
