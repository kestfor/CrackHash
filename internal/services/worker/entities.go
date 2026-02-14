package worker

import "github.com/google/uuid"

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
	Status          Status    `json:"status"`
	IterationsDone  int       `json:"iterations_done"`
	TotalIterations int       `json:"total_iterations"`
	Result          []string  `json:"result"`
}

type TaskResult struct {
	TaskID uuid.UUID `json:"task_id"`
	Words  []string  `json:"words"`
}
