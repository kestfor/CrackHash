package manager

import (
	"github.com/kestfor/CrackHash/internal/services/worker"
)

type SubTaskStatus string

const (
	SubTaskStatusPending SubTaskStatus = "pending"
	SubTaskStatusSent    SubTaskStatus = "sent"
)

type SubTask struct {
	worker.Task `bson:",inline"`
	Status      SubTaskStatus `bson:"status"`
}

type TaskStatus struct {
	Status   worker.Status `json:"status"`
	Progress int           `json:"progress"`
	Data     []string      `json:"data"`
}
