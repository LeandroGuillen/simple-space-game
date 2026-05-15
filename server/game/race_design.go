package game

import (
	"fmt"
	"strings"

	"ssg/model"
)

const (
	racePickBudget        = 10
	raceNegativePickLimit = 10
)

type RaceTraitDefinition struct {
	ID             string
	Name           string
	Category       string
	PickCost       int
	ExclusiveGroup string
	Incompatible   []string
	Enabled        bool
	Description    string
	Grants         model.Grants
	Stats          model.RaceStats
	Homeworld      *RaceHomeworldModifier
}

type RaceHomeworldModifier struct {
	Size                model.PlanetSize
	MineralClass        model.PlanetMineral
	SciencePerScientist int64
}

type CustomRaceDesign struct {
	Name       string   `json:"name"`
	Government string   `json:"government"`
	TraitIDs   []string `json:"trait_ids"`
	Playable   bool     `json:"playable"`
}

var raceTraitDefinitions = []RaceTraitDefinition{
	{ID: "growth_minus_50", Name: "-50% Growth", Category: "Population", PickCost: -4, ExclusiveGroup: "growth", Enabled: true, Stats: model.RaceStats{PopulationGrowthPercent: -50}, Description: "Colonies grow at half speed."},
	{ID: "growth_plus_50", Name: "+50% Growth", Category: "Population", PickCost: 3, ExclusiveGroup: "growth", Enabled: true, Stats: model.RaceStats{PopulationGrowthPercent: 50}, Description: "Colonies grow 50% faster."},
	{ID: "growth_plus_100", Name: "+100% Growth", Category: "Population", PickCost: 6, ExclusiveGroup: "growth", Enabled: true, Stats: model.RaceStats{PopulationGrowthPercent: 100}, Description: "Colonies grow twice as fast."},
	{ID: "food_minus_1", Name: "-1/2 Food", Category: "Farming", PickCost: -3, ExclusiveGroup: "food", Enabled: true, Grants: model.Grants{Modifiers: []model.Modifier{{Name: "food_per_farmer", Value: -1}}}, Stats: model.RaceStats{Farming: -1}, Description: "Farmers produce less food."},
	{ID: "food_plus_1", Name: "+1 Food", Category: "Farming", PickCost: 4, ExclusiveGroup: "food", Enabled: true, Grants: model.Grants{Modifiers: []model.Modifier{{Name: "food_per_farmer", Value: 1}}}, Stats: model.RaceStats{Farming: 1}, Description: "Farmers produce one extra food."},
	{ID: "food_plus_2", Name: "+2 Food", Category: "Farming", PickCost: 7, ExclusiveGroup: "food", Enabled: true, Grants: model.Grants{Modifiers: []model.Modifier{{Name: "food_per_farmer", Value: 2}}}, Stats: model.RaceStats{Farming: 2}, Description: "Farmers produce two extra food."},
	{ID: "production_minus_1", Name: "-1 Production", Category: "Production", PickCost: -3, ExclusiveGroup: "production", Enabled: true, Grants: model.Grants{Modifiers: []model.Modifier{{Name: "industry_per_worker", Value: -1}}}, Stats: model.RaceStats{Production: -1}, Description: "Workers produce less industry."},
	{ID: "production_plus_1", Name: "+1 Production", Category: "Production", PickCost: 3, ExclusiveGroup: "production", Enabled: true, Grants: model.Grants{Modifiers: []model.Modifier{{Name: "industry_per_worker", Value: 1}}}, Stats: model.RaceStats{Production: 1}, Description: "Workers produce one extra industry."},
	{ID: "production_plus_2", Name: "+2 Production", Category: "Production", PickCost: 6, ExclusiveGroup: "production", Enabled: true, Grants: model.Grants{Modifiers: []model.Modifier{{Name: "industry_per_worker", Value: 2}}}, Stats: model.RaceStats{Production: 2}, Description: "Workers produce two extra industry."},
	{ID: "research_minus_1", Name: "-1 Research", Category: "Research", PickCost: -3, ExclusiveGroup: "research", Enabled: true, Grants: model.Grants{Modifiers: []model.Modifier{{Name: "science_per_scientist", Value: -1}}}, Stats: model.RaceStats{Research: -1}, Description: "Scientists produce less research."},
	{ID: "research_plus_1", Name: "+1 Research", Category: "Research", PickCost: 3, ExclusiveGroup: "research", Enabled: true, Grants: model.Grants{Modifiers: []model.Modifier{{Name: "science_per_scientist", Value: 1}}}, Stats: model.RaceStats{Research: 1}, Description: "Scientists produce one extra research."},
	{ID: "research_plus_2", Name: "+2 Research", Category: "Research", PickCost: 6, ExclusiveGroup: "research", Enabled: true, Grants: model.Grants{Modifiers: []model.Modifier{{Name: "science_per_scientist", Value: 2}}}, Stats: model.RaceStats{Research: 2}, Description: "Scientists produce two extra research."},
	{ID: "money_minus_half", Name: "-0.5 BC", Category: "Money", PickCost: -4, ExclusiveGroup: "money", Enabled: true, Grants: model.Grants{Modifiers: []model.Modifier{{Name: "income_bonus", Value: -0.5}}}, Stats: model.RaceStats{MoneyBonus: -0.5}, Description: "Taxable income is reduced."},
	{ID: "money_plus_half", Name: "+0.5 BC", Category: "Money", PickCost: 5, ExclusiveGroup: "money", Enabled: true, Grants: model.Grants{Modifiers: []model.Modifier{{Name: "income_bonus", Value: 0.5}}}, Stats: model.RaceStats{MoneyBonus: 0.5}, Description: "Taxable income is increased."},
	{ID: "money_plus_1", Name: "+1 BC", Category: "Money", PickCost: 8, ExclusiveGroup: "money", Enabled: true, Grants: model.Grants{Modifiers: []model.Modifier{{Name: "income_bonus", Value: 1}}}, Stats: model.RaceStats{MoneyBonus: 1}, Description: "Taxable income is strongly increased."},
	{ID: "subterranean", Name: "Subterranean", Category: "Homeworld", PickCost: 6, ExclusiveGroup: "capacity", Enabled: true, Description: "Planet population capacity is doubled."},
	{ID: "large_homeworld", Name: "Large Home World", Category: "Homeworld", PickCost: 1, ExclusiveGroup: "homeworld_size", Enabled: true, Homeworld: &RaceHomeworldModifier{Size: model.PlanetSizeLarge}, Description: "The capital starts as a large world."},
	{ID: "rich_homeworld", Name: "Rich Home World", Category: "Homeworld", PickCost: 2, ExclusiveGroup: "homeworld_minerals", Enabled: true, Homeworld: &RaceHomeworldModifier{MineralClass: model.PlanetMineralRich}, Description: "The capital starts rich."},
	{ID: "poor_homeworld", Name: "Poor Home World", Category: "Homeworld", PickCost: -1, ExclusiveGroup: "homeworld_minerals", Enabled: true, Homeworld: &RaceHomeworldModifier{MineralClass: model.PlanetMineralPoor}, Description: "The capital starts poor."},
	{ID: "artifacts_homeworld", Name: "Artifacts World", Category: "Homeworld", PickCost: 3, ExclusiveGroup: "homeworld_artifacts", Enabled: true, Homeworld: &RaceHomeworldModifier{SciencePerScientist: 2}, Description: "The capital produces extra research."},
	{ID: "ship_defense", Name: "Ship Defense", Category: "Future Traits", PickCost: 0, Enabled: false, Description: "Requires combat."},
	{ID: "ship_attack", Name: "Ship Attack", Category: "Future Traits", PickCost: 0, Enabled: false, Description: "Requires combat."},
	{ID: "ground_combat", Name: "Ground Combat", Category: "Future Traits", PickCost: 0, Enabled: false, Description: "Requires ground combat."},
	{ID: "spying", Name: "Spying", Category: "Future Traits", PickCost: 0, Enabled: false, Description: "Requires espionage."},
	{ID: "low_g_world", Name: "Low-G World", Category: "Future Traits", PickCost: -5, Enabled: false, Description: "Requires gravity rules."},
	{ID: "high_g_world", Name: "High-G World", Category: "Future Traits", PickCost: 6, Enabled: false, Description: "Requires gravity rules."},
	{ID: "aquatic", Name: "Aquatic", Category: "Future Traits", PickCost: 5, Enabled: false, Description: "Requires aquatic planet capacity rules."},
	{ID: "cybernetic", Name: "Cybernetic", Category: "Future Traits", PickCost: 4, Enabled: false, Description: "Requires food consumption rules."},
	{ID: "lithovore", Name: "Lithovore", Category: "Future Traits", PickCost: 10, Enabled: false, Description: "Requires food consumption rules."},
	{ID: "repulsive", Name: "Repulsive", Category: "Future Traits", PickCost: -6, Enabled: false, Description: "Requires diplomacy."},
	{ID: "charismatic", Name: "Charismatic", Category: "Future Traits", PickCost: 3, Enabled: false, Description: "Requires diplomacy."},
	{ID: "uncreative", Name: "Uncreative", Category: "Future Traits", PickCost: -4, Enabled: false, Description: "Requires research choice rules."},
	{ID: "creative", Name: "Creative", Category: "Future Traits", PickCost: 8, Enabled: false, Description: "Requires research choice rules."},
	{ID: "tolerant", Name: "Tolerant", Category: "Future Traits", PickCost: 10, Enabled: false, Description: "Requires pollution and hostile environment rules."},
	{ID: "fantastic_traders", Name: "Fantastic Traders", Category: "Future Traits", PickCost: 4, Enabled: false, Description: "Requires trade rules."},
	{ID: "telepathic", Name: "Telepathic", Category: "Future Traits", PickCost: 6, Enabled: false, Description: "Requires diplomacy and conquest rules."},
	{ID: "lucky", Name: "Lucky", Category: "Future Traits", PickCost: 3, Enabled: false, Description: "Requires random events."},
	{ID: "omniscient", Name: "Omniscient", Category: "Future Traits", PickCost: 3, Enabled: false, Description: "Requires map intelligence rules."},
	{ID: "stealthy_ships", Name: "Stealthy Ships", Category: "Future Traits", PickCost: 4, Enabled: false, Description: "Requires detection rules."},
	{ID: "trans_dimensional", Name: "Trans-Dimensional", Category: "Future Traits", PickCost: 5, Enabled: false, Description: "Requires movement rules."},
	{ID: "warlord", Name: "Warlord", Category: "Future Traits", PickCost: 4, Enabled: false, Description: "Requires crew experience rules."},
}

func DefaultRace(name string) model.Race {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Human"
	}
	return model.Race{Name: name, Government: "Dictatorship"}
}

func BuildCustomRace(resources *ResourceManager, design CustomRaceDesign) (model.Race, error) {
	return buildCustomRace(resources, design, false)
}

func SaveableCustomRace(resources *ResourceManager, design CustomRaceDesign) (CustomRaceDesign, error) {
	race, err := buildCustomRace(resources, design, true)
	if err != nil {
		return CustomRaceDesign{}, err
	}
	return CustomRaceDesign{
		Name:       race.Name,
		Government: race.Government,
		TraitIDs:   append([]string{}, race.TraitIDs...),
		Playable:   racePlayable(resources, race.Government, race.TraitIDs),
	}, nil
}

func buildCustomRace(resources *ResourceManager, design CustomRaceDesign, allowUnplayable bool) (model.Race, error) {
	name := strings.TrimSpace(design.Name)
	if name == "" {
		return model.Race{}, fmt.Errorf("custom race name is required")
	}
	if len(name) > 32 {
		return model.Race{}, fmt.Errorf("custom race name cannot exceed 32 characters")
	}
	if resources == nil {
		return model.Race{}, fmt.Errorf("resources are required")
	}

	governmentName := strings.TrimSpace(design.Government)
	if governmentName == "" {
		governmentName = defaultGovernmentName(&resources.Resources)
	}
	government, ok := findGovernmentTech(&resources.Resources, governmentName)
	if !ok {
		return model.Race{}, fmt.Errorf("unknown government %q", governmentName)
	}
	if !isBaseGovernment(government) {
		return model.Race{}, fmt.Errorf("government %q is not available at race creation", governmentName)
	}

	byID := raceTraitDefinitionsByID()
	seen := map[string]bool{}
	exclusiveGroups := map[string]string{}
	totalCost := government.PickCost
	negativeCost := max(0, -government.PickCost)
	race := model.Race{Name: name, Government: government.Name, TraitIDs: make([]string, 0, len(design.TraitIDs))}

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
	}

	if !allowUnplayable && totalCost > racePickBudget {
		return model.Race{}, fmt.Errorf("custom race overspends picks by %d", totalCost-racePickBudget)
	}
	if !allowUnplayable && negativeCost > raceNegativePickLimit {
		return model.Race{}, fmt.Errorf("custom race uses %d negative picks; maximum is %d", negativeCost, raceNegativePickLimit)
	}

	return race, nil
}

func racePlayable(resources *ResourceManager, governmentName string, traitIDs []string) bool {
	if resources == nil {
		return false
	}
	government, ok := findGovernmentTech(&resources.Resources, governmentName)
	if !ok || !isBaseGovernment(government) {
		return false
	}
	byID := raceTraitDefinitionsByID()
	totalCost := government.PickCost
	negativeCost := max(0, -government.PickCost)
	for _, traitID := range traitIDs {
		def, ok := byID[traitID]
		if !ok || !def.Enabled {
			return false
		}
		totalCost += def.PickCost
		if def.PickCost < 0 {
			negativeCost += -def.PickCost
		}
	}
	return totalCost <= racePickBudget && negativeCost <= raceNegativePickLimit
}

func raceTraitDefinitionsByID() map[string]RaceTraitDefinition {
	byID := make(map[string]RaceTraitDefinition, len(raceTraitDefinitions))
	for _, def := range raceTraitDefinitions {
		byID[def.ID] = def
	}
	return byID
}

func playerHasRaceTrait(player *model.Player, traitID string) bool {
	if player == nil {
		return false
	}
	for _, id := range player.Race.TraitIDs {
		if id == traitID {
			return true
		}
	}
	return false
}
