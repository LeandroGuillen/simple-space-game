export type MovePopulationCommand = {
  planetId: number;
  from: 'farmers' | 'workers' | 'scientists';
  to: 'farmers' | 'workers' | 'scientists';
  count: number;
};
