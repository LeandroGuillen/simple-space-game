export type Player = {
  id: number;
  name: string;

  race: string;

  treasury: number;
  taxRate: number;
  researchPoints: number;

  fleets: number[];
  freighters: number;
  freightersUsed: number;
}

export type Game = {
  galaxy: Galaxy;
  players: Player[];
  id: string;
}

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

export enum PlanetSize { INVALID, TINY, SMALL, MEDIUM, LARGE, HUGE }

export enum PlanetGravity { INVALID, LOW, NORMAL, HIGH }

export enum PlanetMineral { INVALID, ULTRAPOOR, POOR, ABUNDANT, RICH, ULTRARICH }

export enum PlanetEnvironment {
  INVALID,
  GASGIANT,
  ASTEROIDBELT,
  TOXIC,
  RADIATED,
  BARREN,
  DESERT,
  ARID,
  TUNDRA,
  SWAMP,
  OCEAN,
  TERRAN,
  GAIA,
}

export type Planet = {
  id: number;
  parent_star: number;
  name: string;
  mineral_class: PlanetMineral;
  size: PlanetSize;
  gravity: PlanetGravity;
  environment: PlanetEnvironment;
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
  current_project: number;
  build_list: string[];
  buildings: Building[];
};

export type Building = {
  id: number;
  name: string;
  cost: number;
  maintenance: number;
}

export function emptyGame(): Game {
  return {
    id: '',
    players: [],
    galaxy: {
      stars: []
    },
  }
}

export type Server = {
  id: string;
  name: string;
  url: string;
}

export function emptyServer(): Server {
  return {
    id: '',
    name: '',
    url: '',
  }
}