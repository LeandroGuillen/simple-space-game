import type { BuildingViewModel, PlanetViewModel } from './types/domain';

const AUTOBUILD_EXCLUDED_PROJECTS = new Set([
  'Colony Base',
  'Colony Ship',
  'Artificial Planet Construction',
  'Terraforming',
  'Outpost Ship',
  'Transport',
  'Freighters',
  'Spy',
  'Android Farmers',
  'Android Workers',
  'Android Scientists',
]);

export function isAutobuildProject(project: Pick<BuildingViewModel, 'name'>): boolean {
  if (AUTOBUILD_EXCLUDED_PROJECTS.has(project.name)) {
    return false;
  }
  return !project.name.endsWith(' Ship');
}

export const GRAVITY_GENERATOR_BUILDING_NAME = 'Gravity Generator';

const PLANET_GRAVITY_LOW = 1;
const PLANET_GRAVITY_NORMAL = 2;
const PLANET_GRAVITY_HIGH = 3;

export function preferredGravityForRace(traitIds: string[] | undefined): number {
  if (!traitIds) return PLANET_GRAVITY_NORMAL;
  if (traitIds.includes('low_g_world')) return PLANET_GRAVITY_LOW;
  if (traitIds.includes('high_g_world')) return PLANET_GRAVITY_HIGH;
  return PLANET_GRAVITY_NORMAL;
}

// Toxic, Radiated, and Barren planet environments (see PlanetEnvironment enum).
const HOSTILE_ENVIRONMENTS = new Set([3, 4, 5]);
const BIOMORPHIC_FUNGI_TECH = 'Biomorphic Fungi';

// buildingIsFarmerBonusOnly reports whether a building's sole effect is increasing
// food per farmer (e.g. Soil Enrichment, Weather Controller, Android Farmers).
function buildingIsFarmerBonusOnly(
  project: Partial<Pick<BuildingViewModel, 'grants'>>,
): boolean {
  const modifiers = project.grants?.modifiers ?? [];
  return (
    modifiers.length > 0 &&
    modifiers.every((modifier) => modifier.name === 'food_per_farmer')
  );
}

export function isBuildingApplicableToPlanet(
  project: Pick<BuildingViewModel, 'name'> & Partial<Pick<BuildingViewModel, 'grants'>>,
  planet: Pick<PlanetViewModel, 'gravity' | 'environment' | 'buildings'>,
  raceTraitIds: string[] | undefined,
  researchedTechs?: ReadonlySet<string>,
): boolean {
  if (project.name === GRAVITY_GENERATOR_BUILDING_NAME) {
    return planet.gravity !== preferredGravityForRace(raceTraitIds);
  }
  if (project.name === 'Terraforming') {
    return canTerraformPlanet(planet);
  }
  // Farmer-bonus-only buildings are useless on hostile worlds until Biomorphic
  // Fungi lets farmers produce food there. Flat-food buildings are unaffected.
  if (
    HOSTILE_ENVIRONMENTS.has(planet.environment) &&
    !researchedTechs?.has(BIOMORPHIC_FUNGI_TECH) &&
    buildingIsFarmerBonusOnly(project)
  ) {
    return false;
  }
  return true;
}

const TERRAFORM_TRANSITIONS: Record<number, number[]> = {
  3: [5],
  4: [5],
  5: [6, 8],
  6: [7],
  8: [9],
  7: [11],
  9: [11],
  10: [11],
};

const TERRAFORM_PREREQUISITES: Record<number, string> = {
  3: 'Weather Controller',
  4: 'Planetary Radiation Shield',
};

export function canTerraformPlanet(
  planet: Pick<PlanetViewModel, 'environment' | 'buildings'>,
): boolean {
  const targets = TERRAFORM_TRANSITIONS[planet.environment];
  if (!targets || targets.length === 0) return false;
  const prereq = TERRAFORM_PREREQUISITES[planet.environment];
  if (prereq && !planet.buildings.some((b) => b.name === prereq)) return false;
  return true;
}

type ToggleQueuedProjectOptions = {
  maxQueueSize?: number;
  isRepeatable?: boolean;
  tradeGoodsId?: number;
  terminalProjectIds?: number[];
};

export function toggleQueuedConstructionProject(
  currentProjects: BuildingViewModel[],
  project: BuildingViewModel,
  options: ToggleQueuedProjectOptions = {},
): BuildingViewModel[] {
  const maxQueueSize = options.maxQueueSize ?? 10;
  const terminalProjectIds = options.terminalProjectIds ?? [options.tradeGoodsId ?? 0];
  const isRepeatable = options.isRepeatable ?? false;

  if (!isRepeatable) {
    const existingIndex = currentProjects.findIndex(
      (queuedProject) => queuedProject.id === project.id,
    );
    if (existingIndex >= 0) {
      return currentProjects.filter((_, index) => index !== existingIndex);
    }
  }

  const projects =
    !terminalProjectIds.includes(project.id) &&
    currentProjects.length === 1 &&
    terminalProjectIds.includes(currentProjects[0]?.id ?? Number.NaN)
      ? []
      : currentProjects;

  if (projects.length >= maxQueueSize) {
    return projects;
  }

  return [...projects, project];
}
