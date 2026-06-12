// Command battledemo writes data/scenarios/battle-demo.json: a two-player
// game with armed fleets sharing a neutral star so an interactive tactical
// battle triggers on the first end-of-turn.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"ssg/game"
	. "ssg/model"
)

func main() {
	resources, err := game.NewResourceManager()
	must(err)

	g, err := game.NewGame(resources, game.NewGameOptions{
		PlayerName: "Commander",
		NumStars:   8,
		NumPlayers: 2,
		Race:       game.DefaultRace("Human"),
	})
	must(err)
	g.ID = "battle-demo"

	parts := resources.ShipPartsCollection
	hull, err := game.HullSizeFromName("Frigate", parts.HullSizes)
	must(err)
	bigHull, err := game.HullSizeFromName("Destroyer", parts.HullSizes)
	must(err)
	drive, err := game.DriveFromName("Nuclear", parts.Drives)
	must(err)
	armor, err := game.ArmorFromName("Titanium", parts.Armors)
	must(err)
	computer, err := game.ComputerFromName("Electronic", parts.Computers)
	must(err)
	fuel, err := game.FuelCellFromName("Standard", parts.FuelCells)
	must(err)
	shield, err := game.ShieldFromName("None", parts.Shields)
	must(err)

	laser := findBeam(resources.Beams, "Laser")
	fusion := findBeam(resources.Beams, "Fusion")

	gunship := game.NewShipDesign("Laser Gunship", []Beam{laser, laser, laser}, nil, hull, drive, armor, computer, fuel, shield)
	raider := game.NewShipDesign("Fusion Raider", []Beam{fusion, fusion}, nil, bigHull, drive, armor, computer, fuel, shield)

	playerOne := g.Players[0]
	playerTwo := g.Players[1]
	playerOne.ShipDesigns = append(playerOne.ShipDesigns, gunship)
	playerTwo.ShipDesigns = append(playerTwo.ShipDesigns, raider)

	star := neutralStar(g)
	must(g.ExploreStar(playerOne.ID, star.ID))
	must(g.ExploreStar(playerTwo.ID, star.ID))

	addFleet(g, playerOne, star, []FleetShip{
		{Type: "ship_design", Name: "Dark Blade", DesignName: "Laser Gunship", HullSize: hull.Name, Drive: drive.Name},
		{Type: "ship_design", Name: "Night Falcon", DesignName: "Laser Gunship", HullSize: hull.Name, Drive: drive.Name},
		{Type: "ship_design", Name: "Star Lance", DesignName: "Laser Gunship", HullSize: hull.Name, Drive: drive.Name},
	})
	addFleet(g, playerTwo, star, []FleetShip{
		{Type: "ship_design", Name: "Crimson Maw", DesignName: "Fusion Raider", HullSize: bigHull.Name, Drive: drive.Name},
		{Type: "ship_design", Name: "Void Render", DesignName: "Fusion Raider", HullSize: bigHull.Name, Drive: drive.Name},
	})

	data, err := json.MarshalIndent(g, "", "  ")
	must(err)
	out := filepath.Join("data", "scenarios", "battle-demo.json")
	must(os.WriteFile(out, data, 0o644))
	fmt.Println("wrote", out)
}

func neutralStar(g *game.Game) *Star {
	for _, star := range g.Galaxy.Stars {
		owned := false
		for _, planet := range star.Planets {
			if planet.Owner != 0 {
				owned = true
				break
			}
		}
		if !owned {
			return star
		}
	}
	panic("no neutral star")
}

func addFleet(g *game.Game, player *Player, star *Star, ships []FleetShip) {
	id := uint(len(g.Fleets) + 1)
	fleet := &Fleet{
		ID:      id,
		Owner:   player.ID,
		StarID:  star.ID,
		X:       float64(star.X),
		Y:       float64(star.Y),
		OriginX: float64(star.X),
		OriginY: float64(star.Y),
		Ships:   ships,
	}
	g.Fleets = append(g.Fleets, fleet)
	player.Fleets = append(player.Fleets, id)
}

func findBeam(beams []Beam, name string) Beam {
	for _, beam := range beams {
		if beam.Name == name {
			return beam
		}
	}
	panic("beam not found: " + name)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
