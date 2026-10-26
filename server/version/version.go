package version

import _ "embed"

//go:embed VERSION
var versionString string

// Version returns the application version.
func Version() string {
	return versionString
}
