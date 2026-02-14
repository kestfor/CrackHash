package workerservice

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kestfor/CrackHash/internal/services/worker"
	"github.com/kestfor/CrackHash/internal/services/worker/notifier"
	worker2 "github.com/kestfor/CrackHash/internal/services/worker/worker"
	"github.com/kestfor/CrackHash/internal/services/worker/worker/impl"
)

type workerService struct {
	workerID     uuid.UUID
	maxParallel  int
	notifyPeriod time.Duration

	notifier notifier.Notifier
	workers  map[uuid.UUID]worker2.Worker

	// TODO lru clear
	tasks map[uuid.UUID]*worker.Task
	mu    sync.Mutex
}

func NewService(workerID uuid.UUID, config *worker.Config, notifier notifier.Notifier) *workerService {
	return &workerService{
		workerID:     workerID,
		maxParallel:  config.MaxParallel,
		notifyPeriod: config.NotifyPeriod,
		notifier:     notifier,
		workers:      make(map[uuid.UUID]worker2.Worker),
		tasks:        make(map[uuid.UUID]*worker.Task),
	}
}

func (s *workerService) CreateTask(ctx context.Context, task *worker.Task) error {
	if err := validateTask(task); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.workers) >= s.maxParallel {
		return fmt.Errorf("max parallel tasks reached")
	}

	if _, ok := s.tasks[task.TaskID]; ok {
		return worker.ErrTaskAlreadyExists
	}

	slog.Info("task created", slog.String("task_id", task.TaskID.String()))
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
		wrk = impl.NewWorker(s.workerID, []notifier.Notifier{s.notifier}, s.notifyPeriod)
		s.workers[taskID] = wrk

		slog.Info("starting task execution", slog.String("task_id", taskID.String()))
		// Use background context so task isn't canceled when HTTP request ends
		wrk.Do(context.Background(), task)
	}

	return nil
}

func (s *workerService) DeleteTask(ctx context.Context, taskID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	slog.Info("deleting task", slog.String("task_id", taskID.String()))
	wrk, ok := s.workers[taskID]
	if ok {
		wrk.Cancel()
	}

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
