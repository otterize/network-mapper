package notifier

import (
	"context"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

type NotifierTestSuite struct {
	suite.Suite
}

func (s *NotifierTestSuite) TestNotifierDummy() {
	notifier := NewNotifier()

	myFlag := false
	go func() {
		myFlag = true
		notifier.Notify()
	}()
	err := notifier.Wait(context.Background())
	s.NoError(err)
	s.True(myFlag)
}

func (s *NotifierTestSuite) TestWaitUntilContextCancelled() {
	notifier := NewNotifier()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	go func() {
		notifier.Notify()
	}()
	time.Sleep(5 * time.Millisecond)
	err := notifier.Wait(ctx)
	s.Error(err)
}

func (s *NotifierTestSuite) TestWaitUntilCondition() {
	notifier := NewNotifier()

	condition := false
	go func() {
		notifier.Notify()
	}()
	go func() {
		time.Sleep(50 * time.Millisecond)
		condition = true
		notifier.Notify()
	}()
	for !condition {
		err := notifier.Wait(context.Background())
		s.NoError(err)
	}

	s.Require().True(condition)
}

func TestNotifierTestSuite(t *testing.T) {
	suite.Run(t, new(NotifierTestSuite))
}
