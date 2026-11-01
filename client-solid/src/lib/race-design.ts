import type { ApiGovernment, CustomRaceDesign, NewGameOptions, RaceTraitId } from "./types/api";

export type RaceTraitDefinition = {
  id: string;
  name: string;
  category: string;
  pickCost: number;
  exclusiveGroup?: string;
  enabled: boolean;
  description: string;
  populationGrowthPercent?: number;
};

// These picks have effects that are still awaiting their corresponding systems.
export const partialRaceTraitDetails: Record<string, string> = {
  ground_combat: "The 50% damage bonus works in ground combat for both sides.",
  spying: "Spy bonuses are recorded, but there is no playable espionage system yet.",
  low_g_world: "Homeworld gravity and planetary output penalties work; the troop penalty is not implemented.",
  high_g_world: "Homeworld gravity and planetary output penalties work; the extra troop hit is not implemented.",
  stealthy_ships: "The detection modifier is recorded, but there is no detection system yet.",
  lucky: "There is a 35% chance per turn of one bonus research point; random events are not implemented.",
};

export const RACE_PICK_BUDGET = 10;
export const RACE_NEGATIVE_PICK_LIMIT = 10;

// Evolutionary Mutation grants 4 extra pick points
export const MUTATION_PICK_BUDGET = 14; // 10 base + 4 bonus

export const DEFAULT_GOVERNMENT = "Dictatorship";

/** MoO2 race design screen: left column economic picks. */
export const RACE_EDITOR_LEFT_COLUMN = [
  "Population",
  "Farming",
  "Industry",
  "Science",
  "Money",
] as const;

/** MoO2 race design screen: middle column combat and government. */
export const RACE_EDITOR_MIDDLE_COLUMN = [
  "Ship Defense",
  "Ship Attack",
  "Ground Combat",
  "Spying",
] as const;

/** MoO2 race design screen: right column special abilities. */
export const RACE_EDITOR_RIGHT_COLUMN = ["Special Abilities"] as const;

export const fallbackGovernments: ApiGovernment[] = [
  { name: "Feudal", description: "A decentralized noble hierarchy with cheap ship construction and weak research.", pick_cost: -4 },
  { name: "Dictatorship", description: "A centralized authoritarian government with stable internal security.", pick_cost: 0, default: true },
  { name: "Democracy", description: "A representative government with strong research and tax output but weaker counter-espionage.", pick_cost: 7 },
  { name: "Unification", description: "A unified collective with strong farming and industry that ignores morale.", pick_cost: 6 },
];

export const raceTraitDefinitions: RaceTraitDefinition[] = [
  { id: "growth_minus_50", name: "-50% Growth", category: "Population", pickCost: -4, exclusiveGroup: "growth", enabled: true, populationGrowthPercent: -50, description: "Colonies grow at half speed." },
  { id: "growth_plus_50", name: "+50% Growth", category: "Population", pickCost: 3, exclusiveGroup: "growth", enabled: true, populationGrowthPercent: 50, description: "Colonies grow 50% faster." },
  { id: "growth_plus_100", name: "+100% Growth", category: "Population", pickCost: 6, exclusiveGroup: "growth", enabled: true, populationGrowthPercent: 100, description: "Colonies grow twice as fast." },
  { id: "food_minus_1", name: "-1/2 Food", category: "Farming", pickCost: -3, exclusiveGroup: "food", enabled: true, description: "Farmers produce less food." },
  { id: "food_plus_1", name: "+1 Food", category: "Farming", pickCost: 4, exclusiveGroup: "food", enabled: true, description: "Farmers produce one extra food." },
  { id: "food_plus_2", name: "+2 Food", category: "Farming", pickCost: 7, exclusiveGroup: "food", enabled: true, description: "Farmers produce two extra food." },
  { id: "production_minus_1", name: "-1 Production", category: "Industry", pickCost: -3, exclusiveGroup: "production", enabled: true, description: "Workers produce less industry." },
  { id: "production_plus_1", name: "+1 Production", category: "Industry", pickCost: 3, exclusiveGroup: "production", enabled: true, description: "Workers produce one extra industry." },
  { id: "production_plus_2", name: "+2 Production", category: "Industry", pickCost: 6, exclusiveGroup: "production", enabled: true, description: "Workers produce two extra industry." },
  { id: "research_minus_1", name: "-1 Research", category: "Science", pickCost: -3, exclusiveGroup: "research", enabled: true, description: "Scientists produce less research." },
  { id: "research_plus_1", name: "+1 Research", category: "Science", pickCost: 3, exclusiveGroup: "research", enabled: true, description: "Scientists produce one extra research." },
  { id: "research_plus_2", name: "+2 Research", category: "Science", pickCost: 6, exclusiveGroup: "research", enabled: true, description: "Scientists produce two extra research." },
  { id: "money_minus_half", name: "-0.5 BC", category: "Money", pickCost: -4, exclusiveGroup: "money", enabled: true, description: "Taxable income is reduced." },
  { id: "money_plus_half", name: "+0.5 BC", category: "Money", pickCost: 5, exclusiveGroup: "money", enabled: true, description: "Taxable income is increased." },
  { id: "money_plus_1", name: "+1 BC", category: "Money", pickCost: 8, exclusiveGroup: "money", enabled: true, description: "Taxable income is strongly increased." },
  { id: "ship_defense", name: "+20% Ship Defense", category: "Ship Defense", pickCost: 3, enabled: true, description: "Ships gain +20% beam defense in combat (MoO2 first level, 3 picks)." },
  { id: "ship_attack", name: "+20% Ship Attack", category: "Ship Attack", pickCost: 3, enabled: true, description: "Ships gain +20% beam attack in combat (MoO2 first level, 3 picks)." },
  { id: "ground_combat", name: "Ground Combat", category: "Ground Combat", pickCost: 5, enabled: true, description: "+50% ground combat strength (marines and invasions when implemented)." },
  { id: "spying", name: "Spying", category: "Spying", pickCost: 3, enabled: false, description: "Unavailable until espionage is implemented; its modifiers are stored but have no playable effect." },
  { id: "low_g_world", name: "Low-G World", category: "Special Abilities", pickCost: -5, exclusiveGroup: "gravity", enabled: true, description: "Adapted to Low-G worlds (no penalty there); suffers -25% farming/production/research on Normal-G worlds and -50% on High-G. Troops take a 10% penalty in ground combat (when implemented)." },
  { id: "high_g_world", name: "High-G World", category: "Special Abilities", pickCost: 6, exclusiveGroup: "gravity", enabled: true, description: "Adapted to High-G worlds (no penalty there); still -25% farming/production/research on Low-G worlds. Troops take 1 extra hit before dying in ground combat (when implemented)." },
  { id: "aquatic", name: "Aquatic", category: "Special Abilities", pickCost: 5, enabled: true, description: "Double population capacity on Ocean and Swamp worlds; half capacity on other habitable planets." },
  { id: "subterranean", name: "Subterranean", category: "Special Abilities", pickCost: 6, exclusiveGroup: "capacity", enabled: true, description: "Planet population capacity is doubled." },
  { id: "large_homeworld", name: "Large Home World", category: "Special Abilities", pickCost: 1, exclusiveGroup: "homeworld_size", enabled: true, description: "The capital starts as a large world." },
  { id: "rich_homeworld", name: "Rich Home World", category: "Special Abilities", pickCost: 2, exclusiveGroup: "homeworld_minerals", enabled: true, description: "The capital starts rich." },
  { id: "poor_homeworld", name: "Poor Home World", category: "Special Abilities", pickCost: -1, exclusiveGroup: "homeworld_minerals", enabled: true, description: "The capital starts poor." },
  { id: "artifacts_homeworld", name: "Artifacts World", category: "Special Abilities", pickCost: 3, exclusiveGroup: "homeworld_artifacts", enabled: true, description: "The capital produces extra research." },
  { id: "cybernetic", name: "Cybernetic", category: "Special Abilities", pickCost: 4, enabled: true, description: "Population consumes half as much food (0.5 per 1000 pop)." },
  { id: "lithovore", name: "Lithovore", category: "Special Abilities", pickCost: 10, enabled: true, description: "Population does not consume food; no starvation from food shortage." },
  { id: "repulsive", name: "Repulsive", category: "Special Abilities", pickCost: -6, enabled: false, description: "Requires diplomacy." },
  { id: "charismatic", name: "Charismatic", category: "Special Abilities", pickCost: 3, enabled: false, description: "Requires diplomacy." },
  { id: "uncreative", name: "Uncreative", category: "Special Abilities", pickCost: -4, exclusiveGroup: "creativity", enabled: true, description: "Completing a research level discovers only the selected technology at that breakthrough." },
  { id: "creative", name: "Creative", category: "Special Abilities", pickCost: 8, exclusiveGroup: "creativity", enabled: true, description: "Completing a research level discovers every technology at that breakthrough." },
  { id: "tolerant", name: "Tolerant", category: "Special Abilities", pickCost: 10, enabled: true, description: "Ignores industrial pollution and hostile-environment population penalties (Barren, Toxic, Radiated use Terran capacity)." },
  { id: "fantastic_traders", name: "Fantastic Traders", category: "Special Abilities", pickCost: 4, enabled: true, description: "+50% income from trade goods." },
  { id: "telepathic", name: "Telepathic", category: "Special Abilities", pickCost: 6, enabled: false, description: "Requires diplomacy and conquest rules." },
  { id: "lucky", name: "Lucky", category: "Special Abilities", pickCost: 3, enabled: true, description: "v1: Each turn, 35% chance for +1 bonus research point (10% base + 25% lucky bonus). Full random events planned later." },
  { id: "omniscient", name: "Omniscient", category: "Special Abilities", pickCost: 3, enabled: true, description: "See the entire galaxy map." },
  { id: "stealthy_ships", name: "Stealthy Ships", category: "Special Abilities", pickCost: 4, enabled: false, description: "Unavailable until ship detection is implemented; its modifier currently has no playable effect." },
  { id: "trans_dimensional", name: "Trans-Dimensional", category: "Special Abilities", pickCost: 5, enabled: true, description: "Ships gain +2 combat speed in battle." },
  { id: "warlord", name: "Warlord", category: "Special Abilities", pickCost: 4, enabled: true, description: "Ships start with veteran crews (+25% attack and defense in combat)." },
];

export function raceTraitGroupsForCategories(categories: readonly string[]) {
  return categories
    .map((category) => ({
      category,
      traits: raceTraitDefinitions.filter((trait) => trait.category === category),
    }))
    .filter((group) => group.traits.length > 0);
}

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

/** Race growth bonus percent; prefers API stats when provided. */
export function racePopulationGrowthPercent(
  traitIds: TraitIdInput,
  apiPercent?: number | null,
): number {
  if (apiPercent != null && Number.isFinite(apiPercent)) {
    return apiPercent;
  }
  return selectedRaceTraits(traitIds).reduce(
    (sum, trait) => sum + (trait.populationGrowthPercent ?? 0),
    0,
  );
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

// Mutation budget functions (14 points instead of 10)
export function mutationTotalPickCost(traitIds: TraitIdInput, government: string, governments: readonly ApiGovernment[] = fallbackGovernments) {
  return racePickCost(traitIds) + governmentPickCost(government || defaultGovernment(governments), governments);
}

export function mutationRemainingPicks(traitIds: TraitIdInput, government = DEFAULT_GOVERNMENT, governments: readonly ApiGovernment[] = fallbackGovernments) {
  return MUTATION_PICK_BUDGET - mutationTotalPickCost(traitIds, government, governments);
}

export function mutationNegativePicks(traitIds: TraitIdInput, government = DEFAULT_GOVERNMENT, governments: readonly ApiGovernment[] = fallbackGovernments) {
  return selectedRaceTraits(traitIds).reduce((total, trait) => total + Math.max(0, -trait.pickCost), Math.max(0, -governmentPickCost(government || defaultGovernment(governments), governments)));
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

/** API key for built-in races in new-game requests (e.g. Human -> human). */
export function raceSlugForNewGame(name: string): string {
  return name.trim().toLowerCase().replace(/\s+/g, "_");
}

/** Decode `b:slug` / `c:name` from the server page race toggle. */
export function newGameRaceFieldsFromEncodedSelection(encoded: string): Pick<NewGameOptions, "race" | "custom_race_name"> {
  if (encoded.startsWith("c:")) {
    return { race: "custom", custom_race_name: encoded.slice(2) };
  }
  if (encoded.startsWith("b:")) {
    return { race: encoded.slice(2) };
  }
  throw new Error(`invalid race selection: ${encoded}`);
}

export function isRaceTraitId(id: string): id is RaceTraitId {
  const trait = raceTraitById.get(id);
  return Boolean(trait?.enabled);
}
