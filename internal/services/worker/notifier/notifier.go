package notifier

import "github.com/kestfor/CrackHash/internal/services/worker"

type Notifier interface {
	Notify(result *worker.TaskResult) error
}
