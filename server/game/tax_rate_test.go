package game

import (
	"encoding/json"
	"fmt"
	"math"
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/require"
)

func newTaxRateGame(t *testing.T) (*Game, *ResourceManager) {
	t.Helper()
	resources, err := NewResourceManager()
	require.NoError(t, err)
	g, err := NewGame(resources, NewGameOptions{
		PlayerName: "Ada", PlayerColor: model.DefaultPlayerColor(),
		NumStars: 8, Race: DefaultRace("Human"), NumPlayers: 2,
	})
	require.NoError(t, err)
	return g, resources
}

func taxRateCommands(t *testing.T, rate float64) model.Commands {
	t.Helper()
	var commands model.Commands
	require.NoError(t, json.Unmarshal([]byte(fmt.Sprintf(`{"set_tax_rate":{"rate":%g}}`, rate)), &commands))
	return commands
}

func TestSetTaxRateAppliesOnTurnResolutionAndPersists(t *testing.T) {
	for _, rate := range []float64{0, 0.3, 0.5, 0.25} {
		t.Run(fmt.Sprint(rate), func(t *testing.T) {
			g, resources := newTaxRateGame(t)
			g.Players[0].TaxRate = 0.1
			g.Players[1].TaxRate = 0.2
			_, advanced, err := g.SubmitTurn(1, taxRateCommands(t, rate))
			require.NoError(t, err)
			require.False(t, advanced)
			require.Equal(t, 0.1, g.Players[0].TaxRate, "orders must wait for turn resolution")
			_, advanced, err = g.SubmitTurn(2, model.Commands{})
			require.NoError(t, err)
			require.True(t, advanced)
			require.Equal(t, rate, g.Players[0].TaxRate)
			require.Equal(t, 0.2, g.Players[1].TaxRate, "omitting the order preserves the rate")

			data, err := g.MarshalSave()
			require.NoError(t, err)
			var loaded Game
			require.NoError(t, json.Unmarshal(data, &loaded))
			require.NoError(t, ValidateLoadedGame(&loaded, resources))
			HydrateLoadedGame(&loaded, resources)
			require.Equal(t, rate, loaded.Players[0].TaxRate)
			require.Equal(t, 0.2, loaded.Players[1].TaxRate)
		})
	}
}

func TestSubmitTurnRejectsInvalidTaxRate(t *testing.T) {
	for _, rate := range []float64{-0.1, 0.50001, 1, math.NaN(), math.Inf(1), math.Inf(-1)} {
		t.Run(fmt.Sprint(rate), func(t *testing.T) {
			g, _ := newTaxRateGame(t)
			g.Players[0].TaxRate = 0.2
			_, _, err := g.SubmitTurn(2, model.Commands{})
			require.NoError(t, err)
			turn := g.CurrentTurn
			_, advanced, err := g.SubmitTurn(1, model.Commands{SetTaxRate: &model.SetTaxRateCommand{Rate: rate}})
			require.ErrorContains(t, err, "tax rate")
			require.False(t, advanced)
			require.Equal(t, turn, g.CurrentTurn)
			require.Equal(t, 0.2, g.Players[0].TaxRate)
			require.NotContains(t, g.TurnSubmissions, uint(1))
			require.Contains(t, g.TurnSubmissions, uint(2))
		})
	}
}

func TestLoadLegacySaveWithoutTaxRate(t *testing.T) {
	g, resources := newTaxRateGame(t)
	data, err := g.MarshalSave()
	require.NoError(t, err)
	var saved map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &saved))
	var players []map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(saved["players"], &players))
	for _, player := range players {
		delete(player, "tax_rate")
	}
	saved["players"], err = json.Marshal(players)
	require.NoError(t, err)
	data, err = json.Marshal(saved)
	require.NoError(t, err)
	var loaded Game
	require.NoError(t, json.Unmarshal(data, &loaded))
	require.NoError(t, ValidateLoadedGame(&loaded, resources))
	HydrateLoadedGame(&loaded, resources)
	for _, player := range loaded.Players {
		require.Zero(t, player.TaxRate)
	}
}

func TestSetTaxRateRollsBackWithFailedTurn(t *testing.T) {
	g, _ := newTaxRateGame(t)
	g.Players[0].TaxRate = 0.1
	_, advanced, err := g.SubmitTurn(1, taxRateCommands(t, 0.3))
	require.NoError(t, err)
	require.False(t, advanced)
	advanceTurnTestHook = func(stage string) error {
		if stage == "after_apply:1" {
			return fmt.Errorf("injected turn failure")
		}
		return nil
	}
	t.Cleanup(func() { advanceTurnTestHook = nil })
	_, advanced, err = g.SubmitTurn(2, model.Commands{})
	require.ErrorContains(t, err, "injected turn failure")
	require.False(t, advanced)
	require.Equal(t, 0.1, g.Players[0].TaxRate)
	require.Contains(t, g.TurnSubmissions, uint(1))
	require.Equal(t, 0.3, g.TurnSubmissions[1].SetTaxRate.Rate)
	advanceTurnTestHook = nil
	_, advanced, err = g.SubmitTurn(2, model.Commands{})
	require.NoError(t, err)
	require.True(t, advanced)
	require.Equal(t, 0.3, g.Players[0].TaxRate)
}

func TestTaxRateSnapshotDoesNotAliasOrder(t *testing.T) {
	commands := taxRateCommands(t, 0.3)
	snapshot := copyCommands(commands)
	commands.SetTaxRate.Rate = 0.5
	require.Equal(t, 0.3, snapshot.SetTaxRate.Rate)
}

func TestLoadPreservesHistoricalTaxRateRange(t *testing.T) {
	g, resources := newTaxRateGame(t)
	g.Players[0].TaxRate = 0.75
	data, err := g.MarshalSave()
	require.NoError(t, err)
	var loaded Game
	require.NoError(t, json.Unmarshal(data, &loaded))
	require.NoError(t, ValidateLoadedGame(&loaded, resources))
	HydrateLoadedGame(&loaded, resources)
	require.Equal(t, 0.75, loaded.Players[0].TaxRate)
}
