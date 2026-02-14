package impl

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"sync/atomic"

	"github.com/kestfor/CrackHash/internal/services/worker"
	"github.com/kestfor/CrackHash/internal/services/worker/notifier"
	workerinterface "github.com/kestfor/CrackHash/internal/services/worker/worker"
)

type workerImpl struct {
	progress atomic.Pointer[worker.TaskProgress]
	task     *worker.Task
	result   *worker.TaskResult

	notifiers []notifier.Notifier
}

var _ workerinterface.Worker = (*workerImpl)(nil)

func NewWorker(notifiers []notifier.Notifier) *workerImpl {
	return &workerImpl{
		progress:  atomic.Pointer[worker.TaskProgress]{},
		notifiers: notifiers,
		result:    &worker.TaskResult{},
	}
}

func (w *workerImpl) Do(ctx context.Context, task *worker.Task) {
	w.task = task
	w.progress.Store(&worker.TaskProgress{
		TaskID:         task.TaskID,
		IterationsDone: 0,
		Status:         worker.StatusNotStarted,
	})

	go func() {
		w.do(ctx)
	}()
}

func (w *workerImpl) Progress() *worker.TaskProgress {
	return w.progress.Load()
}

func (w *workerImpl) totalIterations() int {
	if len(w.task.IterationAlphabet) == 1 {
		return 1
	}

	m := len(w.task.IterationAlphabet)
	return m * (m ^ w.task.MaxLength - 1) / (m - 1)
}

func (w *workerImpl) do(ctx context.Context) {
	progress := &worker.TaskProgress{
		TaskID:          w.task.TaskID,
		IterationsDone:  0,
		Status:          worker.StatusInProgress,
		TotalIterations: w.totalIterations(),
	}

	w.progress.Store(progress)

	iterationsDone := 0

	matches := make([]string, 0)

	// TODO configure
	progressFreq := 10

	generator := WordGenerator(w.task.MaxLength, w.task.IterationAlphabet)
	for word := range generator.Iterate() {
		select {
		case <-ctx.Done():
			progress.Status = worker.StatusError
			w.progress.Store(progress)
			return
		default:
		}

		iterationsDone++

		if iterationsDone%progressFreq == 0 {
			progress.IterationsDone = iterationsDone
			w.progress.Store(progress)
		}

		hash := md5.Sum([]byte(word))
		encoded := hex.EncodeToString(hash[:])

		if w.task.TargetHash == encoded {
			matches = append(matches, word)
		}

	}

	progress.Status = worker.StatusReady
	progress.IterationsDone = iterationsDone
	w.progress.Store(progress)

	for _, n := range w.notifiers {
		_ = n.Notify(progress)
	}
}
