package game

import (
	"slices"
	. "ssg/model"
)

func Science(p *Planet) int64 {
	return p.Science.Scientists*p.Science.ProducedPerScientist + p.Science.ProducedPerBuilding
}

func (g *Game) selectResearch(player *Player, cmd SelectResearchCommand) error {
	// tech, ok := g.findTechByName(cmd.TechName)
	// if !ok {
	// 	return fmt.Errorf("tech %q does not exist", cmd.TechName)
	// }
	// if _, ok := nextResearchLevel(player, tech); !ok {
	// 	return fmt.Errorf("tech %q is already complete", tech.Name)
	// }
	// if player.CurrentResearch != nil && player.CurrentResearch.Subject == tech.Name {
	// 	return nil
	// }

	// player.CurrentResearch = &ResearchProject{
	// 	Subject:  tech.Name,
	// 	Progress: 0,
	// }
	return nil
}

func (g *Game) research(player *Player, techName string) {
	researchBuilding(g, player, techName)
}

func researchBuilding(g *Game, player *Player, techName string) {
	if hasAvailableBuilding(player, techName) {
		return
	}
	if _, err := getBuildingFromName(g, techName); err != nil {
		return
	}
	player.Available.Buildings = append(player.Available.Buildings, techName)
}

func hasAvailableBuilding(player *Player, buildingName string) bool {
	return slices.Contains(player.Available.Buildings, buildingName)
}
