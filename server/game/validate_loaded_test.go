package game

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/require"
)

func TestValidateLoadedGameAcceptsNewGame(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)
	g, err := NewGame(resources, NewGameOptions{
		PlayerName:  "Ada",
		NumStars:    6,
		PlayerColor: model.DefaultPlayerColor(),
		Race:        DefaultRace("Human"),
	})
	require.NoError(t, err)
	require.NoError(t, ValidateLoadedGame(g, resources))
}

func TestValidateLoadedGameAcceptsRepoScenarios(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	dir := filepath.Join("data", "scenarios")
	entries, err := os.ReadDir(dir)
	if err != nil {
		// Tests may run with cwd=server/; also try relative from module root.
		dir = filepath.Join("..", "data", "scenarios")
		entries, err = os.ReadDir(dir)
	}
	require.NoError(t, err)

	found := 0
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		found++
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		require.NoError(t, err, entry.Name())
		var loaded Game
		require.NoError(t, json.Unmarshal(data, &loaded), entry.Name())
		require.NoError(t, ValidateLoadedGame(&loaded, resources), entry.Name())
	}
	require.Greater(t, found, 0)
}

func TestValidateLoadedGameRejectsInvalidCases(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	base := func(t *testing.T) *Game {
		t.Helper()
		g, err := NewGame(resources, NewGameOptions{
			PlayerName:  "Ada",
			NumStars:    6,
			PlayerColor: model.DefaultPlayerColor(),
			Race:        DefaultRace("Human"),
		})
		require.NoError(t, err)
		return g
	}

	t.Run("no players", func(t *testing.T) {
		g := base(t)
		g.Players = nil
		err := ValidateLoadedGame(g, resources)
		require.Error(t, err)
		require.Contains(t, err.Error(), "no players")
	})

	t.Run("too many players", func(t *testing.T) {
		g := base(t)
		for id := uint(2); id <= model.MaxPlayers+1; id++ {
			g.Players = append(g.Players, NewPlayer(id, "X", model.DefaultPlayerColor(), DefaultRace("Human")))
		}
		err := ValidateLoadedGame(g, resources)
		require.Error(t, err)
		require.Contains(t, err.Error(), "too many players")
	})

	t.Run("duplicate player id", func(t *testing.T) {
		g := base(t)
		g.Players = append(g.Players, NewPlayer(g.Players[0].ID, "Dup", model.DefaultPlayerColor(), DefaultRace("Human")))
		err := ValidateLoadedGame(g, resources)
		require.Error(t, err)
		require.Contains(t, err.Error(), "duplicate player id")
	})

	t.Run("zero player id", func(t *testing.T) {
		g := base(t)
		g.Players[0].ID = 0
		err := ValidateLoadedGame(g, resources)
		require.Error(t, err)
		require.Contains(t, err.Error(), "zero id")
	})

	t.Run("duplicate star id", func(t *testing.T) {
		g := base(t)
		require.GreaterOrEqual(t, len(g.Galaxy.Stars), 2)
		g.Galaxy.Stars[1].ID = g.Galaxy.Stars[0].ID
		err := ValidateLoadedGame(g, resources)
		require.Error(t, err)
		require.Contains(t, err.Error(), "duplicate star id")
	})

	t.Run("duplicate planet id", func(t *testing.T) {
		g := base(t)
		var first, second *model.Planet
		for _, star := range g.Galaxy.Stars {
			for _, p := range star.Planets {
				if first == nil {
					first = p
					continue
				}
				second = p
				break
			}
			if second != nil {
				break
			}
		}
		require.NotNil(t, first)
		require.NotNil(t, second)
		second.ID = first.ID
		err := ValidateLoadedGame(g, resources)
		require.Error(t, err)
		require.Contains(t, err.Error(), "duplicate planet id")
	})

	t.Run("duplicate fleet id", func(t *testing.T) {
		g := base(t)
		starID := g.Galaxy.Stars[0].ID
		g.Fleets = []*model.Fleet{
			{ID: 7, Owner: g.Players[0].ID, StarID: starID},
			{ID: 7, Owner: g.Players[0].ID, StarID: starID},
		}
		err := ValidateLoadedGame(g, resources)
		require.Error(t, err)
		require.Contains(t, err.Error(), "duplicate fleet id")
	})

	t.Run("negative population", func(t *testing.T) {
		g := base(t)
		planet := firstOwnedPlanet(t, g)
		planet.Population = -1
		err := ValidateLoadedGame(g, resources)
		require.Error(t, err)
		require.Contains(t, err.Error(), "negative population")
	})

	t.Run("unknown tech", func(t *testing.T) {
		g := base(t)
		g.Players[0].ResearchedTechs = append(g.Players[0].ResearchedTechs, "Totally Fake Tech")
		err := ValidateLoadedGame(g, resources)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unknown tech")
	})

	t.Run("unknown race trait", func(t *testing.T) {
		g := base(t)
		g.Players[0].Race.TraitIDs = []string{"not_a_real_trait"}
		err := ValidateLoadedGame(g, resources)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unknown race trait")
	})

	t.Run("unknown ship design", func(t *testing.T) {
		g := base(t)
		starID := g.Galaxy.Stars[0].ID
		g.Fleets = []*model.Fleet{{
			ID:     9,
			Owner:  g.Players[0].ID,
			StarID: starID,
			Ships:  []model.FleetShip{{Type: "ship_design", Name: "X", DesignName: "No Such Design"}},
		}}
		err := ValidateLoadedGame(g, resources)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unknown ship design")
	})

	t.Run("fleet missing player", func(t *testing.T) {
		g := base(t)
		starID := g.Galaxy.Stars[0].ID
		g.Fleets = []*model.Fleet{{ID: 3, Owner: 99, StarID: starID}}
		err := ValidateLoadedGame(g, resources)
		require.Error(t, err)
		require.Contains(t, err.Error(), "missing player")
	})

	t.Run("planet missing player", func(t *testing.T) {
		g := base(t)
		planet := firstOwnedPlanet(t, g)
		planet.Owner = 99
		err := ValidateLoadedGame(g, resources)
		require.Error(t, err)
		require.Contains(t, err.Error(), "missing player")
	})

	t.Run("fleet missing star", func(t *testing.T) {
		g := base(t)
		g.Fleets = []*model.Fleet{{ID: 3, Owner: g.Players[0].ID, StarID: 99999}}
		err := ValidateLoadedGame(g, resources)
		require.Error(t, err)
		require.Contains(t, err.Error(), "missing star")
	})

	t.Run("negative current turn", func(t *testing.T) {
		g := base(t)
		g.CurrentTurn = -1
		err := ValidateLoadedGame(g, resources)
		require.Error(t, err)
		require.Contains(t, err.Error(), "current turn")
	})

	t.Run("unknown current research level", func(t *testing.T) {
		g := base(t)
		g.Players[0].CurrentResearch = &model.ResearchProject{
			Subject:   "Bogus",
			TechName:  "Bogus Tech",
			TechNames: []string{"Bogus Tech"},
			LevelName: "Bogus Level",
		}
		err := ValidateLoadedGame(g, resources)
		require.Error(t, err)
		require.Contains(t, err.Error(), "Bogus Level")
	})

	t.Run("unknown hyper advanced subject", func(t *testing.T) {
		g := base(t)
		g.Players[0].CurrentResearch = &model.ResearchProject{
			Subject:   "NoSuchSubject",
			TechName:  "Hyper Advanced NoSuchSubject",
			LevelName: "Hyper Advanced 1",
		}
		err := ValidateLoadedGame(g, resources)
		require.Error(t, err)
		require.Contains(t, err.Error(), "hyper advanced subject")
	})

	t.Run("unknown tech in current research", func(t *testing.T) {
		g := base(t)
		g.Players[0].CurrentResearch = &model.ResearchProject{
			Subject:   "Not A Real Tech",
			TechName:  "Not A Real Tech",
			TechNames: []string{"Not A Real Tech"},
			LevelName: "Electronics",
		}
		err := ValidateLoadedGame(g, resources)
		require.Error(t, err)
		require.Contains(t, err.Error(), "not selectable")
	})

	t.Run("tech_names not in research level", func(t *testing.T) {
		g := base(t)
		g.Players[0].CurrentResearch = &model.ResearchProject{
			Subject:   "Electronics",
			TechNames: []string{"Automated Factory"},
			LevelName: "Electronics",
		}
		err := ValidateLoadedGame(g, resources)
		require.Error(t, err)
		require.Contains(t, err.Error(), "not selectable")
	})

	t.Run("current research level before incomplete earlier level", func(t *testing.T) {
		g := base(t)
		subject := g.Resources.TechTree.Subjects[0]
		require.GreaterOrEqual(t, len(subject.Levels), 2)
		second := subject.Levels[1]
		tech := second.AvailableTechs[0]
		g.Players[0].CurrentResearch = &model.ResearchProject{
			Subject:   tech,
			TechName:  tech,
			TechNames: []string{tech},
			LevelName: second.Name,
		}
		err := ValidateLoadedGame(g, resources)
		require.Error(t, err)
		require.Contains(t, err.Error(), "requires")
		require.Contains(t, err.Error(), subject.Levels[0].Name)
	})

	t.Run("valid in-progress research accepted", func(t *testing.T) {
		g := base(t)
		g.Players[0].CurrentResearch = &model.ResearchProject{
			Subject:   "Electronic Computer",
			TechName:  "Electronic Computer",
			TechNames: []string{"Electronic Computer"},
			LevelName: "Electronics",
			Progress:  3,
		}
		require.NoError(t, ValidateLoadedGame(g, resources))
	})

	t.Run("nil current research accepted", func(t *testing.T) {
		g := base(t)
		g.Players[0].CurrentResearch = nil
		require.NoError(t, ValidateLoadedGame(g, resources))
	})
}

func firstOwnedPlanet(t *testing.T, g *Game) *model.Planet {
	t.Helper()
	for _, star := range g.Galaxy.Stars {
		for _, p := range star.Planets {
			if p != nil && p.Owner != 0 {
				return p
			}
		}
	}
	t.Fatal("no owned planet")
	return nil
}
