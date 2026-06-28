package workerservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/kestfor/CrackHash/internal/services/broker"
	entities "github.com/kestfor/CrackHash/internal/services/worker"
	worker "github.com/kestfor/CrackHash/internal/services/worker/worker"
)

type Fabric interface {
	NewWorker() worker.Worker
}

type workerService struct {
	workerFabric  Fabric
	tasksConsumer broker.Consumer
}

func NewService(workerFabric Fabric, tasksConsumer broker.Consumer) *workerService {
	return &workerService{
		workerFabric:  workerFabric,
		tasksConsumer: tasksConsumer,
	}
}

func (s *workerService) Run(ctx context.Context) error {
	if err := s.tasksConsumer.Consume(ctx, s); err != nil {
		return err
	}
	return nil
}

func (s *workerService) Handle(msg broker.Message) error {
	task := entities.Task{}
	if err := json.Unmarshal(msg.Body, &task); err != nil {
		return err
	}

	if err := validateTask(&task); err != nil {
		return err
	}

	slog.Info("received task", slog.String("task", task.String()))

	go func() {
		wrk := s.workerFabric.NewWorker()
		wrk.Do(context.Background(), &task)

		slog.Info("task execution completed", slog.String("task", task.String()))
		if err := msg.Ack(); err != nil {
			slog.Error("failed to ack message", slog.String("task", task.String()), slog.Any("error", err))
		}
	}()

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
