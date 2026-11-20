package game_test

import (
	"os"
	"path/filepath"
	"testing"

	"pacfan/internal/game"
)

func TestHighScorePersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "highscore")

	// Missing file returns 0 gracefully.
	if got := game.LoadHighScore(path); got != 0 {
		t.Fatalf("missing file high score = %d, want 0", got)
	}

	// Saving and loading round-trips cleanly.
	if err := game.SaveHighScore(path, 12340); err != nil {
		t.Fatalf("SaveHighScore failed: %v", err)
	}
	if got := game.LoadHighScore(path); got != 12340 {
		t.Fatalf("loaded high score = %d, want 12340", got)
	}

	// Corrupt file degrades gracefully to 0.
	if err := os.WriteFile(path, []byte("not-a-number\n"), 0o644); err != nil {
		t.Fatalf("write corrupt file: %v", err)
	}
	if got := game.LoadHighScore(path); got != 0 {
		t.Fatalf("corrupt file high score = %d, want 0", got)
	}
}
