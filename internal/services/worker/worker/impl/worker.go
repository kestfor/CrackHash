package impl

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/kestfor/CrackHash/internal/services/worker"
	"github.com/kestfor/CrackHash/internal/services/worker/notifier"
	workerinterface "github.com/kestfor/CrackHash/internal/services/worker/worker"
)

type workerImpl struct {
	workerID     uuid.UUID
	task         *worker.Task
	notifiers    []notifier.Notifier
	progress     atomic.Pointer[worker.TaskProgress]
	notifyPeriod time.Duration
	cancel       context.CancelFunc
}

var _ workerinterface.Worker = (*workerImpl)(nil)

func NewWorker(workerID uuid.UUID, notifiers []notifier.Notifier, notifyPeriod time.Duration) *workerImpl {
	return &workerImpl{
		workerID:     workerID,
		notifyPeriod: notifyPeriod,
		notifiers:    notifiers,
	}
}

func (w *workerImpl) Do(ctx context.Context, task *worker.Task) {
	w.task = task
	ctx, w.cancel = context.WithCancel(ctx)

	go func() {
		w.do(ctx)
	}()
}

func (w *workerImpl) Cancel() {
	w.cancel()
}

func (w *workerImpl) do(ctx context.Context) {
	slog.Info("worker started processing",
		slog.String("task_id", w.task.TaskID.String()),
		slog.String("target_hash", w.task.TargetHash),
		slog.String("alphabet", w.task.Alphabet),
		slog.Int("max_length", w.task.MaxLength),
		slog.Uint64("start_index", w.task.StartIndex),
		slog.Uint64("end_index", w.task.EndIndex),
	)

	wrkContext := &workerContext{}
	wrkContext.SetStatus(worker.StatusInProgress)
	wrkContext.TotalIterations = int(w.task.EndIndex - w.task.StartIndex)

	failed := false

	notifyContext, cancelNotify := context.WithCancel(ctx)
	go w.backgroundNotify(notifyContext, wrkContext)

	searchSpace := NewSearchSpace(w.task.Alphabet, w.task.MaxLength)
	buf := make([]byte, w.task.MaxLength)

	for idx := w.task.StartIndex; idx < w.task.EndIndex; idx++ {
		select {
		case <-ctx.Done():
			slog.Warn("task cancelled",
				slog.Any("task_id", w.task.TaskID),
				slog.String("reason", ctx.Err().Error()))
			wrkContext.SetStatus(worker.StatusError)
			failed = true
			break
		default:
		}

		if failed {
			break
		}

		wordLen := searchSpace.FillWord(idx, buf)
		if wordLen == 0 {
			continue
		}

		word := buf[:wordLen]
		hash := md5.Sum(word)
		encoded := hex.EncodeToString(hash[:])

		if w.task.TargetHash == encoded {
			slog.Info("match found",
				slog.String("task_id", w.task.TaskID.String()),
				slog.String("word", string(word)),
			)
			wrkContext.AddMatch(string(word))
		}

		wrkContext.IterationsDone.Add(1)
	}

	if !failed {
		wrkContext.SetStatus(worker.StatusReady)
	}

	progress := w.contextToProgress(wrkContext)

	slog.Info("worker finished processing",
		slog.Any("task_id", w.task.TaskID),
		slog.Any("matches_found", progress.Result),
	)

	cancelNotify() // stop background notification

	w.notifyProgress(progress) // send final notification
}

func (w *workerImpl) backgroundNotify(ctx context.Context, wrkContext *workerContext) {

	progress := w.contextToProgress(wrkContext)

	ticker := time.NewTicker(w.notifyPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:

			progress.IterationsDone = int(wrkContext.IterationsDone.Load())
			progress.Result = wrkContext.Matches()
			progress.Status = wrkContext.Status()

			slog.Info("worker progress", slog.Any("progress", progress))

			w.notifyProgress(progress)
		}
	}
}

func (w *workerImpl) notifyProgress(progress *worker.TaskProgress) {
	for _, n := range w.notifiers {
		if err := n.Notify(progress); err != nil {
			slog.Warn("failed to notify subscriber",
				slog.String("task_id", progress.TaskID.String()),
				slog.Any("error", err),
			)
		}
	}
}

func (w *workerImpl) contextToProgress(wrkContext *workerContext) *worker.TaskProgress {
	return &worker.TaskProgress{
		TaskID:          w.task.TaskID,
		WorkerID:        w.workerID,
		IterationsDone:  int(wrkContext.IterationsDone.Load()),
		TotalIterations: wrkContext.TotalIterations,
		Result:          wrkContext.Matches(),
		Status:          wrkContext.Status(),
	}
}
