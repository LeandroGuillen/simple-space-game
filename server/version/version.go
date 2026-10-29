package version

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var versionString string

// Version returns the application version with surrounding whitespace trimmed.
func Version() string {
	return strings.TrimSpace(versionString)
}
