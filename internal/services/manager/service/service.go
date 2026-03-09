package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/kestfor/CrackHash/internal/services/broker"
	"github.com/kestfor/CrackHash/internal/services/broker/rabbitmq"
	"github.com/kestfor/CrackHash/internal/services/manager"
	"github.com/kestfor/CrackHash/internal/services/manager/storage"
	"github.com/kestfor/CrackHash/internal/services/worker"
	utils "github.com/kestfor/CrackHash/pkg"
	"github.com/kestfor/CrackHash/pkg/search_space"
	"github.com/kestfor/CrackHash/pkg/set"
)

type managerService struct {
	alphabet string

	progressStorage     storage.ProgressStorage
	subTaskStorage      storage.SubTaskStorage
	tasksPublisher      broker.Publisher
	progressConsumer    broker.Consumer
	deadLettersConsumer broker.Consumer
	retrySendPeriod     time.Duration
}

func NewService(
	alphabet string,
	progressStorage storage.ProgressStorage,
	subTaskStorage storage.SubTaskStorage,
	tasksPublisher broker.Publisher,
	progressConsumer broker.Consumer,
	deadLettersConsumer broker.Consumer,
	retrySendPeriod time.Duration,
) *managerService {
	return &managerService{
		alphabet:            alphabet,
		progressStorage:     progressStorage,
		subTaskStorage:      subTaskStorage,
		tasksPublisher:      tasksPublisher,
		progressConsumer:    progressConsumer,
		deadLettersConsumer: deadLettersConsumer,
		retrySendPeriod:     retrySendPeriod,
	}
}

func (s *managerService) Run(ctx context.Context) error {
	slog.Info("starting manager service")

	go func() {
		slog.Info("starting progress consumer")
		err := s.progressConsumer.Consume(ctx, s)
		if err != nil {
			slog.Error("consume msg", slog.Any("error", err))
		}
	}()

	go func() {
		slog.Info("starting dead letters consumer")
		err := s.deadLettersConsumer.Consume(ctx, s)
		if err != nil {
			slog.Error("consume msg", slog.Any("error", err))
		}
	}()

	go s.runSubTaskSender(ctx)

	return nil
}

func (s *managerService) runSubTaskSender(ctx context.Context) {
	slog.Info("starting subtask sender", slog.Duration("period", s.retrySendPeriod))

	ticker := time.NewTicker(s.retrySendPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("subtask sender stopped")
			return
		case <-ticker.C:
			s.sendPendingSubTasks(ctx)
		}
	}
}

func (s *managerService) sendPendingSubTasks(ctx context.Context) {
	pending, err := s.subTaskStorage.FindPending(ctx)
	if err != nil {
		slog.Error("find pending subtasks", slog.Any("error", err))
		return
	}

	if len(pending) == 0 {
		return
	}

	slog.Info("found pending subtasks", slog.Int("count", len(pending)))

	for _, sub := range pending {
		data, err := json.Marshal(sub.Task)
		if err != nil {
			slog.Error("marshal subtask", slog.Any("error", err))
			continue
		}

		if err := s.tasksPublisher.Publish(ctx, rabbitmq.TasksQueue, data); err != nil {
			slog.Error("publish subtask", slog.Any("subtask", sub.Task), slog.Any("error", err))
			continue
		}

		if err := s.subTaskStorage.MarkSent(ctx, sub.TaskID, sub.StartIndex, sub.EndIndex); err != nil {
			slog.Error("mark subtask sent", slog.Any("subtask", sub.Task), slog.Any("error", err))
			continue
		}

		slog.Info("subtask sent", slog.Any("task_id", sub.TaskID), slog.Uint64("start", sub.StartIndex), slog.Uint64("end", sub.EndIndex))
	}
}

// Handle handles progress messages from workers
func (s *managerService) Handle(msg broker.Message) error {

	var err error
	ctx := context.Background()

	switch msg.RoutingKey {
	case rabbitmq.TasksProgressQueue:
		err = s.handleProgress(ctx, msg)
	case rabbitmq.DeadLetterQueue:
		err = s.handleDeadLetter(ctx, msg)
	default:
		err = fmt.Errorf("unknown routing key: %s", msg.RoutingKey)
	}

	if err != nil {
		slog.Error("handle message", slog.Any("error", err))
		return fmt.Errorf("handle message: %w", err)
	}

	return msg.Ack()
}

func (s *managerService) handleProgress(ctx context.Context, msg broker.Message) error {
	var progress worker.TaskProgress

	if err := json.Unmarshal(msg.Body, &progress); err != nil {
		slog.Error("unmarshal progress", slog.Any("error", err))
		return fmt.Errorf("unmarshal progress: %w", err)
	}

	slog.Info("got progress message", slog.String("task", progress.String()))

	if err := s.progressStorage.Upsert(ctx, progress); err != nil {
		slog.Error("upsert progress", slog.Any("error", err))
		return fmt.Errorf("upsert progress: %w", err)
	}

	return nil
}

func (s *managerService) handleDeadLetter(ctx context.Context, msg broker.Message) error {
	deadSubTask := worker.Task{}
	if err := json.Unmarshal(msg.Body, &deadSubTask); err != nil {
		slog.Error("unmarshal dead letter", slog.Any("error", err))
		return fmt.Errorf("unmarshal dead letter: %w", err)
	}

	slog.Info("got dead letter", slog.String("task", deadSubTask.String()))

	failedProgress := worker.TaskProgress{
		TaskID: deadSubTask.TaskID,
		Status: worker.StatusError,
	}

	if err := s.progressStorage.Upsert(ctx, failedProgress); err != nil {
		slog.Error("upsert progress", slog.Any("error", err))
		return fmt.Errorf("upsert progress: %w", err)
	}

	return nil

}

func (s *managerService) TaskProgress(ctx context.Context, taskID uuid.UUID) (manager.TaskStatus, error) {
	progresses, err := s.progressStorage.Collect(ctx, taskID)

	if err != nil {
		slog.Error("collect progress", slog.Any("error", err))
		return manager.TaskStatus{}, fmt.Errorf("collect progress: %w", err)
	}

	merged := mergeProgress(progresses...)
	slog.Info("task progress", slog.Any("task_id", taskID), slog.Any("progress", merged))

	return merged, nil
}

func (s *managerService) SubmitTask(ctx context.Context, targetHash string, maxLength int) (uuid.UUID, error) {
	if maxLength <= 0 {
		return uuid.Nil, manager.ErrInvalidMaxLength
	}

	totalSize := search_space.NewSearchSpace(s.alphabet, maxLength).TotalSize()
	ranges, err := utils.SplitRange(totalSize, maxLength) // total size > max length => no repeated ranges

	if err != nil {
		return uuid.Nil, fmt.Errorf("split search space: %w", err)
	}

	slog.Info("submitting task",
		slog.String("target_hash", targetHash),
		slog.Int("max_length", maxLength),
		slog.Int("subtasks_count", maxLength),
		slog.Uint64("total_search_space", totalSize),
	)

	taskID := uuid.New()

	tasks := make([]worker.Task, 0, len(ranges))
	for _, r := range ranges {
		tasks = append(tasks, worker.Task{
			TaskID:     taskID,
			TargetHash: targetHash,
			Alphabet:   s.alphabet,
			MaxLength:  maxLength,
			StartIndex: r.Start,
			EndIndex:   r.End,
		})
	}

	if err := s.subTaskStorage.CreateBatch(ctx, tasks); err != nil {
		return uuid.Nil, fmt.Errorf("save subtasks: %w", err)
	}

	slog.Info("task submitted successfully",
		slog.Any("task_id", taskID),
	)

	return taskID, nil
}

func mergeProgress(progresses ...worker.TaskProgress) manager.TaskStatus {
	mergedProgress := manager.TaskStatus{
		Progress: 0,
		Status:   worker.StatusNotStarted,
		Data:     []string{},
	}

	if len(progresses) == 0 {
		return mergedProgress
	}

	data := set.New[string]()
	totalIterations := 0
	iterationsDone := 0
	failed := false
	allDone := true

	for _, progress := range progresses {
		totalIterations += progress.TotalIterations
		iterationsDone += progress.IterationsDone

		data.Add(progress.Result...)

		if progress.Status != worker.StatusReady {
			allDone = false
		}

		if progress.Status == worker.StatusError {
			failed = true
		}
	}

	if failed {
		mergedProgress.Status = worker.StatusError
	} else if allDone {
		mergedProgress.Status = worker.StatusReady
	} else {
		mergedProgress.Status = worker.StatusInProgress
	}

	if totalIterations > 0 {
		mergedProgress.Progress = int(float64(iterationsDone) / float64(totalIterations) * 100)
	} else {
		mergedProgress.Progress = 0
	}

	mergedProgress.Data = data.Slice()
	return mergedProgress
}
