package audio_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/yohamta/donburi"

	"cheddarhead/internal/archetype"
	"cheddarhead/internal/audio"
	"cheddarhead/internal/component"
	"cheddarhead/internal/config"
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
		for _, rel := range audio.LoopFirstPaths(loop) {
			if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
				t.Errorf("missing firstloop WAV %s for loop=%v: %v", rel, loop, err)
			}
		}
	}
}

func TestMissingAndHasAnyAssetsVerification(t *testing.T) {
	emptyFS := fstest.MapFS{}
	if audio.HasAnyAssets(emptyFS, false) || audio.HasAnyAssets(emptyFS, true) {
		t.Fatalf("expected HasAnyAssets=false for empty FS")
	}
	if missing := audio.MissingAssets(emptyFS, false); len(missing) == 0 {
		t.Fatalf("expected MissingAssets to report missing files for empty FS")
	}

	// Zero-byte files and directories must not count as valid audio files.
	badFS := fstest.MapFS{
		"assets/ms-pacman/start.wav": {Data: nil},
		"assets/ms-pacman/death.wav": {Mode: fs.ModeDir},
	}
	if audio.HasAnyAssets(badFS, false) {
		t.Fatalf("expected HasAnyAssets=false for empty file / directory entries")
	}

	// Populate every primary Ms. Pac-Man sound + loop -> 0 missing.
	fullFS := fstest.MapFS{}
	for _, id := range audio.AllSounds {
		fullFS[audio.SoundPaths(false, id)[0]] = &fstest.MapFile{Data: []byte("RIFF")}
	}
	for _, loop := range audio.AllLoops {
		fullFS[audio.LoopPaths(loop)[0]] = &fstest.MapFile{Data: []byte("RIFF")}
	}
	if !audio.HasAnyAssets(fullFS, false) {
		t.Fatalf("expected HasAnyAssets=true for populated FS")
	}
	if missing := audio.MissingAssets(fullFS, false); len(missing) != 0 {
		t.Fatalf("expected no missing assets for populated FS, got %v", missing)
	}

	// Removing one file reports only that file as missing.
	delete(fullFS, "assets/ms-pacman/death.wav")
	missing := audio.MissingAssets(fullFS, false)
	if len(missing) != 1 || missing[0] != "assets/ms-pacman/death.wav" {
		t.Fatalf("MissingAssets = %v, want [assets/ms-pacman/death.wav]", missing)
	}
}
