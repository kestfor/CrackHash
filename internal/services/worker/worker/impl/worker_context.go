package impl

import (
	"sync"
	"sync/atomic"

	"github.com/kestfor/CrackHash/internal/services/worker"
)

type workerContext struct {
	matchesMutex    sync.Mutex
	matches         []string
	IterationsDone  atomic.Int64
	TotalIterations int
	status          atomic.Pointer[worker.Status]
}

func (w *workerContext) Status() worker.Status {
	return *w.status.Load()
}

func (w *workerContext) SetStatus(status worker.Status) {
	w.status.Store(&status)
}

func (w *workerContext) AddMatch(match string) {
	w.matchesMutex.Lock()
	defer w.matchesMutex.Unlock()
	w.matches = append(w.matches, match)
}

func (w *workerContext) Matches() []string {
	w.matchesMutex.Lock()
	defer w.matchesMutex.Unlock()
	return w.matches
}
