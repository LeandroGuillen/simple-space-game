package game

import (
	"slices"
	. "ssg/model"
)

func (g *Game) hydratePlayerAvailability(player *Player) {
	available := AvailableTech{}
	g.applyAvailableUnlocks(player, &available, player.Race.Grants.Unlocks)
	for _, techName := range player.ResearchedTechs {
		if tech, ok := g.findTechDefinition(techName); ok {
			g.applyAvailableUnlocks(player, &available, tech.Grants.Unlocks)
		}
	}
	player.Available = available
}

func (g *Game) applyAvailableUnlocks(player *Player, available *AvailableTech, unlocks Unlocks) {
	if !governmentMoraleNeutral(g, player) {
		applyUnlocks(available, unlocks)
		return
	}
	filtered := unlocks
	filtered.Buildings = make([]string, 0, len(unlocks.Buildings))
	for _, buildingName := range unlocks.Buildings {
		building, err := getBuildingFromName(g, buildingName)
		if err == nil && buildingHasMoraleModifier(building) {
			continue
		}
		filtered.Buildings = append(filtered.Buildings, buildingName)
	}
	applyUnlocks(available, filtered)
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
