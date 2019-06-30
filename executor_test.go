package executor

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ExecutorTestSuite struct {
	suite.Suite
}

func (suite *ExecutorTestSuite) SetupSuite() {
	flag.Set("logtostderr", "true")
	flag.Parse()
}

func (s *ExecutorTestSuite) TestInitExecutor() {
	executor, err := NewExecutor(DefaultExecutorParams())
	assert.Nil(s.T(), err)

	x := 1
	err = executor.Submit(func() { x++ })
	assert.Nil(s.T(), err)
	executor.Stop()
	assert.Equal(s.T(), x, 2)
}

func TestExecutorTestSuite(t *testing.T) {
	suite.Run(t, new(ExecutorTestSuite))
}
