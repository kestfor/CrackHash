package service

import (
	"sort"
	"testing"

	"github.com/google/uuid"
	"github.com/kestfor/CrackHash/internal/services/worker"
	"github.com/stretchr/testify/assert"
)

func TestMergeProgress(t *testing.T) {
	tests := []struct {
		name         string
		progresses   []*worker.TaskProgress
		wantStatus   worker.Status
		wantProgress int
		wantData     []string
	}{
		{
			name:         "empty progresses",
			progresses:   []*worker.TaskProgress{},
			wantStatus:   worker.StatusNotStarted,
			wantProgress: 0,
			wantData:     []string{},
		},
		{
			name: "single progress in progress",
			progresses: []*worker.TaskProgress{
				{
					TaskID:          uuid.New(),
					WorkerID:        uuid.New(),
					Status:          worker.StatusInProgress,
					IterationsDone:  50,
					TotalIterations: 100,
					Result:          []string{},
				},
			},
			wantStatus:   worker.StatusInProgress,
			wantProgress: 50,
			wantData:     []string{},
		},
		{
			name: "single progress ready",
			progresses: []*worker.TaskProgress{
				{
					TaskID:          uuid.New(),
					WorkerID:        uuid.New(),
					Status:          worker.StatusReady,
					IterationsDone:  100,
					TotalIterations: 100,
					Result:          []string{"password"},
				},
			},
			wantStatus:   worker.StatusReady,
			wantProgress: 100,
			wantData:     []string{"password"},
		},
		{
			name: "multiple progresses all ready",
			progresses: []*worker.TaskProgress{
				{
					TaskID:          uuid.New(),
					WorkerID:        uuid.New(),
					Status:          worker.StatusReady,
					IterationsDone:  100,
					TotalIterations: 100,
					Result:          []string{"pass1"},
				},
				{
					TaskID:          uuid.New(),
					WorkerID:        uuid.New(),
					Status:          worker.StatusReady,
					IterationsDone:  100,
					TotalIterations: 100,
					Result:          []string{"pass2"},
				},
			},
			wantStatus:   worker.StatusReady,
			wantProgress: 100,
			wantData:     []string{"pass1", "pass2"},
		},
		{
			name: "multiple progresses one in progress",
			progresses: []*worker.TaskProgress{
				{
					TaskID:          uuid.New(),
					WorkerID:        uuid.New(),
					Status:          worker.StatusReady,
					IterationsDone:  100,
					TotalIterations: 100,
					Result:          []string{"pass1"},
				},
				{
					TaskID:          uuid.New(),
					WorkerID:        uuid.New(),
					Status:          worker.StatusInProgress,
					IterationsDone:  50,
					TotalIterations: 100,
					Result:          []string{},
				},
			},
			wantStatus:   worker.StatusInProgress,
			wantProgress: 75,
			wantData:     []string{"pass1"},
		},
		{
			name: "multiple progresses one error",
			progresses: []*worker.TaskProgress{
				{
					TaskID:          uuid.New(),
					WorkerID:        uuid.New(),
					Status:          worker.StatusReady,
					IterationsDone:  100,
					TotalIterations: 100,
					Result:          []string{"pass1"},
				},
				{
					TaskID:          uuid.New(),
					WorkerID:        uuid.New(),
					Status:          worker.StatusError,
					IterationsDone:  30,
					TotalIterations: 100,
					Result:          []string{},
				},
			},
			wantStatus:   worker.StatusError,
			wantProgress: 65,
			wantData:     []string{"pass1"},
		},
		{
			name: "progresses with zero total iterations",
			progresses: []*worker.TaskProgress{
				{
					TaskID:          uuid.New(),
					WorkerID:        uuid.New(),
					Status:          worker.StatusNotStarted,
					IterationsDone:  0,
					TotalIterations: 0,
					Result:          []string{},
				},
			},
			wantStatus:   worker.StatusInProgress,
			wantProgress: 0,
			wantData:     []string{},
		},
		{
			name: "merge results removes duplicates",
			progresses: []*worker.TaskProgress{
				{
					TaskID:          uuid.New(),
					WorkerID:        uuid.New(),
					Status:          worker.StatusReady,
					IterationsDone:  100,
					TotalIterations: 100,
					Result:          []string{"pass1", "pass2"},
				},
				{
					TaskID:          uuid.New(),
					WorkerID:        uuid.New(),
					Status:          worker.StatusReady,
					IterationsDone:  100,
					TotalIterations: 100,
					Result:          []string{"pass2", "pass3"},
				},
			},
			wantStatus:   worker.StatusReady,
			wantProgress: 100,
			wantData:     []string{"pass1", "pass2", "pass3"},
		},
		{
			name: "three workers different states",
			progresses: []*worker.TaskProgress{
				{
					TaskID:          uuid.New(),
					WorkerID:        uuid.New(),
					Status:          worker.StatusReady,
					IterationsDone:  100,
					TotalIterations: 100,
					Result:          []string{},
				},
				{
					TaskID:          uuid.New(),
					WorkerID:        uuid.New(),
					Status:          worker.StatusInProgress,
					IterationsDone:  50,
					TotalIterations: 100,
					Result:          []string{},
				},
				{
					TaskID:          uuid.New(),
					WorkerID:        uuid.New(),
					Status:          worker.StatusNotStarted,
					IterationsDone:  0,
					TotalIterations: 100,
					Result:          []string{},
				},
			},
			wantStatus:   worker.StatusInProgress,
			wantProgress: 50,
			wantData:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeProgress(tt.progresses...)

			assert.Equal(t, tt.wantStatus, result.Status)
			assert.Equal(t, tt.wantProgress, result.Progress)

			// sort both slices for comparison since set order is not guaranteed
			gotData := make([]string, len(result.Data))
			copy(gotData, result.Data)
			sort.Strings(gotData)

			wantData := make([]string, len(tt.wantData))
			copy(wantData, tt.wantData)
			sort.Strings(wantData)

			assert.Equal(t, wantData, gotData)
		})
	}
}
