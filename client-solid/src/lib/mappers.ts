import type {
  ApiArmor,
  ApiBuilding,
  ApiComputer,
  ApiDrive,
  ApiFleet,
  ApiFleetShip,
  ApiFuelCell,
  ApiGame,
  ApiGrants,
  ApiHullSize,
  ApiPlanet,
  ApiPlayer,
  ApiShield,
  ApiShipDesign,
  ApiShipSystem,
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
  FleetShipViewModel,
  FleetViewModel,
  GameViewModel,
  HullSizeViewModel,
  ModifierDefinitionViewModel,
  PlanetViewModel,
  PlayerColorOptions,
  PlayerViewModel,
  ShieldViewModel,
  ShipDesignViewModel,
  ShipSystemViewModel,
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
  const modifiers =
    apiGame.resources.modifier_collection?.modifiers?.map(mapModifierDefinition) ?? [];
  const modifierByName = new Map(
    modifiers.map((modifier) => [modifier.name.toLowerCase(), modifier]),
  );
  const buildings =
    apiGame.resources.building_collection?.buildings?.map((building, index) =>
      mapBuilding(building, index + 1, modifierByName),
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
    fleets: (apiGame.fleets ?? []).map(mapFleet),
    resources: {
      shipParts: {
        hullSizes: apiGame.resources.ship_parts.hull_sizes.map(mapHullSize),
        drives: apiGame.resources.ship_parts.drives.map(mapDrive),
        armors: apiGame.resources.ship_parts.armors.map(mapArmor),
        computers: apiGame.resources.ship_parts.computers.map(mapComputer),
        fuelCells: apiGame.resources.ship_parts.fuel_cells.map(mapFuelCell),
        shields: apiGame.resources.ship_parts.shields.map(mapShield),
      },
      shipSystemCollection: {
        shipSystems:
          apiGame.resources.ship_system_collection?.ship_systems?.map(mapShipSystem) ?? [],
      },
      buildingCollection: {
        buildings,
      },
      modifierCollection: {
        modifiers,
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
        techs:
          apiGame.resources.tech_tree?.all_techs?.map(mapTech) ??
          apiGame.resources.tech_collection?.techs?.map(mapTech) ??
          [],
      },
    },
    pendingColonyBases: apiGame.pending_colony_bases ?? {},
  };
}

function mapFleet(fleet: ApiFleet): FleetViewModel {
  return {
    id: fleet.id,
    ownerId: fleet.owner,
    starId: fleet.star_id,
    x: fleet.x,
    y: fleet.y,
    originStarId: fleet.origin_star_id ?? null,
    destinationStarId: fleet.destination_star_id ?? null,
    totalTurns: fleet.total_turns ?? 0,
    turnsRemaining: fleet.turns_remaining ?? 0,
    ships: (fleet.ships ?? []).map(mapFleetShip),
  };
}

function mapFleetShip(ship: ApiFleetShip): FleetShipViewModel {
  return {
    type: ship.type,
    name: ship.name,
    icon: ship.icon ?? "🚀",
  };
}

function mapPlayer(player: ApiPlayer): PlayerViewModel {
  const available = {
    buildings: player.available?.buildings ?? [],
    hullSizes: player.available?.hull_sizes ?? [],
    drives: player.available?.drives ?? [],
    armors: player.available?.armors ?? [],
    computers: player.available?.computers ?? [],
    fuelCells: player.available?.fuel_cells ?? [],
    shields: player.available?.shields ?? [],
    beams: player.available?.beams ?? [],
    shipSystems: player.available?.ship_systems ?? [],
  };

  return {
    id: player.id,
    name: player.name,
    color: player.color,
    raceName: player.race?.name || "Unassigned",
    treasury: player.treasury,
    taxRate: player.tax_rate,
    researchPoints: player.research_points,
    researchedTechs: player.researched_techs ?? [],
    hyperAdvanced: player.hyper_advanced ?? {},
    available,
    availableBuildings: available.buildings,
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
    hidden: Boolean(tech.hidden),
    cost: tech.cost ?? 0,
    maintenance: tech.maintenance ?? 0,
    builtOn: tech.built_on ?? [],
    unique: Boolean(tech.unique),
    grants: mapGrants(tech.grants),
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
    industry: {
      ...planet.industry,
      produced_per_colonist: planet.industry.produced_per_colonist ?? 0,
    },
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
    maxPopulation: planet.max_population ?? planet.size * 4 * 1000,
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
      bonus: planet.income.bonus ?? 0,
    };
  }

  const taxes = Math.floor(planet.population / 1000);
  const food =
    planet.food.farmers * planet.food.produced_per_farmer +
    planet.food.produced_per_building;
  const foodSurplus = Math.max(0, food - taxes);
  const grossIndustry =
    planet.industry.workers * planet.industry.produced_per_worker +
    planet.industry.produced_per_building +
    Math.floor(planet.population / 1000) *
      (planet.industry.produced_per_colonist ?? 0);
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
      building.modifiers.reduce(
        (modifierTotal, modifier) =>
          modifierTotal + (modifier.name === "income_flat" ? modifier.bonus : 0),
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
    bonus: 0,
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

function mapShipSystem(shipSystem: ApiShipSystem): ShipSystemViewModel {
  return {
    name: shipSystem.name,
    description: shipSystem.description ?? "",
    beamAttackBonus: shipSystem.beam_attack_bonus ?? 0,
    beamDefenseBonus: shipSystem.beam_defense_bonus ?? 0,
  };
}

function mapBuilding(
  building: ApiBuilding,
  fallbackId = 0,
  modifierByName: Map<string, ModifierDefinitionViewModel> = new Map(),
): BuildingViewModel {
  const grants = mapGrants(building.grants);
  return {
    id: building.id ?? fallbackId,
    name: building.name ?? "Unknown building",
    description: building.description ?? "",
    icon: building.icon ?? "🏗️",
    cost: building.cost ?? 0,
    maintenance: building.maintenance ?? 0,
    unique: Boolean(building.unique),
    grants,
    modifiers: grants.modifiers.map((modifier) =>
      mapBuildingModifier(modifier, modifierByName),
    ),
  };
}

function mapBuildingModifier(
  modifier: { name: string; value: number },
  modifierByName: Map<string, ModifierDefinitionViewModel>,
) {
  const definition = modifierByName.get(modifier.name.toLowerCase());
  return {
    name: modifier.name,
    description: definition?.description ?? "",
    bonus: modifier.value,
  };
}

function mapModifierDefinition(modifier: {
  name?: string;
  description?: string;
  scope?: string;
  stat?: string;
  operation?: string;
}): ModifierDefinitionViewModel {
  return {
    name: modifier.name ?? "",
    description: modifier.description ?? "",
    scope: modifier.scope ?? "",
    stat: modifier.stat ?? "",
    operation: modifier.operation ?? "",
  };
}

function mapGrants(grants: ApiGrants | undefined) {
  return {
    unlocks: {
      buildings: grants?.unlocks?.buildings ?? [],
      hullSizes: grants?.unlocks?.hull_sizes ?? [],
      drives: grants?.unlocks?.drives ?? [],
      armors: grants?.unlocks?.armors ?? [],
      computers: grants?.unlocks?.computers ?? [],
      fuelCells: grants?.unlocks?.fuel_cells ?? [],
      shields: grants?.unlocks?.shields ?? [],
      beams: grants?.unlocks?.beams ?? [],
      shipSystems: grants?.unlocks?.ship_systems ?? [],
    },
    modifiers: grants?.modifiers?.map((modifier) => ({
      name: modifier.name,
      value: modifier.value ?? 0,
    })) ?? [],
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
