package game

import (
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArtificialPlanetBuildRequiresTech(t *testing.T) {
	g, source, _ := newArtificialPlanetGame(t, model.PlanetEnvironmentGasGiant)

	assert.False(t, Build(source, artificialPlanetProject, g.Players[0].Available.Buildings, g))
	assert.False(t, source.IsBuilding)
}

func TestArtificialPlanetBuildRequiresSameSystemTarget(t *testing.T) {
	g, source, target := newArtificialPlanetGame(t, model.PlanetEnvironmentGasGiant)
	g.Players[0].Available.Buildings = append(g.Players[0].Available.Buildings, artificialPlanetProject)
	for _, planet := range g.Galaxy.Stars[0].Planets {
		if planet.Owner == 0 {
			planet.Environment = model.PlanetEnvironmentTerran
		}
	}
	target.Environment = model.PlanetEnvironmentTerran

	assert.False(t, Build(source, artificialPlanetProject, g.Players[0].Available.Buildings, g))
	assert.False(t, source.IsBuilding)
}

func TestArtificialPlanetBuildStartsWhenTargetExists(t *testing.T) {
	g, source, _ := newArtificialPlanetGame(t, model.PlanetEnvironmentGasGiant)
	g.Players[0].Available.Buildings = append(g.Players[0].Available.Buildings, artificialPlanetProject)

	require.True(t, Build(source, artificialPlanetProject, g.Players[0].Available.Buildings, g))

	assert.True(t, source.IsBuilding)
	assert.Equal(t, artificialPlanetProject, source.CurrentProject)
	assert.Equal(t, int64(800), source.BuildBucket)
}

func TestArtificialPlanetCompletionCreatesPendingChoice(t *testing.T) {
	g, source, _ := newArtificialPlanetGame(t, model.PlanetEnvironmentGasGiant)
	g.Players[0].Available.Buildings = append(g.Players[0].Available.Buildings, artificialPlanetProject)
	source.CurrentProject = artificialPlanetProject
	source.IsBuilding = true
	source.BuildBucket = 1

	require.NoError(t, g.Turn(1, model.Commands{}))

	assert.Contains(t, g.PendingArtificialPlanets[1], source.ID)
	assert.NotContains(t, source.Buildings, artificialPlanetProject)
}

func TestArtificialPlanetCompletionLosesProductionWhenNoTargetRemains(t *testing.T) {
	g, source, target := newArtificialPlanetGame(t, model.PlanetEnvironmentGasGiant)
	g.Players[0].Available.Buildings = append(g.Players[0].Available.Buildings, artificialPlanetProject)
	source.CurrentProject = artificialPlanetProject
	source.IsBuilding = true
	source.BuildBucket = 1
	for _, planet := range g.Galaxy.Stars[0].Planets {
		if planet.Owner == 0 {
			planet.Environment = model.PlanetEnvironmentTerran
		}
	}
	target.Environment = model.PlanetEnvironmentTerran

	require.NoError(t, g.Turn(1, model.Commands{}))

	assert.Empty(t, g.PendingArtificialPlanets[1])
	assert.Equal(t, model.PlanetEnvironmentTerran, target.Environment)
}

func TestArtificialPlanetCommandTransformsGasGiant(t *testing.T) {
	g, source, target := newArtificialPlanetGame(t, model.PlanetEnvironmentGasGiant)
	g.PendingArtificialPlanets[1] = []uint{source.ID}

	require.NoError(t, g.Turn(1, model.Commands{
		ArtificialPlanet: &model.ArtificialPlanetCommand{
			SourcePlanetID: source.ID,
			TargetPlanetID: target.ID,
		},
	}))

	assert.Equal(t, model.PlanetEnvironmentBarren, target.Environment)
	assert.Equal(t, model.PlanetSizeHuge, target.Size)
	assert.Equal(t, model.PlanetMineralAbundant, target.MineralClass)
	assert.Equal(t, model.PlanetGravityNormal, target.Gravity)
	assert.Equal(t, uint(0), target.Owner)
	assert.Empty(t, g.PendingArtificialPlanets[1])
}

func TestArtificialPlanetCommandTransformsAsteroidBelt(t *testing.T) {
	g, source, target := newArtificialPlanetGame(t, model.PlanetEnvironmentAsteroidBelt)
	g.PendingArtificialPlanets[1] = []uint{source.ID}

	require.NoError(t, g.Turn(1, model.Commands{
		ArtificialPlanet: &model.ArtificialPlanetCommand{
			SourcePlanetID: source.ID,
			TargetPlanetID: target.ID,
		},
	}))

	assert.Equal(t, model.PlanetEnvironmentBarren, target.Environment)
	assert.Equal(t, model.PlanetSizeLarge, target.Size)
	assert.Equal(t, model.PlanetMineralAbundant, target.MineralClass)
	assert.Equal(t, model.PlanetGravityNormal, target.Gravity)
	assert.Equal(t, uint(0), target.Owner)
}

func TestArtificialPlanetCommandRejectsInvalidTarget(t *testing.T) {
	t.Run("owned target", func(t *testing.T) {
		g, source, target := newArtificialPlanetGame(t, model.PlanetEnvironmentGasGiant)
		g.PendingArtificialPlanets[1] = []uint{source.ID}
		target.Owner = 2

		err := g.Turn(1, model.Commands{ArtificialPlanet: &model.ArtificialPlanetCommand{SourcePlanetID: source.ID, TargetPlanetID: target.ID}})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid artificial planet target")
	})

	t.Run("different system", func(t *testing.T) {
		g, source, target := newArtificialPlanetGame(t, model.PlanetEnvironmentGasGiant)
		g.PendingArtificialPlanets[1] = []uint{source.ID}
		target.ParentStar = 2

		err := g.Turn(1, model.Commands{ArtificialPlanet: &model.ArtificialPlanetCommand{SourcePlanetID: source.ID, TargetPlanetID: target.ID}})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "same system")
	})

	t.Run("non pending source", func(t *testing.T) {
		g, source, target := newArtificialPlanetGame(t, model.PlanetEnvironmentGasGiant)

		err := g.Turn(1, model.Commands{ArtificialPlanet: &model.ArtificialPlanetCommand{SourcePlanetID: source.ID, TargetPlanetID: target.ID}})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no pending artificial planet construction")
	})

	t.Run("regular planet", func(t *testing.T) {
		g, source, target := newArtificialPlanetGame(t, model.PlanetEnvironmentTerran)
		g.PendingArtificialPlanets[1] = []uint{source.ID}

		err := g.Turn(1, model.Commands{ArtificialPlanet: &model.ArtificialPlanetCommand{SourcePlanetID: source.ID, TargetPlanetID: target.ID}})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid artificial planet target")
	})
}

func TestArtificialPlanetPendingPrunedWhenUnavailable(t *testing.T) {
	g, source, target := newArtificialPlanetGame(t, model.PlanetEnvironmentGasGiant)
	g.PendingArtificialPlanets[1] = []uint{source.ID}
	for _, planet := range g.Galaxy.Stars[0].Planets {
		if planet.Owner == 0 {
			planet.Environment = model.PlanetEnvironmentBarren
		}
	}
	target.Environment = model.PlanetEnvironmentBarren

	require.NoError(t, g.Turn(1, model.Commands{}))

	assert.Empty(t, g.PendingArtificialPlanets[1])
}

func newArtificialPlanetGame(t *testing.T, targetEnvironment model.PlanetEnvironment) (*Game, *model.Planet, *model.Planet) {
	t.Helper()
	resources, err := NewResourceManager()
	require.NoError(t, err)
	g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: DefaultRace("Human")})
	require.NoError(t, err)
	source := g.Galaxy.Stars[0].Planets[0]
	source.Population = 10000
	source.Industry.Workers = 1
	source.CurrentProject = "Trade Goods"
	source.IsBuilding = false
	source.BuildBucket = 0
	target := &model.Planet{
		ID:           999,
		Name:         "Ada II",
		ParentStar:   source.ParentStar,
		Owner:        0,
		Environment:  targetEnvironment,
		Size:         model.PlanetSizeHuge,
		MineralClass: model.PlanetMineralRich,
		Gravity:      model.PlanetGravityHigh,
	}
	g.Galaxy.Stars[0].Planets = append(g.Galaxy.Stars[0].Planets, target)
	if g.PendingArtificialPlanets == nil {
		g.PendingArtificialPlanets = make(map[uint][]uint)
	}
	return g, source, target
}
