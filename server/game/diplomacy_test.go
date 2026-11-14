package game

import (
	"encoding/json"
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// contactIDsJSON reads a player's contacts through the save/API JSON so the
// assertion is about the serialized contract.
func contactIDsJSON(t *testing.T, p *model.Player) []uint {
	t.Helper()
	data, err := json.Marshal(p)
	require.NoError(t, err)
	var decoded struct {
		Contacts []struct {
			PlayerID uint `json:"player_id"`
			Relation *int `json:"relation"`
		} `json:"contacts"`
	}
	require.NoError(t, json.Unmarshal(data, &decoded))
	ids := []uint{}
	for _, c := range decoded.Contacts {
		ids = append(ids, c.PlayerID)
	}
	return ids
}

func firstContactEventOwners(g *Game) []uint {
	owners := []uint{}
	for _, event := range g.TurnEvents {
		if event.Kind == "first_contact" {
			owners = append(owners, event.Owner)
		}
	}
	return owners
}

func playerByID(t *testing.T, g *Game, id uint) *model.Player {
	t.Helper()
	p, err := g.getPlayer(id)
	require.NoError(t, err)
	return p
}

func visiblePlayerIDs(t *testing.T, g *Game, viewer uint) []uint {
	t.Helper()
	visible, err := g.GetVisibleGameForPlayer(viewer)
	require.NoError(t, err)
	ids := []uint{}
	for _, p := range visible.Players {
		ids = append(ids, p.ID)
	}
	return ids
}

func testStarIsEmpty(g *Game, star *model.Star) bool {
	for _, planet := range star.Planets {
		if planet != nil && planet.Owner != 0 {
			return false
		}
	}
	for _, fleet := range g.Fleets {
		if fleet != nil && fleet.StarID == star.ID {
			return false
		}
	}
	return true
}

// emptyStarAwayFrom returns a star with no colonies and no fleets.
func emptyStarAwayFrom(t *testing.T, g *Game) *model.Star {
	t.Helper()
	for _, star := range g.Galaxy.Stars {
		if testStarIsEmpty(g, star) {
			return star
		}
	}
	t.Fatal("no empty star")
	return nil
}

// arrivingFleet places a fleet one turn away from starID.
func arrivingFleet(g *Game, owner, starID uint) *model.Fleet {
	fleet := &model.Fleet{
		ID: g.nextFleetID(), Owner: owner, DestinationStarID: starID,
		TurnsRemaining: 1, TotalTurns: 1,
		Ships: []model.FleetShip{{Type: "colony_ship", Name: "Colony Ship"}},
	}
	g.Fleets = append(g.Fleets, fleet)
	return fleet
}

func orbitingFleet(g *Game, owner, starID uint) *model.Fleet {
	fleet := &model.Fleet{
		ID: g.nextFleetID(), Owner: owner, StarID: starID,
		Ships: []model.FleetShip{{Type: "colony_ship", Name: "Colony Ship"}},
	}
	g.Fleets = append(g.Fleets, fleet)
	return fleet
}

func TestFirstContactOnArrivalAtRivalColony(t *testing.T) {
	g := multiPlayerGame(t, 3, 30)
	rivalHome := getPlayerHomeworld(g, 2)
	require.NotNil(t, rivalHome)
	arrivingFleet(g, 1, rivalHome.ParentStar)

	require.NoError(t, g.AdvanceTurn(emptySubsFor(g)))

	assert.Equal(t, []uint{2}, contactIDsJSON(t, playerByID(t, g, 1)))
	assert.Equal(t, []uint{1}, contactIDsJSON(t, playerByID(t, g, 2)), "both sides get the contact")
	assert.Empty(t, contactIDsJSON(t, playerByID(t, g, 3)), "an absent empire is not contacted")
	assert.ElementsMatch(t, []uint{1, 2}, firstContactEventOwners(g))
	firstContactAt, battleAt := -1, -1
	for i, event := range g.TurnEvents {
		if event.Kind == "first_contact" {
			assert.Equal(t, rivalHome.ParentStar, event.StarID)
			assert.Contains(t, event.Text, "First contact with")
			if firstContactAt < 0 {
				firstContactAt = i
			}
		}
		if event.Kind == "battle" && event.StarID == rivalHome.ParentStar && battleAt < 0 {
			battleAt = i
		}
	}
	if battleAt >= 0 {
		assert.Less(t, firstContactAt, battleAt, "contact happens before battles at the same star")
	}

	// Contact is recorded once; later turns at the same star do not repeat it.
	require.NoError(t, g.AdvanceTurn(emptySubsFor(g)))
	assert.Equal(t, []uint{2}, contactIDsJSON(t, playerByID(t, g, 1)))
	assert.Empty(t, firstContactEventOwners(g))
}

func TestFirstContactWithOrbitingRivalFleet(t *testing.T) {
	g := multiPlayerGame(t, 2, 30)
	star := emptyStarAwayFrom(t, g)
	orbitingFleet(g, 2, star.ID)
	arrivingFleet(g, 1, star.ID)

	require.NoError(t, g.AdvanceTurn(emptySubsFor(g)))

	assert.Equal(t, []uint{2}, contactIDsJSON(t, playerByID(t, g, 1)))
	assert.Equal(t, []uint{1}, contactIDsJSON(t, playerByID(t, g, 2)))
	assert.ElementsMatch(t, []uint{1, 2}, firstContactEventOwners(g))
}

func TestFirstContactWhenRivalArrivesWhereFleetAlreadyOrbits(t *testing.T) {
	g := multiPlayerGame(t, 2, 30)
	star := emptyStarAwayFrom(t, g)
	orbitingFleet(g, 1, star.ID)
	arrivingFleet(g, 2, star.ID)

	require.NoError(t, g.AdvanceTurn(emptySubsFor(g)))

	assert.Equal(t, []uint{2}, contactIDsJSON(t, playerByID(t, g, 1)))
	assert.Equal(t, []uint{1}, contactIDsJSON(t, playerByID(t, g, 2)))
}

func TestNoFirstContactAtEmptyStarOrWithPassingFleets(t *testing.T) {
	g := multiPlayerGame(t, 2, 30)
	star := emptyStarAwayFrom(t, g)
	arrivingFleet(g, 1, star.ID)
	// A rival fleet that is only passing through (in transit) does not count,
	// and neither does the rival having explored the star before.
	other := emptyStarAwayFrom(t, g)
	for _, s := range g.Galaxy.Stars {
		if s.ID != star.ID && testStarIsEmpty(g, s) {
			other = s
			break
		}
	}
	require.NotEqual(t, star.ID, other.ID)
	g.Fleets = append(g.Fleets, &model.Fleet{
		ID: g.nextFleetID(), Owner: 2, StarID: star.ID, DestinationStarID: other.ID,
		TurnsRemaining: 5, TotalTurns: 6,
		Ships: []model.FleetShip{{Type: "colony_ship", Name: "Colony Ship"}},
	})
	require.NoError(t, g.ExploreStar(2, star.ID))

	require.NoError(t, g.AdvanceTurn(emptySubsFor(g)))

	assert.Empty(t, contactIDsJSON(t, playerByID(t, g, 1)))
	assert.Empty(t, contactIDsJSON(t, playerByID(t, g, 2)))
	assert.Empty(t, firstContactEventOwners(g))
	assert.Equal(t, []uint{1}, visiblePlayerIDs(t, g, 1))
}

func TestFirstContactWhenColonizingSharedSystem(t *testing.T) {
	g := multiPlayerGame(t, 2, 30)
	star := emptyStarAwayFrom(t, g)
	rivalColony := &model.Planet{
		ID: nextTestPlanetID(g), Name: "Rival Outpost", ParentStar: star.ID, Owner: 2,
		Environment: model.PlanetEnvironmentTerran, Size: model.PlanetSizeMedium,
		MineralClass: model.PlanetMineralAbundant, Population: 1000,
	}
	star.Planets = append(star.Planets, rivalColony)
	target := &model.Planet{
		ID: nextTestPlanetID(g), Name: "New Home", ParentStar: star.ID,
		Environment: model.PlanetEnvironmentTerran, Size: model.PlanetSizeMedium,
		MineralClass: model.PlanetMineralAbundant,
	}
	star.Planets = append(star.Planets, target)
	require.NoError(t, g.ExploreStar(1, star.ID))
	colonyFleet := orbitingFleet(g, 1, star.ID)
	playerByID(t, g, 1).Fleets = append(playerByID(t, g, 1).Fleets, colonyFleet.ID)

	subs := emptySubsFor(g)
	subs[1] = model.Commands{Colonize: &model.ColonizeCommand{FleetID: colonyFleet.ID, PlanetID: target.ID}}
	require.NoError(t, g.AdvanceTurn(subs))

	require.Equal(t, uint(1), target.Owner)
	for _, f := range g.Fleets {
		require.NotEqual(t, colonyFleet.ID, f.ID, "the colony ship is consumed, so only the colony meets the rival")
	}
	assert.Equal(t, []uint{2}, contactIDsJSON(t, playerByID(t, g, 1)))
	assert.Equal(t, []uint{1}, contactIDsJSON(t, playerByID(t, g, 2)))
	assert.ElementsMatch(t, []uint{1, 2}, firstContactEventOwners(g))
}

func TestVisibleGameOmitsUncontactedRivals(t *testing.T) {
	g := multiPlayerGame(t, 3, 30)
	assert.Equal(t, []uint{1}, visiblePlayerIDs(t, g, 1))
	assert.Equal(t, []uint{2}, visiblePlayerIDs(t, g, 2))

	rivalHome := getPlayerHomeworld(g, 3)
	require.NotNil(t, rivalHome)
	arrivingFleet(g, 1, rivalHome.ParentStar)
	require.NoError(t, g.AdvanceTurn(emptySubsFor(g)))

	assert.Equal(t, []uint{1, 3}, visiblePlayerIDs(t, g, 1))
	assert.Equal(t, []uint{1, 3}, visiblePlayerIDs(t, g, 3))
	assert.Equal(t, []uint{2}, visiblePlayerIDs(t, g, 2), "player 2 has met nobody")

	// The viewer sees its own contacts (relation 0) but not the rival's.
	visible, err := g.GetVisibleGameForPlayer(1)
	require.NoError(t, err)
	data, err := json.Marshal(visible)
	require.NoError(t, err)
	var decoded struct {
		Players []struct {
			ID       uint              `json:"id"`
			Contacts []json.RawMessage `json:"contacts"`
		} `json:"players"`
	}
	require.NoError(t, json.Unmarshal(data, &decoded))
	for _, p := range decoded.Players {
		if p.ID == 1 {
			require.Len(t, p.Contacts, 1)
			var c map[string]any
			require.NoError(t, json.Unmarshal(p.Contacts[0], &c))
			assert.Equal(t, float64(3), c["player_id"])
			assert.Nil(t, c["relation"], "relation is 0 and omitted")
		} else {
			assert.Empty(t, p.Contacts)
		}
	}
}

func TestOldSaveWithoutContactsLoadsWithEmptyContacts(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)
	g := multiPlayerGame(t, 2, 30)

	data, err := json.Marshal(g)
	require.NoError(t, err)
	var raw map[string]any
	require.NoError(t, json.Unmarshal(data, &raw))
	for _, p := range raw["players"].([]any) {
		delete(p.(map[string]any), "contacts")
	}
	data, err = json.Marshal(raw)
	require.NoError(t, err)
	require.NotContains(t, string(data), `"contacts"`)

	var loaded Game
	require.NoError(t, json.Unmarshal(data, &loaded))
	HydrateLoadedGame(&loaded, resources)
	require.NoError(t, ValidateLoadedGame(&loaded, resources))

	for _, p := range loaded.Players {
		assert.Empty(t, contactIDsJSON(t, p))
	}
	assert.Equal(t, []uint{1}, visiblePlayerIDs(t, &loaded, 1))
	require.NoError(t, loaded.AdvanceTurn(emptySubsFor(&loaded)))
	assert.Empty(t, firstContactEventOwners(&loaded))
}

func TestContactsSurviveSaveRoundTrip(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)
	g := multiPlayerGame(t, 2, 30)
	rivalHome := getPlayerHomeworld(g, 2)
	arrivingFleet(g, 1, rivalHome.ParentStar)
	require.NoError(t, g.AdvanceTurn(emptySubsFor(g)))

	data, err := g.MarshalSave()
	require.NoError(t, err)
	var loaded Game
	require.NoError(t, json.Unmarshal(data, &loaded))
	HydrateLoadedGame(&loaded, resources)
	require.NoError(t, ValidateLoadedGame(&loaded, resources))

	assert.Equal(t, []uint{2}, contactIDsJSON(t, loaded.Players[0]))
	assert.Equal(t, []uint{1}, contactIDsJSON(t, loaded.Players[1]))
	assert.Equal(t, []uint{1, 2}, visiblePlayerIDs(t, &loaded, 1))
}
