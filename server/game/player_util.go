package game

import (
	. "ssg/model"
)

var defaultBuildings = []string{
	"Capitol",
	"Colony Base",
	"Star Base",
	"Marine Barracks",
}

func NewPlayer(ID uint, playerName, playerColor string) *Player {
	return &Player{
		ID:    ID,
		Name:  playerName,
		Color: playerColor,
		Available: AvailableTech{
			Buildings: defaultBuildings,
		},
	}
}
