package worker

import (
	"context"

	entities "github.com/kestfor/CrackHash/internal/services/worker"
)

type Worker interface {
	Do(ctx context.Context, task *entities.Task)
	Progress() *entities.TaskProgress
}
