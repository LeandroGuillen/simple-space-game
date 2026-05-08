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
  ApiTech,
  ApiTechLevel,
  ApiTechSubject,
} from "./types/api";
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
  TechLevelViewModel,
  TechSubjectViewModel,
  TechViewModel,
} from "./types/domain";

export function mapGame(apiGame: ApiGame): GameViewModel {
  const players = apiGame.players.map(mapPlayer);
  const playerColorById = new Map(
    players.map((player) => [player.id, player.color]),
  );
  const buildings =
    apiGame.resources.building_collection?.buildings?.map((building, index) =>
      mapBuilding(building, index + 1),
    ) ?? [];
  const buildingByName = new Map(
    buildings.map((building) => [building.name.toLowerCase(), building]),
  );

  return {
    id: apiGame.id,
    currentTurn: apiGame.current_turn,
    galaxy: {
      stars: apiGame.galaxy.stars.map((star) =>
        mapStar(star, playerColorById, buildingByName),
      ),
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
        buildings,
      },
      shipDesignCollection: {
        shipDesigns:
          apiGame.resources.ship_design_collection?.ship_designs?.map(
            (design) => mapShipDesign(design, true),
          ) ?? [],
      },
      techTree: {
        subjects:
          apiGame.resources.tech_tree?.subjects?.map(mapTechSubject) ?? [],
      },
      techCollection: {
        techs: apiGame.resources.tech_collection?.techs?.map(mapTech) ?? [],
      },
    },
  };
}

function mapPlayer(player: ApiPlayer): PlayerViewModel {
  return {
    id: player.id,
    name: player.name,
    color: player.color,
    raceName: player.race?.name || "Unassigned",
    treasury: player.treasury,
    taxRate: player.tax_rate,
    researchPoints: player.research_points,
    knownTechs: player.known_techs ?? [],
    availableBuildings: player.available?.buildings ?? [],
    currentResearch: player.current_research
      ? {
          subject: player.current_research.subject,
          techName: player.current_research.tech_name ?? "",
          techNames: player.current_research.tech_names ?? [],
          levelName: player.current_research.level_name ?? "",
          progress: player.current_research.progress,
        }
      : null,
    fleets: player.fleets ?? [],
    freighters: player.freighters,
    freightersUsed: player.freighters_used,
    shipDesigns:
      player.ship_designs?.map((design) => mapShipDesign(design, false)) ?? [],
  };
}

function mapTechSubject(subject: ApiTechSubject): TechSubjectViewModel {
  return {
    name: subject.name,
    levels: subject.levels.map(mapTechLevel),
  };
}

function mapTechLevel(level: ApiTechLevel): TechLevelViewModel {
  return {
    name: level.name,
    cost: level.cost,
    forceAll: Boolean(level.force_all),
    availableTechs: level.available_techs,
  };
}

function mapTech(tech: ApiTech): TechViewModel {
  return {
    name: tech.name,
    description: tech.description ?? "",
    cost: tech.cost ?? 0,
    maintenance: tech.maintenance ?? 0,
    effects: tech.effects ?? null,
    builtOn: tech.built_on ?? [],
    unique: Boolean(tech.unique),
  };
}

function mapStar(
  star: ApiStar,
  playerColorById: Map<number, string>,
  buildingByName: Map<string, BuildingViewModel>,
): StarSummary {
  const explored = Boolean(star.name);
  const planets = (star.planets ?? []).map((planet) =>
    mapPlanet(planet, buildingByName),
  );
  const ownedBy = planets.find((planet) => planet.owner > 0)?.owner;
  return {
    id: star.id,
    name: star.name || "",
    planets,
    color: star.color,
    x: star.x,
    y: star.y,
    ownedPlanets: planets.filter((planet) => planet.owner > 0).length,
    explored,
    ownerColor: ownedBy ? (playerColorById.get(ownedBy) ?? null) : null,
  };
}

export function mapPlayerColorOptions(options: {
  max_players: number;
  colors: string[];
}): PlayerColorOptions {
  return {
    maxPlayers: options.max_players,
    colors: options.colors,
  };
}

function mapPlanet(
  planet: ApiPlanet,
  buildingByName: Map<string, BuildingViewModel>,
): PlanetViewModel {
  const buildings =
    planet.buildings
      ?.map((building) => mapPlanetBuilding(building, buildingByName))
      .filter((building): building is BuildingViewModel => Boolean(building)) ??
    [];

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
    income: mapIncome(planet, buildings),
    pollution: planet.pollution ?? 0,
    marines: planet.marines,
    armor: planet.armor,
    currentProject: mapPlanetBuilding(planet.current_project, buildingByName),
    buildQueue:
      planet.build_queue
        ?.map((building) => mapPlanetBuilding(building, buildingByName))
        .filter((building): building is BuildingViewModel => Boolean(building)) ??
      [],
    buildings,
    industryStack: planet.industry_stack,
    buildBucket: planet.build_bucket,
    isBuilding: planet.is_building,
    maxPopulation: planet.size * 4 * 1000,
  };
}

function mapPlanetBuilding(
  building: ApiBuilding | string | undefined,
  buildingByName: Map<string, BuildingViewModel>,
): BuildingViewModel | null {
  if (!building) {
    return null;
  }
  if (typeof building === "string") {
    return buildingByName.get(building.toLowerCase()) ?? null;
  }
  if (!building.name) {
    return null;
  }
  return mapBuilding(building);
}

function mapIncome(planet: ApiPlanet, buildingsOnPlanet: BuildingViewModel[]) {
  if (planet.income) {
    return {
      taxes: planet.income.taxes,
      buildings: planet.income.buildings,
      tradeGoods: planet.income.trade_goods,
      maintenance: planet.income.maintenance,
      total: planet.income.total,
    };
  }

  const taxes = Math.floor(planet.population / 1000);
  const food =
    planet.food.farmers * planet.food.produced_per_farmer +
    planet.food.produced_per_building;
  const foodSurplus = Math.max(0, food - taxes);
  const grossIndustry =
    planet.industry.workers * planet.industry.produced_per_worker +
    planet.industry.produced_per_building;
  const industry = Math.max(
    0,
    grossIndustry - mapPlanetPollution(planet, planet.industry.workers),
  );
  const tradeGoods = planet.is_building
    ? 0
    : Math.floor(foodSurplus / 2) + Math.floor(industry / 2);
  const maintenance = buildingsOnPlanet.reduce(
    (total, building) => total + building.maintenance,
    0,
  );
  const buildings = buildingsOnPlanet.reduce(
    (total, building) =>
      total +
      building.effects.reduce(
        (effectTotal, effect) =>
          effectTotal + (effect.name === "income" ? effect.bonus : 0),
        0,
      ),
    0,
  );

  return {
    taxes,
    buildings,
    tradeGoods,
    maintenance,
    total: taxes + buildings + tradeGoods - maintenance,
  };
}

function mapPlanetPollution(planet: ApiPlanet, workers: number) {
  const buildingNames = (planet.buildings ?? []).map((building) =>
    (typeof building === "string" ? building : (building.name ?? "")).toLowerCase(),
  );
  const hasProcessor = buildingNames.includes("pollution processor");
  const hasRenewer = buildingNames.includes("atmospheric renewer");
  const divisor =
    hasProcessor && hasRenewer ? 8 : hasRenewer ? 4 : hasProcessor ? 2 : 1;
  const hasNanoDisassemblers = buildingNames.includes("nano disassemblers");
  const pollutionSize = planet.size * 2 * (hasNanoDisassemblers ? 2 : 1);
  const dirtyProduction =
    Math.max(0, workers) * planet.industry.produced_per_worker;
  return Math.max(
    0,
    Math.ceil((dirtyProduction / divisor - pollutionSize) / 2),
  );
}

function mapShipDesign(
  ship: ApiShipDesign,
  isCommon: boolean,
): ShipDesignViewModel {
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

function mapBuilding(building: ApiBuilding, fallbackId = 0): BuildingViewModel {
  return {
    id: building.id ?? fallbackId,
    name: building.name ?? "Unknown building",
    description: building.description ?? "",
    icon: building.icon ?? "🏗️",
    cost: building.cost ?? 0,
    maintenance: building.maintenance ?? 0,
    effects: building.effects?.map(mapBuildingEffect) ?? [],
  };
}

function mapBuildingEffect(effect: ApiEffect) {
  if (effect.name) {
    return {
      name: effect.name,
      description: effect.description ?? "",
      bonus: effect.bonus ?? 0,
    };
  }

  const dynamicNames = [
    "industryBonus",
    "flatIndustryBonus",
    "flatFoodBonus",
    "scienceBonus",
    "flatScienceBonus",
  ] as const;
  const name =
    dynamicNames.find((candidate) => effect[candidate] !== undefined) ??
    "unknown";
  return {
    name,
    description: effect.description ?? "",
    bonus: name === "unknown" ? 0 : (effect[name] ?? 0),
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
