package executor

import (
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type ExecutorParams struct {
	NumWorkers          int
	MaxJobQueueCapacity int
	MaxJobQueueWaitTime time.Duration
	ShutdownTimeout     time.Duration
}

/*
DefaultExecutorParams generates a default param struct for
creating Executor.

NumWorkers: number of CPU report by go runtime library
MaxJobQueueCapacity: 1000
MaxJobQueueWaitTime: 30 seconds
ShutdownTimeout: 3 seconds
*/
func DefaultExecutorParams() ExecutorParams {
	return ExecutorParams{
		NumWorkers:          runtime.NumCPU(),
		MaxJobQueueCapacity: 1000,
		MaxJobQueueWaitTime: 30 * time.Second,
		ShutdownTimeout:     3 * time.Second,
	}
}

func (p ExecutorParams) validate() error {
	if p.NumWorkers <= 0 {
		return errors.New("executor params: non positive NumWorkers")
	}
	if p.MaxJobQueueCapacity < 0 {
		return errors.New("executor params: negative MaxJobQueueCapacity")
	}
	return nil
}

type executorJob struct {
	runnable func()
	ts       time.Time
	done     func()
}

func (ej *executorJob) invoke() {
	ej.runnable()
	ej.done()
}

/*
worker internal states:

Unstarted
Idle: waiting for a job to execute
Running: executing a job
Stopped
*/
type worker struct {
	done    chan struct{}
	jobChan chan *executorJob

	executor *Executor
}

func newWorker(e *Executor) *worker {
	return &worker{
		done:     make(chan struct{}),
		jobChan:  make(chan *executorJob),
		executor: e,
	}
}

func (w *worker) Run() {
	for {
		job := w.executor.tryGetJobAndRegister(w)
		if job != nil {
			job.invoke()
		} else {
			select {
			case job := <-w.jobChan:
				job.invoke()
			case <-w.done:
				return
			}
		}
	}
}

func (w *worker) Stop() {
	close(w.done)
}

func (w *worker) Take(j *executorJob) {
	w.jobChan <- j
}

type Executor struct {
	// parameters, immutable fields
	numWorkers          int
	maxJobQueueCapacity int
	maxJobQueueWaitTime time.Duration
	shutdownTimeout     time.Duration

	// core objects
	mu             sync.Mutex
	jobQueue       []*executorJob
	stopChan       chan struct{}
	workerWaitList chan *worker
	stopped        bool
	workers        []*worker

	// stats
	inflightJobs int32
}

func NewExecutor(params ExecutorParams) (*Executor, error) {
	exec, err := newExecutor(params)
	if err != nil {
		return nil, err
	}

	for i := 0; i < params.NumWorkers; i++ {
		w := newWorker(exec)
		exec.workers[i] = w
		go w.Run()
	}
	go exec.truncateLoop()

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
		jobQueue:            make([]*executorJob, 0),
		stopChan:            make(chan struct{}),
		workerWaitList:      make(chan *worker, params.NumWorkers),
		workers:             make([]*worker, params.NumWorkers),
		inflightJobs:        0,
		stopped:             false,
	}, nil
}

func (e *Executor) truncateLoop() {
	ticker := time.NewTicker(e.maxJobQueueWaitTime)
	for {
		select {
		case <-e.stopChan:
			// truncate all
			jobs := e.cleanJobQueue()
			go e.dropJobs(jobs)
			return
		case <-ticker.C:
			jobs := e.removeTimeoutJobsFromQueue()
			go e.dropJobs(jobs)
		}

	}
}

func (e *Executor) removeTimeoutJobsFromQueue() []*executorJob {
	e.mu.Lock()
	defer e.mu.Unlock()
	now := time.Now()
	var i int
	for i = 0; i < len(e.jobQueue); i++ {
		if now.Sub(e.jobQueue[i].ts) < e.maxJobQueueWaitTime {
			break
		}
	}
	jobs := e.jobQueue[:i]
	e.jobQueue = e.jobQueue[i:]
	return jobs
}

func (e *Executor) cleanJobQueue() []*executorJob {
	e.mu.Lock()
	defer e.mu.Unlock()
	jobs := e.jobQueue
	e.jobQueue = nil
	return jobs
}

func (e *Executor) dropJobs(jobs []*executorJob) {
	for _, j := range jobs {
		j.done()
	}
}

func (e *Executor) tryGetJobAndRegister(w *worker) *executorJob {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.stopped {
		return nil
	}
	if len(e.jobQueue) > 0 {
		head := e.jobQueue[0]
		e.jobQueue = e.jobQueue[1:]
		return head
	}
	e.workerWaitList <- w
	return nil
}

func (e *Executor) pushLocked(job *executorJob) error {
	if len(e.jobQueue) == e.maxJobQueueCapacity {
		return errors.New("executor: queue over flow")
	}
	e.jobQueue = append(e.jobQueue, job)
	return nil
}

func (e *Executor) Submit(runnable func()) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.stopped {
		return errors.New("executor: submit a job to stopped executor")
	}
	atomic.AddInt32(&e.inflightJobs, 1)
	job := &executorJob{
		runnable: runnable,
		ts:       time.Now(),
		done:     func() { atomic.AddInt32(&e.inflightJobs, -1) },
	}
	select {
	case w := <-e.workerWaitList:
		w.Take(job)
	default:
		return e.pushLocked(job)
	}
	return nil
}

func (e *Executor) Stop() {
	e.mu.Lock()
	// stop new job submittion and workers acquiring job first
	e.stopped = true
	e.mu.Unlock()

	close(e.stopChan)
	for _, w := range e.workers {
		w.Stop()
	}

	// Graceful shutdown
	afterC := time.After(e.shutdownTimeout)
	for {
		inflightJobs := atomic.LoadInt32(&e.inflightJobs)
		if inflightJobs == 0 {
			break
		}

		select {
		case <-afterC:
			break
		case <-time.After(100 * time.Millisecond):
			continue
		}
	}
}
