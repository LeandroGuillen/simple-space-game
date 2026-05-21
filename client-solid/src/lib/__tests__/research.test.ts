import { describe, expect, it } from "vitest";

import { researchLevelUnlocksAllTechs } from "../research";

describe("research", () => {
  it("treats creative races as unlocking every tech in a level", () => {
    expect(
      researchLevelUnlocksAllTechs({ forceAll: false }, ["creative"]),
    ).toBe(true);
  });

  it("keeps normal races on single-tech levels", () => {
    expect(researchLevelUnlocksAllTechs({ forceAll: false }, [])).toBe(false);
  });

  it("honors force_all levels without creative", () => {
    expect(researchLevelUnlocksAllTechs({ forceAll: true }, [])).toBe(true);
  });
});
