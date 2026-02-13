package worker

import "github.com/google/uuid"

const (
	StatusNotStarted Status = "not_started"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
	StatusTimeout    Status = "timeout"
)

type Status string

type Task struct {
	TaskID            uuid.UUID `json:"task_id"`
	TargetHash        string    `json:"target_hash"`
	IterationAlphabet string    `json:"iteration_alphabet"`
	MaxLength         int       `json:"max_length"`
}

type TaskProgress struct {
	TaskID         uuid.UUID `json:"task_id"`
	IterationsDone int       `json:"iterations_done"`
}

type TaskResult struct {
	TaskID uuid.UUID `json:"task_id"`
	Words  []string  `json:"words"`
}
