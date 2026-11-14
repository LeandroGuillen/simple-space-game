const ART_ROOT = "/art";
const MAX_PLANET_ART_VARIANTS = 3;

export const planetArtSlugByEnvironment: Record<number, string> = {
  1: "gas-giant",
  2: "asteroid-belt",
  3: "toxic",
  4: "radiated",
  5: "barren",
  6: "desert",
  7: "arid",
  8: "tundra",
  9: "swamp",
  10: "ocean",
  11: "terran",
  12: "gaia",
};

export const shipArtByType: Record<string, string> = {
  colony_ship: `${ART_ROOT}/ships/colony-ship.png`,
  scout: `${ART_ROOT}/ships/scout.png`,
  default: `${ART_ROOT}/ships/default-ship.png`,
};

const discoveredPlanetArt = new Map<number, Promise<string[]>>();

export function planetArtForEnvironment(
  environment: number,
  planetId = 0,
  variants = [planetArtFallbackForEnvironment(environment)],
) {
  const index = stableVariantIndex(environment, planetId, variants.length);
  return variants[index];
}

export function planetArtFallbackForEnvironment(environment: number) {
  const slug = planetArtSlugForEnvironment(environment);
  return `${ART_ROOT}/planets/${slug}1.png`;
}

export function planetArtCandidatesForEnvironment(environment: number) {
  const slug = planetArtSlugForEnvironment(environment);
  const candidates = Array.from(
    { length: MAX_PLANET_ART_VARIANTS },
    (_, index) => `${ART_ROOT}/planets/${slug}${index + 1}.png`,
  );

  return [...candidates, `${ART_ROOT}/planets/${slug}.png`];
}

export function discoverPlanetArtForEnvironment(environment: number) {
  const normalizedEnvironment = planetArtSlugByEnvironment[environment]
    ? environment
    : 5;

  const cached = discoveredPlanetArt.get(normalizedEnvironment);
  if (cached) {
    return cached;
  }

  const discovered = Promise.all(
    planetArtCandidatesForEnvironment(normalizedEnvironment).map((path) =>
      imageExists(path).then((exists) => exists ? path : null),
    ),
  ).then((paths) => {
    const variants = paths.filter((path): path is string => path != null);
    return variants.length > 0
      ? variants
      : [planetArtFallbackForEnvironment(normalizedEnvironment)];
  });

  discoveredPlanetArt.set(normalizedEnvironment, discovered);
  return discovered;
}

export function shipArtForType(type: string) {
  return shipArtByType[type] ?? shipArtByType.default;
}

function planetArtSlugForEnvironment(environment: number) {
  return planetArtSlugByEnvironment[environment] ?? planetArtSlugByEnvironment[5];
}

function imageExists(path: string) {
  if (typeof Image === "undefined") {
    return Promise.resolve(true);
  }

  return new Promise<boolean>((resolve) => {
    const image = new Image();
    image.onload = () => resolve(true);
    image.onerror = () => resolve(false);
    image.src = path;
  });
}

function stableVariantIndex(environment: number, planetId: number, variantCount: number) {
  if (variantCount <= 1) {
    return 0;
  }

  const seed = Math.imul(planetId || environment, 2654435761) ^ Math.imul(environment, 2246822519);
  return Math.abs(seed) % variantCount;
}
