package game

import (
	. "ssg/model"
)

const startingColonyInfrastructureTech = "Starting Colony Infrastructure"

func NewPlayer(ID uint, playerName, playerColor string, race Race) *Player {
	return &Player{
		ID:              ID,
		Name:            playerName,
		Color:           playerColor,
		Race:            race,
		ResearchedTechs: []string{startingColonyInfrastructureTech},
		HyperAdvanced:   map[string]uint64{},
		Treasury:        10,
	}
}
