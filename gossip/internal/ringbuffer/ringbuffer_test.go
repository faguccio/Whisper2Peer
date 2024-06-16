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
