// Package scheduler implements a system for running actions after an specified
// time.
package scheduler

import (
	"errors"
	"sync/atomic"
	"time"

	"github.com/alan-b-lima/pkg/heap"
)

// Scheduler is a system that schedules functions, called tickets, to be
// executed after an specified timestamp. Each ticket is run on their own
// goroutines.
//
// The zero value of [Scheduler] is NOT valid, use [New] to create a new
// schedulers.
//
// A scheduler created with [New] is not yet timing-out tickets,
// [Scheduler.Start] has to be called first. To stop the scheduler, call
// [Scheduler.Stop]. A scheduler can be restarted after being stopped, without
// clearing or altering its state, by calling [Scheduler.Start] again.
//
// Individuals tickets cannot be peaked once posted.
//
// If a ticket expires while the scheduler is stopped, it will not be executed
// until soon after the scheduler is restarted.
//
// Schedulers are safe for concurrent use by multiple goroutines.
type Scheduler struct {
	heap heap.Heap[ticket]

	running atomic.Bool
	push    chan ticket
	close   chan struct{}
}

var (
	ErrAlreadyRunning = errors.New("scheduler: already running")
	ErrNotRunning     = errors.New("scheduler: not running")
)

// New creates a new scheduler. To start the scheduler, call [Scheduler.Start].
func New() *Scheduler {
	return &Scheduler{
		push:  make(chan ticket, 32),
		close: make(chan struct{}, 1),
	}
}

// Start starts the scheduler, allowing it to execute posted tickets. If the
// scheduler is already running, an error is returned.
//
// Start can be used to restart, without clearing its state, an previously
// stopped scheduler.
func (t *Scheduler) Start() error {
	if t.running.Load() {
		return ErrAlreadyRunning
	}

	go flush(t)
	return nil
}

// Stop stops the scheduler, preventing it from executing posted tickets. If
// the scheduler is not running, an error is returned.
//
// Stopping a scheduler does not clear its queue, nor makes it invalid, so the
// scheduler can be started again. Effectively pausing the scheduler.
func (t *Scheduler) Stop() error {
	if !t.running.Load() {
		return ErrNotRunning
	}

	t.close <- struct{}{}
	<-t.close

	return nil
}

// Post posts an ticket to be executed after the specified expiration. If the
// scheduler is not running and the ticket expires, it will be executed soon
// after the scheduler is restarted.
//
// If multiple tickets are posted with the same expiration, they are NOT
// guaranteed to be executed in the same order they were posted.
//
// Nor are ticket tickets guaranteed to be executed in a known window of time,
// only that it is executed after the specified expiration.
func (t *Scheduler) Post(action func(), expires time.Time) {
	t.push <- ticket{
		Expires: expires,
		Action:  action,
	}
}

// PostWithDuration posts an ticked to be executed after the specified
// duration. Equivalent to:
//
//	t.Post(action, time.Now().Add(duration))
//
// See [Scheduler.Post] for more details.
func (t *Scheduler) PostWithDuration(action func(), duration time.Duration) {
	t.Post(action, time.Now().Add(duration))
}

type ticket struct {
	Expires time.Time
	Action  func()
}

func (t ticket) Less(o ticket) bool {
	return t.Expires.Before(o.Expires)
}

func flush(t *Scheduler) {
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
