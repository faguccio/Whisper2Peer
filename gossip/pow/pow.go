package pow

import (
	"context"
	"crypto/sha256"
	"encoding"
	"io"
	"sync"

	"golang.org/x/exp/constraints"
)

const WORKERS = 32

type Marshaller[T constraints.Integer] interface {
	Marshal([]byte) ([]byte, error)
	Nonce() T
	SetNonce(T)
	AddToNonce(T)
	StripPrefixLen() uint
	PrefixLen() uint
	WriteNonce(io.Writer)
	Clone() Marshaller[T]
}

// implementation which is exposed to the outside
func ProofOfWork[T constraints.Integer](pred func(digest []byte) bool, e Marshaller[T]) T {
	return parallelProofOfWork3(pred, e)
}

// internal implementation 2
func parallelProofOfWork2[T constraints.Integer](pred func(digest []byte) bool, e Marshaller[T]) T {
	result := make(chan T)
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	for i := 0; i < WORKERS; i++ {
		wg.Add(1)
		// c,_ := context.WithCancel(ctx)
		go func(ctx context.Context, id T, e Marshaller[T]) {
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
func parallelProofOfWork3[T constraints.Integer](pred func(digest []byte) bool, e Marshaller[T]) T {
	result := make(chan T)
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	for i := 0; i < WORKERS; i++ {
		wg.Add(1)
		// c,_ := context.WithCancel(ctx)
		go func(ctx context.Context, id T, e Marshaller[T]) {
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
