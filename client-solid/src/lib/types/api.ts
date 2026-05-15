export type ApiStatusResponse = {
  ok: boolean;
};

export type ApiGameListItem = {
  game_id: string;
  num_stars: number;
  current_turn: number;
  /** ISO-8601 UTC when the game was created */
  created_at?: string | null;
  /** ISO-8601 UTC when the last turn completed; omit or null if none yet */
  last_turn_at?: string | null;
  players: ApiGameListPlayer[];
};

export type ApiGameListPlayer = {
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
  | "subterranean"
  | "large_homeworld"
  | "rich_homeworld"
  | "poor_homeworld"
  | "artifacts_homeworld";

export type CustomRaceDesign = {
  name: string;
  government: string;
  trait_ids: RaceTraitId[] | null;
  playable?: boolean;
};

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
  race: "human" | "psilon" | "custom";
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
  marines: number;
  armor: number;
  current_project?: ApiBuilding | string;
  build_queue?: (ApiBuilding | string)[] | null;
  buildings?: (ApiBuilding | string)[] | null;
  industry_stack: number;
  build_bucket: number;
  is_building: boolean;
  autobuild?: boolean;
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
  treasury: number;
  tax_rate: number;
  research_points: number;
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
  freighters: number;
  freighters_used: number;
  ship_designs?: ApiShipDesign[] | null;
  command_points_capacity?: number;
  command_points_used?: number;
  command_points_remaining?: number;
  command_overdraw_bc?: number;
  command_points_starting_bonus?: number;
};

export type ApiFleetShip = {
  type: string;
  name: string;
  icon?: string;
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

export type ApiGame = {
  galaxy: ApiGalaxy;
  players: ApiPlayer[];
  fleets?: ApiFleet[] | null;
  id: string;
  current_turn: number;
  created_at?: string | null;
  last_turn_at?: string | null;
  resources: ApiResources;
  pending_colony_bases?: Record<number, number[]>; // playerID -> planetIDs
  pending_colony_ships?: Record<number, number[]>; // playerID -> fleetIDs
};

export type ApiGameUpdateMessage = {
  game_update: ApiGame;
};
