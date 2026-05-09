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
