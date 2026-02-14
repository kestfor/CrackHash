package handler

import "github.com/google/uuid"

type CreateTaskRequest struct {
	Hash      string `json:"hash"`
	MaxLength int    `json:"maxLength"`
}

type CreateTaskResponse struct {
	RequestID uuid.UUID `json:"requestId"`
}

type RegisterWorkerResponse struct {
	WorkerID uuid.UUID `json:"id"`
}
