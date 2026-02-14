package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/kestfor/CrackHash/internal/services/manager"
	"github.com/kestfor/CrackHash/internal/services/worker"
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
	if err := json.NewEncoder(w).Encode(CreateTaskResponse{TaskID: taskID}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *managerHTTPHandler) HandleGetTaskProgress(w http.ResponseWriter, r *http.Request) {
	taskID := r.URL.Query().Get("task_id")

	if taskID == "" {
		http.Error(w, "task_id is required", http.StatusBadRequest)
		return
	}

	parsedID, err := uuid.Parse(taskID)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *managerHTTPHandler) HandleAddTaskResult(w http.ResponseWriter, r *http.Request) {
	taskResult := &worker.TaskProgress{}

	err := json.NewDecoder(r.Body).Decode(taskResult)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	workerAddr := r.Host

	if err := h.managerService.AddTaskResult(r.Context(), workerAddr, taskResult); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
