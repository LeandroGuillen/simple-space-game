import { existsSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

import {
  planetArtCandidatesForEnvironment,
  planetArtFallbackForEnvironment,
  planetArtForEnvironment,
  planetArtSlugByEnvironment,
} from "../art";

const publicDir = join(process.cwd(), "public");

describe("planet art", () => {
  it("has a file naming convention for every planet environment", () => {
    for (let environment = 1; environment <= 12; environment += 1) {
      expect(planetArtSlugByEnvironment[environment]).toBeTruthy();
    }
  });

  it("can discover at least one current asset for every planet environment", () => {
    for (let environment = 1; environment <= 12; environment += 1) {
      const existingCandidates = planetArtCandidatesForEnvironment(environment)
        .filter((path) => existsSync(join(publicDir, path.replace(/^\//, ""))));

      expect(existingCandidates.length, `environment ${environment}`).toBeGreaterThan(0);
    }
  });

  it("keeps variant selection stable for the same planet", () => {
    const variants = planetArtCandidatesForEnvironment(11);
    expect(planetArtForEnvironment(11, 42, variants)).toBe(
      planetArtForEnvironment(11, 42, variants),
    );
  });

  it("falls back to barren art for unknown environments", () => {
    expect(planetArtFallbackForEnvironment(999)).toBe("/art/planets/barren1.png");
  });
});
