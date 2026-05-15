import type { ApiGovernment, CustomRaceDesign, RaceTraitId } from "./types/api";

export type RaceTraitDefinition = {
  id: string;
  name: string;
  category: string;
  pickCost: number;
  exclusiveGroup?: string;
  enabled: boolean;
  description: string;
};

export const RACE_PICK_BUDGET = 10;
export const RACE_NEGATIVE_PICK_LIMIT = 10;

export const DEFAULT_GOVERNMENT = "Dictatorship";

export const fallbackGovernments: ApiGovernment[] = [
  { name: "Feudal", description: "A decentralized noble hierarchy with cheap ship construction and weak research.", pick_cost: -4 },
  { name: "Dictatorship", description: "A centralized authoritarian government with stable internal security.", pick_cost: 0, default: true },
  { name: "Democracy", description: "A representative government with strong research and tax output but weaker counter-espionage.", pick_cost: 7 },
  { name: "Unification", description: "A unified collective with strong farming and industry that ignores morale.", pick_cost: 6 },
];

export const raceTraitDefinitions: RaceTraitDefinition[] = [
  { id: "growth_minus_50", name: "-50% Growth", category: "Population", pickCost: -4, exclusiveGroup: "growth", enabled: true, description: "Colonies grow at half speed." },
  { id: "growth_plus_50", name: "+50% Growth", category: "Population", pickCost: 3, exclusiveGroup: "growth", enabled: true, description: "Colonies grow 50% faster." },
  { id: "growth_plus_100", name: "+100% Growth", category: "Population", pickCost: 6, exclusiveGroup: "growth", enabled: true, description: "Colonies grow twice as fast." },
  { id: "food_minus_1", name: "-1/2 Food", category: "Farming", pickCost: -3, exclusiveGroup: "food", enabled: true, description: "Farmers produce less food." },
  { id: "food_plus_1", name: "+1 Food", category: "Farming", pickCost: 4, exclusiveGroup: "food", enabled: true, description: "Farmers produce one extra food." },
  { id: "food_plus_2", name: "+2 Food", category: "Farming", pickCost: 7, exclusiveGroup: "food", enabled: true, description: "Farmers produce two extra food." },
  { id: "production_minus_1", name: "-1 Production", category: "Production", pickCost: -3, exclusiveGroup: "production", enabled: true, description: "Workers produce less industry." },
  { id: "production_plus_1", name: "+1 Production", category: "Production", pickCost: 3, exclusiveGroup: "production", enabled: true, description: "Workers produce one extra industry." },
  { id: "production_plus_2", name: "+2 Production", category: "Production", pickCost: 6, exclusiveGroup: "production", enabled: true, description: "Workers produce two extra industry." },
  { id: "research_minus_1", name: "-1 Research", category: "Research", pickCost: -3, exclusiveGroup: "research", enabled: true, description: "Scientists produce less research." },
  { id: "research_plus_1", name: "+1 Research", category: "Research", pickCost: 3, exclusiveGroup: "research", enabled: true, description: "Scientists produce one extra research." },
  { id: "research_plus_2", name: "+2 Research", category: "Research", pickCost: 6, exclusiveGroup: "research", enabled: true, description: "Scientists produce two extra research." },
  { id: "money_minus_half", name: "-0.5 BC", category: "Money", pickCost: -4, exclusiveGroup: "money", enabled: true, description: "Taxable income is reduced." },
  { id: "money_plus_half", name: "+0.5 BC", category: "Money", pickCost: 5, exclusiveGroup: "money", enabled: true, description: "Taxable income is increased." },
  { id: "money_plus_1", name: "+1 BC", category: "Money", pickCost: 8, exclusiveGroup: "money", enabled: true, description: "Taxable income is strongly increased." },
  { id: "subterranean", name: "Subterranean", category: "Homeworld", pickCost: 6, exclusiveGroup: "capacity", enabled: true, description: "Planet population capacity is doubled." },
  { id: "large_homeworld", name: "Large Home World", category: "Homeworld", pickCost: 1, exclusiveGroup: "homeworld_size", enabled: true, description: "The capital starts as a large world." },
  { id: "rich_homeworld", name: "Rich Home World", category: "Homeworld", pickCost: 2, exclusiveGroup: "homeworld_minerals", enabled: true, description: "The capital starts rich." },
  { id: "poor_homeworld", name: "Poor Home World", category: "Homeworld", pickCost: -1, exclusiveGroup: "homeworld_minerals", enabled: true, description: "The capital starts poor." },
  { id: "artifacts_homeworld", name: "Artifacts World", category: "Homeworld", pickCost: 3, exclusiveGroup: "homeworld_artifacts", enabled: true, description: "The capital produces extra research." },
  { id: "ship_defense", name: "Ship Defense", category: "Future Traits", pickCost: 0, enabled: false, description: "Requires combat." },
  { id: "ship_attack", name: "Ship Attack", category: "Future Traits", pickCost: 0, enabled: false, description: "Requires combat." },
  { id: "ground_combat", name: "Ground Combat", category: "Future Traits", pickCost: 0, enabled: false, description: "Requires ground combat." },
  { id: "spying", name: "Spying", category: "Future Traits", pickCost: 0, enabled: false, description: "Requires espionage." },
  { id: "low_g_world", name: "Low-G World", category: "Future Traits", pickCost: -5, enabled: false, description: "Requires gravity rules." },
  { id: "high_g_world", name: "High-G World", category: "Future Traits", pickCost: 6, enabled: false, description: "Requires gravity rules." },
  { id: "aquatic", name: "Aquatic", category: "Future Traits", pickCost: 5, enabled: false, description: "Requires aquatic planet capacity rules." },
  { id: "cybernetic", name: "Cybernetic", category: "Future Traits", pickCost: 4, enabled: false, description: "Requires food consumption rules." },
  { id: "lithovore", name: "Lithovore", category: "Future Traits", pickCost: 10, enabled: false, description: "Requires food consumption rules." },
  { id: "repulsive", name: "Repulsive", category: "Future Traits", pickCost: -6, enabled: false, description: "Requires diplomacy." },
  { id: "charismatic", name: "Charismatic", category: "Future Traits", pickCost: 3, enabled: false, description: "Requires diplomacy." },
  { id: "uncreative", name: "Uncreative", category: "Future Traits", pickCost: -4, enabled: false, description: "Requires research choice rules." },
  { id: "creative", name: "Creative", category: "Future Traits", pickCost: 8, enabled: false, description: "Requires research choice rules." },
  { id: "tolerant", name: "Tolerant", category: "Future Traits", pickCost: 10, enabled: false, description: "Requires pollution and hostile environment rules." },
  { id: "fantastic_traders", name: "Fantastic Traders", category: "Future Traits", pickCost: 4, enabled: false, description: "Requires trade rules." },
  { id: "telepathic", name: "Telepathic", category: "Future Traits", pickCost: 6, enabled: false, description: "Requires diplomacy and conquest rules." },
  { id: "lucky", name: "Lucky", category: "Future Traits", pickCost: 3, enabled: false, description: "Requires random events." },
  { id: "omniscient", name: "Omniscient", category: "Future Traits", pickCost: 3, enabled: false, description: "Requires map intelligence rules." },
  { id: "stealthy_ships", name: "Stealthy Ships", category: "Future Traits", pickCost: 4, enabled: false, description: "Requires detection rules." },
  { id: "trans_dimensional", name: "Trans-Dimensional", category: "Future Traits", pickCost: 5, enabled: false, description: "Requires movement rules." },
  { id: "warlord", name: "Warlord", category: "Future Traits", pickCost: 4, enabled: false, description: "Requires crew experience rules." },
];

export const raceTraitById = new Map(raceTraitDefinitions.map((trait) => [trait.id, trait]));

type TraitIdInput = readonly string[] | null | undefined;

function normalizeTraitIds(traitIds: TraitIdInput) {
  return Array.isArray(traitIds) ? traitIds : [];
}

export function selectedRaceTraits(traitIds: TraitIdInput) {
  return normalizeTraitIds(traitIds).map((id) => raceTraitById.get(id)).filter((trait): trait is RaceTraitDefinition => Boolean(trait));
}

export function racePickCost(traitIds: TraitIdInput) {
  return selectedRaceTraits(traitIds).reduce((total, trait) => total + trait.pickCost, 0);
}

export function governmentPickCost(government: string, governments: readonly ApiGovernment[] = fallbackGovernments) {
  return governments.find((item) => item.name === government)?.pick_cost ?? 0;
}

export function defaultGovernment(governments: readonly ApiGovernment[] = fallbackGovernments) {
  return governments.find((item) => item.default)?.name ?? DEFAULT_GOVERNMENT;
}

export function raceTotalPickCost(traitIds: TraitIdInput, government: string, governments: readonly ApiGovernment[] = fallbackGovernments) {
  return racePickCost(traitIds) + governmentPickCost(government || defaultGovernment(governments), governments);
}

export function raceRemainingPicks(traitIds: TraitIdInput, government = DEFAULT_GOVERNMENT, governments: readonly ApiGovernment[] = fallbackGovernments) {
  return RACE_PICK_BUDGET - raceTotalPickCost(traitIds, government, governments);
}

export function raceNegativePicks(traitIds: TraitIdInput, government = DEFAULT_GOVERNMENT, governments: readonly ApiGovernment[] = fallbackGovernments) {
  return selectedRaceTraits(traitIds).reduce((total, trait) => total + Math.max(0, -trait.pickCost), Math.max(0, -governmentPickCost(government || defaultGovernment(governments), governments)));
}

export function isRaceTraitBlocked(trait: RaceTraitDefinition) {
  if (!trait.enabled) return true;
  return false;
}

export function swapExclusiveRaceTrait(trait: RaceTraitDefinition, traitIds: readonly RaceTraitId[]) {
  if (!trait.exclusiveGroup) return [...traitIds, trait.id as RaceTraitId];
  return [
    ...selectedRaceTraits(traitIds)
      .filter((selected) => selected.exclusiveGroup !== trait.exclusiveGroup)
      .map((selected) => selected.id as RaceTraitId),
    trait.id as RaceTraitId,
  ];
}

export function validateCustomRaceDesign(design: CustomRaceDesign) {
  const errors: string[] = [];
  if (!design.name.trim()) errors.push("Enter a race name.");
  if (design.name.trim().length > 32) errors.push("Race name must be 32 characters or fewer.");
  if (!design.government?.trim()) errors.push("Select a government.");
  return errors;
}

export function validatePlayableCustomRaceDesign(design: CustomRaceDesign, governments: readonly ApiGovernment[] = fallbackGovernments) {
  const errors = validateCustomRaceDesign(design);
  if (raceRemainingPicks(design.trait_ids, design.government, governments) < 0) errors.push("Race spends more than 10 picks.");
  if (raceNegativePicks(design.trait_ids, design.government, governments) > RACE_NEGATIVE_PICK_LIMIT) errors.push("Race uses more than 10 negative picks.");
  return errors;
}

export function isPlayableCustomRaceDesign(design: CustomRaceDesign, governments: readonly ApiGovernment[] = fallbackGovernments) {
  return validatePlayableCustomRaceDesign(design, governments).length === 0;
}

export function isRaceTraitId(id: string): id is RaceTraitId {
  const trait = raceTraitById.get(id);
  return Boolean(trait?.enabled);
}
