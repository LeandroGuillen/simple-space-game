import type { BuildingViewModel } from './types/domain';

const AUTOBUILD_EXCLUDED_PROJECTS = new Set([
  'Colony Base',
  'Colony Ship',
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

type ToggleQueuedProjectOptions = {
  maxQueueSize?: number;
  isRepeatable?: boolean;
  tradeGoodsId?: number;
};

export function toggleQueuedConstructionProject(
  currentProjects: BuildingViewModel[],
  project: BuildingViewModel,
  options: ToggleQueuedProjectOptions = {},
): BuildingViewModel[] {
  const maxQueueSize = options.maxQueueSize ?? 10;
  const tradeGoodsId = options.tradeGoodsId ?? 0;
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
    project.id !== tradeGoodsId &&
    currentProjects.length === 1 &&
    currentProjects[0]?.id === tradeGoodsId
      ? []
      : currentProjects;

  if (projects.length >= maxQueueSize) {
    return projects;
  }

  return [...projects, project];
}
