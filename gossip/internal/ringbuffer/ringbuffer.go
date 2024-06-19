package ringbuffer

import (
	"container/ring"
	"errors"
)

var ErrNotPresent = errors.New("value not present")

// This struct represents a ringbuffer with a given capacity.
//
// If the capacity is exceeded, the oldest entry will be deleted.
// Internally this uses the [container/ring] implementation of golang (so a linked circular list).
//
// The struct contains various internal fields, thus it should only be created
// by using the [NewRingbuffer] function!
type Ringbuffer[T comparable] struct {
	// store the actual data contained in the buffer
	data *ring.Ring
	// amount of values stored in the buffer
	len uint
	// total amount of values that can be stored in the buffer
	cap uint
}

// Use this function to instantiate the ringbuffer
func NewRingbuffer[T comparable](capacity uint) *Ringbuffer[T] {
	return &Ringbuffer[T]{
		data: nil,
		len:  0,
		cap:  capacity,
	}
}

// Insert a value into the ringbuffer.
//
// If this insert exceeds the capacity, the oldest element will (silently) be overwritten
func (r *Ringbuffer[T]) Insert(v T) {
	if r.len == 0 {
		// create the ringbuffer
		r.data = ring.New(1)
		r.data.Value = v
	} else {
		if r.len >= r.cap {
			// remove the next element
			r.data.Unlink(1)
			r.len--
		}
		// plain insert
		x := ring.New(1)
		x.Value = v
		r.data.Link(x)
		r.data = x
	}
	r.len++
}

// Remove a value from the ringbuffer.
//
// This will throw a [ErrNotPresent] error if the value is not contained in the ringbuffer.
func (r *Ringbuffer[T]) Remove(v T) error {
	if r.len == 0 {
		return ErrNotPresent
	}
	start := r.data
	for i := r.data; ; i = i.Next() {
		if i.Value.(T) == v {
			if r.data == i {
				i = i.Prev()
				i.Unlink(1)
				r.data = i.Next()
			} else {
				i.Prev().Unlink(1)
			}
			r.len--
			return nil
		}
		if i.Next() == start {
			break
		}
	}
	return ErrNotPresent
}

// Do calls function f on each element of the ringbuffer in forward order.
// The behavior of Do is undefined if f changes *r.
func (r *Ringbuffer[T]) Do(f func(T)) {
	if r.len == 0 {
		return
	}

	f(r.data.Value.(T))
	for i := r.data.Next(); i != r.data; i = i.Next() {
		f(i.Value.(T))
	}
}

// Filter out calls a function f on each element, expected to return a boolean.
// A slice of all element wich the return value was true is returned
func (r *Ringbuffer[T]) Filter(f func(T) bool) []T {
	result := make([]T, 0)

	if r.len == 0 {
		return result
	}

	if f(r.data.Value.(T)) {
		result = append(result, r.data.Value.(T))
	}
	for i := r.data.Next(); i != r.data; i = i.Next() {
		if f(i.Value.(T)) {
			result = append(result, i.Value.(T))
		}
	}
	return result
}

// FindFirs returns the first occurance on which the funciton f returns true
func (r *Ringbuffer[T]) FindFirst(f func(T) bool) (T, error) {
	var ret T

	if r.len == 0 {
		return ret, ErrNotPresent
	}

	for i := r.data.Next(); i != r.data; i = i.Next() {
		if f(i.Value.(T)) {
			return i.Value.(T), nil
		}
	}

	return r.data.Value.(T), ErrNotPresent
}
