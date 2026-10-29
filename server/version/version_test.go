package version

import (
	"os"
	"strings"
	"testing"
)

func TestVersionMatchesVERSIONFile(t *testing.T) {
	content, err := os.ReadFile("../../VERSION")
	if err != nil {
		t.Fatalf("failed to read VERSION file: %v", err)
	}
	expected := strings.TrimSpace(string(content))

	got := Version()

	if got == "" {
		t.Fatal("Version() returned empty string")
	}

	if got != expected {
		t.Errorf("Version() = %q, want %q (from VERSION file)", got, expected)
	}
}

func TestVersionHasNoSurroundingWhitespace(t *testing.T) {
	got := Version()
	if got != strings.TrimSpace(got) {
		t.Fatalf("Version() has leading/trailing whitespace: %q", got)
	}
	if strings.ContainsAny(got, "\n\r\t") {
		t.Fatalf("Version() contains whitespace characters: %q", got)
	}
}
