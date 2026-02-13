package handler

import "github.com/google/uuid"

type TaskProgressRequest struct {
	TaskID uuid.UUID `json:"task_id"`
}
