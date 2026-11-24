package game_test

import (
	"path/filepath"
	"strings"
	"testing"

	"cheddarhead/assets"
	"cheddarhead/internal/component"
	"cheddarhead/internal/game"
	"cheddarhead/internal/sim"
)

func TestReplayPlayerVerifiesFixturesAndControls(t *testing.T) {
	fixturesDir := filepath.Join("..", "sim", "testdata")

	player, err := game.NewReplayPlayer(assets.FS, fixturesDir, "", 4, 1.0)
	if err != nil {
		t.Fatalf("NewReplayPlayer(%s): %v", fixturesDir, err)
	}
	if player.FileCount() < 3 {
		t.Fatalf("FileCount() = %d, want >= 3", player.FileCount())
	}

	// Verify Zoom is set from argument without breaking HashWorld checkpoints.
	if got := component.MustFindGameState(player.Env().World()).Zoom; got != 4 {
		t.Fatalf("Zoom = %v, want 4", got)
	}

	// Pause + single tick.
	player.TogglePause()
	if !player.Paused() {
		t.Fatalf("expected Paused() = true")
	}
	player.Advance(1.0)
	if player.Env().Tick() != 0 {
		t.Fatalf("Advance while paused moved tick to %d", player.Env().Tick())
	}
	if !player.StepTick() || player.Env().Tick() != 1 {
		t.Fatalf("StepTick failed: tick=%d", player.Env().Tick())
	}

	// Speed controls clamp to [0.25, 16].
	for range 10 {
		player.Faster()
	}
	if player.Speed() != game.MaxReplaySpeed {
		t.Fatalf("Speed after Faster = %v, want %v", player.Speed(), game.MaxReplaySpeed)
	}
	for range 10 {
		player.Slower()
	}
	if player.Speed() != game.MinReplaySpeed {
		t.Fatalf("Speed after Slower = %v, want %v", player.Speed(), game.MinReplaySpeed)
	}

	// Play every fixture in the playlist to completion with zero hash errors.
	player.TogglePause()
	for i := range player.FileCount() {
		if err := player.LoadIndex(i); err != nil {
			t.Fatalf("LoadIndex(%d): %v", i, err)
		}
		for !player.Finished() {
			player.StepTick()
			if err := player.HashError(); err != nil {
				t.Fatalf("fixture %s hash error: %v", player.FilePath(), err)
			}
		}
		if player.CheckpointsOK() == 0 {
			t.Fatalf("fixture %s checked 0 checkpoints", player.FilePath())
		}
		hud := player.HUDLine()
		if !strings.Contains(hud, "DONE") {
			t.Fatalf("HUDLine %q missing DONE", hud)
		}
	}

	// PrevFile / NextFile wrap cleanly.
	if err := player.NextFile(); err != nil {
		t.Fatalf("NextFile: %v", err)
	}
	if err := player.PrevFile(); err != nil {
		t.Fatalf("PrevFile: %v", err)
	}
}

func TestResolveReplayArtRejectsMazeHashMismatch(t *testing.T) {
	bad := &sim.Replay{
		Header: sim.ReplayHeader{
			PFR:      sim.ReplayFormatVersion,
			Art:      sim.ArtInfo{File: "game.png"},
			MazeHash: "deadbeef00000000000000000000000000000000000000000000000000000000",
		},
	}
	_, err := game.ResolveReplayArt(assets.FS, "", bad)
	if err == nil {
		t.Fatalf("expected error on maze_hash mismatch, got nil")
	}
	if !strings.Contains(err.Error(), "maze_hash mismatch") || !strings.Contains(err.Error(), "game.png") {
		t.Fatalf("unexpected error message: %v", err)
	}
}
