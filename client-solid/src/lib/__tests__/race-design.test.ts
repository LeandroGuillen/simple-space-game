import {
  isRaceTraitBlocked,
  raceNegativePicks,
  raceRemainingPicks,
  swapExclusiveRaceTrait,
  validateCustomRaceDesign,
  validatePlayableCustomRaceDesign,
} from "../race-design";

describe("race design", () => {
  it("calculates remaining and negative picks", () => {
    const traitIds = ["food_plus_1", "production_minus_1"];

    expect(raceRemainingPicks(traitIds)).toBe(9);
    expect(raceNegativePicks(traitIds)).toBe(3);
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
      id: "creative",
      name: "Creative",
      category: "Future Traits",
      pickCost: 8,
      enabled: false,
      description: "",
    };

    expect(isRaceTraitBlocked(trait)).toBe(true);
  });

  it("validates overspending and negative pick limits", () => {
    expect(validateCustomRaceDesign({ name: "Ada", trait_ids: ["money_plus_1", "food_plus_2"] })).toEqual([]);
    expect(validatePlayableCustomRaceDesign({ name: "Ada", trait_ids: ["money_plus_1", "food_plus_2"] })).toContain("Race spends more than 10 picks.");
    expect(validatePlayableCustomRaceDesign({ name: "Ada", trait_ids: ["growth_minus_50", "food_minus_1", "production_minus_1", "money_minus_half"] })).toContain("Race uses more than 10 negative picks.");
  });
});
