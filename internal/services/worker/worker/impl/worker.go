package impl

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"sync/atomic"

	"github.com/kestfor/CrackHash/internal/services/worker"
	"github.com/kestfor/CrackHash/internal/services/worker/notifier"
	workerinterface "github.com/kestfor/CrackHash/internal/services/worker/worker"
	"github.com/kestfor/CrackHash/pkg"
)

type workerImpl struct {
	progress atomic.Pointer[worker.TaskProgress]
	status   atomic.Pointer[worker.Status]
	task     *worker.Task
	result   *worker.TaskResult

	notifiers []notifier.Notifier
}

var _ workerinterface.Worker = (*workerImpl)(nil)

func NewWorker(notifiers []notifier.Notifier) *workerImpl {
	return &workerImpl{
		progress:  atomic.Pointer[worker.TaskProgress]{},
		status:    atomic.Pointer[worker.Status]{},
		notifiers: notifiers,
		result:    &worker.TaskResult{},
	}
}

func (w *workerImpl) Do(ctx context.Context, task *worker.Task) {
	w.task = task
	w.progress.Store(&worker.TaskProgress{
		TaskID:         task.TaskID,
		IterationsDone: 0,
	})

	go func() {
		w.do(ctx)
	}()
}

func (w *workerImpl) Progress() *worker.TaskProgress {
	return w.progress.Load()
}

func (w *workerImpl) Result() (result *worker.TaskResult, status worker.Status) {
	stat := w.status.Load()
	if stat == nil {
		return nil, worker.StatusNotStarted
	}

	if *stat == worker.StatusDone {
		return w.result, worker.StatusDone
	}

	return nil, *stat

}

func (w *workerImpl) do(ctx context.Context) {
	w.status.Store(pkg.ToPtr(worker.StatusInProgress))

	iterationsDone := 0
	matches := make([]string, 0)

	// TODO configure
	progressFreq := 10

	generator := WordGenerator(w.task.MaxLength, w.task.IterationAlphabet)
	for word := range generator.Iterate() {
		select {
		case <-ctx.Done():
			w.status.Store(pkg.ToPtr(worker.StatusTimeout))
			return
		default:
		}

		iterationsDone++

		if iterationsDone%progressFreq == 0 {
			w.progress.Store(&worker.TaskProgress{
				TaskID:         w.task.TaskID,
				IterationsDone: iterationsDone,
			})
		}

		hash := md5.Sum([]byte(word))
		encoded := hex.EncodeToString(hash[:])

		if w.task.TargetHash == encoded {
			matches = append(matches, word)
		}

	}

	w.result = &worker.TaskResult{
		TaskID: w.task.TaskID,
		Words:  matches,
	}

	w.status.Store(pkg.ToPtr(worker.StatusDone))
	for _, n := range w.notifiers {
		_ = n.Notify(w.result)
	}
}
