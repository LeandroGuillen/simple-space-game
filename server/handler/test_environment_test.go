package server

import (
	"os"
	"testing"
)

// Handler tests must never read or write a developer's real autosaves.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "ssg-handler-saves-*")
	if err != nil {
		panic(err)
	}
	if err := os.Setenv("SSG_SAVES_DIR", dir); err != nil {
		panic(err)
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}
