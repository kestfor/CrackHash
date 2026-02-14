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
	tasks    map[uuid.UUID]*worker.Task
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
func (s *workerService) Notify(result *worker.TaskProgress) error {
	taskID := result.TaskID
	return s.DeleteTask(context.Background(), taskID)
}

func (s *workerService) CreateTask(ctx context.Context, task *worker.Task) error {
	if len(s.workers) >= s.maxParallel {
		return fmt.Errorf("max parallel tasks reached")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[task.TaskID]; ok {
		return worker.ErrTaskAlreadyExists
	}

	s.tasks[task.TaskID] = task

	return nil
}

func (s *workerService) DoTask(ctx context.Context, taskID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[taskID]
	if !ok {
		return worker.ErrTaskNotFound
	}

	wrk, ok := s.workers[taskID]
	if !ok {
		wrk = impl.NewWorker([]notifier.Notifier{s, s.notifier})
		s.workers[taskID] = wrk

		go func() {
			wrk.Do(ctx, task)
		}()
	}

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

func (s *workerService) DeleteTask(ctx context.Context, taskID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.workers, taskID)
	delete(s.tasks, taskID)
	return nil
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
