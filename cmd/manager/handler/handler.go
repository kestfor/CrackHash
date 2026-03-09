package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/kestfor/CrackHash/internal/services/manager"
)

type managerHTTPHandler struct {
	managerService manager.Service
}

func NewHandler(managerService manager.Service) *managerHTTPHandler {
	return &managerHTTPHandler{
		managerService: managerService,
	}
}

func (h *managerHTTPHandler) HandleCreateTask(w http.ResponseWriter, r *http.Request) {
	task := &CreateTaskRequest{}

	err := json.NewDecoder(r.Body).Decode(task)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	taskID, err := h.managerService.SubmitTask(r.Context(), task.Hash, task.MaxLength)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(CreateTaskResponse{RequestID: taskID}); err != nil {
		slog.Warn("failed to encode response", slog.Any("error", err))
	}
}

func (h *managerHTTPHandler) HandleGetTaskProgress(w http.ResponseWriter, r *http.Request) {
	requestID := r.URL.Query().Get("requestId")

	if requestID == "" {
		http.Error(w, "requestId is required", http.StatusBadRequest)
		return
	}

	parsedID, err := uuid.Parse(requestID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	progress, err := h.managerService.TaskProgress(r.Context(), parsedID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(progress); err != nil {
		slog.Warn("failed to encode response", slog.Any("error", err))
	}
}
