package game

import (
	"path/filepath"
	"testing"
)

func TestHighScoreFilesAreSeparatePerMode(t *testing.T) {
	if highScoreFileName(true) == highScoreFileName(false) {
		t.Fatalf("Pac-Man and Ms. Pac-Man share high-score file %q", highScoreFileName(true))
	}

	dir := t.TempDir()
	pacPath := filepath.Join(dir, highScoreFileName(true))
	msPath := filepath.Join(dir, highScoreFileName(false))

	if err := SaveHighScore(pacPath, 5000); err != nil {
		t.Fatalf("save Pac-Man high score: %v", err)
	}
	if err := SaveHighScore(msPath, 7000); err != nil {
		t.Fatalf("save Ms. Pac-Man high score: %v", err)
	}
	if got := loadModeHighScore(pacPath, true); got != 5000 {
		t.Errorf("Pac-Man high score = %d, want 5000", got)
	}
	if got := loadModeHighScore(msPath, false); got != 7000 {
		t.Errorf("Ms. Pac-Man high score = %d, want 7000", got)
	}
}

func TestLegacySharedHighScoreSeedsOnlyMsPacman(t *testing.T) {
	dir := t.TempDir()
	if err := SaveHighScore(filepath.Join(dir, "highscore"), 9000); err != nil {
		t.Fatalf("save legacy high score: %v", err)
	}
	pacPath := filepath.Join(dir, highScoreFileName(true))
	msPath := filepath.Join(dir, highScoreFileName(false))

	if got := loadModeHighScore(pacPath, true); got != 0 {
		t.Errorf("Pac-Man high score with only legacy file = %d, want 0", got)
	}
	if got := loadModeHighScore(msPath, false); got != 9000 {
		t.Errorf("Ms. Pac-Man high score with only legacy file = %d, want 9000", got)
	}

	// Once a mode-specific file exists, the legacy file is ignored.
	if err := SaveHighScore(msPath, 100); err != nil {
		t.Fatalf("save Ms. Pac-Man high score: %v", err)
	}
	if got := loadModeHighScore(msPath, false); got != 100 {
		t.Errorf("Ms. Pac-Man high score after split = %d, want 100", got)
	}
}
