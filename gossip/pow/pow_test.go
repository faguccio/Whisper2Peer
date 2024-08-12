package pow

import (
	"crypto/sha256"
	"math/rand"
	"testing"
)

func BenchmarkPoW1(b *testing.B) {
	randomSource_ := rand.NewSource(1337).(rand.Source64)
	randomSource := rand.New(randomSource_)

	for i:= 0; i < b.N; i++ {
		x := TM{
			header: uint16(randomSource.Int()),
			data:   [16]byte{0xc0, 0xff, 0xfe},
			nonce:  0,
		}

		print("1\n")
		x.nonce = parallelProofOfWork2(func(digest []byte) bool {
			return digest[0] == 0 && digest[1] == 0 && digest[2] == 0
		}, &x)
		print("2\n")

		buf,_ := x.Marshal(nil)
		digest := sha256.Sum256(buf)
		if !first24bits0(digest[:]) {
			b.Fatalf("Returned nonce does not fulfull the predicate")
		}
	}
}

func BenchmarkPoW2(b *testing.B) {
	randomSource_ := rand.NewSource(1337).(rand.Source64)
	randomSource := rand.New(randomSource_)

	for i:= 0; i < b.N; i++ {
		x := TM{
			header: uint16(randomSource.Int()),
			data:   [16]byte{},
			nonce:  0,
		}

		x.nonce = parallelProofOfWork3(func(digest []byte) bool {
			return digest[0] == 0 && digest[1] == 0 && digest[2] == 0
		}, &x)

		buf,_ := x.Marshal(nil)
		digest := sha256.Sum256(buf)
		if !first24bits0(digest[:]) {
			b.Fatalf("Returned nonce does not fulfull the predicate")
		}
	}
}

func first24bits0(digest []byte) bool {
	return digest[0] == 0 && digest[1] == 0 && digest[2] == 0
}
