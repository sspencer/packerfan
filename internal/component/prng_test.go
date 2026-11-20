package component_test

import (
	"bytes"
	"testing"

	"pacfan/internal/component"
)

func TestPRNGSeedDeterminismAndDivergence(t *testing.T) {
	pDefault := component.NewPRNGData()
	var pSeedDefault component.PRNGData
	pSeedDefault.InitWithSeed(component.DefaultPRNGSeed)

	if pDefault.ROM != pSeedDefault.ROM {
		t.Fatalf("Init() ROM differs from InitWithSeed(DefaultPRNGSeed)")
	}

	var pA, pB, pOther component.PRNGData
	pA.InitWithSeed(12345)
	pB.InitWithSeed(12345)
	pOther.InitWithSeed(54321)

	seqA := make([]byte, 256)
	seqB := make([]byte, 256)
	seqOther := make([]byte, 256)
	for i := range seqA {
		seqA[i] = pA.NextByte()
		seqB[i] = pB.NextByte()
		seqOther[i] = pOther.NextByte()
	}

	if !bytes.Equal(seqA, seqB) {
		t.Fatalf("same seed produced different byte sequences")
	}
	if bytes.Equal(seqA, seqOther) {
		t.Fatalf("different seeds produced identical byte sequences")
	}

	// Reset rewinds the sequence for the current ROM seed.
	pA.Reset()
	for i, want := range seqA {
		if got := pA.NextByte(); got != want {
			t.Fatalf("after Reset(), byte %d = %d, want %d", i, got, want)
		}
	}
}
