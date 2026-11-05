package game

import (
	"errors"
	"fmt"
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// invalidInputError is implemented by errors that mark setup values the caller
// can fix. The API layer detects them with errors.As to answer 400 instead of
// 500.
type invalidInputError interface {
	error
	IsInvalidInput() bool
}

func TestNewGameMarksCallerFixableSetupErrors(t *testing.T) {
	resources := testResources(t)
	cases := []struct {
		name    string
		options NewGameOptions
		want    string
	}{
		{
			name: "invalid player color",
			options: NewGameOptions{
				PlayerName:  "Ada",
				NumStars:    8,
				PlayerColor: "neon_pink",
				Race:        DefaultRace("Human"),
			},
			want: `invalid player color "neon_pink"`,
		},
		{
			name: "more players than stars",
			options: NewGameOptions{
				PlayerName: "Ada",
				NumStars:   2,
				NumPlayers: 5,
				Race:       DefaultRace("Human"),
			},
			want: "players (5) cannot exceed stars (2)",
		},
		{
			name: "more players than allowed",
			options: NewGameOptions{
				PlayerName: "Ada",
				NumStars:   40,
				NumPlayers: model.MaxPlayers + 1,
				Race:       DefaultRace("Human"),
			},
			want: fmt.Sprintf("players must be between 1 and %d", model.MaxPlayers),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g, err := NewGame(resources, tc.options)
			require.Error(t, err)
			assert.Nil(t, g)
			assert.EqualError(t, err, tc.want)

			var invalid invalidInputError
			assert.True(t, errors.As(err, &invalid), "error should be marked as invalid input: %v", err)
		})
	}
}
