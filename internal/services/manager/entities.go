package manager

import (
	"github.com/kestfor/CrackHash/internal/services/worker"
)

type TaskStatus struct {
	Status   worker.Status `json:"status"`
	Progress int           `json:"progress"`
	Data     []string      `json:"data"`
}
