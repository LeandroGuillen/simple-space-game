package game

import (
	"os"
	"path/filepath"
	"runtime"
)

func DefaultSavesDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return filepath.Join("server", "data", "scenarios")
	}
	return filepath.Join(filepath.Dir(file), "..", "data", "scenarios")
}

func SavesDir() string {
	if p := os.Getenv("SSG_SAVES_DIR"); p != "" {
		return p
	}
	return DefaultSavesDir()
}
