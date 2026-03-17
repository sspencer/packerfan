package main

import "math/rand"

const romSize = 8192

var rom [romSize]byte
var prngIndex uint = 0 // Global PRNG state (shared by all ghosts)

// Advance PRNG and get next "random" byte from simulated ROM
func getPrngByte() byte {
	prngIndex = (prngIndex*5 + 1) % romSize
	return rom[prngIndex]
}

func initPrng() {
	// Seed for reproducibility (simulating fixed ROM patterns)
	r := rand.New(rand.NewSource(42))

	// Fill simulated ROM with pseudo-random bytes
	for i := range rom {
		rom[i] = byte(r.Intn(256))
	}
}

// Reset PRNG for a new "level"
func resetPrng() {
	prngIndex = 0
}

// Get preferred direction from lowest 2 bits of the PRNG byte
func getPreferredDirection() Direction {
	b := getPrngByte()
	bits := b & 0x03 // lowest 2 bits: 0=UP, 1=RIGHT, 2=DOWN, 3=LEFT
	switch bits {
	case 0:
		return Up
	case 1:
		return Right
	case 2:
		return Down
	case 3:
		return Left
	default:
		return Up // unreachable
	}
}

// Clockwise next direction: UP -> RIGHT -> DOWN -> LEFT
func clockwiseNext(d Direction) Direction {
	switch d {
	case Up:
		return Right
	case Right:
		return Down
	case Down:
		return Left
	case Left:
		return Up
	default:
		return Up
	}
}
