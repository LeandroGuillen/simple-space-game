import type { BuildingViewModel, PlanetViewModel } from './types/domain';

const AUTOBUILD_EXCLUDED_PROJECTS = new Set([
  'Colony Base',
  'Colony Ship',
  'Artificial Planet Construction',
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

export function isBuildingApplicableToPlanet(
  project: Pick<BuildingViewModel, 'name'>,
  planet: Pick<PlanetViewModel, 'gravity'>,
  raceTraitIds: string[] | undefined,
): boolean {
  if (project.name === GRAVITY_GENERATOR_BUILDING_NAME) {
    return planet.gravity !== preferredGravityForRace(raceTraitIds);
  }
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
