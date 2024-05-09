package main

import (
	"math/rand"
	"testing"
)

func BenchmarkMainParallel2(b *testing.B) {
	randomSource := rand.NewSource(1337).(rand.Source64)
	var enrollRegGen EnrollRegister
	enrollRegGen.FillFromIni(CONFIG_FILE)
	// calculate size + add space for \r\n
	enrollRegGen.Size = uint16(enrollRegGen.CalcSize()) + 2*3

	for i:= 0; i < b.N; i++ {
		// make a copy
		var enrollReg EnrollRegister = enrollRegGen
		enrollReg.Challenge = randomSource.Uint64()

		enrollReg.Nonce = parallelProofOfWork2(first24bits0, enrollReg)
	}
}

// func BenchmarkMainParallel(b *testing.B) {
// 	randomSource := rand.NewSource(1337).(rand.Source64)
// 	var enrollRegGen EnrollRegister
// 	enrollRegGen.FillFromIni(CONFIG_FILE)
// 	// calculate size + add space for \r\n
// 	enrollRegGen.Size = uint16(enrollRegGen.CalcSize()) + 2*3
//
// 	for i := 0; i < b.N; i++ {
// 		// make a copy
// 		var enrollReg EnrollRegister = enrollRegGen
// 		enrollReg.Challenge = randomSource.Uint64()
//
// 		enrollReg.Nonce = parallelProofOfWork(first24bits0, enrollReg)
// 	}
// }

func BenchmarkMain(b *testing.B) {
	randomSource := rand.NewSource(1337).(rand.Source64)
	var enrollRegGen EnrollRegister
	enrollRegGen.FillFromIni(CONFIG_FILE)
	// calculate size + add space for \r\n
	enrollRegGen.Size = uint16(enrollRegGen.CalcSize()) + 2*3

	for i:= 0; i < b.N; i++ {
		// make a copy
		var enrollReg EnrollRegister = enrollRegGen
		enrollReg.Challenge = randomSource.Uint64()

		enrollReg.Nonce = sequentialProofOfWork(first24bits0, enrollReg)
	}
}
