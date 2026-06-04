import type {
  ApiArmor,
  ApiBuilding,
  ApiComputer,
  ApiDrive,
  ApiFleet,
  ApiFleetShip,
  ApiFuelCell,
  ApiGame,
  ApiPopulationTransfer,
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
  PopulationTransferViewModel,
  PlayerViewModel,
  ShieldViewModel,
  ShipDesignViewModel,
  ShipSystemViewModel,
  StarSummary,
  TechLevelViewModel,
  TechSubjectViewModel,
  TechViewModel,
} from "./types/domain";

const DEFAULT_GALAXY_WIDTH = 700;
const DEFAULT_GALAXY_HEIGHT = 400;

const VIRTUAL_PROJECTS = new Map<string, BuildingViewModel>([
  [
    "trade goods",
    {
      id: 0,
      name: "Trade Goods",
      description: "Converts surplus colony output into treasury income.",
      icon: "💱",
      cost: 0,
      maintenance: 0,
      unique: false,
      upgrades: [],
      grants: emptyGrants(),
      modifiers: [],
    },
  ],
  [
    "housing",
    {
      id: -1,
      name: "Housing",
      description: "Converts colony production into population growth.",
      icon: "🏘️",
      cost: 0,
      maintenance: 0,
      unique: false,
      upgrades: [],
      grants: emptyGrants(),
      modifiers: [],
    },
  ],
]);

export function mapGame(apiGame: ApiGame): GameViewModel {
  if (!apiGame.players) {
    throw new Error("Game payload is missing players.");
  }
  if (!apiGame.resources?.ship_parts) {
    throw new Error("Game payload is missing resources.");
  }
  const players = apiGame.players.map(mapPlayer);
  const commonShipDesigns =
    apiGame.resources.ship_design_collection?.ship_designs?.map((design) =>
      mapShipDesign(design, true),
    ) ?? [];
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
  const shipDesignProjectByName = new Map(
    [...commonShipDesigns, ...players.flatMap((player) => player.shipDesigns)].map(
      (design) => [design.name.toLowerCase(), mapShipDesignProject(design)],
    ),
  );
  const techs =
    apiGame.resources.tech_tree?.all_techs?.map(mapTech) ??
    apiGame.resources.tech_collection?.techs?.map(mapTech) ??
    [];
  const governmentByPlayerId = new Map(
    players.map((player) => [player.id, activeGovernmentForPlayer(player, techs)]),
  );

  return {
    id: apiGame.id,
    currentTurn: apiGame.current_turn,
    galaxy: {
      stars: apiGame.galaxy.stars.map((star) =>
        mapStar(
          star,
          playerColorById,
          buildingByName,
          shipDesignProjectByName,
          governmentByPlayerId,
        ),
      ),
      width: apiGame.galaxy.width ?? DEFAULT_GALAXY_WIDTH,
      height: apiGame.galaxy.height ?? DEFAULT_GALAXY_HEIGHT,
    },
    players,
    fleets: (apiGame.fleets ?? []).map(mapFleet),
    populationTransfers: (apiGame.population_transfers ?? []).map(mapPopulationTransfer),
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
        shipDesigns: commonShipDesigns,
      },
      techTree: {
        subjects:
          apiGame.resources.tech_tree?.subjects?.map(mapTechSubject) ?? [],
      },
      techCollection: {
        techs,
      },
    },
    pendingColonyBases: apiGame.pending_colony_bases ?? {},
    pendingColonyShips: apiGame.pending_colony_ships ?? {},
    pendingArtificialPlanets: apiGame.pending_artificial_planets ?? {},
    turnStatus: {
      mode: apiGame.turn_status?.mode ?? "simultaneous",
      readyPlayerIds: apiGame.turn_status?.ready_player_ids ?? [],
      waitingPlayerIds: apiGame.turn_status?.waiting_player_ids ?? [],
      requiredPlayerIds: apiGame.turn_status?.required_player_ids ?? players.map((player) => player.id),
    },
  };
}

function mapPopulationTransfer(transfer: ApiPopulationTransfer): PopulationTransferViewModel {
  return {
    id: transfer.id,
    owner: transfer.owner,
    sourcePlanetId: transfer.source_planet_id,
    destinationPlanetId: transfer.destination_planet_id,
    populationUnits: transfer.population_units,
    freightersReserved: transfer.freighters_reserved,
    originStarId: transfer.origin_star_id,
    destinationStarId: transfer.destination_star_id,
    totalTurns: transfer.total_turns,
    turnsRemaining: transfer.turns_remaining,
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
    designName: ship.design_name ?? "",
    hullSize: ship.hull_size ?? "",
    drive: ship.drive ?? "",
    interstellarSpeed: ship.interstellar_speed ?? 0,
    fuelCell: ship.fuel_cell ?? "",
    fuelRangeParsecs: ship.fuel_range_parsecs ?? 0,
    unlimitedFuelRange: Boolean(ship.unlimited_fuel_range),
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
    raceGovernment: player.race?.government || "Dictatorship",
    raceTraitIds: player.race?.trait_ids ?? [],
    racePopulationGrowthPercent:
      player.race?.stats?.population_growth_percent ??
      player.race?.Stats?.PopulationGrowth ??
      0,
    raceGrants: mapGrants(player.race?.grants),
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
    commandPointsCapacity: player.command_points_capacity ?? 0,
    commandPointsUsed: player.command_points_used ?? 0,
    commandPointsRemaining: player.command_points_remaining ?? 0,
    commandOverdrawBc: player.command_overdraw_bc ?? 0,
    commandPointsStartingBonus: player.command_points_starting_bonus ?? 0,
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
    type: tech.type ?? "",
    pickCost: tech.pick_cost ?? 0,
    default: Boolean(tech.default),
    upgrades: tech.upgrades ?? [],
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
  shipDesignProjectByName: Map<string, BuildingViewModel>,
  governmentByPlayerId: Map<number, TechViewModel | null>,
): StarSummary {
  const explored = Boolean(star.name);
  const planets = (star.planets ?? []).map((planet) =>
    mapPlanet(
      planet,
      buildingByName,
      shipDesignProjectByName,
      governmentByPlayerId.get(planet.owner) ?? null,
    ),
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
  shipDesignProjectByName: Map<string, BuildingViewModel>,
  government: TechViewModel | null,
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
    morale: {
      bonus: planet.morale?.bonus ?? 0,
      multiplier: planet.morale?.multiplier ?? 1,
      sources:
        planet.morale?.sources?.map((source) => ({
          name: source.name,
          type: source.type,
          bonus: source.bonus,
        })) ?? [],
    },
    government: mapPlanetGovernment(planet, government),
    pollution: planet.pollution ?? 0,
    gravityMultiplier: planet.gravity_multiplier ?? 1,
    marines: planet.marines,
    armor: planet.armor,
    currentProject: mapPlanetProject(
      planet.current_project,
      buildingByName,
      shipDesignProjectByName,
    ),
    buildQueue:
      planet.build_queue
        ?.map((building) =>
          mapPlanetProject(building, buildingByName, shipDesignProjectByName),
        )
        .filter((building): building is BuildingViewModel => Boolean(building)) ??
      [],
    buildings,
    industryStack: planet.industry_stack,
    buildBucket: planet.build_bucket,
    isBuilding: planet.is_building,
    autobuild: planet.autobuild ?? false,
    maxPopulation: planet.max_population ?? planet.size * 4 * 1000,
    terraformingCount: planet.terraforming_count ?? 0,
  };
}

function activeGovernmentForPlayer(player: PlayerViewModel, techs: TechViewModel[]) {
  let active =
    techs.find((tech) => tech.type === "government" && tech.name === player.raceGovernment) ??
    techs.find((tech) => tech.type === "government" && tech.default) ??
    null;
  if (!active) return null;
  const researched = new Set(player.researchedTechs);
  const seen = new Set([active.name]);
  let changed = true;
  while (changed) {
    changed = false;
    for (const tech of techs) {
      if (tech.type !== "government" || tech.upgrades.length === 0 || !researched.has(tech.name)) {
        continue;
      }
      if (!tech.upgrades.includes(active.name)) {
        continue;
      }
      if (seen.has(tech.name)) {
        return active;
      }
      seen.add(tech.name);
      active = tech;
      changed = true;
      break;
    }
  }
  return active;
}

function mapPlanetGovernment(planet: ApiPlanet, government: TechViewModel | null) {
  if (planet.government) {
    return {
      name: planet.government.name ?? government?.name ?? "",
      foodLaborMultiplier: planet.government.food_labor_multiplier ?? 1,
      industryLaborMultiplier: planet.government.industry_labor_multiplier ?? 1,
      scienceLaborMultiplier: planet.government.science_labor_multiplier ?? 1,
      taxMultiplier: planet.government.tax_multiplier ?? 1,
      moraleNeutral: Boolean(planet.government.morale_neutral),
    };
  }
  return mapGovernmentOutputModifiers(government);
}

function mapGovernmentOutputModifiers(government: TechViewModel | null) {
  return {
    name: government?.name ?? "",
    foodLaborMultiplier: governmentModifierValue(government, "food_labor_multiplier", 1),
    industryLaborMultiplier: governmentModifierValue(government, "industry_labor_multiplier", 1),
    scienceLaborMultiplier: governmentModifierValue(government, "science_labor_multiplier", 1),
    taxMultiplier: governmentModifierValue(government, "tax_multiplier", 1),
    moraleNeutral: Boolean(government?.grants.modifiers.some((modifier) => modifier.name === "morale_neutral")),
  };
}

function governmentModifierValue(government: TechViewModel | null, modifierName: string, fallback: number) {
  return government?.grants.modifiers.find((modifier) => modifier.name === modifierName)?.value ?? fallback;
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

function mapPlanetProject(
	project: ApiBuilding | string | undefined,
	buildingByName: Map<string, BuildingViewModel>,
	shipDesignProjectByName: Map<string, BuildingViewModel>,
): BuildingViewModel | null {
	if (!project) {
		return null;
	}
	if (typeof project === "string") {
		return (
			VIRTUAL_PROJECTS.get(project.toLowerCase()) ??
			buildingByName.get(project.toLowerCase()) ??
			shipDesignProjectByName.get(project.toLowerCase()) ??
			null
		);
	}
	if (!project.name) {
		return null;
	}
	return mapBuilding(project);
}

function emptyGrants() {
  return {
    unlocks: {
      buildings: [],
      hullSizes: [],
      drives: [],
      armors: [],
      computers: [],
      fuelCells: [],
      shields: [],
      beams: [],
      shipSystems: [],
    },
    modifiers: [],
  };
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

  const moraleMultiplier = planet.morale?.multiplier ?? 1;
  const taxes = Math.floor(Math.floor(planet.population / 1000) * moraleMultiplier);
  const food =
    Math.floor(planet.food.farmers * planet.food.produced_per_farmer * moraleMultiplier) +
    planet.food.produced_per_building;
  const foodSurplus = Math.max(0, food - taxes);
  const grossIndustry =
    Math.floor(planet.industry.workers * planet.industry.produced_per_worker * moraleMultiplier) +
    planet.industry.produced_per_building +
    Math.floor(planet.population / 1000) *
      (planet.industry.produced_per_colonist ?? 0);
  const industry = Math.max(
    0,
    grossIndustry - (planet.pollution ?? mapPlanetPollution(planet, planet.industry.workers)),
  );
  const currentProjectName =
    typeof planet.current_project === "string"
      ? planet.current_project
      : planet.current_project?.name;
  const tradeGoods = planet.is_building || currentProjectName !== "Trade Goods"
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

function mapShipDesignProject(design: ShipDesignViewModel): BuildingViewModel {
	return {
		id: shipDesignProjectId(design.name),
		name: design.name,
		description: `${design.hullSize.name} ship design`,
		icon: "🚢",
		cost: design.hullSize.name === "Frigate" ? 100 : 0,
		maintenance: 0,
		unique: false,
		upgrades: [],
		grants: {
			unlocks: {
				buildings: [],
				hullSizes: [],
				drives: [],
				armors: [],
				computers: [],
				fuelCells: [],
				shields: [],
				beams: [],
				shipSystems: [],
			},
			modifiers: [],
		},
		modifiers: [],
	};
}

function shipDesignProjectId(name: string) {
	let hash = 0;
	for (let index = 0; index < name.length; index += 1) {
		hash += name.charCodeAt(index) * (index + 1);
	}
	return -10000 - hash;
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
    upgrades: building.upgrades ?? [],
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
