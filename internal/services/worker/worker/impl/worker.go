package impl

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"log/slog"
	"math"
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
	m := len(w.task.IterationAlphabet)
	if m == 0 {
		return 0
	}
	if m == 1 {
		return w.task.MaxLength
	}

	total := 0
	for i := 1; i <= w.task.MaxLength; i++ {
		total += int(math.Pow(float64(m), float64(i)))
	}
	return total
}

func (w *workerImpl) do(ctx context.Context) {
	slog.Info("worker started processing",
		slog.String("task_id", w.task.TaskID.String()),
		slog.String("target_hash", w.task.TargetHash),
		slog.String("iteration_alphabet", w.task.IterationAlphabet),
		slog.Int("max_length", w.task.MaxLength),
	)

	progress := &worker.TaskProgress{
		TaskID:          w.task.TaskID,
		IterationsDone:  0,
		Status:          worker.StatusInProgress,
		TotalIterations: w.totalIterations(),
		Result:          []string{},
	}

	w.progress.Store(progress)

	iterationsDone := 0
	matches := make([]string, 0)

	progressFreq := 1000

	generator := WordGenerator(w.task.MaxLength, w.task.IterationAlphabet)
	for word := range generator.Iterate() {
		select {
		case <-ctx.Done():
			slog.Warn("task cancelled", slog.String("task_id", w.task.TaskID.String()))
			progress.Status = worker.StatusError
			progress.Result = matches
			w.progress.Store(progress)
			return
		default:
		}

		iterationsDone++

		if iterationsDone%progressFreq == 0 {
			progress.IterationsDone = iterationsDone
			progress.Result = matches
			w.progress.Store(progress)
		}

		hash := md5.Sum([]byte(word))
		encoded := hex.EncodeToString(hash[:])

		if w.task.TargetHash == encoded {
			slog.Info("match found",
				slog.String("task_id", w.task.TaskID.String()),
				slog.String("word", word),
			)
			matches = append(matches, word)
		}
	}

	progress.Status = worker.StatusReady
	progress.IterationsDone = iterationsDone
	progress.Result = matches
	w.progress.Store(progress)

	slog.Info("worker finished processing",
		slog.String("task_id", w.task.TaskID.String()),
		slog.Int("matches_found", len(matches)),
	)

	for _, n := range w.notifiers {
		_ = n.Notify(progress)
	}
}
