package game_test

import (
	"ssg/game"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewGalaxy(t *testing.T) {
	g := game.NewGalaxy(1)
	assert.Equal(t, len(g.Stars), 1)
}

func TestNewEmptyGalaxy(t *testing.T) {
	galaxy := game.NewEmptyGalaxy()
	assert.Equal(t, len(galaxy.Stars), 0, "galaxy should be empty")
}
