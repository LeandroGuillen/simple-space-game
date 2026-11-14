import type { ApiInvasionReport, ApiMissile, ApiTurnEvent } from "./api";
import type { ApiTacticalBattle } from "./battle";

export type ShipDesignForm = {
  beam?: string;
  /** Missile rack in the one weapon slot; never set together with beam. */
  missile?: string;
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
    missiles?: string[];
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
  unique: boolean;
  upgrades: string[];
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
    industryTax?: number;
    tradeGoods: number;
    maintenance: number;
    total: number;
    bonus: number;
  };
  morale: {
    bonus: number;
    multiplier: number;
    sources: MoraleSourceViewModel[];
  };
  government?: {
    name: string;
    foodLaborMultiplier: number;
    industryLaborMultiplier: number;
    scienceLaborMultiplier: number;
    taxMultiplier: number;
    moraleNeutral: boolean;
  };
  pollution: number;
  gravityMultiplier: number;
  currentProject: BuildingViewModel | null;
  buildQueue: BuildingViewModel[];
  buildings: BuildingViewModel[];
  industryStack: number;
  buildBucket: number;
  isBuilding: boolean;
  autobuild: boolean;
  maxPopulation: number;
  terraformingCount: number;
};

export type MoraleSourceViewModel = {
  name: string;
  type: string;
  bonus: number;
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
  width: number;
  height: number;
};

export type PlayerViewModel = {
  id: number;
  eliminated?: boolean;
  name: string;
  color: string;
  raceName: string;
  raceGovernment: string;
  raceTraitIds: string[];
  racePopulationGrowthPercent: number;
  raceGrants: GrantsViewModel;
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
  commandPointsCapacity: number;
  commandPointsUsed: number;
  commandPointsRemaining: number;
  commandOverdrawBc: number;
  commandPointsStartingBonus: number;
  evolutionaryMutationPoints: number;
  hasUsedEvolutionaryMutation: boolean;
};

export type FleetShipViewModel = {
  type: string;
  name: string;
  icon: string;
  designName: string;
  hullSize: string;
  drive: string;
  interstellarSpeed: number;
  fuelCell: string;
  fuelRangeParsecs: number;
  unlimitedFuelRange: boolean;
};

export type FleetViewModel = {
  id: number;
  ownerId: number;
  starId: number;
  x: number;
  y: number;
  originStarId: number | null;
  destinationStarId: number | null;
  totalTurns: number;
  turnsRemaining: number;
  ships: FleetShipViewModel[];
};

export type PopulationTransferViewModel = {
  id: number;
  owner: number;
  sourcePlanetId: number;
  destinationPlanetId: number;
  populationUnits: number;
  freightersReserved: number;
  originStarId: number;
  destinationStarId: number;
  totalTurns: number;
  turnsRemaining: number;
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
  missiles?: string[];
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
  beam?: string;
  missile?: string;
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
    beams?: { name: string; minDamage: number; maxDamage: number }[];
    missiles?: ApiMissile[];
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
  type: string;
  pickCost: number;
  default: boolean;
  upgrades: string[];
  cost: number;
  maintenance: number;
  builtOn: string[];
  unique: boolean;
  grants: GrantsViewModel;
};

export type TransferLossNotice = {
  owner: number;
  turn: number;
  transferId: number;
  sourcePlanetId: number;
  sourcePlanetName: string;
  destinationPlanetId: number;
  destinationPlanetName: string;
  populationLost: number;
};

export type GameViewModel = {
  id: string;
  status?: "active" | "finished";
  winnerId?: number;
  turnsPlayed?: number;
  currentTurn: number;
  galaxy: GalaxyViewModel;
  players: PlayerViewModel[];
  fleets: FleetViewModel[];
  populationTransfers?: PopulationTransferViewModel[];
  transferLosses?: TransferLossNotice[];
  turnEvents?: ApiTurnEvent[];
  resources: ResourcesViewModel;
  pendingColonyBases: Record<number, number[]>; // playerID -> planetIDs
  pendingColonyShips: Record<number, number[]>; // playerID -> fleetIDs
  pendingArtificialPlanets: Record<number, number[]>; // playerID -> source planet IDs
  invasionReports?: ApiInvasionReport[];
  pendingBattles?: ApiTacticalBattle[];
  turnStatus?: TurnStatusViewModel;
};

export type TurnStatusViewModel = {
  mode: "simultaneous";
  readyPlayerIds: number[];
  waitingPlayerIds: number[];
  requiredPlayerIds: number[];
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
  morale: number;
  currentProject: string;
};
