import { existsSync, readdirSync, readFileSync } from "node:fs";
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

  for (const [environment, slug] of Object.entries(planetArtSlugByEnvironment)) {
    it(`ships at least three numbered 256px PNGs with alpha for ${slug}`, () => {
      const directory = join(publicDir, "art/planets");
      const numbered = readdirSync(directory).filter((name) =>
        new RegExp(`^${slug}[1-9]\\d*\\.png$`).test(name),
      );
      expect(numbered.length, `environment ${environment}`).toBeGreaterThanOrEqual(3);
      for (const name of numbered) {
        expect(planetArtCandidatesForEnvironment(Number(environment))).toContain(`/art/planets/${name}`);
        const png = readFileSync(join(directory, name));
        expect(png.subarray(0, 8).toString("hex"), name).toBe("89504e470d0a1a0a");
        expect(png.subarray(12, 16).toString("ascii"), name).toBe("IHDR");
        expect(png.readUInt32BE(16), `${name} width`).toBe(256);
        expect(png.readUInt32BE(20), `${name} height`).toBe(256);
        // PNG color types 4 (gray + alpha) and 6 (RGBA) have explicit alpha.
        expect([4, 6], `${name} alpha channel`).toContain(png[25]);
      }
    });
  }

  it("falls back to barren art for unknown environments", () => {
    expect(planetArtFallbackForEnvironment(999)).toBe("/art/planets/barren1.png");
  });
});
