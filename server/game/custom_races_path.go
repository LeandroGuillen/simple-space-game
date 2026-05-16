package game

import (
	"os"
	"path/filepath"
	"runtime"
)

// DefaultCustomRacesPath returns the on-disk path for custom-races.json next to this package's resources folder.
func DefaultCustomRacesPath() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return filepath.Join("game", "resources", "custom-races.json")
	}
	return filepath.Join(filepath.Dir(file), "resources", "custom-races.json")
}

// CustomRacesPath returns SSG_CUSTOM_RACES_PATH when set, otherwise DefaultCustomRacesPath().
func CustomRacesPath() string {
	if p := os.Getenv("SSG_CUSTOM_RACES_PATH"); p != "" {
		return p
	}
	return DefaultCustomRacesPath()
}
