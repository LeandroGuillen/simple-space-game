package game

import (
	. "ssg/model"
)

// shipRaceAttackBonus returns extra beam attack from race stats (+N% of base attack).
func shipRaceAttackBonus(player *Player, beamAttack int64) int64 {
	if player == nil || beamAttack == 0 {
		return 0
	}
	percent := player.Race.Stats.ShipAttackBonusPercent
	if percent == 0 {
		return 0
	}
	return beamAttack * percent / 100
}

// shipRaceDefenseBonus returns extra beam defense from race stats (+N% of base defense).
func shipRaceDefenseBonus(player *Player, beamDefense int64) int64 {
	if player == nil || beamDefense == 0 {
		return 0
	}
	percent := player.Race.Stats.ShipDefenseBonusPercent
	if percent == 0 {
		return 0
	}
	return beamDefense * percent / 100
}
