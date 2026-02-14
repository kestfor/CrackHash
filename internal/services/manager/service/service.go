package service

import (
	"context"
	"fmt"
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

	m       sync.Mutex
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
	}
}

func (s *managerService) AddWorker(ctx context.Context, workerAddress string) uuid.UUID {
	s.m.Lock()
	defer s.m.Unlock()

	workerID := uuid.New()
	s.workers[workerID] = workerclient.NewWorkerClient(workerAddress)
	s.addrToID[workerAddress] = workerID
	go s.checkWorker(ctx, workerID)
	return workerID
}

// blocks until failure
func (s *managerService) checkWorker(ctx context.Context, workerID uuid.UUID) {
	client := s.workers[workerID]
	healthChecker := s.healthCheckerProvider(client.Address())
	healthChecker.NotifyFailure()
	s.m.Lock()
	defer s.m.Unlock()
	delete(s.workers, workerID)

	// mark this worker tasks as failed if it was not done
	for taskID, workers := range s.tasks {
		for _, wID := range workers {
			if wID == workerID {
				_, ok := s.ready[taskID]
				if !ok {
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
	ready, doneWorkers, err := s.collectReadyResults(ctx, taskID)
	if err != nil {
		return nil, err
	}

	progresses, err := s.collectInProgress(ctx, taskID, doneWorkers)
	if err != nil {
		return nil, err
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
		return fmt.Errorf("worker not found")
	}

	if result.Status != worker.StatusReady {
		return fmt.Errorf("task not done")
	}

	taskID := result.TaskID
	_, ok = s.ready[taskID]
	if !ok {
		s.ready[taskID] = make(map[uuid.UUID]*worker.TaskProgress)
	}
	s.ready[taskID][workerID] = result
	return nil
}

func (s *managerService) SubmitTask(ctx context.Context, targetHash string, maxLength int) (uuid.UUID, error) {
	s.m.Lock()
	defer s.m.Unlock()

	taskID := uuid.New()
	workersNum := len(s.workers)
	busyWorkers := set.New[uuid.UUID]()

	for len(busyWorkers) < workersNum {

		availableWorkers := set.New[uuid.UUID]()
		parts := manager.SplitAlphabet(s.alphabet, workersNum)
		partNum := 0
		failed := false

		for wID, client := range s.workers {
			part := parts[partNum]

			task := &worker.Task{
				TaskID:            taskID,
				TargetHash:        targetHash,
				IterationAlphabet: part,
				MaxLength:         maxLength,
			}

			err := client.CreateTask(ctx, task)
			if err != nil {
				failed = true

				busyWorkers.Add(wID)

				// clear created tasks
				for wID := range availableWorkers {
					client := s.workers[wID]
					_ = client.DeleteTask(ctx, taskID)
				}

				break
			}

			availableWorkers.Add(wID)
			partNum++
		}

		if !failed {
			for wID := range availableWorkers {
				s.tasks[taskID] = append(s.tasks[taskID], wID)
				client := s.workers[wID]
				_ = client.DoTask(ctx, taskID)
			}

			return taskID, nil
		}

	}

	return uuid.Nil, manager.ErrNoAvailableWorkers
}

func (s *managerService) collectReadyResults(ctx context.Context, taskID uuid.UUID) ([]*worker.TaskProgress, set.Set[uuid.UUID], error) {
	s.m.Lock()
	defer s.m.Unlock()

	doneWorkers := set.New[uuid.UUID]()
	readyParts, ok := s.ready[taskID]
	progresses := make([]*worker.TaskProgress, 0)

	if !ok {
		return nil, nil, nil
	}

	for workerID, result := range readyParts {
		doneWorkers.Add(workerID)
		progresses = append(progresses, result)
	}

	return progresses, doneWorkers, nil
}

func (s *managerService) collectInProgress(ctx context.Context, taskID uuid.UUID, readyWorkers set.Set[uuid.UUID]) ([]*worker.TaskProgress, error) {
	s.m.Lock()
	defer s.m.Unlock()

	workerIDs, ok := s.tasks[taskID]
	if !ok {
		return nil, manager.ErrTaskNotFound
	}
	var progresses []*worker.TaskProgress

	for _, wrkID := range workerIDs {
		if readyWorkers.Contains(wrkID) {
			continue
		}
		client, ok := s.workers[wrkID]
		if !ok {
			return nil, fmt.Errorf("client for worker with id %s not found", wrkID)
		}
		progress, err := client.TaskProgress(ctx, taskID)
		if err != nil {
			return nil, err
		}
		progresses = append(progresses, progress)
	}

	return progresses, nil
}

func mergeProgress(progresses ...*worker.TaskProgress) *manager.TaskStatus {
	mergedProgress := &manager.TaskStatus{
		Progress: 0,
		Status:   worker.StatusNotStarted,
		Data:     nil,
	}

	totalIterations := 0
	failed := false
	allDone := true

	for _, progress := range progresses {
		mergedProgress.Status = progress.Status
		totalIterations += progress.TotalIterations
		mergedProgress.Progress += progress.IterationsDone

		if progress.Status == worker.StatusInProgress {
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

	mergedProgress.Progress = int(float64(mergedProgress.Progress) / float64(totalIterations) * 100)
	return mergedProgress

}
