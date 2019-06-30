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

func (s *ExecutorTestSuite) TestInitExecutor() {
	executor, err := NewExecutor(DefaultExecutorParams())
	s.Nil(err)

	x := 1
	err = executor.Submit(func() { x++ })
	s.Nil(err)
	executor.Stop()
	s.Equal(2, x)
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
	executor, err := NewExecutor(p)
	s.Nil(err)
	executor.Submit(func() { time.Sleep(5 * time.Second) })
	executed := int32(0)
	for i := 0; i < 10; i++ {
		executor.Submit(func() { atomic.AddInt32(&executed, 1) })
	}
	executor.Stop()
	s.Equal(int32(0), executed)
}

func (s *ExecutorTestSuite) TestGracefulShutdown() {
	p := DefaultExecutorParams()
	p.ShutdownTimeout = 100 * time.Second

	executor, err := NewExecutor(p)
	s.Nil(err)

	executed := int32(0)
	var wg sync.WaitGroup
	wg.Add(1)
	for i := 0; i < 10; i++ {
		executor.Submit(func() {
			wg.Wait()
			atomic.AddInt32(&executed, 1)
		})
	}

	done := make(chan struct{})
	go func() {
		executor.Stop()
		close(done)
	}()
	wg.Done()
	<-done
	s.Equal(int32(10), executed)
}

func TestExecutorTestSuite(t *testing.T) {
	suite.Run(t, new(ExecutorTestSuite))
}
