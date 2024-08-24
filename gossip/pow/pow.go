package pow

import (
	"context"
	"crypto/sha256"
	"encoding"
	"io"
	"sync"

	"golang.org/x/exp/constraints"
)

// defines how many worker goroutines are used per proof of work invocation
const WORKERS = 32

// Interface which all types must implement over which a PoW should be
// calculated over. The generic parameter T is the type of the nonce which is
// adjusted to fulfil the predicate.
//
// NOTE: The (marshalled) struct must have the following structure:
//
//	| PREFIX_a | PREFIX_b | NONCE |
//	prefix_a is not involved in the PoW computation
//	prefix_b are the remaining data
//	nonce is the number (of generic type T) which is incremented to
//	  fulfil the predicate
type POWMarshaller[T constraints.Integer] interface {
	// marshal the (whole) struct to a bytes slice
	Marshal([]byte) ([]byte, error)
	// obtain the nonce of the struct
	Nonce() T
	// set the nonce of the struct
	SetNonce(T)
	// increment the nonce
	AddToNonce(T)
	// how many bytes in the beginning of the bytes slice (from marshal) should
	// be skipped (and not be included in the PoW)
	StripPrefixLen() uint
	// remaining length of the struct minus the skipped part and minus the
	// length of the nonce
	PrefixLen() uint
	// write the nonce to an arbitrary writer (used to write the nonce to the
	// hash object)
	WriteNonce(io.Writer)
	// must return an exact copy of the struct
	Clone() POWMarshaller[T]
}

// implementation which is exposed to the outside
func ProofOfWork[T constraints.Integer](pred func(digest []byte) bool, e POWMarshaller[T]) T {
	return parallelProofOfWork3(pred, e)
}

// internal implementation 2
// always hashes the complete byte slice
func parallelProofOfWork2[T constraints.Integer](pred func(digest []byte) bool, e POWMarshaller[T]) T {
	result := make(chan T)
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	for i := 0; i < WORKERS; i++ {
		wg.Add(1)
		// c,_ := context.WithCancel(ctx)
		go func(ctx context.Context, id T, e POWMarshaller[T]) {
			h := sha256.New()
			digest := [sha256.Size]byte{}

			m := make([]byte, 0)
			m, _ = e.Marshal(m)
			m = m[e.StripPrefixLen():]
			m1 := m[:e.PrefixLen()]

		loop:
			for e.SetNonce(id); true; e.AddToNonce(WORKERS) {
				select {
				case <-ctx.Done():
					break loop
				default:
					h.Reset()
					h.Write(m1)
					e.WriteNonce(h)
					if pred(h.Sum(digest[:0])) {
						select {
						case result <- e.Nonce():
						default:
						}
						break loop
					}
				}
			}
			wg.Done()
		}(ctx, T(i), e.Clone())
	}
	r := <-result
	cancel()
	wg.Wait()
	return r
}

// internal implementation 3
// makes use of length extension and reuses the previous state
func parallelProofOfWork3[T constraints.Integer](pred func(digest []byte) bool, e POWMarshaller[T]) T {
	result := make(chan T)
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	for i := 0; i < WORKERS; i++ {
		wg.Add(1)
		// c,_ := context.WithCancel(ctx)
		go func(ctx context.Context, id T, e POWMarshaller[T]) {
			h := sha256.New()
			digest := [sha256.Size]byte{}

			m := make([]byte, 0)
			m, _ = e.Marshal(m)
			m = m[e.StripPrefixLen():]
			m1 := m[:e.PrefixLen()]

			h.Reset()
			h.Write(m1)
			hm := h.(encoding.BinaryMarshaler)
			hu := h.(encoding.BinaryUnmarshaler)
			stat, _ := hm.MarshalBinary()
		loop:
			for e.SetNonce(id); true; e.AddToNonce(WORKERS) {
				select {
				case <-ctx.Done():
					break loop
				default:
					_ = hu.UnmarshalBinary(stat)
					e.WriteNonce(h)
					if pred(h.Sum(digest[:0])) {

						select {
						case result <- e.Nonce():
						default:
						}
						break loop
					}
				}
			}
			wg.Done()
		}(ctx, T(i), e.Clone())
	}
	r := <-result
	cancel()
	wg.Wait()
	return r
}

// predicate which checks if the first 24 bits of a slice are 0
func First24bits0(digest []byte) bool {
	return digest[0] == 0 && digest[1] == 0 && digest[2] == 0
}

// predicate which checks if the first 8 bits of a slice are 0
func First8bits0(digest []byte) bool {
	return digest[0] == 0
}

// check weather the input has a valid proof of work for the predicate
func CheckProofOfWork[T constraints.Integer](pred func(digest []byte) bool, e POWMarshaller[T]) bool {
	nonce := e.Nonce()
	e.SetNonce(0)
	h := sha256.New()
	digest := [sha256.Size]byte{}
	m := make([]byte, 0)
	m, _ = e.Marshal(m)
	m = m[e.StripPrefixLen():]
	m1 := m[:e.PrefixLen()]

	e.SetNonce(nonce)
	h.Reset()
	h.Write(m1)
	e.WriteNonce(h)

	return pred(h.Sum(digest[:0]))
}
