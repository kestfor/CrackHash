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

	m                 sync.RWMutex
	workerHTTPClients map[uuid.UUID]workerclient.WorkerClient // workerID -> http worker client to trigger internal api
	taskToWorkersMap  map[uuid.UUID]set.Set[uuid.UUID]        // taskID -> workerHTTPClients: when task created it assigned to list of workerHTTPClients, this struct stores this info

	addrToIDMap map[string]uuid.UUID // reverse mapping for fast lookup of workerID by address

	// progress stores the latest progress from each worker for each task
	// taskID -> workerID -> progress
	progress map[uuid.UUID]map[uuid.UUID]*worker.TaskProgress
}

func NewService(alphabet string, healthCheckerProvider HealthCheckerProvider) *managerService {
	return &managerService{
		alphabet:              alphabet,
		healthCheckerProvider: healthCheckerProvider,
		workerHTTPClients:     make(map[uuid.UUID]workerclient.WorkerClient),
		taskToWorkersMap:      make(map[uuid.UUID]set.Set[uuid.UUID]),
		addrToIDMap:           make(map[string]uuid.UUID),
		progress:              make(map[uuid.UUID]map[uuid.UUID]*worker.TaskProgress),
	}
}

func (s *managerService) AddWorker(ctx context.Context, workerAddress string) uuid.UUID {
	s.m.Lock()
	defer s.m.Unlock()

	if oldID, ok := s.addrToIDMap[workerAddress]; ok {
		slog.Info("worker with same address already exists, removing old worker resources",
			slog.String("worker_address", workerAddress),
		)
		s.clearWorkerResourcesLocked(oldID, workerAddress)
	}

	workerID := uuid.New()
	client := workerclient.NewWorkerClient(workerAddress)
	s.workerHTTPClients[workerID] = client
	s.addrToIDMap[workerAddress] = workerID

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

	s.clearWorkerResources(workerID, client.Address())
}

func (s *managerService) clearWorkerResources(workerID uuid.UUID, workerAddress string) {
	s.m.Lock()
	defer s.m.Unlock()
	s.clearWorkerResourcesLocked(workerID, workerAddress)
}

// clearWorkerResourcesLocked must be called with held lock
func (s *managerService) clearWorkerResourcesLocked(workerID uuid.UUID, workerAddress string) {
	delete(s.addrToIDMap, workerAddress)
	delete(s.workerHTTPClients, workerID)

	// mark this worker's taskToWorkersMap as failed if not already done
	for taskID, workers := range s.taskToWorkersMap {
		for wID := range workers {
			if wID == workerID {
				if _, ok := s.progress[taskID]; !ok {
					s.progress[taskID] = make(map[uuid.UUID]*worker.TaskProgress)
				}

				progress, ok := s.progress[taskID][workerID]
				if !ok {
					progress = &worker.TaskProgress{
						TaskID:   taskID,
						WorkerID: workerID,
						Status:   worker.StatusError,
					}
				}

				if progress.Status != worker.StatusReady {
					progress.Status = worker.StatusError
					s.progress[taskID][workerID] = progress
				}
			}
		}
	}
}

func (s *managerService) UpdateProgress(ctx context.Context, progress *worker.TaskProgress) error {
	s.m.Lock()
	defer s.m.Unlock()

	taskID := progress.TaskID
	workerID := progress.WorkerID

	if _, ok := s.workerHTTPClients[workerID]; !ok {
		slog.Warn("received push update from unknown worker", slog.Any("worker_id", workerID))
		return fmt.Errorf("worker not found: %s", workerID)
	}

	workerIDs, ok := s.taskToWorkersMap[taskID]
	if !ok {
		slog.Warn("received push update for unknown task", slog.Any("task_id", taskID))
		return fmt.Errorf("task not found: %s", taskID)
	}

	if !workerIDs.Contains(workerID) {
		slog.Warn("received push update for task not assigned to worker",
			slog.Any("task_id", taskID),
			slog.Any("worker_id", workerID),
		)
		return fmt.Errorf("worker %s is not assigned to task %s", workerID, taskID)
	}

	if _, ok := s.progress[taskID]; !ok {
		s.progress[taskID] = make(map[uuid.UUID]*worker.TaskProgress)
	}
	s.progress[taskID][workerID] = progress

	slog.Info("task progress received",
		slog.Any("progress", progress),
	)

	return nil
}

func (s *managerService) TaskProgress(ctx context.Context, taskID uuid.UUID) (*manager.TaskStatus, error) {
	s.m.RLock()
	defer s.m.RUnlock()

	workerIDs, ok := s.taskToWorkersMap[taskID]
	if !ok {
		return nil, manager.ErrTaskNotFound
	}

	progresses := make([]*worker.TaskProgress, 0, len(workerIDs))

	taskProgress, ok := s.progress[taskID]

	if ok {
		for _, progress := range taskProgress {
			progresses = append(progresses, progress)
		}
	}

	merged := mergeProgress(progresses...)
	return merged, nil
}

func (s *managerService) SubmitTask(ctx context.Context, targetHash string, maxLength int) (uuid.UUID, error) {
	s.m.Lock()
	defer s.m.Unlock()

	workersNum := len(s.workerHTTPClients)
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

	// now tries assign to all workerHTTPClients

	for wID, client := range s.workerHTTPClients {
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
			// Clean up created taskToWorkersMap on other workerHTTPClients
			for _, createdWID := range availableWorkers {
				if c, ok := s.workerHTTPClients[createdWID]; ok {
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

	// Start all taskToWorkersMap
	s.taskToWorkersMap[taskID] = set.New[uuid.UUID]()
	for _, wID := range availableWorkers {

		s.taskToWorkersMap[taskID].Add(wID)
		client := s.workerHTTPClients[wID]

		if err := client.DoTask(ctx, taskID); err != nil {
			slog.Error("failed to start task on worker",
				slog.Any("worker_id", wID),
				slog.Any("error", err),
			)
		}
	}

	slog.Info("task submitted successfully",
		slog.Any("task_id", taskID),
		slog.Int("workers_assigned", len(availableWorkers)),
	)

	return taskID, nil
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
