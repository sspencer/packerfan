package audio_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yohamta/donburi"

	"pacfan/internal/archetype"
	"pacfan/internal/audio"
	"pacfan/internal/component"
	"pacfan/internal/config"
)

func TestSirenLoopEscalatesAsDotsDecrease(t *testing.T) {
	tests := []struct {
		rem, total int
		want       audio.LoopID
	}{
		{240, 244, audio.LoopSiren0},
		{180, 244, audio.LoopSiren1},
		{100, 244, audio.LoopSiren2},
		{40, 244, audio.LoopSiren3},
		{10, 244, audio.LoopSiren4},
		{0, 0, audio.LoopSiren0},
	}
	for _, tt := range tests {
		if got := audio.SirenLoopForDots(tt.rem, tt.total); got != tt.want {
			t.Errorf("SirenLoopForDots(%d, %d) = %v, want %v", tt.rem, tt.total, got, tt.want)
		}
	}
}

func TestSelectBackgroundLoopPriority(t *testing.T) {
	w := donburi.NewWorld()
	archetype.CreateGameState(w, true, 3)
	archetype.CreateMaze(w, true)
	archetype.CreatePRNG(w)
	ghosts := archetype.SpawnAllGhosts(w)

	gs := component.MustFindGameState(w)
	gs.DotsEaten = 200
	gs.DotsRemaining = 44 // ~18% -> Siren3

	if got := audio.SelectBackgroundLoop(w); got != audio.LoopSiren3 {
		t.Fatalf("normal play loop = %v, want LoopSiren3", got)
	}

	// Frightened mode overrides siren.
	gs.IsFrightMode = true
	if got := audio.SelectBackgroundLoop(w); got != audio.LoopFright {
		t.Fatalf("fright mode loop = %v, want LoopFright", got)
	}

	// Retreating eyes override frightened mode.
	component.Ghost.Get(ghosts[0]).State = config.Eaten
	if got := audio.SelectBackgroundLoop(w); got != audio.LoopEyes {
		t.Fatalf("retreating eyes loop = %v, want LoopEyes", got)
	}

	// Paused silences background loop.
	gs.Paused = true
	if got := audio.SelectBackgroundLoop(w); got != audio.LoopNone {
		t.Fatalf("paused loop = %v, want LoopNone", got)
	}
}

func TestAudioAssetPathsPresentIfLocalWAVsExist(t *testing.T) {
	// Skip when running in a clean repo checkout without gitignored WAV files.
	if _, err := os.Stat(filepath.Join("..", "..", "assets", "ms-pacman", "start.wav")); err != nil {
		t.Skipf("local WAV assets not present: %v", err)
	}

	root := filepath.Join("..", "..")
	for _, pacmanMode := range []bool{true, false} {
		for _, id := range []audio.SoundID{
			audio.SoundStart, audio.SoundCredit, audio.SoundChomp,
			audio.SoundEatGhost, audio.SoundEatFruit, audio.SoundExtraLife,
			audio.SoundDeath, audio.SoundIntermission1, audio.SoundIntermission2, audio.SoundIntermission3,
		} {
			found := false
			for _, rel := range audio.SoundPaths(pacmanMode, id) {
				if _, err := os.Stat(filepath.Join(root, rel)); err == nil {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("no existing WAV found for pacmanMode=%v soundID=%v (candidates: %v)",
					pacmanMode, id, audio.SoundPaths(pacmanMode, id))
			}
		}
	}
	for _, loop := range []audio.LoopID{
		audio.LoopSiren0, audio.LoopSiren1, audio.LoopSiren2,
		audio.LoopSiren3, audio.LoopSiren4, audio.LoopFright, audio.LoopEyes,
	} {
		found := false
		for _, rel := range audio.LoopPaths(loop) {
			if _, err := os.Stat(filepath.Join(root, rel)); err == nil {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("no existing WAV found for loop=%v (candidates: %v)", loop, audio.LoopPaths(loop))
		}
	}
}
