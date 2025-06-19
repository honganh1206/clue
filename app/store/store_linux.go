package store

import (
	"os"
	"path/filepath"
)

func getStorePath() string {
	home := os.Getenv("HOME")
	return filepath.Join(home, ".local", ".clue", "config.json")
}
