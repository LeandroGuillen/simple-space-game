package builder_test

import (
	"ssg/game/builder"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewGalaxy(t *testing.T) {
	g := builder.NewGalaxy(1)
	assert.Equal(t, len(g.Stars), 1)
}

func TestNewEmptyGalaxy(t *testing.T) {
	galaxy := builder.NewEmptyGalaxy()
	assert.Equal(t, len(galaxy.Stars), 0, "galaxy should be empty")
}
