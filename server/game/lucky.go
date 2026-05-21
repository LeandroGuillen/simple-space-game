package game

import (
	. "ssg/model"
)

const (
	luckyEventBonus          = 0.25
	luckyResearchBonusChance = 0.10
	luckyResearchBonusPoints = uint64(1)
)

// LuckyEventBonus returns the extra probability weight for beneficial random events (v1 stub).
func LuckyEventBonus(player *Player) float64 {
	if playerHasRaceTrait(player, "lucky") {
		return luckyEventBonus
	}
	return 0
}

// luckyTurnResearchBonus rolls for a small research windfall at turn end (v1 until random events exist).
func luckyTurnResearchBonus(player *Player) uint64 {
	if player == nil || LuckyEventBonus(player) == 0 {
		return 0
	}
	if randomFloat64() < luckyResearchBonusChance+LuckyEventBonus(player) {
		return luckyResearchBonusPoints
	}
	return 0
}
