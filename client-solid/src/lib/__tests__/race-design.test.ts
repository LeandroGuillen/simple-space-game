import type { RaceTraitId } from "../types/api";
import {
  governmentPickCost,
  isPlayableCustomRaceDesign,
  isRaceTraitBlocked,
  isRaceTraitId,
  newGameRaceFieldsFromEncodedSelection,
  raceNegativePicks,
  racePopulationGrowthPercent,
  raceRemainingPicks,
  raceSlugForNewGame,
  raceTraitById,
  partialRaceTraitDetails,
  swapExclusiveRaceTrait,
  validateCustomRaceDesign,
  validatePlayableCustomRaceDesign,
} from "../race-design";

describe("race design", () => {
  it("explains each incomplete pick", () => {
    for (const id of ["ground_combat", "spying", "low_g_world", "high_g_world", "stealthy_ships", "lucky"]) {
      expect(partialRaceTraitDetails[id]).toBeTruthy();
    }
  });

  it("calculates remaining and negative picks", () => {
    const traitIds = ["food_plus_1", "production_minus_1"];

    expect(raceRemainingPicks(traitIds)).toBe(9);
    expect(raceNegativePicks(traitIds)).toBe(3);
    expect(governmentPickCost("Democracy")).toBe(7);
    expect(raceRemainingPicks(traitIds, "Democracy")).toBe(2);
  });

  it("swaps traits in the same exclusive group", () => {
    const trait = {
      id: "research_plus_2",
      name: "+2 Research",
      category: "Research",
      pickCost: 6,
      exclusiveGroup: "research",
      enabled: true,
      description: "",
    };

    expect(swapExclusiveRaceTrait(trait, ["research_plus_1"])).toEqual(["research_plus_2"]);
  });

  it("treats disabled future traits as blocked", () => {
    const trait = {
      id: "repulsive",
      name: "Repulsive",
      category: "Special Abilities",
      pickCost: -6,
      enabled: false,
      description: "",
    };

    expect(isRaceTraitBlocked(trait)).toBe(true);
  });

  it("sums population growth percent from growth traits", () => {
    expect(racePopulationGrowthPercent(["growth_plus_50"])).toBe(50);
    expect(racePopulationGrowthPercent(["growth_plus_100"])).toBe(100);
    expect(racePopulationGrowthPercent(["growth_minus_50"])).toBe(-50);
    expect(racePopulationGrowthPercent(["food_plus_1"])).toBe(0);
  });

  it("prefers API population growth percent when provided", () => {
    expect(racePopulationGrowthPercent(["growth_plus_50"], 100)).toBe(100);
    expect(racePopulationGrowthPercent(["growth_minus_50"], 0)).toBe(0);
  });

  it("recognizes newly enabled special trait ids", () => {
    expect(isRaceTraitId("low_g_world")).toBe(true);
    expect(isRaceTraitId("high_g_world")).toBe(true);
    expect(isRaceTraitId("poor_homeworld")).toBe(true);
    expect(isRaceTraitId("creative")).toBe(true);
    expect(isRaceTraitId("tolerant")).toBe(true);
    expect(isRaceTraitId("repulsive")).toBe(false);
  });

  it("disables unfinished espionage and stealth picks", () => {
    expect(isRaceTraitBlocked(raceTraitById.get("ground_combat")!)).toBe(false);
    expect(raceTraitById.get("spying")?.enabled).toBe(false);
    expect(raceTraitById.get("stealthy_ships")?.enabled).toBe(false);
    expect(raceTraitById.get("ground_combat")?.pickCost).toBe(5);
    expect(raceTraitById.get("spying")?.pickCost).toBe(3);
  });

  it("treats creative as selectable", () => {
    const trait = {
      id: "creative",
      name: "Creative",
      category: "Special Abilities",
      pickCost: 8,
      exclusiveGroup: "creativity",
      enabled: true,
      description: "",
    };

    expect(isRaceTraitBlocked(trait)).toBe(false);
  });

  it("treats uncreative as selectable", () => {
    const trait = {
      id: "uncreative",
      name: "Uncreative",
      category: "Special Abilities",
      pickCost: -4,
      exclusiveGroup: "creativity",
      enabled: true,
      description: "",
    };

    expect(isRaceTraitBlocked(trait)).toBe(false);
  });

  it("validates overspending and negative pick limits", () => {
    expect(validateCustomRaceDesign({ name: "Ada", government: "Dictatorship", trait_ids: ["money_plus_1", "food_plus_2"] })).toEqual([]);
    expect(validatePlayableCustomRaceDesign({ name: "Ada", government: "Dictatorship", trait_ids: ["money_plus_1", "food_plus_2"] })).toContain(
      "Race spends more than 10 picks.",
    );
    expect(
      validatePlayableCustomRaceDesign({
        name: "Ada",
        government: "Dictatorship",
        trait_ids: ["growth_minus_50", "food_minus_1", "production_minus_1", "money_minus_half"],
      }),
    ).toContain("Race uses more than 10 negative picks.");
  });

  it("maps new-game race selections to API fields", () => {
    expect(raceSlugForNewGame("Human")).toBe("human");
    expect(newGameRaceFieldsFromEncodedSelection("b:human")).toEqual({ race: "human" });
    expect(newGameRaceFieldsFromEncodedSelection("c:Synths")).toEqual({
      race: "custom",
      custom_race_name: "Synths",
    });
    expect(() => newGameRaceFieldsFromEncodedSelection("nope")).toThrow();
  });

  it("disables unplayable race designs for game start", () => {
    const playable = {
      name: "Ok",
      government: "Dictatorship",
      trait_ids: ["research_plus_1"] as RaceTraitId[],
    };
    const overspent = {
      name: "Bad",
      government: "Dictatorship",
      trait_ids: ["money_plus_1", "food_plus_2"] as RaceTraitId[],
    };
    expect(isPlayableCustomRaceDesign(playable)).toBe(true);
    expect(isPlayableCustomRaceDesign(overspent)).toBe(false);
  });
});
