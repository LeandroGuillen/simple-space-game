package game

import (
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/assert"
)

func TestPlanetMoraleBonusStacksBuildingsAndTechs(t *testing.T) {
	player := &model.Player{
		ID:              1,
		ResearchedTechs: []string{"Virtual Reality Network"},
	}
	g := &Game{
		Players: []*model.Player{player},
		Resources: &Resources{
			BuildingCollection: &BuildingCollection{
				Buildings: []model.Building{
					{
						Name: "Holo Simulator",
						Grants: model.Grants{Modifiers: []model.Modifier{
							{Name: "morale_bonus", Value: 0.2},
						}},
					},
				},
			},
			TechTree: &model.TechTree{
				AllTechs: []model.Tech{
					{
						Name: "Virtual Reality Network",
						Grants: model.Grants{Modifiers: []model.Modifier{
							{Name: "morale_bonus", Value: 0.2},
						}},
					},
				},
			},
		},
	}
	planet := &model.Planet{
		Owner:     1,
		Buildings: []string{"Holo Simulator"},
	}

	hydratePlanetMorale(g, planet, player)

	assert.InDelta(t, 0.4, planet.Morale.Bonus, 0.0001)
	assert.InDelta(t, 1.4, planet.Morale.Multiplier, 0.0001)
	assert.Equal(t, []model.PlanetMoraleSource{
		{Name: "Holo Simulator", Type: "building", Bonus: 0.2},
		{Name: "Virtual Reality Network", Type: "tech", Bonus: 0.2},
	}, planet.Morale.Sources)
}

func TestPlanetMoraleDefaultsToNoBonus(t *testing.T) {
	planet := &model.Planet{}

	hydratePlanetMorale(&Game{Resources: &Resources{BuildingCollection: &BuildingCollection{}}}, planet, nil)

	assert.Equal(t, 0.0, planet.Morale.Bonus)
	assert.Equal(t, 1.0, planet.Morale.Multiplier)
	assert.Empty(t, planet.Morale.Sources)
}

func TestPlanetMoraleSourcesIncludeNetZeroEffects(t *testing.T) {
	g := &Game{
		Resources: &Resources{
			BuildingCollection: &BuildingCollection{
				Buildings: []model.Building{
					{
						Name: "Holo Simulator",
						Grants: model.Grants{Modifiers: []model.Modifier{
							{Name: "morale_bonus", Value: 0.2},
						}},
					},
					{
						Name: "Penalty",
						Grants: model.Grants{Modifiers: []model.Modifier{
							{Name: "morale_bonus", Value: -0.2},
						}},
					},
				},
			},
		},
	}
	planet := &model.Planet{Buildings: []string{"Holo Simulator", "Penalty"}}

	hydratePlanetMorale(g, planet, nil)

	assert.InDelta(t, 0.0, planet.Morale.Bonus, 0.0001)
	assert.Equal(t, 1.0, planet.Morale.Multiplier)
	assert.Equal(t, []model.PlanetMoraleSource{
		{Name: "Holo Simulator", Type: "building", Bonus: 0.2},
		{Name: "Penalty", Type: "building", Bonus: -0.2},
	}, planet.Morale.Sources)
}
