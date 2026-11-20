package config_test

import (
	"testing"

	"pacfan/internal/config"
)

func TestGetLevelSpecRanges(t *testing.T) {
	tests := []struct {
		level      int
		wantSymbol config.BonusSymbol
		wantSpeed  uint32
		wantFright int
	}{
		{1, config.BonusCherries, config.Speed80, config.SecondsToFrames(6)},
		{2, config.BonusStrawberry, config.Speed90, config.SecondsToFrames(5)},
		{7, config.BonusGrapes, config.Speed100, config.SecondsToFrames(2)},
		{8, config.BonusGrapes, config.Speed100, config.SecondsToFrames(2)},
		{17, config.BonusKey, config.Speed100, config.FrightTimeNA},
		{21, config.BonusKey, config.Speed90, config.FrightTimeNA},
		{255, config.BonusKey, config.Speed90, config.FrightTimeNA},
	}

	for _, tt := range tests {
		spec := config.GetLevelSpec(tt.level)
		if spec.BonusSymbol != tt.wantSymbol {
			t.Errorf("level %d BonusSymbol = %v, want %v", tt.level, spec.BonusSymbol, tt.wantSymbol)
		}
		if spec.PacmanSpeed != tt.wantSpeed {
			t.Errorf("level %d PacmanSpeed = %#x, want %#x", tt.level, spec.PacmanSpeed, tt.wantSpeed)
		}
		if spec.FrightenedFrames != tt.wantFright {
			t.Errorf("level %d FrightenedFrames = %d, want %d", tt.level, spec.FrightenedFrames, tt.wantFright)
		}
	}
}

func TestGhostWaveStateTransitions(t *testing.T) {
	scatterEnd := config.SecondsToFrames(7)
	if got := config.GetGhostWaveState(1, 1); got != config.Scatter {
		t.Fatalf("frame 1 = %v, want Scatter", got)
	}
	if got := config.GetGhostWaveState(1, scatterEnd); got != config.Scatter {
		t.Fatalf("frame %d = %v, want Scatter", scatterEnd, got)
	}
	if got := config.GetGhostWaveState(1, scatterEnd+1); got != config.Chase {
		t.Fatalf("frame %d = %v, want Chase", scatterEnd+1, got)
	}
}
