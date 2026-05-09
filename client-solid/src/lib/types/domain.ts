export type ShipDesignForm = {
  name: string;
  drive: string;
  computer: string;
  shield: string;
  armor: string;
  hullSize: string;
  fuelCell: string;
};

export type BuildingModifierViewModel = {
  name: string;
  description: string;
  bonus: number;
};

export type ModifierViewModel = {
  name: string;
  value: number;
};

export type GrantsViewModel = {
  unlocks: {
    buildings: string[];
    hullSizes: string[];
    drives: string[];
    armors: string[];
    computers: string[];
    fuelCells: string[];
    shields: string[];
    beams: string[];
    shipSystems: string[];
  };
  modifiers: ModifierViewModel[];
};

export type ModifierDefinitionViewModel = {
  name: string;
  description: string;
  scope: string;
  stat: string;
  operation: string;
};

export type BuildingViewModel = {
  id: number;
  name: string;
  description: string;
  icon: string;
  cost: number;
  maintenance: number;
  grants: GrantsViewModel;
  modifiers: BuildingModifierViewModel[];
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
    produced_per_colonist: number;
  };
  science: {
    scientists: number;
    produced_per_scientist: number;
    produced_per_building: number;
  };
  income: {
    taxes: number;
    buildings: number;
    tradeGoods: number;
    maintenance: number;
    total: number;
    bonus: number;
  };
  pollution: number;
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
  researchedTechs: string[];
  hyperAdvanced: Record<string, number>;
  available: PlayerAvailabilityViewModel;
  availableBuildings: string[];
  currentResearch: ResearchProjectViewModel | null;
  fleets: number[];
  freighters: number;
  freightersUsed: number;
  shipDesigns: ShipDesignViewModel[];
};

export type PlayerAvailabilityViewModel = {
  buildings: string[];
  hullSizes: string[];
  drives: string[];
  armors: string[];
  computers: string[];
  fuelCells: string[];
  shields: string[];
  beams: string[];
  shipSystems: string[];
};

export type ResearchProjectViewModel = {
  subject: string;
  techName: string;
  techNames: string[];
  levelName: string;
  progress: number;
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

export type ShipSystemViewModel = {
  name: string;
  description: string;
  beamAttackBonus: number;
  beamDefenseBonus: number;
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
  shipSystemCollection: {
    shipSystems: ShipSystemViewModel[];
  };
  buildingCollection: {
    buildings: BuildingViewModel[];
  };
  modifierCollection: {
    modifiers: ModifierDefinitionViewModel[];
  };
  shipDesignCollection: {
    shipDesigns: ShipDesignViewModel[];
  };
  techTree: TechTreeViewModel;
  techCollection: {
    techs: TechViewModel[];
  };
};

export type TechTreeViewModel = {
  subjects: TechSubjectViewModel[];
};

export type TechSubjectViewModel = {
  name: string;
  levels: TechLevelViewModel[];
};

export type TechLevelViewModel = {
  name: string;
  cost: number;
  forceAll: boolean;
  availableTechs: string[];
};

export type TechViewModel = {
  name: string;
  description: string;
  hidden: boolean;
  cost: number;
  maintenance: number;
  builtOn: string[];
  unique: boolean;
  grants: GrantsViewModel;
};

export type GameViewModel = {
  id: string;
  currentTurn: number;
  galaxy: GalaxyViewModel;
  players: PlayerViewModel[];
  resources: ResourcesViewModel;
  pendingColonyBases: Record<number, number[]>; // playerID -> planetIDs
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
  income: number;
  currentProject: string;
};
