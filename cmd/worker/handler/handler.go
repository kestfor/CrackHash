package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/kestfor/CrackHash/internal/services/worker"
)

type handler struct {
	service worker.Service
}

func NewHandler(service worker.Service) *handler {
	return &handler{
		service: service,
	}
}

func (h *handler) HandleCreateTask(w http.ResponseWriter, r *http.Request) {
	task := &worker.Task{}

	err := json.NewDecoder(r.Body).Decode(task)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.service.CreateTask(r.Context(), task); err != nil {
		if errors.Is(err, worker.ErrInvalidTask) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if errors.Is(err, worker.ErrTaskAlreadyExists) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return

	}
	w.WriteHeader(http.StatusOK)
}

func (h *handler) HandleDeleteTask(w http.ResponseWriter, r *http.Request) {
	taskID := getTaskID(w, r)
	if taskID == uuid.Nil {
		return
	}

	if err := h.service.DeleteTask(r.Context(), taskID); err != nil {
		if errors.Is(err, worker.ErrTaskNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *handler) HandleDoTask(w http.ResponseWriter, r *http.Request) {
	taskID := getTaskID(w, r)
	if taskID == uuid.Nil {
		return
	}

	if err := h.service.DoTask(r.Context(), taskID); err != nil {
		if errors.Is(err, worker.ErrTaskNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func getTaskID(w http.ResponseWriter, r *http.Request) uuid.UUID {
	taskID := r.PathValue("task_id")

	if taskID == "" {
		http.Error(w, "task_id is required", http.StatusBadRequest)
		return uuid.Nil
	}

	taskUUID, err := uuid.Parse(taskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return uuid.Nil
	}
	return taskUUID
}
