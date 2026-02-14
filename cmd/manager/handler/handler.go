package handler

import (
	"encoding/json"
	"fmt"
	"net"
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
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(CreateTaskResponse{RequestID: taskID})
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
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(progress)
}

func (h *managerHTTPHandler) HandleAddTaskResult(w http.ResponseWriter, r *http.Request) {
	taskResult := &worker.TaskProgress{}

	err := json.NewDecoder(r.Body).Decode(taskResult)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid remote address: %v", err), http.StatusBadRequest)
		return
	}

	workerPort := r.Header.Get("X-Worker-Port")
	if workerPort == "" {
		http.Error(w, "X-Worker-Port header is required", http.StatusBadRequest)
		return
	}

	workerAddr := net.JoinHostPort(remoteIP, workerPort)

	if err := h.managerService.AddTaskResult(r.Context(), workerAddr, taskResult); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *managerHTTPHandler) HandleRegisterWorker(w http.ResponseWriter, r *http.Request) {
	remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid remote address: %v", err), http.StatusBadRequest)
		return
	}

	workerPort := r.Header.Get("X-Worker-Port")
	if workerPort == "" {
		http.Error(w, "X-Worker-Port header is required", http.StatusBadRequest)
		return
	}

	workerAddr := net.JoinHostPort(remoteIP, workerPort)

	workerID := h.managerService.AddWorker(r.Context(), workerAddr)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": workerID})
}
