// Package timeout implements a system for running actions after an specified
// time.
package timeout

import (
	"errors"
	"sync/atomic"
	"time"

	"github.com/alan-b-lima/pkg/heap"
)

// Timeout is a system that consumer ticketed functions that will run after an
// specified timestamp. Those are run on their own goroutines.
//
// The zero value of [Timeout] is NOT valid, use [New] to create a new timeout.
//
// A timeout created with [New] is not yet timing-out tickets, [Timeout.Start]
// has to be called first. To stop a timeout, call [Timeout.Stop]. A timeout
// can be restarted after being stopped, without clearing or altering its
// state, by calling [Timeout.Start] again.
//
// If a ticket expires while the timeout is stopped, it will not be executed
// until soon after the timeout is restarted.
//
// Timeouts are safe for concurrent use by multiple goroutines.
type Timeout struct {
	heap heap.Heap[ticket]

	running atomic.Bool
	push    chan ticket
	close   chan struct{}
}

var (
	ErrAlreadyRunning = errors.New("timeout: already running")
	ErrNotRunning     = errors.New("timeout: not running")
)

// New creates a new timeout. To start the timeout, call [Timeout.Start].
func New() *Timeout {
	return &Timeout{
		push:  make(chan ticket, 32),
		close: make(chan struct{}, 1),
	}
}

// Start starts the timeout, allowing it to execute posted tickets. If the
// timeout is already running, an error is returned.
//
// Start can be used to restart, without clearing its state, an previously
// stopped timeout.
func (t *Timeout) Start() error {
	if t.running.Load() {
		return ErrAlreadyRunning
	}

	go flush(t)
	return nil
}

// Stop stops the timeout, preventing it from executing posted tickets. If the
// timeout is not running, an error is returned.
//
// Stopping a timeout does not clear its queue, nor makes it invalid, so the
// timeout can be started again. Effectively pausing the timeout.
func (t *Timeout) Stop() error {
	if !t.running.Load() {
		return ErrNotRunning
	}

	t.close <- struct{}{}
	<-t.close

	return nil
}

// Post posts an action to be executed after the specified expiration. If the
// timeout is not running and the action expires, it will be executed soon
// after the timeout is restarted.
//
// If multiple actions are posted with the same expiration, they are NOT
// guaranteed to be executed in the same order they were posted.
//
// Nor are ticket actions guaranteed to be executed in a known window of time,
// only that it is executed after the specified expiration.
func (t *Timeout) Post(action func(), expires time.Time) {
	t.push <- ticket{
		Expires: expires,
		Action:  action,
	}
}

// PostWithDuration posts an action to be executed after the specified
// duration. Equivalent to:
//
//	t.Post(action, time.Now().Add(duration))
//
// See [Timeout.Post] for more details.
func (t *Timeout) PostWithDuration(action func(), duration time.Duration) {
	t.Post(action, time.Now().Add(duration))
}

type ticket struct {
	Expires time.Time
	Action  func()
}

func (t ticket) Less(o ticket) bool {
	return t.Expires.Before(o.Expires)
}

func flush(t *Timeout) {
	t.running.Store(true)

	for {
		var after <-chan time.Time
		if t.heap.Len() > 0 {
			delay := time.Until(t.heap.Peek().Expires)
			after = time.After(delay)
		}

		select {
		case <-after:
			ticket := t.heap.Pop()
			go ticket.Action()

		case ticket := <-t.push:
			t.heap.Push(ticket)

		case <-t.close:
			t.running.Store(false)
			t.close <- struct{}{}

			return
		}
	}
}
