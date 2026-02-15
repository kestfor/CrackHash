package workerservice

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	expirable "github.com/hashicorp/golang-lru/v2/expirable"
	entities "github.com/kestfor/CrackHash/internal/services/worker"
	worker "github.com/kestfor/CrackHash/internal/services/worker/worker"
)

type Fabric interface {
	NewWorker() worker.Worker
}

type workerService struct {
	maxParallel  int
	workerFabric Fabric

	mu           sync.Mutex
	tasksContext *expirable.LRU[uuid.UUID, taskExecutionContext]
}

type taskExecutionContext struct {
	executionContext context.Context
	task             *entities.Task
	worker           worker.Worker
}

func NewService(config *Config, workerFabric Fabric) *workerService {
	return &workerService{
		maxParallel:  config.MaxParallel,
		workerFabric: workerFabric,

		tasksContext: expirable.NewLRU[uuid.UUID, taskExecutionContext](config.MaxParallel, nil, time.Minute*5),
	}
}

func (s *workerService) CreateTask(ctx context.Context, task *entities.Task) error {
	if err := validateTask(task); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.tasksContext.Len() >= s.maxParallel {
		return fmt.Errorf("max parallel tasks reached")
	}

	if s.tasksContext.Contains(task.TaskID) {
		return entities.ErrTaskAlreadyExists
	}

	s.tasksContext.Add(task.TaskID, taskExecutionContext{
		task: task,
	})

	slog.Info("task created", slog.String("task_id", task.TaskID.String()))

	return nil
}

func (s *workerService) DoTask(ctx context.Context, taskID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	taskContext, ok := s.tasksContext.Get(taskID)
	if !ok {
		return entities.ErrTaskNotFound
	}

	wrk := taskContext.worker
	if wrk == nil {
		slog.Info("starting task execution", slog.Any("task_id", taskID))

		wrk = s.workerFabric.NewWorker()
		taskContext.worker = wrk
		taskContext.executionContext = context.Background()

		s.tasksContext.Add(taskID, taskContext)

		wrk.Do(taskContext.executionContext, taskContext.task)
	}

	return nil
}

func (s *workerService) DeleteTask(ctx context.Context, taskID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	taskContext, ok := s.tasksContext.Get(taskID)
	if ok {
		taskContext.worker.Cancel()
	}

	s.tasksContext.Remove(taskID)

	slog.Info("task deleted", slog.Any("task_id", taskID))

	return nil
}

func validateTask(task *entities.Task) error {
	if task.TargetHash == "" {
		return fmt.Errorf("%w: target hash is required", entities.ErrInvalidTask)
	}

	if task.Alphabet == "" {
		return fmt.Errorf("%w: alphabet is required", entities.ErrInvalidTask)
	}

	if task.MaxLength <= 0 {
		return fmt.Errorf("%w: max length must be greater than 0", entities.ErrInvalidTask)
	}

	if task.EndIndex <= task.StartIndex {
		return fmt.Errorf("%w: end index must be greater than start index", entities.ErrInvalidTask)
	}

	return nil
}
