package handler

import "github.com/google/uuid"

type CreateTaskRequest struct {
	Hash      string `json:"hash"`
	MaxLength int    `json:"max_length"`
}

type CreateTaskResponse struct {
	TaskID uuid.UUID `json:"task_id"`
}
