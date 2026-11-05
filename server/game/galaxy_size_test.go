package game

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeGalaxySize(t *testing.T) {
	cases := []struct {
		size string
		want string
		ok   bool
	}{
		{"", DefaultGalaxySize, true},
		{"tiny", "tiny", true},
		{"medium", "medium", true},
		{"huge", "huge", true},
		{"TINY", "tiny", true},
		{" medium ", "medium", true},
		{"massive", "", false},
		{"small", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.size, func(t *testing.T) {
			got, ok := NormalizeGalaxySize(tc.size)
			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestBuiltinGalaxySizesMatchKnownSizes(t *testing.T) {
	assert.Equal(t, []string{"tiny", "medium", "huge"}, BuiltinGalaxySizes)
	for _, size := range BuiltinGalaxySizes {
		normalized, ok := NormalizeGalaxySize(size)
		assert.True(t, ok, size)
		assert.Equal(t, size, normalized)
	}
}
