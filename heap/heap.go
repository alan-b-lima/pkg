// Package heap implements the minheap abstract data
// structure using the functions and interface given by
// the [container/heap] package.
package heap

import (
	"container/heap"
	"fmt"
)

// Heap is an implementation of a minheap abstract data
// structure. Elements of this data structure must
// implement the [Lesser] interface on and for itself. The
// zero value of Heap is an empty heap.
//
// Heap is NOT safe for concurrent access by multiple
// goroutines.
type Heap[T Lesser[T]] struct {
	_heap _heap[T]
}

// Lesser is an interface that implements a ordering
// relation amongst the set of values of type T.
type Lesser[T any] interface {
	Less(T) bool
}

// Make preallocates memory for a heap of type T with the
// given size. If no size is given, it defaults to 0.
//
// Make panics if the size is negative, or if more than one
// argument is given.
func Make[T Lesser[T]](size ...int) Heap[T] {
	if len(size) == 0 {
		size = []int{0}
	}
	if size[0] < 0 {
		panic("heap: size cannot be negative")
	}
	if len(size) > 1 {
		panic("heap: too many arguments")
	}

	return Heap[T]{
		_heap: _heap[T]{make([]T, 0, size[0])},
	}
}

// Len returns the number of elements in the heap.
func (h *Heap[T]) Len() int {
	return h._heap.Len()
}

// Push inserts an element onto the heap.
//
// The complexity is O(log n) where n = h.Len().
func (h *Heap[T]) Push(v T) {
	heap.Push(&h._heap, v)
}

// PushMany inserts multiple elements onto the heap.
//
// The complexity is O(m + n) where m = len(v) and n = h.Len().
func (h *Heap[T]) PushMany(v ...T) {
	h._heap.ess = append(h._heap.ess, v...)
	heap.Init(&h._heap)
}

// Pop removes the smaller element, determined by [Lesser],
// of the heap, and returns it. If multiple elements have
// the same smallness, any of them may be returned.
//
// Pop panics if the heap is empty.
//
// The complexity is O(log n) where n = h.Len().
func (h *Heap[T]) Pop() T {
	if h.Len() == 0 {
		panic("heap: cannot pop from an empty heap")
	}

	return heap.Pop(&h._heap).(T)
}

// Peek returns the smaller element, determined by [Lesser],
// without removing it from the heap. If multiple elements
// have the same smallness, any of them may be returned.
//
// Peek panics if the heap is empty.
//
// The complexity is O(1).
func (h *Heap[T]) Peek() T {
	if h.Len() == 0 {
		panic("heap: cannot peek into an empty heap")
	}

	return h._heap.ess[0]
}

// Clear removes all elements from the heap.
//
// The complexity is O(n) where n = h.Len().
func (h *Heap[T]) Clear() {
	clear(h._heap.ess)
	h._heap.ess = h._heap.ess[:0]
}

// String implements the [fmt.Stringer] interface, it
// formats the heap as a slice.
func (h Heap[T]) String() string {
	return fmt.Sprint(h._heap.ess)
}

type _heap[T Lesser[T]] struct {
	ess []T
}

func (h *_heap[T]) Len() int           { return len(h.ess) }
func (h *_heap[T]) Swap(i, j int)      { h.ess[i], h.ess[j] = h.ess[j], h.ess[i] }
func (h *_heap[T]) Less(i, j int) bool { return h.ess[i].Less(h.ess[j]) }

func (h *_heap[T]) Push(v any) {
	h.ess = append(h.ess, v.(T))
}

func (h *_heap[T]) Pop() any {
	var zero T
	last := h.ess[len(h.ess)-1]
	h.ess[len(h.ess)-1] = zero

	h.ess = h.ess[:len(h.ess)-1]
	return last
}
