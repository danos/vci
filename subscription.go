// Copyright (c) 2017,2019, AT&T Intellectual Property. All rights reserved.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package vci

import (
	"errors"
	"github.com/danos/vci/internal/queue"
	"reflect"
	"sync"
)

// The Subscription type represents a process that listens
// for notifications and reports them to the subscriber.
// The input for this process is a queue of messages delivered
// from the bus. By default the queue for the subscriber is unbounded.
// Flow control policies can be set before the process is started or may
// be set at any point during processing.
type Subscription struct {
	client           *Client
	subscriber       func(interface{})
	moduleName       string
	notificationName string
	inputType        reflect.Type
	err              error

	running *multiWriterValue
	done    *multiWriterValue
	cache   *multiWriterValue
	queue   *protectedQueue
	last    *multiWriterValue
}

func newSubscription(
	client *Client,
	moduleName, notificationName string,
	subscriber func(interface{}),
	inputType reflect.Type,
	err error,
) *Subscription {
	return &Subscription{
		client:           client,
		moduleName:       moduleName,
		notificationName: notificationName,
		subscriber:       subscriber,
		inputType:        inputType,
		err:              err,
		done:             newMultiWriterValue(false),
		running:          newMultiWriterValue(false),
		cache:            newMultiWriterValue(false),
		queue:            newProtectedQueue(queue.NewUnbounded()),
		last:             newMultiWriterValue(""),
	}
}

// Cancel cancels the subscription, stopping the process.
func (s *Subscription) Cancel() error {
	if !s.isRunning() {
		return nil
	}
	err := s.client.transport.Unsubscribe(s.moduleName,
		s.notificationName, s)
	if err != nil {
		return err
	}
	s.done.Update(func(_ interface{}) interface{} { return true })
	s.queue.Update(func(in queue.Queue) queue.Queue {
		q := in.(queue.Queue)
		q.Close()
		return q
	})
	return nil
}

// ToggleCaching toggles cacheing of the last notification.
func (s *Subscription) ToggleCaching() *Subscription {
	s.cache.Update(func(cache interface{}) interface{} {
		if cache.(bool) {
			s.last.Update(func(interface{}) interface{} {
				return ""
			})
		}
		return !cache.(bool)
	})
	return s
}

// Coalesce collapses notifications if the sender overruns
// the receiver. In some situations one need not be concerned with
// intermediate states so they can be collapsed so the last notification
// is always received by the subscriber.
func (s *Subscription) Coalesce() *Subscription {
	s.swapQueue(queue.NewCoalesced())
	return s
}

// DropAfterLimit causes a limit to be placed on the backlog
// of notifications that the subscriber will receive, any overrun will
// be dropped.
func (s *Subscription) DropAfterLimit(limit int) *Subscription {
	s.swapQueue(queue.NewBounded(limit))
	return s
}

// BlockAfterLimit causes a limit to be placed on the backlog
// of notifications that the subscriber will receive, any overrun will
// block the sender.
func (s *Subscription) BlockAfterLimit(limit int) *Subscription {
	s.swapQueue(queue.NewBlocking(limit))
	return s
}

// RemoveLimit lifts the limits imposed by Coalesce, DropAfterLimit, or
// BlockAfterLimit. This resets the Subscription's queue to be unbounded.
func (s *Subscription) RemoveLimit() *Subscription {
	s.swapQueue(queue.NewUnbounded())
	return s
}

// StoreLastNotificationInto allows one to retrieve the last
// notification that was sent if caching is enabled.
func (s *Subscription) StoreLastNotificationInto(object interface{}) error {
	last := s.last.Load().(string)
	if last == "" {
		return errors.New("no notification cached")
	}
	return s.client.unmarshalObject(last, object)
}

// Run causes the process to start.
func (s *Subscription) Run() error {
	if s.err != nil {
		return s.err
	}
	if s.isRunning() {
		return nil
	}
	err := s.client.transport.Subscribe(
		s.moduleName, s.notificationName,
		s)
	if err != nil {
		return err
	}
	s.running.Update(func(interface{}) interface{} { return true })
	go s.processNotifications()
	return nil
}

// Deliver will place the notificaiton on the input queue for
// the subscription.
func (s *Subscription) Deliver(encodedData string) error {
	queue := s.queue.Load()
	queue.Enqueue(encodedData)
	return nil
}

// isDone allows one to test whether the Subscription has been canceled.
func (s *Subscription) isDone() bool {
	return s.done.Load().(bool)
}

func (s *Subscription) swapQueue(new queue.Queue) {
	s.queue.Update(func(in queue.Queue) queue.Queue {
		old := in.(queue.Queue)
		old.Close()
		queue.Move(new, old)
		return new
	})
}

func (s *Subscription) cacheNotification(encodedData string) {
	if !s.cache.Load().(bool) {
		return
	}
	s.last.Update(func(_ interface{}) interface{} {
		return encodedData
	})
}

func (s *Subscription) validateNotification(
	encodedData string,
) (string, error) {
	in := map[string]interface{}{
		yangdModuleName + ":module-name": s.moduleName,
		yangdModuleName + ":name":        s.notificationName,
		yangdModuleName + ":input":       encodedData,
	}

	var result map[string]interface{}
	err := s.client.Call(yangdModuleName, "validate-notification", in).
		StoreOutputInto(&result)
	if err != nil {
		return "", err
	}

	return result[yangdModuleName+":output"].(string), nil
}

func (s *Subscription) processNotifications() {
	for !s.isDone() {
		q := s.queue.Load()
		queue.Range(q, func(v interface{}) {
			encodedData := v.(string)
			encodedData, err := s.validateNotification(encodedData)
			if err != nil {
				return
			}
			s.cacheNotification(encodedData)
			val, err := s.decodeInput(encodedData)
			if err != nil {
				return
			}
			s.subscriber(val)
		})
	}
	s.running.Update(func(interface{}) interface{} { return false })
}

func (s *Subscription) decodeInput(encodedData string) (interface{}, error) {
	return decodeValue(s.inputType, encodedData)
}

func (s *Subscription) isRunning() bool {
	return s.running.Load().(bool)
}

type protectedQueue struct {
	mu sync.RWMutex
	q  queue.Queue
}

func newProtectedQueue(q queue.Queue) *protectedQueue {
	return &protectedQueue{
		q: q,
	}
}

func (q *protectedQueue) Load() queue.Queue {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.q
}

func (q *protectedQueue) Update(fn func(queue.Queue) queue.Queue) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.q = fn(q.q)
}
