export type ApiStatusResponse = {
  ok: boolean
}

export type ApiGameListItem = {
  game_id: string
  num_stars: number
  current_turn: number
  players: ApiGameListPlayer[]
}

export type ApiGameListPlayer = {
  name: string
  color: string
  num_stars: number
}

export type ApiRace = {
  id: number
  name: string
  Stats?: {
    PopulationGrowth?: number
    Farming?: number
  }
}

export type ApiEffect = {
  id?: number
  name?: string
  description?: string
  bonus?: number
  industryBonus?: number
  flatIndustryBonus?: number
  flatAgricultureBonus?: number
}

export type ApiBuilding = {
  id?: number
  name?: string
  cost?: number
  maintenance?: number
  effects?: ApiEffect[]
}

export type ApiPlanet = {
  id: number
  name: string
  owner: number
  parent_star: number
  mineral_class: number
  size: number
  gravity: number
  environment: number
  population: number
  farmers: number
  workers: number
  scientists: number
  marines: number
  armor: number
  current_project?: ApiBuilding
  build_queue?: ApiBuilding[] | null
  buildings?: ApiBuilding[] | null
  industry_stack: number
  build_bucket: number
  is_building: boolean
}

export type ApiStar = {
  id: number
  name?: string
  planets?: ApiPlanet[]
  color: number
  x: number
  y: number
}

export type ApiGalaxy = {
  stars: ApiStar[]
}

export type ApiWeapon = Record<string, never>
export type ApiSpecialSystem = Record<string, never>

export type ApiDrive = {
  name: string
  interstellar_speed: number
  combat_speed: number
}

export type ApiComputer = {
  name: string
  beam_attack_bonus: number
}

export type ApiShield = {
  name: string
  recharge_rate: number
  reduce_damage: number
  absortion_factor: number
}

export type ApiArmor = {
  name: string
  multiplier: number
  ground_troop_strength: number
}

export type ApiHullSize = {
  name: string
  size_rating: number
  space: number
  marines: number
  base_speed: number
  base_hp: number
  computer_hp: number
  drive_hp: number
  shield_hp: number
}

export type ApiFuelCell = {
  name: string
  autonomy: number
}

export type ApiShipDesign = {
  name: string
  weapons?: ApiWeapon[]
  special_systems?: ApiSpecialSystem[]
  base_space?: number
  drive: ApiDrive
  computer: ApiComputer
  shield: ApiShield
  armor: ApiArmor
  hull_size: ApiHullSize
  fuel_cell: ApiFuelCell
}

export type ApiShipDesignDef = {
  name: string
  drive: string
  computer: string
  shield: string
  armor: string
  hull_size: string
  fuel_cell: string
}

export type ApiShipParts = {
  hull_sizes: ApiHullSize[]
  drives: ApiDrive[]
  armors: ApiArmor[]
  computers: ApiComputer[]
  fuel_cells: ApiFuelCell[]
  shields: ApiShield[]
}

export type ApiResources = {
  ship_parts: ApiShipParts
  building_collection?: {
    buildings: ApiBuilding[]
  }
  ship_design_collection?: {
    ship_design_defs?: ApiShipDesignDef[]
    ship_designs: ApiShipDesign[]
  }
}

export type ApiPlayer = {
  id: number
  name: string
  color: string
  race?: ApiRace
  treasury: number
  tax_rate: number
  research_points: number
  fleets?: number[] | null
  freighters: number
  freighters_used: number
  ship_designs?: ApiShipDesign[] | null
}

export type ApiPlayerColorOptions = {
  max_players: number
  colors: string[]
}

export type ApiGame = {
  galaxy: ApiGalaxy
  players: ApiPlayer[]
  id: string
  current_turn: number
  resources: ApiResources
}

export type ApiGameUpdateMessage = {
  game_update: ApiGame
}
