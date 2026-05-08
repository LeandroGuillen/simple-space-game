package game

import (
	"slices"
	. "ssg/model"
)

func (g *Game) hydratePlayerAvailability(player *Player) {
	available := AvailableTech{}
	applyUnlocks(&available, player.Race.Grants.Unlocks)
	for _, techName := range player.ResearchedTechs {
		if tech, ok := g.findTechDefinition(techName); ok {
			applyUnlocks(&available, tech.Grants.Unlocks)
		}
	}
	player.Available = available
}

func applyUnlocks(available *AvailableTech, unlocks Unlocks) {
	appendUniqueStrings(&available.Buildings, unlocks.Buildings...)
	appendUniqueStrings(&available.HullSizes, unlocks.HullSizes...)
	appendUniqueStrings(&available.Drives, unlocks.Drives...)
	appendUniqueStrings(&available.Armors, unlocks.Armors...)
	appendUniqueStrings(&available.Computers, unlocks.Computers...)
	appendUniqueStrings(&available.FuelCells, unlocks.FuelCells...)
	appendUniqueStrings(&available.Shields, unlocks.Shields...)
	appendUniqueStrings(&available.Beams, unlocks.Beams...)
	appendUniqueStrings(&available.ShipSystems, unlocks.ShipSystems...)
}

func appendUniqueStrings(target *[]string, values ...string) {
	for _, value := range values {
		if value == "" || slices.Contains(*target, value) {
			continue
		}
		*target = append(*target, value)
	}
}
