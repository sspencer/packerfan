package game

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// DefaultHighScorePath returns the user-scoped file path used to persist the
// high score across runs, or "" if the config directory cannot be resolved.
func DefaultHighScorePath() string {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		return ""
	}
	return filepath.Join(dir, "pacfan", "highscore")
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
