package notifier

import (
	"context"
	"sync"
)

type Notifier interface {
	Notify()
	Wait(ctx context.Context) error
}

type NotifierImpl struct {
	lock                sync.Mutex
	notificationContext context.Context
	notificationFunc    context.CancelFunc
}

func NewNotifier() Notifier {
	notificationContext, notificationFunc := context.WithCancel(context.Background())
	return &NotifierImpl{
		notificationContext: notificationContext,
		notificationFunc:    notificationFunc,
	}
}

func (n *NotifierImpl) getNotificationContext() context.Context {
	n.lock.Lock()
	defer n.lock.Unlock()

	return n.notificationContext
}

func (n *NotifierImpl) Wait(ctx context.Context) error {
	waitCtx := n.getNotificationContext()
	select {
	case <-waitCtx.Done():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (n *NotifierImpl) Notify() {
	n.lock.Lock()
	defer n.lock.Unlock()

	n.notificationFunc()
	n.notificationContext, n.notificationFunc = context.WithCancel(context.Background())
}
