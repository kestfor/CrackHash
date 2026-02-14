package worker

import (
	"fmt"

	"github.com/google/uuid"
)

const (
	StatusNotStarted Status = "NOT_STARTED"
	StatusInProgress Status = "IN_PROGRESS"
	StatusReady      Status = "READY"
	StatusError      Status = "ERROR"
)

type Status string

type Task struct {
	TaskID     uuid.UUID `json:"task_id"`
	TargetHash string    `json:"target_hash"`
	Alphabet   string    `json:"alphabet"`
	MaxLength  int       `json:"max_length"`
	StartIndex uint64    `json:"start_index"`
	EndIndex   uint64    `json:"end_index"` // exclusive
}

type TaskProgress struct {
	TaskID          uuid.UUID `json:"task_id"`
	WorkerID        uuid.UUID `json:"worker_id"`
	Status          Status    `json:"status"`
	IterationsDone  int       `json:"iterations_done"`
	TotalIterations int       `json:"total_iterations"`
	Result          []string  `json:"result"`
}

func (t *Task) String() string {
	return fmt.Sprintf("Task<ID: %s, Hash: %s, Alphabet: %s, MaxLength: %d, Range: [%d, %d)>",
		t.TaskID, t.TargetHash, t.Alphabet, t.MaxLength, t.StartIndex, t.EndIndex)
}

func (t *TaskProgress) String() string {
	return fmt.Sprintf("TaskProgress<ID: %s, WorkerID: %s, Status: %s, IterDone: %d, IterTotal: %d, Found: %v>", t.TaskID, t.WorkerID, t.Status, t.IterationsDone, t.TotalIterations, t.Result)
}
