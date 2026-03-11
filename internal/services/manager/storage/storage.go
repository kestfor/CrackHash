package storage

import (
	"context"

	"github.com/google/uuid"
	"github.com/kestfor/CrackHash/internal/services/manager"
	"github.com/kestfor/CrackHash/internal/services/worker"
)

type ProgressStorage interface {
	Upsert(ctx context.Context, progress worker.TaskProgress) error
	Collect(ctx context.Context, taskID uuid.UUID) ([]worker.TaskProgress, error)
}

type SubTaskStorage interface {
	Has(ctx context.Context, taskID uuid.UUID) (bool, error)
	CreateBatch(ctx context.Context, tasks []worker.Task) error
	FindPending(ctx context.Context) ([]manager.SubTask, error)
	MarkSent(ctx context.Context, taskID uuid.UUID, startIndex, endIndex uint64) error
}
