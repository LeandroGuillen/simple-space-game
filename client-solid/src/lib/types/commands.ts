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

export type PendingCommands = {
  build: BuildCommand[];
  movePopulation: MovePopulationCommand[];
  selectResearch?: SelectResearchCommand;
};

export type TurnMovePopulationCommand = {
  planetId: number;
  farmers: number;
  workers: number;
  scientists: number;
};

export type TurnCommands = {
  build: BuildCommand[];
  movePopulation?: TurnMovePopulationCommand;
  selectResearch?: SelectResearchCommand;
};
