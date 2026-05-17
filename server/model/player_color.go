package model

import "slices"

const MaxPlayers = 8

var PlayerColors = []string{
	"#FFFFFF", // white
	"#FF3B30", // red
	"#007AFF", // blue
	"#34C759", // green
	"#FFCC00", // yellow
	"#AF52DE", // purple
	"#FF9500", // orange
	"#FF2D55", // pink
}

func IsValidPlayerColor(color string) bool {
	return slices.Contains(PlayerColors, color)
}

func DefaultPlayerColor() string {
	return PlayerColors[0]
}
