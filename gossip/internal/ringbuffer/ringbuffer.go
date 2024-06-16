package ringbuffer

import (
	"container/ring"
	"errors"
)

var ErrNotPresent = errors.New("value not present")

type Ringbuffer[T comparable] struct {
	data *ring.Ring
	len uint
	cap uint
}

func NewRingbuffer[T comparable](capacity uint) *Ringbuffer[T] {
	return &Ringbuffer[T]{
		data: nil,
		len: 0,
		cap: capacity,
	}
}

func (r *Ringbuffer[T]) Insert(v T) {
	if(r.len == 0) {
		// create the ringbuffer
		r.data = ring.New(1)
		r.data.Value = v
	} else {
		if(r.len >= r.cap) {
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

func (r *Ringbuffer[T]) Remove(v T) error {
	if(r.len == 0) {
		return ErrNotPresent
	}
	start := r.data
	for i := r.data;; i = i.Next() {
		if i.Value.(T) == v {
			if r.data == i {
				i = i.Prev()
				i.Unlink(1)
				r.data = i.Next()
			} else {
				i.Prev().Unlink(1)
			}
			r.len--;
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
