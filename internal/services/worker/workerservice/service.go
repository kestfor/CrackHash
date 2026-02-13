package workerservice

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/kestfor/CrackHash/internal/services/worker"
	"github.com/kestfor/CrackHash/internal/services/worker/notifier"
	worker2 "github.com/kestfor/CrackHash/internal/services/worker/worker"
	"github.com/kestfor/CrackHash/internal/services/worker/worker/impl"
)

type workerService struct {
	maxParallel int

	notifier notifier.Notifier
	workers  map[uuid.UUID]worker2.Worker
	mu       sync.Mutex
}

func NewService(config *worker.Config, notifier notifier.Notifier) *workerService {
	return &workerService{
		maxParallel: config.MaxParallel,
		notifier:    notifier,
		workers:     make(map[uuid.UUID]worker2.Worker),
	}
}

// Notify implements Notifier
// Used to know when task is done to remove it from the map
func (s *workerService) Notify(result *worker.TaskResult) error {
	taskID := result.TaskID

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.workers, taskID)
	return nil
}

func (s *workerService) SubmitTask(ctx context.Context, task *worker.Task) error {
	if len(s.workers) >= s.maxParallel {
		return fmt.Errorf("max parallel tasks reached")
	}

	wrk := impl.NewWorker([]notifier.Notifier{s, s.notifier})

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.workers[task.TaskID]; ok {
		return worker.ErrTaskAlreadyExists
	}

	s.workers[task.TaskID] = wrk
	go func() {
		wrk.Do(ctx, task)
	}()

	return nil
}

func (s *workerService) TaskProgress(ctx context.Context, taskID uuid.UUID) (*worker.TaskProgress, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	wrk, ok := s.workers[taskID]
	if !ok {
		return nil, worker.ErrTaskNotFound
	}

	progress := wrk.Progress()
	if progress == nil {
		// сомнительно, но безопасно
		return nil, fmt.Errorf("task not started")
	}

	return progress, nil
}

func validateTask(task *worker.Task) error {
	if task.TargetHash == "" {
		return fmt.Errorf("%w: target hash is required", worker.ErrInvalidTask)
	}

	if task.IterationAlphabet == "" {
		return fmt.Errorf("%w: iteration alphabet is required", worker.ErrInvalidTask)
	}

	if task.MaxLength <= 0 {
		return fmt.Errorf("%w: max length must be greater than 0", worker.ErrInvalidTask)
	}

	return nil
}
