package game

import (
	"os"
	"path/filepath"
	"runtime"
)

// DefaultCustomShipDesignsPath returns the on-disk path for custom-ship-designs.json next to resources.
func DefaultCustomShipDesignsPath() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return filepath.Join("game", "resources", "custom-ship-designs.json")
	}
	return filepath.Join(filepath.Dir(file), "resources", "custom-ship-designs.json")
}

// CustomShipDesignsPath returns SSG_CUSTOM_SHIP_DESIGNS_PATH when set, otherwise DefaultCustomShipDesignsPath().
func CustomShipDesignsPath() string {
	if p := os.Getenv("SSG_CUSTOM_SHIP_DESIGNS_PATH"); p != "" {
		return p
	}
	return DefaultCustomShipDesignsPath()
}
