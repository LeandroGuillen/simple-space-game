export type Galaxy = {
  stars: Star[];
};

export type Star = {
  id: number;
  name: string;
  planets: Planet[];
  color: number;
  x: number;
  y: number;
  // fe-only
  position?: string;
  numPlanets: number;
};

export enum PlanetMineral { POOR, ABUNDANT, RICH}

export enum PlanetSize { SMALL, MEDIUM, LARGE}

export enum PlanetGravity { LOW, NORMAL, HIGH}

export type Planet = {
  id: number;
  parentStar: number;
  name: string;
  mineral_class: PlanetMineral;
  size: PlanetSize;
  gravity: PlanetGravity;
  owner: number;
  money: number;
  food: number;
  industry: number;
  research: number;
  morale: number;
  current_pop: number;
  pop_growth: number;
  max_pop: number;
  farmers: number;
  workers: number;
  scientists: number;
  marines: number;
  armor: number;
  current_project: string;
  build_list: string[];
  buildings: string[];
};
