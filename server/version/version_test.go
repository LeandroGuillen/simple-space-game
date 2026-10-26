package version

import (
	"os"
	"strings"
	"testing"
)

func TestVersionMatchesVERSIONFile(t *testing.T) {
	// Read the root VERSION file
	content, err := os.ReadFile("../../VERSION")
	if err != nil {
		t.Fatalf("failed to read VERSION file: %v", err)
	}
	expected := strings.TrimSpace(string(content))

	got := strings.TrimSpace(Version())

	if got == "" {
		t.Fatal("Version() returned empty string")
	}

	if got != expected {
		t.Errorf("Version() = %q, want %q (from VERSION file)", got, expected)
	}
}
