package game

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdvanceStardateUsesTenths(t *testing.T) {
	assert.Equal(t, 3500.1, advanceStardate(3500.0))
	assert.Equal(t, 3500.9, advanceStardate(3500.8))
	assert.Equal(t, 3501.0, advanceStardate(3500.9))
}
