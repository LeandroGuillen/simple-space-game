import { BEShipDesigns } from './combat';

export type BEGame = {
  galaxy: BEGalaxy;
  players: BEPlayer[];
  id: string;
  current_turn: number;
};

export type BEPlayer = {
  id: number;
  name: string;

  race: string;

  treasury: number;
  taxRate: number;
  researchPoints: number;

  fleets: number[];
  freighters: number;
  freightersUsed: number;

  shipDesigns: BEShipDesigns[];
};

export type BEGalaxy = {
  stars: BEStar[];
};

export type BEStar = {
  id: number;
  name: string;
  planets: BEPlanet[];
  color: number;
  x: number;
  y: number;
};

export type BEPlanet = {
  id: number;
  name: string;
  parent_star: number;
  mineral_class: number;
  size: number;
  gravity: number;
  environment: number;
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
  buildings: BEBuilding[];
};

export type BEBuilding = {
  id: number;
  name: string;
  cost: number;
  maintenance: number;
};
