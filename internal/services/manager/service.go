package manager

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var ErrInvalidMaxLength = errors.New("max length must be greater than 0")

type Service interface {
	SubmitTask(ctx context.Context, targetHash string, maxLength int) (uuid.UUID, error)
	TaskProgress(ctx context.Context, taskID uuid.UUID) (TaskStatus, error)
	Run(ctx context.Context) error
}
