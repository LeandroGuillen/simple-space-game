package game

import (
	. "ssg/model"
)

const startingColonyInfrastructureTech = "Starting Colony Infrastructure"

func NewPlayer(ID uint, playerName, playerColor string) *Player {
	return &Player{
		ID:              ID,
		Name:            playerName,
		Color:           playerColor,
		ResearchedTechs: []string{startingColonyInfrastructureTech},
		Treasury:        10,
	}
}
