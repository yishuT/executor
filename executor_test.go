package executor

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type ExecutorTestSuite struct {
	suite.Suite
}

func (s *ExecutorTestSuite) TestJobExecuted() {
	executor, err := NewExecutor(DefaultExecutorParams())
	s.Nil(err)

	x := 1
	invC := make(chan struct{})
	err = executor.Submit(func() {
		x++
		close(invC)
	})
	s.Nil(err)
	<-invC
	s.Equal(2, x)
	executor.Stop()
}

func (s *ExecutorTestSuite) TestJobsFinished() {
	executor, err := NewExecutor(DefaultExecutorParams())
	s.Nil(err)

	var wg sync.WaitGroup
	wg.Add(10)
	for i := 0; i < 10; i++ {
		executor.Submit(func() {
			wg.Done()
		})
	}
	wg.Wait()
}

func (s *ExecutorTestSuite) TestJobTruncated() {
	p := DefaultExecutorParams()
	p.ShutdownTimeout = 5 * time.Second
	p.MaxJobQueueWaitTime = 1 * time.Second
	p.NumWorkers = 1

	executed := int32(0)

	executor, err := NewExecutor(p)
	s.Nil(err)
	invC := make(chan struct{})
	executor.Submit(func() {
		close(invC)
		atomic.AddInt32(&executed, 1)
		time.Sleep(5 * time.Second)
	})
	<-invC

	for i := 0; i < 10; i++ {
		executor.Submit(func() { atomic.AddInt32(&executed, 1) })
	}
	executor.Stop()
	s.Equal(int32(1), executed)
}

func (s *ExecutorTestSuite) TestJobRejected() {
	p := DefaultExecutorParams()
	p.ShutdownTimeout = 5 * time.Second
	p.MaxJobQueueWaitTime = 1 * time.Second
	p.MaxJobQueueCapacity = 1
	p.NumWorkers = 1

	executor, err := NewExecutor(p)
	s.Nil(err)
	executed := int32(0)
	invC := make(chan struct{})
	executor.Submit(func() {
		close(invC)
		atomic.AddInt32(&executed, 1)
		time.Sleep(5 * time.Second)
	})
	<-invC

	err = executor.Submit(func() { atomic.AddInt32(&executed, 1) })
	s.Nil(err)
	err = executor.Submit(func() { atomic.AddInt32(&executed, 1) })
	s.NotNil(err)

	executor.Stop()
	s.Equal(int32(1), executed)
}

func (s *ExecutorTestSuite) TestGracefulShutdown() {
	p := DefaultExecutorParams()
	p.ShutdownTimeout = 100 * time.Second
	p.MaxJobQueueWaitTime = 100 * time.Second

	executor, err := NewExecutor(p)
	s.Nil(err)

	executed := int32(0)
	var wg, wg2 sync.WaitGroup
	wg.Add(1)
	wg2.Add(p.NumWorkers)
	for i := 0; i < p.NumWorkers; i++ {
		executor.Submit(func() {
			wg2.Done()
			wg.Wait()
			atomic.AddInt32(&executed, 1)
		})
	}

	done := make(chan struct{})
	wg2.Wait()
	go func() {
		executor.Stop()
		close(done)
	}()

	wg.Done()
	<-done
	s.Equal(int32(p.NumWorkers), executed)
}

func TestExecutorTestSuite(t *testing.T) {
	suite.Run(t, new(ExecutorTestSuite))
}
