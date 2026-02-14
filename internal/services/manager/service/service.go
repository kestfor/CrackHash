package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	"github.com/kestfor/CrackHash/internal/services/manager"
	"github.com/kestfor/CrackHash/internal/services/manager/healthchecker"
	"github.com/kestfor/CrackHash/internal/services/manager/workerclient"
	"github.com/kestfor/CrackHash/internal/services/worker"
	"github.com/kestfor/CrackHash/pkg/set"
)

type HealthCheckerProvider func(workerAddress string) healthchecker.HealthChecker

type managerService struct {
	alphabet string

	healthCheckerProvider HealthCheckerProvider

	m       sync.RWMutex
	workers map[uuid.UUID]workerclient.WorkerClient // workerID -> clients
	tasks   map[uuid.UUID][]uuid.UUID               // taskID -> workers

	addrToID map[string]uuid.UUID // address -> workerID

	ready map[uuid.UUID]map[uuid.UUID]*worker.TaskProgress // taskID -> workerID -> result
}

func NewService(alphabet string, healthCheckerProvider HealthCheckerProvider) *managerService {
	return &managerService{
		alphabet:              alphabet,
		healthCheckerProvider: healthCheckerProvider,
		workers:               make(map[uuid.UUID]workerclient.WorkerClient),
		tasks:                 make(map[uuid.UUID][]uuid.UUID),
		addrToID:              make(map[string]uuid.UUID),
		ready:                 make(map[uuid.UUID]map[uuid.UUID]*worker.TaskProgress),
	}
}

func (s *managerService) AddWorker(ctx context.Context, workerAddress string) uuid.UUID {
	s.m.Lock()
	defer s.m.Unlock()

	workerID := uuid.New()
	client := workerclient.NewWorkerClient(workerAddress)
	s.workers[workerID] = client
	s.addrToID[workerAddress] = workerID

	slog.Info("worker registered",
		slog.String("worker_id", workerID.String()),
		slog.String("worker_address", workerAddress),
	)

	go s.checkWorker(ctx, workerID, client)
	return workerID
}

// checkWorker blocks until worker failure, then removes it
func (s *managerService) checkWorker(ctx context.Context, workerID uuid.UUID, client workerclient.WorkerClient) {
	healthChecker := s.healthCheckerProvider(client.Address())
	healthChecker.NotifyFailure() // blocks until failure

	slog.Warn("worker health check failed, removing worker",
		slog.String("worker_id", workerID.String()),
	)

	s.m.Lock()
	defer s.m.Unlock()

	delete(s.workers, workerID)
	delete(s.addrToID, client.Address())

	// mark this worker's tasks as failed if not already done
	for taskID, workers := range s.tasks {
		for _, wID := range workers {
			if wID == workerID {
				if _, ok := s.ready[taskID]; !ok {
					s.ready[taskID] = make(map[uuid.UUID]*worker.TaskProgress)
				}

				if _, ok := s.ready[taskID][workerID]; !ok {
					s.ready[taskID][workerID] = &worker.TaskProgress{
						TaskID: taskID,
						Status: worker.StatusError,
					}
				}
			}
		}
	}
}

func (s *managerService) TaskProgress(ctx context.Context, taskID uuid.UUID) (*manager.TaskStatus, error) {
	ready, doneWorkers, err := s.collectReadyResults(taskID)
	if err != nil {
		return nil, err
	}

	progresses, err := s.collectInProgress(ctx, taskID, doneWorkers)
	if err != nil {
		// If we have ready results, don't fail just because we can't query active workers
		if err == manager.ErrTaskNotFound && len(ready) > 0 {
			progresses = []*worker.TaskProgress{}
		} else {
			return nil, err
		}
	}

	progresses = append(progresses, ready...)

	merged := mergeProgress(progresses...)
	return merged, nil
}

func (s *managerService) AddTaskResult(ctx context.Context, workerAddress string, result *worker.TaskProgress) error {
	s.m.Lock()
	defer s.m.Unlock()

	workerID, ok := s.addrToID[workerAddress]
	if !ok {
		return fmt.Errorf("worker not found for address: %s", workerAddress)
	}

	if result.Status != worker.StatusReady {
		return fmt.Errorf("task not done, status: %s", result.Status)
	}

	taskID := result.TaskID
	if _, ok := s.ready[taskID]; !ok {
		s.ready[taskID] = make(map[uuid.UUID]*worker.TaskProgress)
	}
	s.ready[taskID][workerID] = result

	slog.Info("task result received",
		slog.String("task_id", taskID.String()),
		slog.String("worker_id", workerID.String()),
		slog.Int("results_count", len(result.Result)),
	)

	return nil
}

func (s *managerService) SubmitTask(ctx context.Context, targetHash string, maxLength int) (uuid.UUID, error) {
	s.m.Lock()
	defer s.m.Unlock()

	workersNum := len(s.workers)
	if workersNum == 0 {
		return uuid.Nil, manager.ErrNoAvailableWorkers
	}

	if maxLength <= 0 {
		return uuid.Nil, manager.ErrInvalidMaxLength
	}

	slog.Info("submitting task",
		slog.String("target_hash", targetHash),
		slog.Int("max_length", maxLength),
		slog.Int("workers_count", workersNum),
	)

	taskID := uuid.New()
	parts := manager.SplitAlphabet(s.alphabet, workersNum)

	availableWorkers := make([]uuid.UUID, 0, workersNum)
	partNum := 0

	for wID, client := range s.workers {
		if partNum >= len(parts) {
			break
		}

		part := parts[partNum]
		if part == "" {
			partNum++
			continue
		}

		task := &worker.Task{
			TaskID:            taskID,
			TargetHash:        targetHash,
			IterationAlphabet: part,
			MaxLength:         maxLength,
		}

		err := client.CreateTask(ctx, task)
		if err != nil {
			slog.Error("failed to create task on worker",
				slog.String("worker_id", wID.String()),
				slog.Any("error", err),
			)
			// Clean up created tasks on other workers
			for _, createdWID := range availableWorkers {
				if c, ok := s.workers[createdWID]; ok {
					_ = c.DeleteTask(ctx, taskID)
				}
			}
			return uuid.Nil, fmt.Errorf("failed to create task on worker %s: %w", wID.String(), err)
		}

		availableWorkers = append(availableWorkers, wID)
		partNum++
	}

	if len(availableWorkers) == 0 {
		return uuid.Nil, manager.ErrNoAvailableWorkers
	}

	// Start all tasks
	for _, wID := range availableWorkers {
		s.tasks[taskID] = append(s.tasks[taskID], wID)
		client := s.workers[wID]
		if err := client.DoTask(ctx, taskID); err != nil {
			slog.Error("failed to start task on worker",
				slog.String("worker_id", wID.String()),
				slog.Any("error", err),
			)
		}
	}

	slog.Info("task submitted successfully",
		slog.String("task_id", taskID.String()),
		slog.Int("workers_assigned", len(availableWorkers)),
	)

	return taskID, nil
}

func (s *managerService) collectReadyResults(taskID uuid.UUID) ([]*worker.TaskProgress, set.Set[uuid.UUID], error) {
	s.m.RLock()
	defer s.m.RUnlock()

	doneWorkers := set.New[uuid.UUID]()
	progresses := make([]*worker.TaskProgress, 0)

	readyParts, ok := s.ready[taskID]
	if !ok {
		return progresses, doneWorkers, nil
	}

	for workerID, result := range readyParts {
		doneWorkers.Add(workerID)
		progresses = append(progresses, result)
	}

	return progresses, doneWorkers, nil
}

func (s *managerService) collectInProgress(ctx context.Context, taskID uuid.UUID, readyWorkers set.Set[uuid.UUID]) ([]*worker.TaskProgress, error) {
	s.m.RLock()
	workerIDs, ok := s.tasks[taskID]
	if !ok {
		s.m.RUnlock()
		return nil, manager.ErrTaskNotFound
	}

	type workerInfo struct {
		id     uuid.UUID
		client workerclient.WorkerClient
	}

	workersToQuery := make([]workerInfo, 0)
	for _, wrkID := range workerIDs {
		if readyWorkers != nil && readyWorkers.Contains(wrkID) {
			continue
		}
		client, ok := s.workers[wrkID]
		if !ok {
			continue
		}
		workersToQuery = append(workersToQuery, workerInfo{id: wrkID, client: client})
	}
	s.m.RUnlock()

	var progresses []*worker.TaskProgress
	for _, w := range workersToQuery {
		progress, err := w.client.TaskProgress(ctx, taskID)
		if err != nil {
			slog.Error("failed to get task progress from worker",
				slog.String("worker_id", w.id.String()),
				slog.Any("error", err),
			)
			continue
		}
		progresses = append(progresses, progress)
	}

	return progresses, nil
}

func mergeProgress(progresses ...*worker.TaskProgress) *manager.TaskStatus {
	mergedProgress := &manager.TaskStatus{
		Progress: 0,
		Status:   worker.StatusNotStarted,
		Data:     []string{},
	}

	if len(progresses) == 0 {
		return mergedProgress
	}

	totalIterations := 0
	iterationsDone := 0
	failed := false
	allDone := true

	for _, progress := range progresses {
		totalIterations += progress.TotalIterations
		iterationsDone += progress.IterationsDone

		// Collect results
		if progress.Result != nil {
			mergedProgress.Data = append(mergedProgress.Data, progress.Result...)
		}

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

	return mergedProgress
}
