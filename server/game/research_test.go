package game

import (
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResearchMakesBuildingAvailable(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: DefaultRace("Human")})
	require.NoError(t, err)

	player := g.Players[0]
	g.research(player, "Automated Factory")
	g.research(player, "Automated Factory")

	assert.Contains(t, player.Available.Buildings, "Capitol")
	assert.Contains(t, player.Available.Buildings, "Colony Base")
	assert.Contains(t, player.Available.Buildings, "Star Base")
	assert.Contains(t, player.Available.Buildings, "Marine Barracks")
	assert.Contains(t, player.Available.Buildings, "Automated Factory")
	assert.Equal(t, 1, countStrings(player.Available.Buildings, "Automated Factory"))
	assert.Contains(t, player.ResearchedTechs, "Automated Factory")
}

func TestTurnAccumulatesResearchPointsForSelectedBreakthrough(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: DefaultRace("Human")})
	require.NoError(t, err)

	player := g.Players[0]
	require.NoError(t, g.Turn(player.ID, model.Commands{
		SelectResearch: &model.SelectResearchCommand{TechName: "Electronic Computer"},
	}))

	require.NotNil(t, player.CurrentResearch)
	assert.Equal(t, "Electronic Computer", player.CurrentResearch.TechName)
	assert.Equal(t, "Electronics", player.CurrentResearch.LevelName)
	assert.Equal(t, uint64(3), player.ResearchPoints)
	assert.Equal(t, uint64(3), player.CurrentResearch.Progress)
}

func TestTurnCompletesSelectedTechAndSpendsAccumulatedResearch(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: DefaultRace("Human")})
	require.NoError(t, err)

	player := g.Players[0]
	player.ResearchPoints = 77
	player.CurrentResearch = &model.ResearchProject{
		Subject:   "Hydroponic Farm",
		TechName:  "Hydroponic Farm",
		TechNames: []string{"Hydroponic Farm"},
		LevelName: "Astro Biology",
		Progress:  77,
	}

	require.NoError(t, g.Turn(player.ID, model.Commands{}))

	assert.Nil(t, player.CurrentResearch)
	assert.Contains(t, player.ResearchedTechs, "Hydroponic Farm")
	assert.NotContains(t, player.ResearchedTechs, "Biospheres")
	assert.Contains(t, player.Available.Buildings, "Hydroponic Farm")
	assert.Equal(t, uint64(0), player.ResearchPoints)
}

func TestTurnCompletesForceAllResearchAndUnlocksEveryTechInLevel(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: DefaultRace("Human")})
	require.NoError(t, err)

	player := g.Players[0]
	player.ResearchPoints = 47
	require.NoError(t, g.Turn(player.ID, model.Commands{
		SelectResearch: &model.SelectResearchCommand{TechName: "Nuclear Fission"},
	}))

	assert.Nil(t, player.CurrentResearch)
	assert.Contains(t, player.ResearchedTechs, "Freighters")
	assert.Contains(t, player.ResearchedTechs, "Nuclear Drive")
	assert.Contains(t, player.ResearchedTechs, "Nuclear Bomb")
	assert.Contains(t, player.Available.Drives, "Nuclear")
	assert.Equal(t, uint64(0), player.ResearchPoints)
}

func TestCanChangeResearchWithoutLosingAccumulatedPoints(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: DefaultRace("Human")})
	require.NoError(t, err)

	player := g.Players[0]
	player.ResearchPoints = 40
	require.NoError(t, g.Turn(player.ID, model.Commands{
		SelectResearch: &model.SelectResearchCommand{TechName: "Electronic Computer"},
	}))
	require.NotNil(t, player.CurrentResearch)
	assert.Equal(t, uint64(43), player.ResearchPoints)

	require.NoError(t, g.Turn(player.ID, model.Commands{
		SelectResearch: &model.SelectResearchCommand{TechName: "Anti-Missile Rockets"},
	}))

	require.NotNil(t, player.CurrentResearch)
	assert.Equal(t, "Anti-Missile Rockets", player.CurrentResearch.TechName)
	assert.Equal(t, uint64(46), player.ResearchPoints)
}

func TestHyperAdvancedResearchRequiresCompletedSubject(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: DefaultRace("Human")})
	require.NoError(t, err)

	player := g.Players[0]
	err = g.Turn(player.ID, model.Commands{
		SelectResearch: &model.SelectResearchCommand{TechName: "Hyper Advanced Construction"},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires completed subject")
}

func TestHyperAdvancedResearchDoesNotCompleteWithoutResearchPoints(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: DefaultRace("Human")})
	require.NoError(t, err)

	player := g.Players[0]
	researchSubject(g, player, "Construction")
	player.ResearchPoints = 0
	require.NoError(t, g.selectResearch(player, model.SelectResearchCommand{TechName: "Hyper Advanced Construction"}))

	require.NoError(t, g.applyResearch(player, 0))

	require.NotNil(t, player.CurrentResearch)
	assert.Equal(t, uint64(0), player.HyperAdvanced["Construction"])
	assert.Equal(t, uint64(0), player.ResearchPoints)
}

func TestHyperAdvancedResearchDoesNotCompleteFromBankedResearchWithoutNewResearch(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: DefaultRace("Human")})
	require.NoError(t, err)

	player := g.Players[0]
	researchSubject(g, player, "Construction")
	cost, ok := g.hyperAdvancedResearchCost(player, "Construction")
	require.True(t, ok)
	player.ResearchPoints = cost
	require.NoError(t, g.selectResearch(player, model.SelectResearchCommand{TechName: "Hyper Advanced Construction"}))

	require.NoError(t, g.applyResearch(player, 0))

	require.NotNil(t, player.CurrentResearch)
	assert.Equal(t, uint64(0), player.HyperAdvanced["Construction"])
	assert.Equal(t, cost, player.ResearchPoints)
}

func TestHyperAdvancedResearchCompletesAndDoublesNextCost(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: DefaultRace("Human")})
	require.NoError(t, err)

	player := g.Players[0]
	researchSubject(g, player, "Construction")
	cost, ok := g.hyperAdvancedResearchCost(player, "Construction")
	require.True(t, ok)

	require.NoError(t, g.selectResearch(player, model.SelectResearchCommand{TechName: "Hyper Advanced Construction"}))
	require.NoError(t, g.applyResearch(player, cost))

	assert.Nil(t, player.CurrentResearch)
	assert.Equal(t, uint64(1), player.HyperAdvanced["Construction"])
	assert.Equal(t, uint64(0), player.ResearchPoints)

	nextCost, ok := g.hyperAdvancedResearchCost(player, "Construction")
	require.True(t, ok)
	assert.Equal(t, cost*2, nextCost)
}

func TestCapitalDoesNotStartWithColonyBaseBuilt(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: DefaultRace("Human")})
	require.NoError(t, err)

	colonies, err := g.GetColonyList(g.Players[0].ID)
	require.NoError(t, err)
	require.NotEmpty(t, colonies)

	assert.NotContains(t, colonies[0].Buildings, "Colony Base")
	assert.Contains(t, colonies[0].Buildings, "Capitol")
	assert.Contains(t, colonies[0].Buildings, "Star Base")
}

func researchSubject(g *Game, player *model.Player, subjectName string) {
	for _, subject := range g.Resources.TechTree.Subjects {
		if subject.Name != subjectName {
			continue
		}
		for _, level := range subject.Levels {
			for _, techName := range level.AvailableTechs {
				g.research(player, techName)
			}
		}
		return
	}
}

func countStrings(values []string, needle string) int {
	count := 0
	for _, value := range values {
		if value == needle {
			count++
		}
	}
	return count
}
