package game

import (
	"fmt"
	"slices"
	. "ssg/model"
)

func Science(p *Planet) int64 {
	return p.Science.Scientists*p.Science.ProducedPerScientist + p.Science.ProducedPerBuilding
}

func (g *Game) selectResearch(player *Player, cmd SelectResearchCommand) error {
	level, ok := g.findResearchLevelForTech(cmd.TechName)
	if !ok {
		level, ok = g.findResearchLevel(cmd.TechName)
		if !ok {
			return fmt.Errorf("research breakthrough %q does not exist", cmd.TechName)
		}
	}
	if isResearchLevelComplete(player, level) {
		return fmt.Errorf("research level %q is already complete", level.Name)
	}
	if level.ForceAll {
		if player.CurrentResearch != nil && player.CurrentResearch.LevelName == level.Name {
			return nil
		}
		player.CurrentResearch = &ResearchProject{
			Subject:   level.Name,
			TechNames: append([]string(nil), level.AvailableTechs...),
			LevelName: level.Name,
			Progress:  player.ResearchPoints,
		}
		return nil
	}

	if !slices.Contains(level.AvailableTechs, cmd.TechName) {
		return fmt.Errorf("tech %q is not selectable in research level %q", cmd.TechName, level.Name)
	}
	if isTechKnown(player, cmd.TechName) {
		return fmt.Errorf("tech %q is already known", cmd.TechName)
	}
	if player.CurrentResearch != nil && player.CurrentResearch.TechName == cmd.TechName {
		return nil
	}

	player.CurrentResearch = &ResearchProject{
		Subject:   cmd.TechName,
		TechName:  cmd.TechName,
		TechNames: []string{cmd.TechName},
		LevelName: level.Name,
		Progress:  player.ResearchPoints,
	}
	return nil
}

func (g *Game) applyResearch(player *Player, generatedResearch uint64) error {
	player.ResearchPoints += generatedResearch
	if player.CurrentResearch == nil {
		return nil
	}

	level, ok := g.findResearchLevel(player.CurrentResearch.LevelName)
	if !ok {
		return fmt.Errorf("research level %q does not exist", player.CurrentResearch.LevelName)
	}

	player.CurrentResearch.Progress = player.ResearchPoints
	if player.ResearchPoints < level.Cost {
		return nil
	}

	for _, techName := range selectedResearchTechs(player.CurrentResearch) {
		researchTech(g, player, techName)
	}
	player.ResearchPoints -= level.Cost
	player.CurrentResearch = nil
	return nil
}

func (g *Game) research(player *Player, techName string) {
	researchTech(g, player, techName)
}

func researchTech(g *Game, player *Player, techName string) {
	if !slices.Contains(player.KnownTechs, techName) {
		player.KnownTechs = append(player.KnownTechs, techName)
	}
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

func (g *Game) findResearchLevel(name string) (TechLevel, bool) {
	for _, subject := range g.Resources.TechTree.Subjects {
		for _, level := range subject.Levels {
			if level.Name == name {
				return level, true
			}
		}
	}
	return TechLevel{}, false
}

func (g *Game) findResearchLevelForTech(techName string) (TechLevel, bool) {
	for _, subject := range g.Resources.TechTree.Subjects {
		for _, level := range subject.Levels {
			if slices.Contains(level.AvailableTechs, techName) {
				return level, true
			}
		}
	}
	return TechLevel{}, false
}

func isResearchLevelComplete(player *Player, level TechLevel) bool {
	if len(level.AvailableTechs) == 0 {
		return false
	}
	if !level.ForceAll {
		return slices.ContainsFunc(level.AvailableTechs, func(techName string) bool {
			return isTechKnown(player, techName)
		})
	}
	for _, techName := range level.AvailableTechs {
		if !isTechKnown(player, techName) {
			return false
		}
	}
	return true
}

func isTechKnown(player *Player, techName string) bool {
	return slices.Contains(player.KnownTechs, techName) || hasAvailableBuilding(player, techName)
}

func selectedResearchTechs(project *ResearchProject) []string {
	if len(project.TechNames) > 0 {
		return project.TechNames
	}
	if project.TechName != "" {
		return []string{project.TechName}
	}
	return nil
}
