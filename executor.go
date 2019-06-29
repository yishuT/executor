package callback

import (
	"runtime"
	"sync"
	"time"
)

type ExecutorParams struct {
	NumWorkers          int
	MaxJobQueueCapacity int
	MaxJobQueueWaitTime time.Duration
}

func DefaultExecutorParams() ExecutorParams {
	return ExecutorParams{
		NumWorkers:          runtime.NumCPU(),
		MaxJobQueueCapacity: 1000,
		MaxJobQueueWaitTime: 30 * time.Second,
	}
}

func (p ExecutorParams) validate() error {
	return nil
}

type Executor struct {
	numWorkers          int
	maxJobQueueCapacity int
	maxJobQueueWaitTime time.Duration

	mu       sync.Mutex
	jobQueue []func()
	stopChan chan struct{}
}

func NewExecutor(params ExecutorParams) (*Executor, error) {
	exec, err := newExecutor(params)
	if err != nil {
		return nil, err
	}
	return exec, err
}

func newExecutor(params ExecutorParams) (*Executor, error) {
	if err := params.validate(); err != nil {
		return nil, err
	}

	return &Executor{
		numWorkers:          params.NumWorkers,
		maxJobQueueCapacity: params.MaxJobQueueCapacity,
		maxJobQueueWaitTime: params.MaxJobQueueWaitTime,
		jobQueue:            make([]func(), 0),
		stopChan:            make(chan struct{}),
	}, nil
}

func (e *Executor) truncateLoop() {
	for {
		ticker := time.NewTicker(e.maxJobQueueWaitTime)
		select {
		case <-e.stopChan:
			// truncate all
			e.jobQueue = nil
		case <-ticker.C:
			e.removeTimeoutJobsFromQueueLocked()
		}
	}
}

func (e *Executor) removeTimeoutJobsFromQueueLocked() {
	e.mu.Lock()
	defer e.mu.Unlock()
}

func (e *Executor) Submit(runnable func()) error {
	return nil
}

func (e *Executor) Stop() {
	close(e.stopChan)
}
