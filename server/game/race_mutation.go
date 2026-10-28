package game

import (
	"fmt"
	"slices"

	"ssg/model"
)

const (
	mutationPickBudget        = 14 // 10 base + 4 bonus
	mutationNegativePickLimit = 10 // Same as base
)

// validateMutateRaceOrder performs all read-only checks for a race mutation.
// Both SubmitTurn and ApplyRaceMutation call this so the rules cannot drift.
func (g *Game) validateMutateRaceOrder(player *model.Player, cmd model.MutateRaceCommand) (model.Race, error) {
	if player.HasUsedEvolutionaryMutation {
		return model.Race{}, fmt.Errorf("evolutionary mutation has already been used")
	}
	if player.EvolutionaryMutationPoints <= 0 {
		return model.Race{}, fmt.Errorf("no evolutionary mutation points available")
	}

	// Only trait_ids can change; name and government stay the same.
	mutationDesign := RaceDesign{
		Name:       player.Race.Name,
		Government: player.Race.Government,
		TraitIDs:   cmd.TraitIDs,
	}
	resourceManager := &ResourceManager{Resources: *g.Resources}
	mutatedRace, err := buildMutationRace(resourceManager, mutationDesign, player.Race)
	if err != nil {
		return model.Race{}, fmt.Errorf("invalid race mutation: %w", err)
	}
	return mutatedRace, nil
}

// ApplyRaceMutation validates and applies a race mutation command.
// The player can only change trait IDs, not the race name or government.
// The mutation uses 14 pick points (10 base + 4 bonus from Evolutionary Mutation).
// Mutation can only be used once per game.
func (g *Game) ApplyRaceMutation(playerID uint, cmd model.MutateRaceCommand) error {
	player, err := g.getPlayer(playerID)
	if err != nil {
		return err
	}

	mutatedRace, err := g.validateMutateRaceOrder(player, cmd)
	if err != nil {
		return err
	}

	player.Race = mutatedRace
	player.HasUsedEvolutionaryMutation = true
	g.hydratePlayerAvailability(player)
	return nil
}

// buildMutationRace builds a race for mutation, using 14 pick points and
// keeping the original race name and government.
func buildMutationRace(resources *ResourceManager, design RaceDesign, originalRace model.Race) (model.Race, error) {
	byID := raceTraitDefinitionsByID()
	seen := map[string]bool{}
	exclusiveGroups := map[string]string{}

	// Start with base government cost
	government, ok := findGovernmentTech(&resources.Resources, design.Government)
	if !ok {
		return model.Race{}, fmt.Errorf("unknown government %q", design.Government)
	}

	totalCost := government.PickCost
	negativeCost := max(0, -government.PickCost)

	race := model.Race{
		Name:       originalRace.Name,       // Keep original name
		Government: originalRace.Government, // Keep original government
		TraitIDs:   make([]string, 0, len(design.TraitIDs)),
		Grants:     model.Grants{}, // Will be rebuilt
		Stats:      model.RaceStats{},
	}

	for _, traitID := range design.TraitIDs {
		def, ok := byID[traitID]
		if !ok {
			return model.Race{}, fmt.Errorf("unknown race trait %q", traitID)
		}
		if seen[traitID] {
			return model.Race{}, fmt.Errorf("duplicate race trait %q", traitID)
		}
		if !def.Enabled {
			return model.Race{}, fmt.Errorf("race trait %q is not implemented yet", traitID)
		}
		if def.ExclusiveGroup != "" {
			if previous, ok := exclusiveGroups[def.ExclusiveGroup]; ok {
				return model.Race{}, fmt.Errorf("race traits %q and %q are mutually exclusive", previous, traitID)
			}
			exclusiveGroups[def.ExclusiveGroup] = traitID
		}
		for _, incompatible := range def.Incompatible {
			if seen[incompatible] {
				return model.Race{}, fmt.Errorf("race traits %q and %q are incompatible", traitID, incompatible)
			}
		}

		seen[traitID] = true
		totalCost += def.PickCost
		if def.PickCost < 0 {
			negativeCost += -def.PickCost
		}
		race.TraitIDs = append(race.TraitIDs, traitID)
		race.Grants.Modifiers = append(race.Grants.Modifiers, def.Grants.Modifiers...)
		race.Stats.PopulationGrowthPercent += def.Stats.PopulationGrowthPercent
		race.Stats.Farming += def.Stats.Farming
		race.Stats.Production += def.Stats.Production
		race.Stats.Research += def.Stats.Research
		race.Stats.MoneyBonus += def.Stats.MoneyBonus
		race.Stats.ShipAttackBonusPercent += def.Stats.ShipAttackBonusPercent
		race.Stats.ShipDefenseBonusPercent += def.Stats.ShipDefenseBonusPercent
		race.Stats.SpyBonus += def.Stats.SpyBonus
		race.Stats.DetectionModifier += def.Stats.DetectionModifier
		race.Stats.CombatSpeedBonus += def.Stats.CombatSpeedBonus
	}

	if totalCost > mutationPickBudget {
		return model.Race{}, fmt.Errorf("mutation overspends picks by %d", totalCost-mutationPickBudget)
	}
	if negativeCost > mutationNegativePickLimit {
		return model.Race{}, fmt.Errorf("mutation uses %d negative picks; maximum is %d", negativeCost, mutationNegativePickLimit)
	}

	return race, nil
}

// ValidateMutationDesign validates a race mutation design without applying it.
// Returns the remaining picks and any validation errors.
func ValidateMutationDesign(resources *ResourceManager, design RaceDesign, originalRace model.Race) (remainingPicks int, negativePicks int, errors []string) {
	byID := raceTraitDefinitionsByID()
	seen := map[string]bool{}
	exclusiveGroups := map[string]string{}

	// Start with base government cost
	government, ok := findGovernmentTech(&resources.Resources, design.Government)
	if !ok {
		return 0, 0, []string{fmt.Sprintf("unknown government %q", design.Government)}
	}

	totalCost := government.PickCost
	negativeCost := max(0, -government.PickCost)

	for _, traitID := range design.TraitIDs {
		def, ok := byID[traitID]
		if !ok {
			errors = append(errors, fmt.Sprintf("unknown race trait %q", traitID))
			continue
		}
		if seen[traitID] {
			errors = append(errors, fmt.Sprintf("duplicate race trait %q", traitID))
			continue
		}
		if !def.Enabled {
			errors = append(errors, fmt.Sprintf("race trait %q is not implemented yet", traitID))
			continue
		}
		if def.ExclusiveGroup != "" {
			if previous, ok := exclusiveGroups[def.ExclusiveGroup]; ok {
				errors = append(errors, fmt.Sprintf("race traits %q and %q are mutually exclusive", previous, traitID))
				continue
			}
			exclusiveGroups[def.ExclusiveGroup] = traitID
		}
		for _, incompatible := range def.Incompatible {
			if seen[incompatible] {
				errors = append(errors, fmt.Sprintf("race traits %q and %q are incompatible", traitID, incompatible))
				break
			}
		}

		seen[traitID] = true
		totalCost += def.PickCost
		if def.PickCost < 0 {
			negativeCost += -def.PickCost
		}
	}

	remainingPicks = mutationPickBudget - totalCost
	negativePicks = negativeCost

	if totalCost > mutationPickBudget {
		errors = append(errors, fmt.Sprintf("mutation overspends picks by %d", totalCost-mutationPickBudget))
	}
	if negativeCost > mutationNegativePickLimit {
		errors = append(errors, fmt.Sprintf("mutation uses %d negative picks; maximum is %d", negativeCost, mutationNegativePickLimit))
	}

	return remainingPicks, negativePicks, errors
}

// HasEvolutionaryMutation checks if the player has researched Evolutionary Mutation.
func HasEvolutionaryMutation(player *model.Player) bool {
	if player == nil {
		return false
	}
	return slices.Contains(player.ResearchedTechs, "Evolutionary Mutation")
}
