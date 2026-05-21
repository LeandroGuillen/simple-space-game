package game

import "ssg/model"

// RaceDetectionModifier returns the race's detection range modifier (e.g. Stealthy Ships: −2).
func RaceDetectionModifier(player *model.Player) int64 {
	if player == nil {
		return 0
	}
	return player.Race.Stats.DetectionModifier
}

// RaceCombatSpeedBonus returns extra combat speed squares per turn from race traits.
func RaceCombatSpeedBonus(player *model.Player) int64 {
	if player == nil {
		return 0
	}
	return player.Race.Stats.CombatSpeedBonus
}
