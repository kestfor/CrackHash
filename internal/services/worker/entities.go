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
	TaskID            uuid.UUID `json:"task_id"`
	TargetHash        string    `json:"target_hash"`
	IterationAlphabet string    `json:"iteration_alphabet"`
	MaxLength         int       `json:"max_length"`
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
	return fmt.Sprintf("Task<ID: %s, Hash: %s, Alphabet: %s, MaxLength: %d>", t.TaskID, t.TargetHash, t.IterationAlphabet, t.MaxLength)
}

func (t *TaskProgress) String() string {
	return fmt.Sprintf("TaskProgress<ID: %s, WorkerID: %s, Status: %s, IterDone: %d, IterTotal: %d, Found: %v>", t.TaskID, t.WorkerID, t.Status, t.IterationsDone, t.TotalIterations, t.Result)
}
