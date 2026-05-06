import type {
  ApiArmor,
  ApiBuilding,
  ApiComputer,
  ApiDrive,
  ApiEffect,
  ApiFuelCell,
  ApiGame,
  ApiHullSize,
  ApiPlanet,
  ApiPlayer,
  ApiShield,
  ApiShipDesign,
  ApiStar,
} from './types/api';
import type {
  ArmorViewModel,
  BuildingViewModel,
  ComputerViewModel,
  DriveViewModel,
  FuelCellViewModel,
  GameViewModel,
  HullSizeViewModel,
  PlanetViewModel,
  PlayerColorOptions,
  PlayerViewModel,
  ShieldViewModel,
  ShipDesignViewModel,
  StarSummary,
} from './types/domain';

export function mapGame(apiGame: ApiGame): GameViewModel {
  const players = apiGame.players.map(mapPlayer);
  const playerColorById = new Map(players.map((player) => [player.id, player.color]));

  return {
    id: apiGame.id,
    currentTurn: apiGame.current_turn,
    galaxy: {
      stars: apiGame.galaxy.stars.map((star) => mapStar(star, playerColorById)),
    },
    players,
    resources: {
      shipParts: {
        hullSizes: apiGame.resources.ship_parts.hull_sizes.map(mapHullSize),
        drives: apiGame.resources.ship_parts.drives.map(mapDrive),
        armors: apiGame.resources.ship_parts.armors.map(mapArmor),
        computers: apiGame.resources.ship_parts.computers.map(mapComputer),
        fuelCells: apiGame.resources.ship_parts.fuel_cells.map(mapFuelCell),
        shields: apiGame.resources.ship_parts.shields.map(mapShield),
      },
      buildingCollection: {
        buildings: apiGame.resources.building_collection?.buildings?.map(mapBuilding) ?? [],
      },
      shipDesignCollection: {
        shipDesigns:
          apiGame.resources.ship_design_collection?.ship_designs?.map((design) => mapShipDesign(design, true)) ?? [],
      },
    },
  };
}

function mapPlayer(player: ApiPlayer): PlayerViewModel {
  return {
    id: player.id,
    name: player.name,
    color: player.color,
    raceName: player.race?.name || 'Unassigned',
    treasury: player.treasury,
    taxRate: player.tax_rate,
    researchPoints: player.research_points,
    fleets: player.fleets ?? [],
    freighters: player.freighters,
    freightersUsed: player.freighters_used,
    shipDesigns: player.ship_designs?.map((design) => mapShipDesign(design, false)) ?? [],
  };
}

function mapStar(star: ApiStar, playerColorById: Map<number, string>): StarSummary {
  const explored = Boolean(star.name);
  const planets = (star.planets ?? []).map(mapPlanet);
  const ownedBy = planets.find((planet) => planet.owner > 0)?.owner;
  return {
    id: star.id,
    name: star.name || '',
    planets,
    color: star.color,
    x: star.x,
    y: star.y,
    ownedPlanets: planets.filter((planet) => planet.owner > 0).length,
    explored,
    ownerColor: ownedBy ? (playerColorById.get(ownedBy) ?? null) : null,
  };
}

export function mapPlayerColorOptions(options: { max_players: number; colors: string[] }): PlayerColorOptions {
  return {
    maxPlayers: options.max_players,
    colors: options.colors,
  };
}

function mapPlanet(planet: ApiPlanet): PlanetViewModel {
  return {
    id: planet.id,
    name: planet.name,
    owner: planet.owner,
    parentStar: planet.parent_star,
    mineralClass: planet.mineral_class,
    size: planet.size,
    gravity: planet.gravity,
    environment: planet.environment,
    population: planet.population,
    last_turn_pop_growth: planet.last_turn_pop_growth,
    food: planet.food,
    industry: planet.industry,
    science: planet.science,
    marines: planet.marines,
    armor: planet.armor,
    currentProject: hasBuildingName(planet.current_project) ? mapBuilding(planet.current_project) : null,
    buildQueue: planet.build_queue?.map(mapBuilding) ?? [],
    buildings: planet.buildings?.map(mapBuilding) ?? [],
    industryStack: planet.industry_stack,
    buildBucket: planet.build_bucket,
    isBuilding: planet.is_building,
    maxPopulation: planet.size * 4 * 1000,
  };
}

function hasBuildingName(building: ApiBuilding | undefined): building is ApiBuilding {
  return Boolean(building?.name);
}

function mapShipDesign(ship: ApiShipDesign, isCommon: boolean): ShipDesignViewModel {
  return {
    name: ship.name,
    baseSpace: ship.base_space ?? 0,
    drive: mapDrive(ship.drive),
    computer: mapComputer(ship.computer),
    shield: mapShield(ship.shield),
    armor: mapArmor(ship.armor),
    hullSize: mapHullSize(ship.hull_size),
    fuelCell: mapFuelCell(ship.fuel_cell),
    isCommon,
  };
}

function mapBuilding(building: ApiBuilding): BuildingViewModel {
  return {
    id: building.id ?? 0,
    name: building.name ?? 'Unknown building',
    description: building.description ?? '',
    icon: building.icon ?? '🏗️',
    cost: building.cost ?? 0,
    maintenance: building.maintenance ?? 0,
    effects: building.effects?.map(mapBuildingEffect) ?? [],
  };
}

function mapBuildingEffect(effect: ApiEffect) {
  if (effect.name) {
    return {
      name: effect.name,
      description: effect.description ?? '',
      bonus: effect.bonus ?? 0,
    };
  }

  const dynamicNames = ['industryBonus', 'flatIndustryBonus', 'flatAgricultureBonus', 'scienceBonus', 'flatScienceBonus'] as const;
  const name = dynamicNames.find((candidate) => effect[candidate] !== undefined) ?? 'unknown';
  return {
    name,
    description: effect.description ?? '',
    bonus: name === 'unknown' ? 0 : (effect[name] ?? 0),
  };
}

function mapDrive(drive: ApiDrive): DriveViewModel {
  return {
    name: drive.name,
    interstellarSpeed: drive.interstellar_speed,
    combatSpeed: drive.combat_speed,
  };
}

function mapComputer(computer: ApiComputer): ComputerViewModel {
  return {
    name: computer.name,
    beamAttackBonus: computer.beam_attack_bonus,
  };
}

function mapShield(shield: ApiShield): ShieldViewModel {
  return {
    name: shield.name,
    rechargeRate: shield.recharge_rate,
    reduceDamage: shield.reduce_damage,
    absortionFactor: shield.absortion_factor,
  };
}

function mapArmor(armor: ApiArmor): ArmorViewModel {
  return {
    name: armor.name,
    multiplier: armor.multiplier,
    groundTroopStrength: armor.ground_troop_strength,
  };
}

function mapHullSize(hullSize: ApiHullSize): HullSizeViewModel {
  return {
    name: hullSize.name,
    sizeRating: hullSize.size_rating,
    space: hullSize.space,
    marines: hullSize.marines,
    baseSpeed: hullSize.base_speed,
    baseHp: hullSize.base_hp,
    computerHp: hullSize.computer_hp,
    driveHp: hullSize.drive_hp,
    shieldHp: hullSize.shield_hp,
  };
}

function mapFuelCell(fuelCell: ApiFuelCell): FuelCellViewModel {
  return {
    name: fuelCell.name,
    autonomy: fuelCell.autonomy,
  };
}
