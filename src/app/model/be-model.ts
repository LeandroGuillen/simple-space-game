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
  buildings: string[];
};
