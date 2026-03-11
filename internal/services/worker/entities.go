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
	TaskID     uuid.UUID `json:"task_id" bson:"task_id"`
	TargetHash string    `json:"target_hash" bson:"target_hash"`
	Alphabet   string    `json:"alphabet" bson:"alphabet"`
	MaxLength  int       `json:"max_length" bson:"max_length"`
	StartIndex uint64    `json:"start_index" bson:"start_index"`
	EndIndex   uint64    `json:"end_index" bson:"end_index"` // exclusive
}

type TaskProgress struct {
	TaskID          uuid.UUID `json:"task_id" bson:"task_id"`
	WorkerID        uuid.UUID `json:"worker_id" bson:"worker_id"`
	StartIndex      uint64    `json:"start_index" bson:"start_index"`
	Status          Status    `json:"status" bson:"status"`
	IterationsDone  int       `json:"iterations_done" bson:"iterations_done"`
	TotalIterations int       `json:"total_iterations" bson:"total_iterations"`
	Result          []string  `json:"result" bson:"result"`
}

func (t *Task) String() string {
	return fmt.Sprintf("Task<ID: %s, Hash: %s, Alphabet: %s, MaxLength: %d, Range: [%d, %d)>",
		t.TaskID, t.TargetHash, t.Alphabet, t.MaxLength, t.StartIndex, t.EndIndex)
}

func (t *TaskProgress) String() string {
	return fmt.Sprintf("TaskProgress<ID: %s, WorkerID: %s, Status: %s, IterDone: %d, IterTotal: %d, Found: %v>", t.TaskID, t.WorkerID, t.Status, t.IterationsDone, t.TotalIterations, t.Result)
}
