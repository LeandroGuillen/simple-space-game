package game

import (
	"fmt"
	"slices"
	"strings"

	. "ssg/model"
)

const hyperAdvancedResearchPrefix = "Hyper Advanced "

func Science(p *Planet) int64 {
	return PlanetScienceProduced(p)
}

// selectResearchPlan holds the read-only resolution of a select-research order
// so submit-time validation and apply-time mutation share one rule set.
type selectResearchPlan struct {
	noop         bool
	hyperSubject string
	level        TechLevel
	unlockAll    bool
}

// validateSelectResearchOrder performs all read-only checks for selecting research.
// Both SubmitTurn and selectResearch call this so the rules cannot drift.
func (g *Game) validateSelectResearchOrder(player *Player, cmd SelectResearchCommand) (selectResearchPlan, error) {
	if subjectName, ok := parseHyperAdvancedResearch(cmd.TechName); ok {
		if err := g.validateHyperAdvancedResearchOrder(player, subjectName); err != nil {
			return selectResearchPlan{}, err
		}
		if player.CurrentResearch != nil && player.CurrentResearch.TechName == hyperAdvancedResearchName(subjectName) {
			return selectResearchPlan{noop: true, hyperSubject: subjectName}, nil
		}
		return selectResearchPlan{hyperSubject: subjectName}, nil
	}

	level, ok := g.findResearchLevelForTech(cmd.TechName)
	if !ok {
		level, ok = g.findResearchLevel(cmd.TechName)
		if !ok {
			return selectResearchPlan{}, fmt.Errorf("research breakthrough %q does not exist", cmd.TechName)
		}
	}
	if isResearchLevelComplete(player, level) {
		return selectResearchPlan{}, fmt.Errorf("research level %q is already complete", level.Name)
	}

	subject, levelIdx, ok := g.findSubjectAndLevelIndex(level.Name)
	if !ok {
		return selectResearchPlan{}, fmt.Errorf("research level %q does not belong to any subject", level.Name)
	}
	if !arePreviousLevelsComplete(player, subject, levelIdx) {
		if levelIdx > 0 {
			return selectResearchPlan{}, fmt.Errorf("research level %q requires %q to be completed first", level.Name, subject.Levels[levelIdx-1].Name)
		}
	}

	unlockAll := researchLevelUnlocksAllTechs(level, player)
	if tech, ok := g.findTechDefinition(cmd.TechName); ok {
		if err := g.validateResearchTechForGovernment(player, tech); err != nil {
			return selectResearchPlan{}, err
		}
	}
	if unlockAll {
		if player.CurrentResearch != nil && player.CurrentResearch.LevelName == level.Name {
			return selectResearchPlan{noop: true, level: level, unlockAll: true}, nil
		}
		for _, techName := range level.AvailableTechs {
			tech, ok := g.findTechDefinition(techName)
			if !ok {
				continue
			}
			if err := g.validateResearchTechForGovernment(player, tech); err != nil {
				return selectResearchPlan{}, err
			}
		}
		return selectResearchPlan{level: level, unlockAll: true}, nil
	}

	if !slices.Contains(level.AvailableTechs, cmd.TechName) {
		return selectResearchPlan{}, fmt.Errorf("tech %q is not selectable in research level %q", cmd.TechName, level.Name)
	}
	if isTechResearched(player, cmd.TechName) {
		return selectResearchPlan{}, fmt.Errorf("tech %q is already known", cmd.TechName)
	}
	if player.CurrentResearch != nil && player.CurrentResearch.TechName == cmd.TechName {
		return selectResearchPlan{noop: true, level: level}, nil
	}
	return selectResearchPlan{level: level}, nil
}

func (g *Game) validateHyperAdvancedResearchOrder(player *Player, subjectName string) error {
	subject, ok := g.findResearchSubject(subjectName)
	if !ok {
		return fmt.Errorf("hyper advanced subject %q does not exist", subjectName)
	}
	if !isResearchSubjectComplete(player, subject) {
		return fmt.Errorf("hyper advanced research requires completed subject %q", subjectName)
	}
	return nil
}

func (g *Game) selectResearch(player *Player, cmd SelectResearchCommand) error {
	plan, err := g.validateSelectResearchOrder(player, cmd)
	if err != nil {
		return err
	}
	if plan.noop {
		return nil
	}
	if plan.hyperSubject != "" {
		player.CurrentResearch = &ResearchProject{
			Subject:   plan.hyperSubject,
			TechName:  hyperAdvancedResearchName(plan.hyperSubject),
			TechNames: []string{},
			LevelName: g.hyperAdvancedResearchLevelName(player, plan.hyperSubject),
			Progress:  player.ResearchPoints,
		}
		return nil
	}
	if plan.unlockAll {
		player.CurrentResearch = &ResearchProject{
			Subject:   plan.level.Name,
			TechNames: append([]string(nil), plan.level.AvailableTechs...),
			LevelName: plan.level.Name,
			Progress:  player.ResearchPoints,
		}
		return nil
	}
	player.CurrentResearch = &ResearchProject{
		Subject:   cmd.TechName,
		TechName:  cmd.TechName,
		TechNames: []string{cmd.TechName},
		LevelName: plan.level.Name,
		Progress:  player.ResearchPoints,
	}
	return nil
}

func (g *Game) validateResearchTechForGovernment(player *Player, tech Tech) error {
	if isGovernmentTech(tech) {
		if isBaseGovernment(tech) {
			return fmt.Errorf("base government %q cannot be selected as research", tech.Name)
		}
		active, ok := activeGovernmentTech(g, player)
		if !ok || !isGovernmentUpgradeAllowed(player, tech, active) {
			activeName := ""
			if ok {
				activeName = active.Name
			}
			return fmt.Errorf("advanced government %q does not upgrade current government %q", tech.Name, activeName)
		}
		return nil
	}
	if governmentMoraleNeutral(g, player) && techUnlocksMoraleBuilding(g.Resources, tech) {
		return fmt.Errorf("government %q cannot research morale building tech %q", player.Race.Government, tech.Name)
	}
	return nil
}

// validateCurrentResearchProject checks that player.CurrentResearch is a
// project applyResearch can make progress on: known hyper-advanced subject,
// or known level with prior levels complete and selected techs in that level.
// Shared by applyResearch and ValidateLoadedGame so the rules cannot drift.
func (g *Game) validateCurrentResearchProject(player *Player) error {
	if player == nil || player.CurrentResearch == nil {
		return nil
	}
	proj := player.CurrentResearch

	if subjectName, ok := parseHyperAdvancedResearch(proj.TechName); ok {
		if _, ok := g.hyperAdvancedResearchCost(player, subjectName); !ok {
			return fmt.Errorf("hyper advanced subject %q does not exist", subjectName)
		}
		return nil
	}

	level, ok := g.findResearchLevel(proj.LevelName)
	if !ok {
		return fmt.Errorf("research level %q does not exist", proj.LevelName)
	}

	subject, levelIdx, ok := g.findSubjectAndLevelIndex(proj.LevelName)
	if !ok {
		return fmt.Errorf("research level %q does not belong to any subject", proj.LevelName)
	}
	if !arePreviousLevelsComplete(player, subject, levelIdx) {
		if levelIdx > 0 {
			return fmt.Errorf("research level %q requires %q to be completed first", level.Name, subject.Levels[levelIdx-1].Name)
		}
	}

	for _, techName := range selectedResearchTechs(proj) {
		if !slices.Contains(level.AvailableTechs, techName) {
			return fmt.Errorf("tech %q is not selectable in research level %q", techName, level.Name)
		}
	}
	return nil
}

func (g *Game) applyResearch(player *Player, generatedResearch uint64) error {
	player.ResearchPoints += generatedResearch
	if player.CurrentResearch == nil {
		return nil
	}

	if err := g.validateCurrentResearchProject(player); err != nil {
		return err
	}

	if subjectName, ok := parseHyperAdvancedResearch(player.CurrentResearch.TechName); ok {
		cost, _ := g.hyperAdvancedResearchCost(player, subjectName)
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

	level, _ := g.findResearchLevel(player.CurrentResearch.LevelName)

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
	g.ensureMu()
	g.mu.Lock()
	defer g.mu.Unlock()
	player, err := g.getPlayer(playerID)
	if err != nil {
		return err
	}
	// Append all techs first, then hydrate availability once. Hydrating per-tech
	// would be O(M^3): each researchTech rescans the growing ResearchedTechs list,
	// and each lookup linearly scans AllTechs.
	known := make(map[string]struct{}, len(player.ResearchedTechs)+len(g.Resources.TechTree.AllTechs))
	for _, name := range player.ResearchedTechs {
		known[name] = struct{}{}
	}
	hadEvolutionaryMutation := slices.Contains(player.ResearchedTechs, "Evolutionary Mutation")
	for _, tech := range g.Resources.TechTree.AllTechs {
		if _, ok := known[tech.Name]; ok {
			continue
		}
		known[tech.Name] = struct{}{}
		player.ResearchedTechs = append(player.ResearchedTechs, tech.Name)
	}
	// Award mutation points if Evolutionary Mutation was newly researched
	if !hadEvolutionaryMutation && slices.Contains(player.ResearchedTechs, "Evolutionary Mutation") {
		player.EvolutionaryMutationPoints += 4
	}
	player.CurrentResearch = nil
	if player.HyperAdvanced == nil {
		player.HyperAdvanced = map[string]uint64{}
	}
	g.hydratePlayerAvailability(player)
	// Research-all is a debug-only mutation; persist the flag for saves/list.
	g.Debug = true
	return nil
}

func researchTech(g *Game, player *Player, techName string) {
	if !slices.Contains(player.ResearchedTechs, techName) {
		player.ResearchedTechs = append(player.ResearchedTechs, techName)
	}
	if techName == "Evolutionary Mutation" {
		player.EvolutionaryMutationPoints += 4
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

func researchLevelUnlocksAllTechs(level TechLevel, player *Player) bool {
	if playerHasRaceTrait(player, "uncreative") {
		return false
	}
	return level.ForceAll || playerHasRaceTrait(player, "creative")
}

func isResearchLevelComplete(player *Player, level TechLevel) bool {
	if len(level.AvailableTechs) == 0 {
		return false
	}
	if !researchLevelUnlocksAllTechs(level, player) {
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
	return g.selectResearch(player, SelectResearchCommand{TechName: hyperAdvancedResearchName(subjectName)})
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

func (g *Game) findSubjectAndLevelIndex(levelName string) (TechSubject, int, bool) {
	for _, subject := range g.Resources.TechTree.Subjects {
		for idx, level := range subject.Levels {
			if level.Name == levelName {
				return subject, idx, true
			}
		}
	}
	return TechSubject{}, 0, false
}

func arePreviousLevelsComplete(player *Player, subject TechSubject, levelIdx int) bool {
	for i := 0; i < levelIdx; i++ {
		if !isResearchLevelComplete(player, subject.Levels[i]) {
			return false
		}
	}
	return true
}
