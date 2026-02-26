package main

import "testing"

func TestBitIsSet(t *testing.T) {
	tests := []struct {
		n    uint32
		bit  int
		want bool
	}{
		{0, 0, false},
		{1, 0, true},
		{1, 1, false},
		{2, 1, true},
		{2, 0, false},
		{0x80000000, 31, true},
		{0x80000000, 30, false},
		{0xFFFFFFFF, 0, true},
		{0xFFFFFFFF, 31, true},
	}

	for _, tt := range tests {
		if got := bitIsSet(tt.n, tt.bit); got != tt.want {
			t.Errorf("bitIsSet(%v, %v) = %v, want %v", tt.n, tt.bit, got, tt.want)
		}
	}
}
