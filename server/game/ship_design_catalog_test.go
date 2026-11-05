package game

import (
	"sync"
	"sync/atomic"
	"testing"

	. "ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShipDesignCatalogIncludesMergedCustomDesign(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)
	builtinCount := len(resources.ShipDesignCatalog())

	custom := ShipDesign{
		Name:     "Catalog Test Scout",
		HullSize: HullSize{Name: "Frigate"},
		Drive:    Drive{Name: "Nuclear"},
		Armor:    Armor{Name: "Titanium"},
		Computer: Computer{Name: "None"},
		FuelCell: FuelCell{Name: "Standard"},
		Shield:   Shield{Name: "None"},
	}
	resources.ApplyMergedShipDesignCatalog([]ShipDesign{custom})

	catalog := resources.ShipDesignCatalog()
	require.Len(t, catalog, builtinCount+1)
	assert.Equal(t, "Catalog Test Scout", catalog[builtinCount].Name)
}

func TestConcurrentShipDesignCatalogReadWhileMerge(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	var stop atomic.Bool
	var wg sync.WaitGroup
	for r := 0; r < 8; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for !stop.Load() {
				catalog := resources.ShipDesignCatalog()
				for _, d := range catalog {
					_ = d.Name
				}
			}
		}()
	}

	for i := 0; i < 200; i++ {
		name := "Stress " + string(rune('A'+i%26))
		resources.ApplyMergedShipDesignCatalog([]ShipDesign{{
			Name:     name,
			HullSize: HullSize{Name: "Frigate"},
			Drive:    Drive{Name: "Nuclear"},
			Armor:    Armor{Name: "Titanium"},
			Computer: Computer{Name: "None"},
			FuelCell: FuelCell{Name: "Standard"},
			Shield:   Shield{Name: "None"},
		}})
	}
	stop.Store(true)
	wg.Wait()
}

func TestFuelRangeLookupUsesShipDesignCatalog(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)
	g, err := NewGame(resources, NewGameOptions{
		PlayerName:  "Ada",
		NumStars:    4,
		PlayerColor: DefaultPlayerColor(),
		Race:        DefaultRace("Human"),
	})
	require.NoError(t, err)
	player := g.Players[0]
	lookup := g.fuelRangeLookup(player)
	require.NotEmpty(t, lookup.StockDesigns)
	require.Equal(t, resources.ShipDesignCatalog(), lookup.StockDesigns)
}
