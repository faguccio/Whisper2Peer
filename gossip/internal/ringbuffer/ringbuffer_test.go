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

package ringbuffer_test

import (
	"gossip/internal/ringbuffer"
	"reflect"
	"testing"
)

func extractToSlice[T comparable](rb *ringbuffer.Ringbuffer[T]) []T {
	ret := make([]T, 0)
	rb.Do(func(x T) {
		ret = append(ret, x)
	})
	return ret
}

func TestRingbuffer(t *testing.T) {
	rb := ringbuffer.NewRingbuffer[int](3)
	should := []int{}
	is := extractToSlice(rb)
	if !reflect.DeepEqual(is, should) {
		t.Fatalf("After creation ringbuffer should be %v but is %v", should, is)
	}

	rb.Insert(0)
	should = []int{0}
	is = extractToSlice(rb)
	if !reflect.DeepEqual(is, should) {
		t.Fatalf("Ringbuffer should be %v but is %v", should, is)
	}

	rb.Insert(1)
	should = []int{1, 0}
	is = extractToSlice(rb)
	if !reflect.DeepEqual(is, should) {
		t.Fatalf("Ringbuffer should be %v but is %v", should, is)
	}

	rb.Insert(2)
	should = []int{2, 0, 1}
	is = extractToSlice(rb)
	if !reflect.DeepEqual(is, should) {
		t.Fatalf("Ringbuffer should be %v but is %v", should, is)
	}

	rb.Insert(3)
	should = []int{3, 1, 2}
	is = extractToSlice(rb)
	if !reflect.DeepEqual(is, should) {
		t.Fatalf("Ringbuffer should be %v but is %v", should, is)
	}

	err := rb.Remove(0)
	if err != ringbuffer.ErrNotPresent {
		t.Fatalf("Removing 0 in state %v should have returned %v but was %v", is, ringbuffer.ErrNotPresent, err)
	}

	err = rb.Remove(2)
	if err != nil {
		t.Fatalf("Removing 2 in state %v should have returned an error but was %v", is, err)
	}
	should = []int{3, 1}
	is = extractToSlice(rb)
	if !reflect.DeepEqual(is, should) {
		t.Fatalf("Ringbuffer should be %v but is %v", should, is)
	}

	rb.Insert(4)
	should = []int{4, 1, 3}
	is = extractToSlice(rb)
	if !reflect.DeepEqual(is, should) {
		t.Fatalf("Ringbuffer should be %v but is %v", should, is)
	}

	err = rb.Remove(4)
	if err != nil {
		t.Fatalf("Removing 4 in state %v should have returned an error but was %v", is, err)
	}
	should = []int{1, 3}
	is = extractToSlice(rb)
	if !reflect.DeepEqual(is, should) {
		t.Fatalf("Ringbuffer should be %v but is %v", should, is)
	}

	err = rb.Remove(3)
	if err != nil {
		t.Fatalf("Removing 3 in state %v should have returned an error but was %v", is, err)
	}
	should = []int{1}
	is = extractToSlice(rb)
	if !reflect.DeepEqual(is, should) {
		t.Fatalf("Ringbuffer should be %v but is %v", should, is)
	}

	err = rb.Remove(1)
	if err != nil {
		t.Fatalf("Removing 1 in state %v should have returned an error but was %v", is, err)
	}
	should = []int{}
	is = extractToSlice(rb)
	if !reflect.DeepEqual(is, should) {
		t.Fatalf("Ringbuffer should be %v but is %v", should, is)
	}
}

func TestFilter(t *testing.T) {
	rb := ringbuffer.NewRingbuffer[int](6)

	for i := 1; i < 8; i++ {
		rb.Insert(i)
	}

	res := rb.Filter(func(a int) bool {
		return a%2 == 0
	})

	should := []int{2, 4, 6}

	if !reflect.DeepEqual(res, should) {
		t.Fatalf("After filter application ringbuffer should be %v but is %v", should, res)
	}
}

func TestFindFirst(t *testing.T) {
	rb := ringbuffer.NewRingbuffer[int](30)

	for i := 1; i < 30; i++ {
		rb.Insert(i)
	}

	res, _ := rb.FindFirst(func(a int) bool {
		return a == 29
	})

	should := 29

	if !reflect.DeepEqual(res, should) {
		t.Fatalf("Find first should have found %v but found %v", should, res)
	}

	res, _ = rb.FindFirst(func(a int) bool {
		return a == 23
	})

	should = 23

	if !reflect.DeepEqual(res, should) {
		t.Fatalf("Find first should have found %v but found %v", should, res)
	}

	_, err := rb.FindFirst(func(a int) bool {
		return a == 50
	})

	if err == nil {
		t.Fatalf("Found first should have found nothing")
	}
}
