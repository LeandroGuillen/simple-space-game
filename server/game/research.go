package game

import (
	"fmt"
	"slices"
	"strings"

	. "ssg/model"
)

const hyperAdvancedResearchPrefix = "Hyper Advanced "

func Science(p *Planet) int64 {
	return p.Science.Scientists*p.Science.ProducedPerScientist + p.Science.ProducedPerBuilding
}

func (g *Game) selectResearch(player *Player, cmd SelectResearchCommand) error {
	if subjectName, ok := parseHyperAdvancedResearch(cmd.TechName); ok {
		return g.selectHyperAdvancedResearch(player, subjectName)
	}

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
	if isTechResearched(player, cmd.TechName) {
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

	if subjectName, ok := parseHyperAdvancedResearch(player.CurrentResearch.TechName); ok {
		cost, ok := g.hyperAdvancedResearchCost(player, subjectName)
		if !ok {
			return fmt.Errorf("hyper advanced subject %q does not exist", subjectName)
		}
		player.CurrentResearch.Progress = player.ResearchPoints
		if generatedResearch == 0 {
			return nil
		}
		if player.ResearchPoints < cost {
			return nil
		}
		if player.HyperAdvanced == nil {
			player.HyperAdvanced = map[string]uint64{}
		}
		player.HyperAdvanced[subjectName]++
		player.ResearchPoints -= cost
		player.CurrentResearch = nil
		return nil
	}

	level, ok := g.findResearchLevel(player.CurrentResearch.LevelName)
	if !ok {
		return fmt.Errorf("research level %q does not exist", player.CurrentResearch.LevelName)
	}

	player.CurrentResearch.Progress = player.ResearchPoints
	if generatedResearch == 0 {
		return nil
	}
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

func (g *Game) ResearchAllTechs(playerID uint) error {
	player, err := g.getPlayer(playerID)
	if err != nil {
		return err
	}
	for _, tech := range g.Resources.TechTree.AllTechs {
		researchTech(g, player, tech.Name)
	}
	player.CurrentResearch = nil
	if player.HyperAdvanced == nil {
		player.HyperAdvanced = map[string]uint64{}
	}
	return nil
}

func researchTech(g *Game, player *Player, techName string) {
	if !slices.Contains(player.ResearchedTechs, techName) {
		player.ResearchedTechs = append(player.ResearchedTechs, techName)
	}
	g.hydratePlayerAvailability(player)
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
			return isTechResearched(player, techName)
		})
	}
	for _, techName := range level.AvailableTechs {
		if !isTechResearched(player, techName) {
			return false
		}
	}
	return true
}

func isTechResearched(player *Player, techName string) bool {
	return slices.Contains(player.ResearchedTechs, techName)
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

func (g *Game) findTechDefinition(techName string) (Tech, bool) {
	for _, tech := range g.Resources.TechTree.AllTechs {
		if tech.Name == techName {
			return tech, true
		}
	}
	return Tech{}, false
}

func (g *Game) selectHyperAdvancedResearch(player *Player, subjectName string) error {
	subject, ok := g.findResearchSubject(subjectName)
	if !ok {
		return fmt.Errorf("hyper advanced subject %q does not exist", subjectName)
	}
	if !isResearchSubjectComplete(player, subject) {
		return fmt.Errorf("hyper advanced research requires completed subject %q", subjectName)
	}
	if player.CurrentResearch != nil && player.CurrentResearch.TechName == hyperAdvancedResearchName(subjectName) {
		return nil
	}
	player.CurrentResearch = &ResearchProject{
		Subject:   subjectName,
		TechName:  hyperAdvancedResearchName(subjectName),
		TechNames: []string{},
		LevelName: g.hyperAdvancedResearchLevelName(player, subjectName),
		Progress:  player.ResearchPoints,
	}
	return nil
}

func (g *Game) hyperAdvancedResearchCost(player *Player, subjectName string) (uint64, bool) {
	subject, ok := g.findResearchSubject(subjectName)
	if !ok || len(subject.Levels) == 0 {
		return 0, false
	}
	lastLevelCost := subject.Levels[len(subject.Levels)-1].Cost
	completedLevels := player.HyperAdvanced[subjectName]
	return lastLevelCost << (completedLevels + 1), true
}

func (g *Game) hyperAdvancedResearchLevelName(player *Player, subjectName string) string {
	return fmt.Sprintf("Hyper Advanced %d", player.HyperAdvanced[subjectName]+1)
}

func hyperAdvancedResearchName(subjectName string) string {
	return hyperAdvancedResearchPrefix + subjectName
}

func parseHyperAdvancedResearch(techName string) (string, bool) {
	if !strings.HasPrefix(techName, hyperAdvancedResearchPrefix) {
		return "", false
	}
	subjectName := strings.TrimSpace(strings.TrimPrefix(techName, hyperAdvancedResearchPrefix))
	return subjectName, subjectName != ""
}

func (g *Game) findResearchSubject(name string) (TechSubject, bool) {
	for _, subject := range g.Resources.TechTree.Subjects {
		if subject.Name == name {
			return subject, true
		}
	}
	return TechSubject{}, false
}

func isResearchSubjectComplete(player *Player, subject TechSubject) bool {
	if len(subject.Levels) == 0 {
		return false
	}
	for _, level := range subject.Levels {
		if !isResearchLevelComplete(player, level) {
			return false
		}
	}
	return true
}
