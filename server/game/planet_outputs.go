package game

import (
	"fmt"
	"strings"

	"ssg/model"
)

func hydratePlanetOutputs(p *model.Planet, owner *model.Player) {
	p.Industry = buildIndustryOutput(p)
	p.Food = buildFoodOutput(p, owner)
	p.Science = buildScienceOutput(p)
}

func buildIndustryOutput(p *model.Planet) model.PlanetIndustryOutput {
	basePerWorker := model.IndustryPerMineralClass(p.MineralClass)
	out := model.PlanetIndustryOutput{
		Workers:       p.Industry.Workers,
		BasePerWorker: basePerWorker,
		Contributors: []model.PlanetOutputContributor{
			{
				ID:         "planet:mineral_class",
				SourceType: "planet",
				SourceID:   strings.ToLower(p.MineralClass.String()),
				Label:      fmt.Sprintf("%s minerals", p.MineralClass.String()),
				Effects: []model.PlanetOutputEffect{
					{Kind: "base_per_worker", Value: float64(basePerWorker)},
				},
			},
		},
	}

	perWorker, flat, percent, fromBuildings := gatherOutputBonuses(
		p.Buildings,
		[]string{"industryBonus"},
		[]string{"flatIndustryBonus"},
		[]string{"industryPercentBonus"},
		"worker",
	)
	out.PerWorkerBonusTotal = perWorker
	out.FlatBonusTotal = flat
	out.PercentBonusTotal = percent
	out.Contributors = append(out.Contributors, fromBuildings...)
	out.Total = totalFromOutput(
		out.Workers,
		out.BasePerWorker,
		out.PerWorkerBonusTotal,
		out.FlatBonusTotal,
		out.PercentBonusTotal,
	)
	return out
}

func buildFoodOutput(p *model.Planet, owner *model.Player) model.PlanetFoodOutput {
	out := model.PlanetFoodOutput{
		Farmers:       p.Food.Farmers,
		BasePerFarmer: 1,
		Contributors: []model.PlanetOutputContributor{
			{
				ID:         "planet:base_food",
				SourceType: "planet",
				SourceID:   "food:base",
				Label:      "Base food output",
				Effects: []model.PlanetOutputEffect{
					{Kind: "base_per_farmer", Value: 1},
				},
			},
		},
	}

	perFarmer, flat, percent, fromBuildings := gatherOutputBonuses(
		p.Buildings,
		[]string{"farmingBonus", "agricultureBonus", "foodBonus"},
		[]string{"flatFarmingBonus", "flatAgricultureBonus", "flatFoodBonus"},
		[]string{"farmingPercentBonus", "agriculturePercentBonus", "foodPercentBonus"},
		"farmer",
	)
	out.PerFarmerBonusTotal = perFarmer
	out.FlatBonusTotal = flat
	out.PercentBonusTotal = percent
	out.Contributors = append(out.Contributors, fromBuildings...)

	if owner != nil && owner.Stats.Farming != 0 {
		raceBonus := owner.Stats.Farming
		out.PerFarmerBonusTotal += raceBonus
		out.Contributors = append(out.Contributors, model.PlanetOutputContributor{
			ID:         fmt.Sprintf("race:%d:farming", owner.ID),
			SourceType: "race",
			SourceID:   fmt.Sprintf("race:%d", owner.ID),
			Label:      "Race farming bonus",
			Effects: []model.PlanetOutputEffect{
				{Kind: "per_farmer_bonus", Value: float64(raceBonus)},
			},
		})
	}

	out.Total = totalFromOutput(
		out.Farmers,
		out.BasePerFarmer,
		out.PerFarmerBonusTotal,
		out.FlatBonusTotal,
		out.PercentBonusTotal,
	)
	return out
}

func buildScienceOutput(p *model.Planet) model.PlanetScienceOutput {
	out := model.PlanetScienceOutput{
		Scientists:       p.Science.Scientists,
		BasePerScientist: 1,
		Contributors: []model.PlanetOutputContributor{
			{
				ID:         "planet:base_science",
				SourceType: "planet",
				SourceID:   "science:base",
				Label:      "Base science output",
				Effects: []model.PlanetOutputEffect{
					{Kind: "base_per_scientist", Value: 1},
				},
			},
		},
	}

	perScientist, flat, percent, fromBuildings := gatherOutputBonuses(
		p.Buildings,
		[]string{"scienceBonus", "researchBonus"},
		[]string{"flatScienceBonus", "flatResearchBonus"},
		[]string{"sciencePercentBonus", "researchPercentBonus"},
		"scientist",
	)
	out.PerScientistBonusTotal = perScientist
	out.FlatBonusTotal = flat
	out.PercentBonusTotal = percent
	out.Contributors = append(out.Contributors, fromBuildings...)
	out.Total = totalFromOutput(
		out.Scientists,
		out.BasePerScientist,
		out.PerScientistBonusTotal,
		out.FlatBonusTotal,
		out.PercentBonusTotal,
	)
	return out
}

func gatherOutputBonuses(
	buildings []model.Building,
	perUnitNames []string,
	flatNames []string,
	percentNames []string,
	unitName string,
) (int64, int64, float64, []model.PlanetOutputContributor) {
	perUnitSet := make(map[string]bool, len(perUnitNames))
	flatSet := make(map[string]bool, len(flatNames))
	percentSet := make(map[string]bool, len(percentNames))
	for _, name := range perUnitNames {
		perUnitSet[name] = true
	}
	for _, name := range flatNames {
		flatSet[name] = true
	}
	for _, name := range percentNames {
		percentSet[name] = true
	}

	var perUnit int64
	var flat int64
	var percent float64
	contributors := make([]model.PlanetOutputContributor, 0)

	for _, building := range buildings {
		effects := make([]model.PlanetOutputEffect, 0)
		for _, effect := range building.Effects {
			switch {
			case perUnitSet[effect.Name]:
				perUnit += effect.Bonus
				effects = append(effects, model.PlanetOutputEffect{
					Kind:  fmt.Sprintf("per_%s_bonus", unitName),
					Value: float64(effect.Bonus),
				})
			case flatSet[effect.Name]:
				flat += effect.Bonus
				effects = append(effects, model.PlanetOutputEffect{
					Kind:  "flat_bonus",
					Value: float64(effect.Bonus),
				})
			case percentSet[effect.Name]:
				percent += float64(effect.Bonus) / 100.0
				effects = append(effects, model.PlanetOutputEffect{
					Kind:  "percent_bonus",
					Value: float64(effect.Bonus) / 100.0,
				})
			}
		}
		if len(effects) > 0 {
			contributors = append(contributors, model.PlanetOutputContributor{
				ID:         fmt.Sprintf("building:%d", building.ID),
				SourceType: "building",
				SourceID:   fmt.Sprintf("%d", building.ID),
				Label:      building.Name,
				Effects:    effects,
			})
		}
	}

	return perUnit, flat, percent, contributors
}

func totalFromOutput(units, basePerUnit, perUnitBonus, flatBonus int64, percentBonus float64) int64 {
	subtotal := float64(units*(basePerUnit+perUnitBonus) + flatBonus)
	return int64(subtotal * (1 + percentBonus))
}
