/*
* gossip
* Copyright (C) 2024 Fabio Gaiba and Lukas Heindl
*
* This program is free software: you can redistribute it and/or modify
* it under the terms of the GNU General Public License as published by
* the Free Software Foundation, either version 3 of the License, or
* (at your option) any later version.
*
* This program is distributed in the hope that it will be useful,
* but WITHOUT ANY WARRANTY; without even the implied warranty of
* MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
* GNU General Public License for more details.
*
* You should have received a copy of the GNU General Public License
* along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

// Package ringbuffer implements a ringbuffer. It will have a fixed size and if
// the capacity is exceeded, the oldest value will be overwritten.
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

	if f(r.data.Value.(T)) {
		return r.data.Value.(T), nil
	}
	for i := r.data.Next(); i != r.data; i = i.Next() {
		if f(i.Value.(T)) {
			return i.Value.(T), nil
		}
	}

	return ret, ErrNotPresent
}

func (r *Ringbuffer[T]) ExtractToSlice() []T {
	ret := make([]T, 0)
	r.Do(func(x T) {
		ret = append(ret, x)
	})
	return ret
}
