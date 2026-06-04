package game

import (
	"math"

	. "ssg/model"
)

// baseFoodPerFarmer returns the food a single farmer produces before any race,
// tech, or building modifiers, based on how fertile the planet's environment is.
// Hostile worlds (Toxic, Barren, Radiated) cannot grow food on their own, so
// farmers there start at zero until empire-wide breakthroughs such as Biomorphic
// Fungi make farming viable.
//
// The values are data-driven from resources/planet-environments.json; the
// hardcoded defaults are a fallback for games constructed without loaded
// resources (e.g. unit tests).
func baseFoodPerFarmer(game *Game, env PlanetEnvironment) int64 {
	if game != nil && game.Resources != nil {
		if value, ok := game.Resources.EnvironmentCollection.baseFoodPerFarmer(env); ok {
			return value
		}
	}
	return defaultBaseFoodPerFarmer(env)
}

func defaultBaseFoodPerFarmer(env PlanetEnvironment) int64 {
	switch env {
	case PlanetEnvironmentDesert, PlanetEnvironmentTundra, PlanetEnvironmentArid:
		return 1
	case PlanetEnvironmentOcean, PlanetEnvironmentSwamp, PlanetEnvironmentTerran:
		return 2
	case PlanetEnvironmentGaia:
		return 3
	default:
		// Toxic, Barren, Radiated, and uncolonizable environments: no farming.
		return 0
	}
}

func hydratePlanetOutputs(game *Game, p *Planet) {
	foodPerFarmer := baseFoodPerFarmer(game, p.Environment)
	foodFromBuildings := int64(0)
	industryPerWorker := IndustryPerMineralClass(p.MineralClass)
	industryFromBuildings := int64(0)
	industryPerColonist := int64(0)
	sciencePerScientist := int64(3)
	scienceFromBuildings := int64(0)

	for _, buildingName := range p.Buildings {
		if building, err := getBuildingFromName(game, buildingName); err == nil {
			for _, modifier := range building.Grants.Modifiers {
				switch modifier.Name {
				case "food_per_farmer":
					foodPerFarmer += int64(modifier.Value)
				case "food_flat":
					foodFromBuildings += int64(modifier.Value)
				case "industry_per_worker":
					industryPerWorker += int64(modifier.Value)
				case "industry_flat":
					industryFromBuildings += int64(modifier.Value)
				case "industry_per_colonist":
					industryPerColonist += int64(modifier.Value)
				case "science_per_scientist":
					sciencePerScientist += int64(modifier.Value)
				case "science_flat":
					scienceFromBuildings += int64(modifier.Value)
				}
			}
		}
	}
	if game != nil {
		if player, err := game.getPlayer(p.Owner); err == nil {
			for _, modifier := range player.Race.Grants.Modifiers {
				switch modifier.Name {
				case "food_per_farmer":
					foodPerFarmer += int64(modifier.Value)
				case "industry_per_worker":
					industryPerWorker += int64(modifier.Value)
				case "science_per_scientist":
					sciencePerScientist += int64(modifier.Value)
				}
			}
			for _, techName := range player.ResearchedTechs {
				tech, ok := game.findTechDefinition(techName)
				if !ok {
					continue
				}
				for _, modifier := range tech.Grants.Modifiers {
					switch modifier.Name {
					case "food_per_farmer":
						foodPerFarmer += int64(modifier.Value)
					case "industry_per_worker":
						industryPerWorker += int64(modifier.Value)
					case "science_per_scientist":
						sciencePerScientist += int64(modifier.Value)
					}
				}
			}
		}
	}
	sciencePerScientist += p.HomeworldScienceBonus
	p.GravityMultiplier = gravityOutputMultiplier(p, gamePlayerForPlanet(game, p))
	foodPerFarmer = maxInt64(0, foodPerFarmer)
	industryPerWorker = maxInt64(0, industryPerWorker)
	sciencePerScientist = maxInt64(0, sciencePerScientist)

	p.Food = PlanetFoodOutput{
		Farmers:             p.Food.Farmers,
		ProducedPerFarmer:   foodPerFarmer,
		ProducedPerBuilding: foodFromBuildings,
	}

	p.Industry = PlanetIndustryOutput{
		Workers:             p.Industry.Workers,
		ProducedPerWorker:   industryPerWorker,
		ProducedPerBuilding: industryFromBuildings,
		ProducedPerColonist: industryPerColonist,
	}

	p.Science = PlanetScienceOutput{
		Scientists:           p.Science.Scientists,
		ProducedPerScientist: sciencePerScientist,
		ProducedPerBuilding:  scienceFromBuildings,
	}

	hydratePlanetPollution(game, p)
}

func PlanetGrossIndustry(p *Planet) int64 {
	return moraleAdjustedLaborOutput(p.Industry.Workers, p.Industry.ProducedPerWorker, effectiveMoraleMultiplier(p.Morale)) +
		p.Industry.ProducedPerBuilding +
		colonistUnits(p)*p.Industry.ProducedPerColonist
}

func GamePlanetGrossIndustry(game *Game, p *Planet) int64 {
	player := gamePlayerForPlanet(game, p)
	return governmentAdjustedLaborOutput(
		p.Industry.Workers,
		p.Industry.ProducedPerWorker,
		governmentLaborMultiplier(game, player, "industry_labor_multiplier"),
		governmentMoraleMultiplier(game, player, p.Morale),
		gravityOutputMultiplier(p, player),
	) + p.Industry.ProducedPerBuilding + colonistUnits(p)*p.Industry.ProducedPerColonist
}

func PlanetIndustryProduced(game *Game, p *Planet) int64 {
	return PlanetGrossIndustry(p) - Pollution(game, p)
}

func GamePlanetIndustryProduced(game *Game, p *Planet) int64 {
	return GamePlanetGrossIndustry(game, p) - Pollution(game, p)
}

func PlanetFoodProduced(p *Planet) int64 {
	return moraleAdjustedLaborOutput(p.Food.Farmers, p.Food.ProducedPerFarmer, effectiveMoraleMultiplier(p.Morale)) +
		p.Food.ProducedPerBuilding
}

func GamePlanetFoodProduced(game *Game, p *Planet) int64 {
	player := gamePlayerForPlanet(game, p)
	return governmentAdjustedLaborOutput(
		p.Food.Farmers,
		p.Food.ProducedPerFarmer,
		governmentLaborMultiplier(game, player, "food_labor_multiplier"),
		governmentMoraleMultiplier(game, player, p.Morale),
		gravityOutputMultiplier(p, player),
	) + p.Food.ProducedPerBuilding
}

func PlanetScienceProduced(p *Planet) int64 {
	return moraleAdjustedLaborOutput(p.Science.Scientists, p.Science.ProducedPerScientist, effectiveMoraleMultiplier(p.Morale)) +
		p.Science.ProducedPerBuilding
}

func GamePlanetScienceProduced(game *Game, p *Planet) int64 {
	player := gamePlayerForPlanet(game, p)
	return governmentAdjustedLaborOutput(
		p.Science.Scientists,
		p.Science.ProducedPerScientist,
		governmentLaborMultiplier(game, player, "science_labor_multiplier"),
		governmentMoraleMultiplier(game, player, p.Morale),
		gravityOutputMultiplier(p, player),
	) + p.Science.ProducedPerBuilding
}

func gamePlayerForPlanet(game *Game, p *Planet) *Player {
	if game == nil || p == nil {
		return nil
	}
	player, err := game.getPlayer(p.Owner)
	if err != nil {
		return nil
	}
	return player
}

func governmentMoraleMultiplier(game *Game, player *Player, morale PlanetMoraleOutput) float64 {
	if governmentMoraleNeutral(game, player) {
		return 1
	}
	return effectiveMoraleMultiplier(morale)
}

func colonistUnits(p *Planet) int64 {
	if p.Population <= 0 {
		return 0
	}
	return p.Population / 1000
}

func moraleAdjustedLaborOutput(workers, outputPerWorker int64, multiplier float64) int64 {
	if multiplier == 0 {
		multiplier = 1
	}
	if multiplier < 0 {
		multiplier = 0
	}
	return int64(math.Floor(float64(workers*outputPerWorker) * multiplier))
}

func governmentAdjustedLaborOutput(workers, outputPerWorker int64, governmentMultiplier, moraleMultiplier, gravityMultiplier float64) int64 {
	if governmentMultiplier == 0 {
		governmentMultiplier = 1
	}
	if governmentMultiplier < 0 {
		governmentMultiplier = 0
	}
	if moraleMultiplier == 0 {
		moraleMultiplier = 1
	}
	if moraleMultiplier < 0 {
		moraleMultiplier = 0
	}
	if gravityMultiplier == 0 {
		gravityMultiplier = 1
	}
	if gravityMultiplier < 0 {
		gravityMultiplier = 0
	}
	return int64(math.Floor(float64(workers*outputPerWorker) * governmentMultiplier * moraleMultiplier * gravityMultiplier))
}
