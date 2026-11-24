package config_test

import (
	"testing"

	"cheddarhead/internal/config"
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

func TestGhostScoreProgression(t *testing.T) {
	tests := []struct {
		eatenCount int
		want       int
	}{
		{-1, 0},
		{0, 0},
		{1, 200},
		{2, 400},
		{3, 800},
		{4, 1600},
		{5, 1600},
	}

	for _, tt := range tests {
		if got := config.GhostScore(tt.eatenCount); got != tt.want {
			t.Errorf("GhostScore(%d) = %d, want %d", tt.eatenCount, got, tt.want)
		}
	}
}

func TestIntermissionForLevel(t *testing.T) {
	tests := []struct {
		level      int
		pacmanMode bool
		want       int
	}{
		{1, true, 0},
		{2, true, 1},
		{3, true, 0},
		{4, true, 0},
		{5, true, 2},
		{6, true, 0},
		{9, true, 3},
		{13, true, 3},
		{17, true, 3},
		{21, true, 0},
		{1, false, 0},
		{2, false, 1},
		{5, false, 2},
		{9, false, 3},
		{13, false, 3},
		{17, false, 3},
		{21, false, 3},
		{22, false, 0},
		{25, false, 3},
	}

	for _, tt := range tests {
		if got := config.IntermissionForLevel(tt.level, tt.pacmanMode); got != tt.want {
			t.Errorf("IntermissionForLevel(%d, pacman=%v) = %d, want %d", tt.level, tt.pacmanMode, got, tt.want)
		}
	}
}

func TestFixedStepClockPacing(t *testing.T) {
	clock := config.NewArcadeClock()

	// Over 1.0 second at 120 Hz render frames, should emit ~60 ticks (60.606061 Hz).
	total := 0
	for range 120 {
		total += clock.Advance(1.0 / 120.0)
	}
	if total != 60 {
		t.Fatalf("ticks over 1s at 120Hz = %d, want 60", total)
	}

	// Over 10.0 seconds total at 60 Hz render frames, should emit 606 ticks.
	clock60 := config.NewArcadeClock()
	total60 := 0
	for range 600 {
		total60 += clock60.Advance(1.0 / 60.0)
	}
	if total60 != 606 {
		t.Fatalf("ticks over 10s at 60Hz = %d, want 606", total60)
	}

	// Huge pause delta is clamped to maxDelta (0.25s -> 15 ticks).
	clockClamp := config.NewArcadeClock()
	if got := clockClamp.Advance(5.0); got != 15 {
		t.Fatalf("clamped 5s pause ticks = %d, want 15", got)
	}

	// Non-positive deltas emit 0 steps.
	if got := clockClamp.Advance(-0.1); got != 0 {
		t.Fatalf("negative delta ticks = %d, want 0", got)
	}
}
