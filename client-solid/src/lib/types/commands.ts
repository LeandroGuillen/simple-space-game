export type MovePopulationCommand = {
  planetId: number;
  from: 'farmers' | 'workers' | 'scientists';
  to: 'farmers' | 'workers' | 'scientists';
  count: number;
};

export type BuildCommand = {
  planetId: number;
  building: string;
  build_queue?: string[];
};

export type SelectResearchCommand = {
  tech_name: string;
};

export type ColonizeCommand = {
  planet_id: number;
  fleet_id?: number;
};

export type ArtificialPlanetCommand = {
  source_planet_id?: number;
  target_planet_id: number;
};

export type FleetMoveCommand = {
  fleet_id: number;
  destination_star_id: number;
  /** Required: server treats a missing JSON field as “move entire fleet”. */
  ship_indexes: number[];
};

export type InvadeCommand = {
  fleet_id: number;
  target_planet_id: number;
  ship_indexes: number[];
};

export type BuyProductionCommand = {
  planetId: number;
};

export type TransferPopulationCommand = {
  source_planet_id: number;
  destination_planet_id: number;
  population_units: number;
};

export type MutateRaceCommand = {
  trait_ids: string[];
};

export type PendingCommands = {
  build: BuildCommand[];
  movePopulation: MovePopulationCommand[];
  transferPopulation?: TransferPopulationCommand[];
  selectResearch?: SelectResearchCommand;
  colonize: ColonizeCommand[];
  artificialPlanet: ArtificialPlanetCommand[];
  fleetMove: FleetMoveCommand[];
  invade: InvadeCommand[];
  buyProduction: BuyProductionCommand[];
  mutateRace?: MutateRaceCommand;
};

export type TurnMovePopulationCommand = {
  planetId: number;
  farmers: number;
  workers: number;
  scientists: number;
};

export type TurnCommands = {
  build: BuildCommand[];
  movePopulation?: TurnMovePopulationCommand[];
  selectResearch?: SelectResearchCommand;
  colonize?: ColonizeCommand;
  colonize_many?: ColonizeCommand[];
  artificial_planet?: ArtificialPlanetCommand;
  artificial_planet_many?: ArtificialPlanetCommand[];
  fleet_move?: FleetMoveCommand[];
  invade?: InvadeCommand[];
  buy_production?: BuyProductionCommand[];
  transfer_population?: TransferPopulationCommand[];
  mutate_race?: MutateRaceCommand;
};
