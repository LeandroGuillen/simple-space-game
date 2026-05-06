package game_test

import (
	"ssg/game"
	"ssg/model"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPollution_IgnoresCleanFlatProduction(t *testing.T) {
	planet := &model.Planet{
		Size:         model.PlanetSizeMedium,
		MineralClass: model.PlanetMineralAbundant,
		Industry: model.PlanetIndustryOutput{
			Workers:             10,
			ProducedPerWorker:   4,
			ProducedPerBuilding: 5,
		},
		Buildings: []model.Building{
			{
				Name: "Automated Factory",
				Effects: []model.Effect{
					{Name: "industryBonus", Bonus: 1},
					{Name: "flatIndustryBonus", Bonus: 5},
				},
			},
		},
	}

	assert.Equal(t, int64(45), game.GrossIndustry(planet))
	assert.Equal(t, int64(17), game.Pollution(planet))
	assert.Equal(t, int64(28), game.Industry(planet))
}

func TestPollution_UsesCleanupBuildingsAndPlanetSize(t *testing.T) {
	planet := &model.Planet{
		Size: model.PlanetSizeTiny,
		Industry: model.PlanetIndustryOutput{
			Workers:           12,
			ProducedPerWorker: 4,
		},
	}

	assert.Equal(t, int64(23), game.Pollution(planet))

	planet.Buildings = []model.Building{{Name: "Pollution Processor"}}
	assert.Equal(t, int64(11), game.Pollution(planet))

	planet.Buildings = []model.Building{{Name: "Atmospheric Renewer"}}
	assert.Equal(t, int64(5), game.Pollution(planet))

	planet.Buildings = []model.Building{{Name: "Pollution Processor"}, {Name: "Atmospheric Renewer"}}
	assert.Equal(t, int64(2), game.Pollution(planet))

	planet.Buildings = append(planet.Buildings, model.Building{Name: "Nano Disassemblers"})
	assert.Equal(t, int64(1), game.Pollution(planet))
}
