import type { ApiTacticalBattle } from './battle';

export type ApiStatusResponse = {
  ok: boolean;
  version?: string;
  /** True when the server registered /debug/* endpoints. */
  debug?: boolean;
};

export type ApiGameListItem = {
  game_id: string;
  num_stars: number;
  current_turn: number;
  /** ISO-8601 UTC when the game was created */
  created_at?: string | null;
  /** ISO-8601 UTC when the last turn completed; omit or null if none yet */
  last_turn_at?: string | null;
  /** True after a debug research-all mutation on this game. */
  debug?: boolean;
  players: ApiGameListPlayer[];
};

export type ScenarioInfo = {
  name: string;
  game_id: string;
  current_turn: number;
  num_stars: number;
  saved_at: string;
  size: number;
};

export type ApiGameListPlayer = {
  id?: number;
  name: string;
  race_name?: string;
  color: string;
  num_stars: number;
};

export type RaceTraitId =
  | "growth_minus_50"
  | "growth_plus_50"
  | "growth_plus_100"
  | "food_minus_1"
  | "food_plus_1"
  | "food_plus_2"
  | "production_minus_1"
  | "production_plus_1"
  | "production_plus_2"
  | "research_minus_1"
  | "research_plus_1"
  | "research_plus_2"
  | "money_minus_half"
  | "money_plus_half"
  | "money_plus_1"
  | "ship_defense"
  | "ship_attack"
  | "ground_combat"
  | "spying"
  | "low_g_world"
  | "high_g_world"
  | "aquatic"
  | "subterranean"
  | "large_homeworld"
  | "rich_homeworld"
  | "poor_homeworld"
  | "artifacts_homeworld"
  | "cybernetic"
  | "lithovore"
  | "uncreative"
  | "creative"
  | "tolerant"
  | "fantastic_traders"
  | "omniscient";

export type RaceDesign = {
  name: string;
  government: string;
  trait_ids: RaceTraitId[] | null;
  playable?: boolean;
  built_in?: boolean;
};

/** @deprecated Prefer RaceDesign; kept for readability in older code paths. */
export type CustomRaceDesign = RaceDesign;

export type ApiGovernment = {
  name: string;
  description?: string;
  pick_cost: number;
  default?: boolean;
};

export type NewGameOptions = {
  player_name: string;
  player_color: string;
  num_stars: number;
  difficulty: "easy" | "medium" | "hard";
  galaxy_size: "tiny" | "medium" | "huge";
  galaxy_age: "organic_rich" | "average" | "mineral_rich";
  players: number;
  tech_level: "low" | "average" | "advanced";
  /** Built-in slug (e.g. `human`) or `custom` for saved custom races. */
  race: string;
  custom_race_name?: string;
  custom_race?: CustomRaceDesign;
};

export type ApiCustomRaceListResponse = {
  custom_races: CustomRaceDesign[];
};

export type ApiRace = {
  id: number;
  name: string;
  government?: string;
  trait_ids?: string[];
  grants?: ApiGrants;
  Stats?: {
    PopulationGrowth?: number;
    Farming?: number;
  };
  stats?: {
    population_growth_percent?: number;
    farming?: number;
    production?: number;
    research?: number;
    money_bonus?: number;
  };
};

export type ApiModifier = {
  name: string;
  value?: number;
};

export type ApiUnlocks = {
  buildings?: string[];
  hull_sizes?: string[];
  drives?: string[];
  armors?: string[];
  computers?: string[];
  fuel_cells?: string[];
  shields?: string[];
  beams?: string[];
  ship_systems?: string[];
};

export type ApiGrants = {
  unlocks?: ApiUnlocks;
  modifiers?: ApiModifier[];
};

export type ApiModifierDefinition = {
  name?: string;
  description?: string;
  scope?: string;
  stat?: string;
  operation?: string;
};

export type ApiBuilding = {
  id?: number;
  name?: string;
  description?: string;
  icon?: string;
  cost?: number;
  maintenance?: number;
  unique?: boolean;
  upgrades?: string[];
  grants?: ApiGrants;
};

export type ApiPlanet = {
  id: number;
  name: string;
  owner: number;
  parent_star: number;
  mineral_class: number;
  size: number;
  gravity: number;
  environment: number;
  population: number;
  max_population?: number;
  last_turn_pop_growth: number;
  food: {
    farmers: number;
    produced_per_farmer: number;
    produced_per_building: number;
  };
  industry: {
    workers: number;
    produced_per_worker: number;
    produced_per_building: number;
    produced_per_colonist?: number;
  };
  science: {
    scientists: number;
    produced_per_scientist: number;
    produced_per_building: number;
  };
  income?: {
    taxes: number;
    buildings: number;
    trade_goods: number;
    maintenance: number;
    total: number;
    bonus?: number;
  };
  morale?: {
    bonus?: number;
    multiplier?: number;
    sources?: ApiMoraleSource[];
  };
  government?: {
    name?: string;
    food_labor_multiplier?: number;
    industry_labor_multiplier?: number;
    science_labor_multiplier?: number;
    tax_multiplier?: number;
    morale_neutral?: boolean;
  };
  pollution?: number;
  gravity_multiplier?: number;
  marines: number;
  armor: number;
  current_project?: ApiBuilding | string;
  build_queue?: (ApiBuilding | string)[] | null;
  buildings?: (ApiBuilding | string)[] | null;
  industry_stack: number;
  build_bucket: number;
  is_building: boolean;
  autobuild?: boolean;
  terraforming_count?: number;
};

export type ApiMoraleSource = {
  name: string;
  type: string;
  bonus: number;
};

export type ApiStar = {
  id: number;
  name?: string;
  planets?: ApiPlanet[];
  color: number;
  x: number;
  y: number;
};

export type ApiGalaxy = {
  stars: ApiStar[];
  width?: number;
  height?: number;
};

export type ApiWeapon = Record<string, never>;
export type ApiSpecialSystem = Record<string, never>;

export type ApiDrive = {
  name: string;
  interstellar_speed: number;
  combat_speed: number;
};

export type ApiComputer = {
  name: string;
  beam_attack_bonus: number;
};

export type ApiShield = {
  name: string;
  recharge_rate: number;
  reduce_damage: number;
  absortion_factor: number;
};

export type ApiArmor = {
  name: string;
  multiplier: number;
  ground_troop_strength: number;
};

export type ApiHullSize = {
  name: string;
  size_rating: number;
  space: number;
  marines: number;
  base_speed: number;
  base_hp: number;
  computer_hp: number;
  drive_hp: number;
  shield_hp: number;
};

export type ApiFuelCell = {
  name: string;
  autonomy: number;
};

export type ApiShipSystem = {
  name: string;
  description?: string;
  beam_attack_bonus?: number;
  beam_defense_bonus?: number;
};

export type ApiShipDesign = {
  name: string;
  weapons?: ApiWeapon[];
  special_systems?: ApiSpecialSystem[];
  base_space?: number;
  drive: ApiDrive;
  computer: ApiComputer;
  shield: ApiShield;
  armor: ApiArmor;
  hull_size: ApiHullSize;
  fuel_cell: ApiFuelCell;
};

export type ApiShipDesignDef = {
  name: string;
  drive: string;
  computer: string;
  shield: string;
  armor: string;
  hull_size: string;
  fuel_cell: string;
};

/** Ship design template from GET /ship-designs (built-in + custom). */
export type ShipDesignTemplateRow = {
  name: string;
  drive: string;
  computer: string;
  shield: string;
  armor: string;
  hull_size: string;
  fuel_cell: string;
  built_in: boolean;
};

export type ApiShipParts = {
  hull_sizes: ApiHullSize[];
  drives: ApiDrive[];
  armors: ApiArmor[];
  computers: ApiComputer[];
  fuel_cells: ApiFuelCell[];
  shields: ApiShield[];
};

export type ApiTechLevel = {
  name: string;
  cost: number;
  force_all?: boolean;
  available_techs: string[];
};

export type ApiTechSubject = {
  name: string;
  levels: ApiTechLevel[];
};

export type ApiTechTree = {
  subjects: ApiTechSubject[];
  all_techs?: ApiTech[];
};

export type ApiTech = {
  name: string;
  description?: string;
  hidden?: boolean;
  type?: string;
  pick_cost?: number;
  default?: boolean;
  upgrades?: string[];
  cost?: number;
  maintenance?: number;
  built_on?: string[];
  unique?: boolean;
  grants?: ApiGrants;
};

export type ApiResources = {
  ship_parts: ApiShipParts;
  ship_system_collection?: {
    ship_systems: ApiShipSystem[];
  };
  building_collection?: {
    buildings: ApiBuilding[];
  };
  modifier_collection?: {
    modifiers: ApiModifierDefinition[];
  };
  ship_design_collection?: {
    ship_design_defs?: ApiShipDesignDef[];
    ship_designs: ApiShipDesign[];
  };
  tech_tree?: ApiTechTree;
  tech_collection?: {
    techs: ApiTech[];
  };
};

export type ApiPlayer = {
  id: number;
  name: string;
  color: string;
  race?: ApiRace;
  /** Omitted for rival players in fog-of-war filtered game views. */
  treasury?: number;
  tax_rate?: number;
  research_points?: number;
  researched_techs?: string[] | null;
  hyper_advanced?: Record<string, number> | null;
  current_research?: ApiResearchProject;
  available?: {
    buildings?: string[] | null;
    hull_sizes?: string[] | null;
    drives?: string[] | null;
    armors?: string[] | null;
    computers?: string[] | null;
    fuel_cells?: string[] | null;
    shields?: string[] | null;
    beams?: string[] | null;
    ship_systems?: string[] | null;
  };
  fleets?: number[] | null;
  freighters?: number;
  freighters_used?: number;
  ship_designs?: ApiShipDesign[] | null;
  command_points_capacity?: number;
  command_points_used?: number;
  command_points_remaining?: number;
  command_overdraw_bc?: number;
  command_points_starting_bonus?: number;
  evolutionary_mutation_points?: number;
  has_used_evolutionary_mutation?: boolean;
};

export type ApiFleetShip = {
  type: string;
  name: string;
  icon?: string;
  design_name?: string;
  hull_size?: string;
  drive?: string;
  interstellar_speed?: number;
  fuel_cell?: string;
  fuel_range_parsecs?: number;
  unlimited_fuel_range?: boolean;
};

export type ApiFleet = {
  id: number;
  owner: number;
  star_id: number;
  x: number;
  y: number;
  origin_star_id?: number;
  destination_star_id?: number;
  total_turns?: number;
  turns_remaining?: number;
  ships: ApiFleetShip[];
};

export type ApiPopulationTransfer = {
  id: number;
  owner: number;
  source_planet_id: number;
  destination_planet_id: number;
  population_units: number;
  freighters_reserved: number;
  origin_star_id: number;
  destination_star_id: number;
  total_turns: number;
  turns_remaining: number;
};

export type ApiResearchProject = {
  subject: string;
  tech_name?: string;
  tech_names?: string[];
  level_name?: string;
  progress: number;
};

export type ApiPlayerColorOptions = {
  max_players: number;
  colors: string[];
};

export type ApiTurnStatus = {
  mode: "simultaneous";
  ready_player_ids: number[];
  waiting_player_ids: number[];
  required_player_ids: number[];
};

export type ApiTurnResponse = {
  message?: string;
  ok: boolean;
  advanced: boolean;
  turn_status: ApiTurnStatus;
};

export type ApiGame = {
  galaxy: ApiGalaxy;
  players: ApiPlayer[];
  fleets?: ApiFleet[] | null;
  population_transfers?: ApiPopulationTransfer[] | null;
  id: string;
  current_turn: number;
  created_at?: string | null;
  last_turn_at?: string | null;
  debug?: boolean;
  resources: ApiResources;
  pending_colony_bases?: Record<number, number[]>; // playerID -> planetIDs
  pending_colony_ships?: Record<number, number[]>; // playerID -> fleetIDs
  pending_artificial_planets?: Record<number, number[]>; // playerID -> source planet IDs
  pending_battles?: ApiTacticalBattle[] | null;
  turn_status?: ApiTurnStatus;
};

export type ApiGameUpdateMessage = {
  game_update: ApiGame;
};
