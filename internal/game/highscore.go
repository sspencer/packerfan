package game

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// DefaultHighScorePath returns the user-scoped file path used to persist the
// high score across runs, or "" if the config directory cannot be resolved.
// Pac-Man and Ms. Pac-Man keep separate high scores in separate files.
func DefaultHighScorePath(pacmanMode bool) string {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		return ""
	}
	return filepath.Join(dir, "pacfan", highScoreFileName(pacmanMode))
}

func highScoreFileName(pacmanMode bool) string {
	if pacmanMode {
		return "highscore-pacman"
	}
	return "highscore-mspacman"
}

// legacyHighScorePath returns the pre-split shared high-score file that sits
// next to path. Ms. Pac-Man was the default mode when the shared file was in
// use, so only the Ms. Pac-Man score falls back to it.
func legacyHighScorePath(path string, pacmanMode bool) string {
	if path == "" || pacmanMode {
		return ""
	}
	return filepath.Join(filepath.Dir(path), "highscore")
}

// loadModeHighScore reads the high score at path, falling back to the legacy
// shared file for Ms. Pac-Man until a mode-specific file has been written.
func loadModeHighScore(path string, pacmanMode bool) int {
	if _, err := os.Stat(path); err != nil {
		if legacy := legacyHighScorePath(path, pacmanMode); legacy != "" {
			return LoadHighScore(legacy)
		}
	}
	return LoadHighScore(path)
}

// LoadHighScore reads the persisted integer high score from path.
// Missing or unreadable files gracefully return 0.
func LoadHighScore(path string) int {
	if path == "" {
		return 0
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	v, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || v < 0 {
		return 0
	}
	return v
}

// SaveHighScore writes score atomically to path, creating parent directories as needed.
func SaveHighScore(path string, score int) error {
	if path == "" || score <= 0 {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strconv.Itoa(score)+"\n"), 0o644)
}
