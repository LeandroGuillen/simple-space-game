export type ShipDesignForm = {
  name: string;
  drive: string;
  computer: string;
  shield: string;
  armor: string;
  hullSize: string;
  fuelCell: string;
};

export type BuildingEffectViewModel = {
  name: string;
  description: string;
  bonus: number;
};

export type BuildingViewModel = {
  id: number;
  name: string;
  description: string;
  icon: string;
  cost: number;
  maintenance: number;
  effects: BuildingEffectViewModel[];
};

export type PlanetViewModel = {
  id: number;
  name: string;
  owner: number;
  parentStar: number;
  mineralClass: number;
  size: number;
  gravity: number;
  environment: number;
  population: number;
  last_turn_pop_growth: number;
  marines: number;
  armor: number;
  food: {
    farmers: number;
    produced_per_farmer: number;
    produced_per_building: number;
  };
  industry: {
    workers: number;
    produced_per_worker: number;
    produced_per_building: number;
  };
  science: {
    scientists: number;
    produced_per_scientist: number;
    produced_per_building: number;
  };
  currentProject: BuildingViewModel | null;
  buildQueue: BuildingViewModel[];
  buildings: BuildingViewModel[];
  industryStack: number;
  buildBucket: number;
  isBuilding: boolean;
  maxPopulation: number;
};

export type StarSummary = {
  id: number;
  name: string;
  planets: PlanetViewModel[];
  color: number;
  x: number;
  y: number;
  ownedPlanets: number;
  explored: boolean;
  ownerColor: string | null;
};

export type GalaxyViewModel = {
  stars: StarSummary[];
};

export type PlayerViewModel = {
  id: number;
  name: string;
  color: string;
  raceName: string;
  treasury: number;
  taxRate: number;
  researchPoints: number;
  fleets: number[];
  freighters: number;
  freightersUsed: number;
  shipDesigns: ShipDesignViewModel[];
};

export type PlayerColorOptions = {
  maxPlayers: number;
  colors: string[];
};

export type DriveViewModel = {
  name: string;
  interstellarSpeed: number;
  combatSpeed: number;
};

export type ComputerViewModel = {
  name: string;
  beamAttackBonus: number;
};

export type ShieldViewModel = {
  name: string;
  rechargeRate: number;
  reduceDamage: number;
  absortionFactor: number;
};

export type ArmorViewModel = {
  name: string;
  multiplier: number;
  groundTroopStrength: number;
};

export type HullSizeViewModel = {
  name: string;
  sizeRating: number;
  space: number;
  marines: number;
  baseSpeed: number;
  baseHp: number;
  computerHp: number;
  driveHp: number;
  shieldHp: number;
};

export type FuelCellViewModel = {
  name: string;
  autonomy: number;
};

export type ShipDesignViewModel = {
  name: string;
  drive: DriveViewModel;
  computer: ComputerViewModel;
  shield: ShieldViewModel;
  armor: ArmorViewModel;
  hullSize: HullSizeViewModel;
  fuelCell: FuelCellViewModel;
  baseSpace: number;
  isCommon: boolean;
};

export type ResourcesViewModel = {
  shipParts: {
    hullSizes: HullSizeViewModel[];
    drives: DriveViewModel[];
    armors: ArmorViewModel[];
    computers: ComputerViewModel[];
    fuelCells: FuelCellViewModel[];
    shields: ShieldViewModel[];
  };
  buildingCollection: {
    buildings: BuildingViewModel[];
  };
  shipDesignCollection: {
    shipDesigns: ShipDesignViewModel[];
  };
};

export type GameViewModel = {
  id: string;
  currentTurn: number;
  galaxy: GalaxyViewModel;
  players: PlayerViewModel[];
  resources: ResourcesViewModel;
};

export type ColonyRow = {
  id: number;
  name: string;
  starId: number;
  starName: string;
  population: number;
  maxPopulation: number;
  farmers: number;
  workers: number;
  scientists: number;
  industry: number;
  currentProject: string;
};
