package game

import . "ssg/game/model"

func Industry(p *Planet) int64 {
	// TODO add morale, tech, racial traits, etc. to this industry effort
	var industryBonus int64 = 0
	var flatIndustryBonus int64 = 0
	for _, building := range p.Buildings {
		for _, effect := range building.Effects {

			// bonuses per worker from buildings
			if effect.Name == "industryBonus" {
				industryBonus += effect.Bonus
			}

			// flat bonuses from buildings
			if effect.Name == "flatIndustryBonus" {
				flatIndustryBonus += effect.Bonus
			}
		}
	}

	totalIndustryPerWorker := industryBonus + IndustryPerMineralClass(p.MineralClass)
	totalFlatIndustry := flatIndustryBonus

	totalIndustry := totalFlatIndustry + totalIndustryPerWorker*p.Workers
	return totalIndustry
}
