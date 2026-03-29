// Package scheduler implements a system for running actions after an specified
// time.
package scheduler

import (
	"errors"
	"math"
	"time"

	"github.com/alan-b-lima/pkg/heap"
)

// Scheduler is a system that schedules functions, called tickets, to be
// executed after a deadline or an specified timestamp. Each ticket is run on
// their own goroutines.
//
// The zero value of a scheduler is NOT a valid scheduler. Use [New] to create
// a new scheduler. A new scheduler is always paused, call [Scheduler.Start] to
// start it.
//
// A scheduler can be in one of three states: running, paused, or stopped. When
// running, the scheduler executes tickets as their deadlines are reached. When
// paused, the scheduler does not execute any tickets, but they are still
// accepted and stored. When stopped, the scheduler does not execute any
// tickets, and it does not accept any new tickets.
//
// After a scheduler is stopped, it cannot be restarted and calling
// [Scheduler.Start] returns an error.
//
// A scheduler is safe for concurrent access by multiple goroutines.
type Scheduler struct {
	push  chan ticket
	delta chan state
	state state
}

const min_tickets = 16

var ErrStopped = errors.New("scheduler: stopped")

// New allocated a new paused scheduler and launches its work goroutine.
func New() *Scheduler {
	t := &Scheduler{
		push:  make(chan ticket, min_tickets),
		delta: make(chan state, 1),
		state: paused,
	}

	go flush(t)
	return t
}

// Start starts or unpauses a scheduler, if the scheduler is not stopped, Start
// is idempotent.
//
// Start returns an error if the scheduler is stopped.
func (t *Scheduler) Start() error {
	if t.state == stopped {
		return ErrStopped
	}

	if t.state != running {
		t.delta <- running
	}

	return nil
}

// Pause pauses a scheduler, if the scheduler is not stopped, Pause is idempotent.
//
// Pause returns an error if the scheduler is stopped.
func (t *Scheduler) Pause() error {
	if t.state == stopped {
		return ErrStopped
	}

	if t.state != paused {
		t.delta <- paused
	}

	return nil
}

// Stop stops a scheduler, if the scheduler is not stopped, Stop is idempotent.
func (t *Scheduler) Stop() error {
	if t.state != stopped {
		t.delta <- stopped
	}

	return nil
}

// Post posts a ticket to be executed after the specified deadline. If the
// scheduler is stopped, Post does nothing.
//
// Actions posted are not guaranteed to be executed at the exact deadline. The
// scheduler also does not guarantee the order of execution of tickets.
func (t *Scheduler) Post(action func(), deadline time.Time) {
	ticket := ticket{
		Deadline: deadline,
		Action:   action,
	}

	if push := t.push; push != nil {
		push <- ticket
	}
}

// PostWithDuration posts a ticket to be executed after the specified duration.
// If the scheduler is stopped, PostWithDuration does nothing.
func (t *Scheduler) PostWithDuration(action func(), duration time.Duration) {
	t.Post(action, time.Now().Add(duration))
}

type state byte

const (
	stopped state = iota
	running
	paused
)

type ticket struct {
	Deadline time.Time
	Action   func()
}

func (t ticket) Less(o ticket) bool {
	return t.Deadline.Before(o.Deadline)
}

func flush(t *Scheduler) {
	heap := heap.Make[ticket](min_tickets)

	smallest := time.Duration(math.MaxInt64)
	var abs_after <-chan time.Time

	for {
		if heap.Len() > 0 {
			wait := time.Until(heap.Peek().Deadline)
			if wait <= smallest {
				abs_after = time.After(wait)
			}
		}

		after := abs_after
		if t.state == paused {
			after = nil
		}

		select {
		case <-after:
			ticket := heap.Pop()
			go ticket.Action()

		case ticket := <-t.push:
			heap.Push(ticket)

		case to := <-t.delta:
			switch to {
			case running, paused:
				t.state = to

			case stopped:
				push, delta := t.push, t.delta
				*t = Scheduler{state: stopped}

				clear_chan(push)
				clear_chan(delta)

				return
			}
		}
	}
}

func clear_chan[T any](ch <-chan T) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}
