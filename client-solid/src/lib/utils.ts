import type {
  PendingCommands,
  MovePopulationCommand,
  BuildCommand,
} from "./types/commands";
import type {
  BuildingViewModel,
  ColonyRow,
  FleetShipViewModel,
  FleetViewModel,
  GameViewModel,
  PlanetViewModel,
  PlayerViewModel,
  ResourcesViewModel,
  ShipDesignForm,
  StarSummary,
} from "./types/domain";
import type { ApiShipParts } from "./types/api";
import { racePopulationGrowthPercent } from "./race-design";
import { playerHasRaceTrait } from "./research";
import { tradeGoodsMultiplier } from "./research";

export const STAR_COLORS = [
  "#f1f5f9",
  "#7dd3fc",
  "#fde68a",
  "#fb923c",
  "#f87171",
];

const sizeLabels = ["Invalid", "Tiny", "Small", "Medium", "Large", "Huge"];
const gravityLabels = ["Invalid", "Low", "Normal", "High"];
const mineralLabels = [
  "Invalid",
  "Ultra Poor",
  "Poor",
  "Abundant",
  "Rich",
  "Ultra Rich",
];
const environmentLabels = [
  "Invalid",
  "Gas Giant",
  "Asteroid Belt",
  "Toxic",
  "Radiated",
  "Barren",
  "Desert",
  "Arid",
  "Tundra",
  "Swamp",
  "Ocean",
  "Terran",
  "Gaia",
];

export function normalizeServerUrl(value: string) {
  const trimmed = value.trim().replace(/\/+$/, "");
  if (!trimmed) {
    return "";
  }
  return /^https?:\/\//.test(trimmed) ? trimmed : `http://${trimmed}`;
}

export function isGameCommandPath(pathname: string) {
  return (
    pathname.startsWith("/galaxy") ||
    pathname.startsWith("/stars") ||
    pathname.startsWith("/colonies") ||
    pathname.startsWith("/fleets") ||
    pathname.startsWith("/technology")
  );
}

export function isLobbyPath(pathname: string) {
  return (
    pathname === "/" ||
    pathname.startsWith("/race-editor") ||
    pathname.startsWith("/ship-editor")
  );
}

export function gameSessionPath(
  gameId: string,
  playerId: number,
  route = "/galaxy",
) {
  return `${route}?game=${encodeURIComponent(gameId)}&player=${playerId}`;
}

export function toWebSocketUrl(serverUrl: string, gameId: string, playerId: number) {
  const base = `${serverUrl.replace(/^http/, "ws")}/gameupdates`;
  const params = new URLSearchParams({
    game_id: gameId,
    player_id: String(playerId),
  });
  return `${base}?${params.toString()}`;
}

export function gameIdShort(gameId: string) {
  return gameId.slice(0, 8);
}

export function formatStardate(stardate: number | null | undefined): string {
  return typeof stardate === "number" && Number.isFinite(stardate)
    ? stardate.toFixed(1)
    : "";
}

/** Local short date/time for last completed turn; missing or invalid ISO → em dash */
export function formatGameLastTurnAt(iso: string | null | undefined): string {
  if (iso == null || iso === "") {
    return "—";
  }
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) {
    return "—";
  }
  return d.toLocaleString(undefined, {
    dateStyle: "short",
    timeStyle: "short",
  });
}

export function labelForSize(value: number) {
  return sizeLabels[value] ?? "Unknown";
}

export function labelForGravity(value: number) {
  return gravityLabels[value] ?? "Unknown";
}

export function labelForMinerals(value: number) {
  return mineralLabels[value] ?? "Unknown";
}

export function labelForEnvironment(value: number) {
  return environmentLabels[value] ?? "Unknown";
}

const environmentCssClassMap: Record<number, string> = {
  1: "gasgiant",
  2: "asteroid",
  3: "toxic",
  4: "radiated",
  5: "barren",
  6: "desert",
  7: "arid",
  8: "frozen",
  9: "swamp",
  10: "oceanic",
  11: "terrestrial",
  12: "gaia",
};

export function cssClassForEnvironment(value: number) {
  return environmentCssClassMap[value] ?? "barren";
}

export function getPrimaryPlayer(game?: GameViewModel | null) {
  return game?.players[0] ?? null;
}

export function getPlayerById(game: GameViewModel | null | undefined, playerId: number) {
  return game?.players.find((player) => player.id === playerId) ?? null;
}

export function findFleetInOrbit(game: GameViewModel | null | undefined, playerId: number, starId: number): FleetViewModel | null {
  if (!game) {
    return null;
  }
  return game.fleets.find((fleet) => fleet.ownerId === playerId && fleet.starId === starId && fleet.turnsRemaining === 0) ?? null;
}

/** Fleets owned by the player that are still in orbit at this star. */
export function findPlayerFleetsAtStar(
  game: GameViewModel | null | undefined,
  playerId: number,
  starId: number,
): FleetViewModel[] {
  if (!game) {
    return [];
  }
  return game.fleets.filter((fleet) =>
    fleet.ownerId === playerId && fleet.starId === starId && (fleet.turnsRemaining ?? 0) === 0
  );
}

export function sortFleetsForStarTabs(fleets: FleetViewModel[]): FleetViewModel[] {
  return [...fleets].sort((a, b) => {
    const aOrbit = (a.turnsRemaining ?? 0) === 0;
    const bOrbit = (b.turnsRemaining ?? 0) === 0;
    if (aOrbit !== bOrbit) {
      return aOrbit ? -1 : 1;
    }
    return (a.destinationStarId ?? 0) - (b.destinationStarId ?? 0);
  });
}

export function fleetTabLabel(game: GameViewModel | null | undefined, fleet: FleetViewModel): string {
  if ((fleet.turnsRemaining ?? 0) === 0) {
    const star = findStar(game, fleet.starId);
    return `Orbiting ${star?.name ?? "Unknown system"}`;
  }
  const dest = fleet.destinationStarId != null ? findStar(game, fleet.destinationStarId) : null;
  if (dest && !dest.explored) {
    return "Exploring a new system";
  }
  return `Traveling to ${dest?.name ?? "Unknown system"}`;
}

export function calculateFleetTravelTurns(from: StarSummary, to: StarSummary) {
  return calculateFleetTravelTurnsFromCoordinates(from.x, from.y, to.x, to.y);
}

export function calculateFleetTravelTurnsFromCoordinates(
  fromX: number,
  fromY: number,
  toX: number,
  toY: number,
  speedParsecsPerTurn = 1.5,
) {
  const dx = toX - fromX;
  const dy = toY - fromY;
  const distance = Math.sqrt(dx * dx + dy * dy);
  return Math.max(1, Math.ceil(distance / Math.max(1, speedParsecsPerTurn * 60)));
}

const PARSEC_IN_PIXELS = 60;

export function distanceParsecsFromCoordinates(fromX: number, fromY: number, toX: number, toY: number) {
  const dx = toX - fromX;
  const dy = toY - fromY;
  return Math.sqrt(dx * dx + dy * dy) / PARSEC_IN_PIXELS;
}

export function formatFuelRange(range: { rangeParsecs: number; unlimited: boolean } | null | undefined) {
  if (!range) {
    return "Unknown";
  }
  if (range.unlimited) {
    return "Unlimited";
  }
  return `${range.rangeParsecs} parsecs`;
}

function isUnlimitedFuel(name: string, autonomy: number) {
  return name.toLowerCase() === "thorium" || autonomy >= 100;
}

function bestAvailableFuelRange(player: PlayerViewModel | null | undefined, resources: ResourcesViewModel | null | undefined) {
  if (!player || !resources) {
    return null;
  }
  let best: { rangeParsecs: number; unlimited: boolean } | null = null;
  for (const fuelName of player.available.fuelCells) {
    const fuel = resources.shipParts.fuelCells.find((item) => item.name === fuelName);
    if (!fuel) {
      continue;
    }
    const candidate = {
      rangeParsecs: fuel.autonomy,
      unlimited: isUnlimitedFuel(fuel.name, fuel.autonomy),
    };
    if (!best || candidate.unlimited || (!best.unlimited && candidate.rangeParsecs > best.rangeParsecs)) {
      best = candidate;
    }
  }
  return best;
}

function shipFuelRange(
  ship: FleetShipViewModel,
  player?: PlayerViewModel | null,
  resources?: ResourcesViewModel | null,
) {
  if (ship.unlimitedFuelRange) {
    return { rangeParsecs: 0, unlimited: true };
  }
  if (ship.fuelCell && resources) {
    const fuel = resources.shipParts.fuelCells.find((item) => item.name === ship.fuelCell);
    if (fuel) {
      return { rangeParsecs: fuel.autonomy, unlimited: isUnlimitedFuel(fuel.name, fuel.autonomy) };
    }
  }
  if (ship.fuelRangeParsecs > 0) {
    return { rangeParsecs: ship.fuelRangeParsecs, unlimited: ship.fuelRangeParsecs >= 100 };
  }
  if (ship.designName) {
    const design = [...(player?.shipDesigns ?? []), ...(resources?.shipDesignCollection.shipDesigns ?? [])].find((item) => item.name === ship.designName);
    if (!design) return null;
    return { rangeParsecs: design.fuelCell.autonomy, unlimited: isUnlimitedFuel(design.fuelCell.name, design.fuelCell.autonomy) };
  }
  return bestAvailableFuelRange(player, resources);
}

function bestAvailableDriveSpeed(player: PlayerViewModel | null | undefined, resources: ResourcesViewModel | null | undefined) {
  if (!player || !resources) {
    return null;
  }
  let best: number | null = null;
  for (const driveName of player.available.drives) {
    const drive = resources.shipParts.drives.find((item) => item.name === driveName);
    if (!drive) {
      continue;
    }
    best = best == null ? drive.interstellarSpeed : Math.max(best, drive.interstellarSpeed);
  }
  return best;
}

function shipInterstellarSpeed(
  ship: FleetShipViewModel,
  player?: PlayerViewModel | null,
  resources?: ResourcesViewModel | null,
) {
  if (ship.drive && resources) {
    const drive = resources.shipParts.drives.find((item) => item.name === ship.drive);
    if (drive) {
      return drive.interstellarSpeed;
    }
  }
  if (ship.interstellarSpeed > 0) {
    return ship.interstellarSpeed;
  }
  if (ship.designName) {
    const design = [
      ...(player?.shipDesigns ?? []),
      ...(resources?.shipDesignCollection.shipDesigns ?? []),
    ].find((item) => item.name === ship.designName);
    if (!design) {
      return null;
    }
    const currentDrive = resources?.shipParts.drives.find((item) => item.name === design.drive.name);
    return currentDrive?.interstellarSpeed ?? design.drive.interstellarSpeed;
  }
  return bestAvailableDriveSpeed(player, resources);
}

export function fleetSelectionInterstellarSpeed(
  ships: FleetShipViewModel[],
  player?: PlayerViewModel | null,
  resources?: ResourcesViewModel | null,
) {
  if (ships.length === 0) {
    return null;
  }
  let slowest: number | null = null;
  for (const ship of ships) {
    const speed = shipInterstellarSpeed(ship, player, resources);
    if (speed == null || speed <= 0) {
      return null;
    }
    slowest = slowest == null ? speed : Math.min(slowest, speed);
  }
  return slowest;
}

export function fleetSelectionFuelRange(
  ships: FleetShipViewModel[],
  player?: PlayerViewModel | null,
  resources?: ResourcesViewModel | null,
) {
  if (ships.length === 0) {
    return null;
  }
  let allUnlimited = true;
  let minRange: number | null = null;
  for (const ship of ships) {
    const range = shipFuelRange(ship, player, resources);
    if (!range) {
      return null;
    }
    if (range.unlimited) {
      continue;
    }
    allUnlimited = false;
    minRange = minRange == null ? range.rangeParsecs : Math.min(minRange, range.rangeParsecs);
  }
  if (allUnlimited) {
    return { rangeParsecs: 0, unlimited: true };
  }
  return { rangeParsecs: minRange ?? 0, unlimited: false };
}

export function destinationInSupplyRange(
  game: GameViewModel | null | undefined,
  playerId: number,
  destinationStarId: number,
  range: { rangeParsecs: number; unlimited: boolean } | null | undefined,
) {
  if (!game || !range) {
    return false;
  }
  if (range.unlimited) {
    return true;
  }
  const destination = findStar(game, destinationStarId);
  if (!destination) {
    return false;
  }
  return game.galaxy.stars.some((star) => {
    if (!star.planets.some((planet) => planet.owner === playerId)) {
      return false;
    }
    return distanceParsecsFromCoordinates(star.x, star.y, destination.x, destination.y) <= range.rangeParsecs;
  });
}

// Check if a star has at least one planet owned by the player
export function starHasPlayerColony(
  game: GameViewModel | null | undefined,
  starId: number,
  playerId: number,
): boolean {
  if (!game) return false;
  const star = findStar(game, starId);
  if (!star) return false;
  return star.planets.some((planet) => planet.owner === playerId);
}

// Check if player has researched Star Gate technology
export function hasStarGate(player: PlayerViewModel | null | undefined): boolean {
  if (!player) return false;
  return player.researchedTechs.includes("Star Gate");
}

// Check if Star Gate travel applies (both stars are player-owned colonies with Star Gate tech)
export function isStarGateTravel(
  game: GameViewModel | null | undefined,
  player: PlayerViewModel | null | undefined,
  originStarId: number,
  destStarId: number,
): boolean {
  if (!game || !player) return false;
  if (!hasStarGate(player)) return false;
  const originOwned = starHasPlayerColony(game, originStarId, player.id);
  const destOwned = starHasPlayerColony(game, destStarId, player.id);
  return originOwned && destOwned;
}

// Calculate fleet travel turns, accounting for Star Gate (1 turn between owned colonies)
export function calculateFleetTravelTurnsWithStarGate(
  game: GameViewModel | null | undefined,
  player: PlayerViewModel | null | undefined,
  fromX: number,
  fromY: number,
  toX: number,
  toY: number,
  originStarId: number,
  destStarId: number,
  speedParsecsPerTurn = 1.5,
): number {
  // Star Gate travel: 1 turn between player-owned colonies
  if (isStarGateTravel(game, player, originStarId, destStarId)) {
    return 1;
  }
  // Normal travel calculation
  return calculateFleetTravelTurnsFromCoordinates(fromX, fromY, toX, toY, speedParsecsPerTurn);
}

export function getPlanetIndustry(planet: PlanetViewModel) {
  return getPlanetIndustryForWorkers(planet, planet.industry.workers);
}

export function getMoraleMultiplier(planet: PlanetViewModel) {
  if (planet.government?.moraleNeutral) {
    return 1;
  }
  return planet.morale.multiplier ?? 1;
}

export function getMoraleAdjustedLaborOutput(
  count: number,
  outputPerWorker: number,
  planet: PlanetViewModel,
) {
  return Math.floor(
    Math.max(0, count) * outputPerWorker * getMoraleMultiplier(planet),
  );
}

export function getGravityMultiplier(planet: PlanetViewModel) {
  const value = planet.gravityMultiplier;
  if (value == null || value <= 0) return 1;
  return value;
}

export function getGovernmentAdjustedLaborOutput(
  count: number,
  outputPerWorker: number,
  planet: PlanetViewModel,
  governmentMultiplier: number,
) {
  return Math.floor(
    Math.max(0, count) *
      outputPerWorker *
      Math.max(0, governmentMultiplier || 1) *
      getMoraleMultiplier(planet) *
      getGravityMultiplier(planet),
  );
}

export function getPlanetFoodForFarmers(
  planet: PlanetViewModel,
  farmers: number,
) {
  return (
    getGovernmentAdjustedLaborOutput(
      farmers,
      planet.food.produced_per_farmer,
      planet,
      planet.government?.foodLaborMultiplier ?? 1,
    ) +
    planet.food.produced_per_building
  );
}

export function getPlanetScienceForScientists(
  planet: PlanetViewModel,
  scientists: number,
) {
  return (
    getGovernmentAdjustedLaborOutput(
      scientists,
      planet.science.produced_per_scientist,
      planet,
      planet.government?.scienceLaborMultiplier ?? 1,
    ) +
    planet.science.produced_per_building
  );
}

export function getPlanetGrossIndustryForWorkers(planet: PlanetViewModel, workers: number) {
  return (
    getGovernmentAdjustedLaborOutput(
      workers,
      planet.industry.produced_per_worker,
      planet,
      planet.government?.industryLaborMultiplier ?? 1,
    ) +
    planet.industry.produced_per_building +
    Math.floor(planet.population / 1000) * planet.industry.produced_per_colonist
  );
}

export function getPlanetPollutionDivisor(planet: PlanetViewModel) {
  const hasProcessor = planet.buildings.some((building) => building.name.toLowerCase() === "pollution processor");
  const hasRenewer = planet.buildings.some((building) => building.name.toLowerCase() === "atmospheric renewer");
  if (hasProcessor && hasRenewer) return 8;
  if (hasRenewer) return 4;
  if (hasProcessor) return 2;
  return 1;
}

export function getPlanetPollutionSize(planet: PlanetViewModel) {
  const hasNanoDisassemblers = planet.buildings.some((building) => building.name.toLowerCase() === "nano disassemblers");
  return planet.size * 2 * (hasNanoDisassemblers ? 2 : 1);
}

export function getPlanetPollutionForWorkers(
  planet: PlanetViewModel,
  workers: number,
  traitIds?: readonly string[],
) {
  if ((traitIds ?? []).includes("tolerant")) {
    return 0;
  }
  const dirtyProduction = Math.max(0, workers) * planet.industry.produced_per_worker;
  const raw = (dirtyProduction / getPlanetPollutionDivisor(planet) - getPlanetPollutionSize(planet)) / 2;
  return Math.max(0, Math.ceil(raw));
}

/** Food units required per turn for population (MoO2 cybernetic / lithovore rules). */
export function populationFoodDemand(traitIds: string[] | undefined, population: number): number {
  const popUnits = Math.floor(population / 1000);
  if (playerHasRaceTrait(traitIds, "lithovore")) {
    return 0;
  }
  if (playerHasRaceTrait(traitIds, "cybernetic")) {
    return popUnits * 0.5;
  }
  return popUnits;
}

export function getPlanetIndustryForWorkers(
  planet: PlanetViewModel,
  workers: number,
  traitIds?: readonly string[],
) {
  const grossIndustry = getPlanetGrossIndustryForWorkers(planet, workers);
  const pollution = getPlanetPollutionForWorkers(planet, workers, traitIds);
  return Math.max(0, grossIndustry - pollution);
}

export type PlanetBuildIndustry = {
  /** Industry after pollution, before Food Replicators and empire tax. */
  afterPollution: number;
  /** Industry consumed by Food Replicators (2 production = 1 food). */
  replicators: number;
  /** Food produced by Food Replicators. */
  replicatorFood: number;
  /** Industry after pollution and Food Replicators (the taxable amount). */
  net: number;
  /** Industry converted 1:1 to BC by the empire tax. */
  tax: number;
  /** Industry that actually goes into construction this turn. */
  build: number;
};

/**
 * Mirrors server planetTurn: pollution, then Food Replicators, then the empire
 * tax as floor(net * rate) per colony. When the rate is unknown, the
 * server-hydrated industry tax is used instead.
 */
export function getPlanetBuildIndustryForAllocation(
  planet: PlanetViewModel,
  allocation: { farmers: number; workers: number },
  traitIds?: string[],
  taxRate?: number | null,
): PlanetBuildIndustry {
  const afterPollution = getPlanetIndustryForWorkers(planet, allocation.workers, traitIds);
  const foodDemand = Math.ceil(populationFoodDemand(traitIds, planet.population));
  const replicatorFood = getFoodReplicatorFood(
    planet,
    foodDemand - getPlanetFoodForFarmers(planet, allocation.farmers),
    afterPollution,
  );
  const replicators = replicatorFood * 2;
  const net = Math.max(0, afterPollution - replicators);
  const tax = taxRate == null
    ? Math.min(net, planet.income.industryTax ?? 0)
    : Math.floor(net * taxRate);
  return { afterPollution, replicators, replicatorFood, net, tax, build: net - tax };
}

/**
 * Build industry for a colony with this turn's pending orders: worker moves and
 * a pending Tax change (falling back to the player's current tax rate).
 */
export function getPlanetBuildIndustry(
  planet: PlanetViewModel,
  pendingCommands?: PendingCommands | null,
  player?: Pick<PlayerViewModel, "raceTraitIds" | "taxRate"> | null,
): PlanetBuildIndustry {
  return getPlanetBuildIndustryForAllocation(
    planet,
    getProjectedWorkerAllocation(planet, pendingCommands?.movePopulation ?? []),
    player?.raceTraitIds,
    pendingCommands?.setTaxRate?.rate ?? player?.taxRate,
  );
}

/** Turns to finish `remaining` production at `industryPerTurn`; null if it never finishes. */
export function getBuildTurnsRemaining(remaining: number, industryPerTurn: number) {
  if (industryPerTurn <= 0) {
    return null;
  }
  return Math.max(1, Math.ceil(remaining / industryPerTurn));
}

export function getPlanetProjectedIncome(
  planet: PlanetViewModel,
  pendingCommands?: PendingCommands,
  traitIds?: string[],
  taxRate?: number,
) {
  // Hydrated server income is authoritative until an economic order changes it.
  // Older snapshots without industryTax still use the local projection below.
  if (planet.income.industryTax !== undefined &&
      !pendingCommands?.setTaxRate &&
      !pendingCommands?.movePopulation.some(command => command.planetId === planet.id) &&
      !pendingCommands?.build.some(command => command.planetId === planet.id)) {
    return { ...planet.income, industryTax: planet.income.industryTax };
  }
  const population = getProjectedWorkerAllocation(planet, pendingCommands?.movePopulation ?? []);
  const pendingBuild = getPendingBuild(planet.id, pendingCommands?.build ?? []);
  const food = getPlanetFoodForFarmers(planet, population.farmers);
  const foodSurplus = Math.max(
    0,
    food - Math.ceil(populationFoodDemand(traitIds, planet.population)),
  );
  // Same 1:1, per-colony floor as the server; tax never gets income bonuses.
  const { tax: industryTax, build: industry } = getPlanetBuildIndustryForAllocation(
    planet, population, traitIds, pendingCommands?.setTaxRate?.rate ?? taxRate,
  );
  const producesTradeGoods =
    pendingBuild?.building === "Trade Goods" ||
    (!planet.isBuilding && !pendingBuild && planet.currentProject?.name === "Trade Goods");
  const baseTradeGoods = producesTradeGoods
    ? Math.floor(foodSurplus / 2) + Math.floor(industry / 2)
    : 0;
  const tradeGoods = Math.trunc(baseTradeGoods * tradeGoodsMultiplier(traitIds));
  const taxes = Math.floor(
    Math.floor(planet.population / 1000) *
      getMoraleMultiplier(planet) *
      (planet.government?.taxMultiplier ?? 1),
  );
  const gross = taxes + planet.income.buildings + tradeGoods;
  const total = Math.trunc(gross * (1 + planet.income.bonus)) - planet.income.maintenance + industryTax;

  return {
    taxes,
    industryTax,
    buildings: planet.income.buildings,
    tradeGoods,
    maintenance: planet.income.maintenance,
    total,
    bonus: planet.income.bonus,
  };
}

export type PopulationGrowthContext = {
  /** Food delivered by automatic freighter transfers this turn. */
  foodReceived?: number;
  taxRate?: number;
  /** Empire-wide growth percent from researched medicine techs. */
  medicineGrowthPercent?: number;
};

/**
 * Mirrors the server's planet turn growth (server/game/turn.go): Food Replicators
 * cover deficits before freighter deliveries, and the result is clamped to the
 * planet's minimum (1,000) and maximum population.
 */
export function getPlanetProjectedPopulationGrowth(
  planet: PlanetViewModel,
  pendingCommands?: PendingCommands,
  traitIds?: string[],
  raceGrowthPercent?: number | null,
  context: PopulationGrowthContext = {},
) {
  if (planet.population <= 0) {
    return 0;
  }

  const allocation = getProjectedWorkerAllocation(planet, pendingCommands?.movePopulation ?? []);
  const pendingBuild = getPendingBuild(planet.id, pendingCommands?.build ?? []);
  const planetCapacity = Math.floor(planet.maxPopulation / 1000);
  const popUnits = Math.floor(Math.min(planet.population, planet.maxPopulation) / 1000);
  const freeSpace = planetCapacity - popUnits;
  const basicIncrement = planetCapacity > 0 && freeSpace > 0
    ? Math.floor(Math.sqrt((2000 * popUnits * freeSpace) / planetCapacity))
    : 0;
  const foodDemand = populationFoodDemand(traitIds, planet.population);
  const baseFood = getPlanetFoodForFarmers(planet, allocation.farmers);
  const grossIndustry = getPlanetIndustryForWorkers(planet, allocation.workers, traitIds);
  const replicatorFood = getFoodReplicatorFood(
    planet,
    Math.ceil(foodDemand) - baseFood,
    grossIndustry,
  );
  const foodProduced = baseFood + replicatorFood + (context.foodReceived ?? 0);
  const foodLack = foodDemand - foodProduced;
  const foodLackPenalty = foodLack > 0 ? 50 * foodLack : 0;
  const flatGrowthBonus = planet.buildings.reduce(
    (sum, building) =>
      sum +
      building.modifiers.reduce(
        (modifierSum, modifier) =>
          modifierSum + (modifier.name === "population_growth" ? modifier.bonus : 0),
        0,
      ),
    0,
  );
  const isHousing =
    pendingBuild?.building === "Housing" ||
    (!pendingBuild && planet.currentProject?.name === "Housing");
  const netIndustry = Math.max(0, grossIndustry - replicatorFood * 2);
  const industry = netIndustry - Math.floor(netIndustry * (pendingCommands?.setTaxRate?.rate ?? context.taxRate ?? 0));
  const populationUnits = Math.max(1, Math.floor(planet.population / 1000));
  const housingBonusPercent = isHousing && industry > 0
    ? Math.floor((industry * 40) / populationUnits)
    : 0;
  const raceBonusPercent =
    racePopulationGrowthPercent(traitIds, raceGrowthPercent) +
    (context.medicineGrowthPercent ?? 0);

  const growth = Math.trunc(
    Math.floor((basicIncrement * (100 + housingBonusPercent + raceBonusPercent)) / 100) +
      flatGrowthBonus -
      foodLackPenalty,
  );
  const nextPopulation = Math.min(
    planet.maxPopulation,
    Math.max(1000, planet.population + growth),
  );
  return nextPopulation - planet.population;
}

/** Projected growth for a planet with empire-wide inputs (freighters, medicine techs). */
export function getPlanetPopulationGrowthForecast(
  game: GameViewModel | null | undefined,
  planet: PlanetViewModel,
  pendingCommands?: PendingCommands,
  player?: PlayerViewModel | null,
) {
  return getPlanetProjectedPopulationGrowth(
    planet,
    pendingCommands,
    player?.raceTraitIds,
    player?.racePopulationGrowthPercent,
    {
      foodReceived:
        getProjectedFoodTransfers(game, planet.owner, pendingCommands, player).get(planet.id) ?? 0,
      taxRate: player?.taxRate,
      medicineGrowthPercent: getMedicineGrowthPercent(game, player),
    },
  );
}

/** Food converted from production by Food Replicators (2 production = 1 food). */
function getFoodReplicatorFood(planet: PlanetViewModel, foodDeficit: number, industry: number) {
  if (foodDeficit <= 0 || industry <= 0) {
    return 0;
  }
  if (!planet.buildings.some((building) => building.name === "Food Replicators")) {
    return 0;
  }
  return Math.min(foodDeficit, Math.floor(industry / 2));
}

/** Mirrors server MedicineGrowthPercent: upgraded techs do not stack with their successor. */
export function getMedicineGrowthPercent(
  game?: GameViewModel | null,
  player?: PlayerViewModel | null,
) {
  if (!game || !player) {
    return 0;
  }
  const techByName = new Map(game.resources.techCollection.techs.map((tech) => [tech.name, tech]));
  const researched = player.researchedTechs
    .map((name) => techByName.get(name))
    .filter((tech): tech is NonNullable<typeof tech> => Boolean(tech));
  const upgraded = new Set(researched.flatMap((tech) => tech.upgrades));
  return researched
    .filter((tech) => !upgraded.has(tech.name))
    .reduce(
      (sum, tech) =>
        sum +
        tech.grants.modifiers.reduce(
          (modifierSum, modifier) =>
            modifierSum + (modifier.name === "population_growth_percent" ? modifier.value : 0),
          0,
        ),
      0,
    );
}

/**
 * Mirrors server calculateAutomaticFoodTransfers: idle freighters move surplus food
 * to starving colonies, both visited in planet id order.
 */
export function getProjectedFoodTransfers(
  game?: GameViewModel | null,
  playerId = 1,
  pendingCommands?: PendingCommands,
  player?: PlayerViewModel | null,
) {
  const received = new Map<number, number>();
  if (!game) {
    return received;
  }
  const me = player ?? game.players.find((p) => p.id === playerId) ?? null;
  if (!me) {
    return received;
  }
  let remaining = Math.max(0, me.freighters - me.freightersUsed);
  if (remaining <= 0) {
    return received;
  }
  const surpluses: { id: number; amount: number }[] = [];
  const deficits: { id: number; amount: number }[] = [];
  for (const star of game.galaxy.stars) {
    for (const planet of star.planets) {
      if (planet.owner !== playerId) {
        continue;
      }
      const allocation = getProjectedWorkerAllocation(
        planet,
        pendingCommands?.movePopulation ?? [],
      );
      const produced = getPlanetFoodForFarmers(planet, allocation.farmers);
      const consumed = Math.ceil(populationFoodDemand(me.raceTraitIds, planet.population));
      if (produced > consumed) {
        surpluses.push({ id: planet.id, amount: produced - consumed });
      } else if (consumed > produced) {
        deficits.push({ id: planet.id, amount: consumed - produced });
      }
    }
  }
  surpluses.sort((a, b) => a.id - b.id);
  deficits.sort((a, b) => a.id - b.id);
  for (const deficit of deficits) {
    if (remaining <= 0) {
      break;
    }
    let needed = deficit.amount;
    for (const surplus of surpluses) {
      if (needed <= 0 || remaining <= 0) {
        break;
      }
      if (surplus.amount <= 0) {
        continue;
      }
      const moved = Math.min(needed, surplus.amount, remaining);
      surplus.amount -= moved;
      needed -= moved;
      remaining -= moved;
      received.set(deficit.id, (received.get(deficit.id) ?? 0) + moved);
    }
  }
  return received;
}

export function getColonyRows(
  game?: GameViewModel | null,
  pendingCommands?: PendingCommands,
  playerId = 1,
): ColonyRow[] {
  if (!game) {
    return [];
  }
  return game.galaxy.stars.flatMap((star) =>
    star.planets
      .filter((planet) => planet.owner === playerId)
      .map((planet) => {
        const allocation = getProjectedWorkerAllocation(
          planet,
          pendingCommands?.movePopulation ?? [],
        );
        return {
          id: planet.id,
          name: planet.name,
          starId: star.id,
          starName: star.name,
          population: planet.population,
          maxPopulation: planet.maxPopulation,
          farmers: allocation.farmers,
          workers: allocation.workers,
          scientists: allocation.scientists,
          // Industry that goes into construction; the taxed part is in income.
          industry: getPlanetBuildIndustry(
            planet,
            pendingCommands,
            game.players.find((p) => p.id === playerId),
          ).build,
          income: getPlanetProjectedIncome(
            planet,
            pendingCommands,
            game.players.find((p) => p.id === playerId)?.raceTraitIds,
            game.players.find((p) => p.id === playerId)?.taxRate,
          ).total,
          morale: planet.morale.bonus,
          currentProject: planet.currentProject?.name || "Idle",
        };
      }),
  );
}

export function getEmpireIncome(
  game?: GameViewModel | null,
  playerId = 1,
  pendingCommands?: PendingCommands,
  player?: PlayerViewModel | null,
) {
  if (!game) {
    return 0;
  }
  const colonyTotal = game.galaxy.stars
    .flatMap((star) => star.planets)
    .filter((planet) => planet.owner === playerId)
    .reduce((total, planet) => {
      const me = player ?? game.players.find((p) => p.id === playerId);
      return total + getPlanetProjectedIncome(planet, pendingCommands, me?.raceTraitIds, me?.taxRate).total;
    }, 0);
  const me = player ?? game.players.find((p) => p.id === playerId);
  const commandPenalty = me?.commandOverdrawBc ?? 0;
  return colonyTotal - commandPenalty;
}

export function getEmpireFoodBalance(
  game?: GameViewModel | null,
  playerId = 1,
  pendingCommands?: PendingCommands,
  traitIds?: string[],
) {
  if (!game) {
    return 0;
  }
  return game.galaxy.stars
    .flatMap((star) => star.planets)
    .filter((planet) => planet.owner === playerId)
    .reduce((total, planet) => {
      const allocation = getProjectedWorkerAllocation(
        planet,
        pendingCommands?.movePopulation ?? [],
      );
      const produced = getPlanetFoodForFarmers(planet, allocation.farmers);
      const consumed = Math.ceil(populationFoodDemand(traitIds, planet.population));
      return total + produced - consumed;
    }, 0);
}

export function getProjectedFoodFreightersUsed(
  game?: GameViewModel | null,
  playerId = 1,
  pendingCommands?: PendingCommands,
  player?: PlayerViewModel | null,
) {
  if (!game) {
    return 0;
  }
  const me = player ?? game.players.find((p) => p.id === playerId) ?? null;
  if (!me) {
    return 0;
  }
  const available = Math.max(0, me.freighters - me.freightersUsed);
  if (available <= 0) {
    return 0;
  }
  let surplus = 0;
  let deficit = 0;
  for (const star of game.galaxy.stars) {
    for (const planet of star.planets) {
      if (planet.owner !== playerId) {
        continue;
      }
      const allocation = getProjectedWorkerAllocation(
        planet,
        pendingCommands?.movePopulation ?? [],
      );
      const produced = getPlanetFoodForFarmers(planet, allocation.farmers);
      const consumed = Math.ceil(populationFoodDemand(me.raceTraitIds, planet.population));
      surplus += Math.max(0, produced - consumed);
      deficit += Math.max(0, consumed - produced);
    }
  }
  return Math.min(available, surplus, deficit);
}

export function formatBcValue(value: number) {
  return new Intl.NumberFormat("en-US", {
    minimumFractionDigits: Number.isInteger(value) ? 0 : 1,
    maximumFractionDigits: 1,
  }).format(value);
}

export function formatFreighterDisplay(
  player: PlayerViewModel | null | undefined,
  projectedFoodFreightersUsed: number,
) {
  const total = player?.freighters ?? 0;
  const available = Math.max(
    0,
    total - (player?.freightersUsed ?? 0) - projectedFoodFreightersUsed,
  );
  const sign = available > 0 ? "+" : "";
  return `${sign}${formatInteger(available)} (${formatInteger(total)})`;
}

export function formatFreighterTooltip(
  player: PlayerViewModel | null | undefined,
  projectedFoodFreightersUsed: number,
) {
  if (!player) {
    return "";
  }
  const total = player.freighters;
  const reserved = player.freightersUsed;
  const available = Math.max(0, total - reserved - projectedFoodFreightersUsed);
  const cost = projectedFoodFreightersUsed * 0.5;
  return [
    `Total freighters: ${formatInteger(total)}`,
    `Reserved for population transfers: ${formatInteger(reserved)}`,
    `Projected food transfers: ${formatInteger(projectedFoodFreightersUsed)}`,
    `Projected food operating cost: ${formatBcValue(cost)} BC`,
    `Available: ${formatInteger(available)}`,
  ].join("\n");
}

export function formatEmpireFoodTooltip(
  game: GameViewModel | null | undefined,
  player: PlayerViewModel | null | undefined,
  pendingCommands?: PendingCommands,
): string {
  if (!game || !player) {
    return "";
  }
  const lines: string[] = [];
  let producedTotal = 0;
  let consumedTotal = 0;
  for (const star of game.galaxy.stars) {
    for (const planet of star.planets) {
      if (planet.owner !== player.id) {
        continue;
      }
      const allocation = getProjectedWorkerAllocation(
        planet,
        pendingCommands?.movePopulation ?? [],
      );
      const produced = getPlanetFoodForFarmers(planet, allocation.farmers);
      const consumed = Math.ceil(populationFoodDemand(player.raceTraitIds, planet.population));
      producedTotal += produced;
      consumedTotal += consumed;
      lines.push(`${planet.name} (${star.name}): ${produced} produced, ${consumed} consumed`);
    }
  }
  if (lines.length === 0) {
    return "No colonies";
  }
  const balance = producedTotal - consumedTotal;
  lines.push(`Empire food: ${producedTotal} produced, ${consumedTotal} consumed`);
  lines.push(`Balance: ${formatSigned(balance)}`);
  return lines.join("\n");
}

/** Multi-line tooltip for empire income (colonies + command overcapacity). */
export function formatEmpireIncomeTooltip(
  game: GameViewModel | null | undefined,
  player: PlayerViewModel | null | undefined,
  pendingCommands?: PendingCommands,
): string {
  if (!game || !player) {
    return "";
  }
  const lines: string[] = [];
  let coloniesTotal = 0;
  for (const star of game.galaxy.stars) {
    for (const planet of star.planets) {
      if (planet.owner !== player.id) {
        continue;
      }
      const inc = getPlanetProjectedIncome(planet, pendingCommands, player.raceTraitIds, player.taxRate).total;
      coloniesTotal += inc;
      lines.push(`${planet.name} (${star.name}): ${formatSignedBc(inc)}`);
    }
  }
  if (lines.length === 0) {
    return "No colonies";
  }
  lines.push(`Colonies total: ${formatSignedBc(coloniesTotal)}`);
  const overBc = player.commandOverdrawBc ?? 0;
  if (overBc > 0) {
    const cap = player.commandPointsCapacity ?? 0;
    const used = player.commandPointsUsed ?? 0;
    const overCp = Math.max(0, used - cap);
    lines.push(`Command overcapacity (${overCp} CP): -${overBc} BC`);
  }
  return lines.join("\n");
}

export function formatCommandPointsDisplay(player: PlayerViewModel | null | undefined): string {
  if (!player) {
    return "+0 (0)";
  }
  const rem = player.commandPointsRemaining ?? 0;
  const cap = player.commandPointsCapacity ?? 0;
  const sign = rem > 0 ? "+" : "";
  return `${sign}${formatInteger(rem)} (${formatInteger(cap)})`;
}

function commandPointsPerBuiltBuilding(building: BuildingViewModel): number {
  return building.modifiers
    .filter((modifier) => modifier.name.toLowerCase() === "command_points")
    .reduce((sum, modifier) => sum + modifier.bonus, 0);
}

/** Lowercase building label with simple English plural for CP tooltip lines. */
function pluralBuildingLabel(canonicalName: string, count: number): string {
  const label = canonicalName.trim().toLowerCase();
  if (count === 1) {
    return label;
  }
  if (label.endsWith(" base")) {
    return `${label.slice(0, -5)} bases`;
  }
  if (label.endsWith("ss")) {
    return `${label}es`;
  }
  if (label.endsWith("s") || label.endsWith("x") || label.endsWith("ch") || label.endsWith("sh")) {
    return `${label}es`;
  }
  return `${label}s`;
}

/** Multi-line tooltip: capacity sources (starting bonus, colonies, buildings with command_points). */
export function formatCommandCapacityTooltip(
  game: GameViewModel | null | undefined,
  player: PlayerViewModel | null | undefined,
): string {
  if (!game || !player) {
    return "";
  }

  const lines: string[] = [];
  const starting = player.commandPointsStartingBonus ?? 0;
  if (starting > 0) {
    lines.push(`${starting} from starting allocation`);
  }

  let colonyCount = 0;
  const buildingBuckets = new Map<
    string,
    { displayName: string; count: number; cpPerBuilding: number }
  >();

  for (const star of game.galaxy.stars) {
    for (const planet of star.planets) {
      if (planet.owner !== player.id) {
        continue;
      }
      colonyCount += 1;
      for (const building of planet.buildings) {
        const cp = commandPointsPerBuiltBuilding(building);
        if (cp <= 0) {
          continue;
        }
        const key = building.name.toLowerCase();
        const existing = buildingBuckets.get(key);
        if (existing) {
          existing.count += 1;
        } else {
          buildingBuckets.set(key, {
            displayName: building.name,
            count: 1,
            cpPerBuilding: cp,
          });
        }
      }
    }
  }

  if (colonyCount > 0) {
    const colonyWord = colonyCount === 1 ? "colony" : "colonies";
    lines.push(`${colonyCount} from ${colonyCount} ${colonyWord}`);
  }

  const buildingLines = [...buildingBuckets.values()]
    .map((bucket) => ({
      bucket,
      totalCp: bucket.count * bucket.cpPerBuilding,
      sortKey: bucket.displayName.toLowerCase(),
    }))
    .sort((a, b) => a.sortKey.localeCompare(b.sortKey));

  for (const { bucket, totalCp } of buildingLines) {
    const label = pluralBuildingLabel(bucket.displayName, bucket.count);
    lines.push(`${totalCp} from ${bucket.count} ${label}`);
  }

  return lines.join("\n");
}

export function formatSignedBc(value: number) {
  return `${value > 0 ? "+" : ""}${formatInteger(value)} BC`;
}

/**
 * Treasury cost (in BC) to immediately finish a build project with `remaining` PP left
 * out of `effectiveCost` total. Mirrors the server formula: 4 BC/PP while invested progress
 * is below 50%, 2 BC/PP once it reaches 50% or more.
 */
export function getBuyProductionCost(effectiveCost: number, remaining: number) {
  if (remaining <= 0 || effectiveCost <= 0) {
    return 0;
  }
  const invested = Math.max(0, effectiveCost - remaining);
  const rate = invested * 2 >= effectiveCost ? 2 : 4;
  return remaining * rate;
}

export function formatSigned(value: number) {
  return `${value > 0 ? "+" : ""}${formatInteger(value)}`;
}

export function formatInteger(value: number) {
  return new Intl.NumberFormat("en-US", {
    maximumFractionDigits: 0,
  }).format(value);
}

export function formatTreasury(value: number) {
  const sign = value < 0 ? "-" : "";
  const absValue = Math.abs(value);

  if (absValue >= 1_000_000) {
    const millions = absValue / 1_000_000;
    const formatted = new Intl.NumberFormat("en-US", {
      maximumFractionDigits: 1,
    }).format(millions);
    return `${sign}${formatted}M`;
  }

  if (absValue >= 100_000) {
    return `${sign}${Math.floor(absValue / 1_000)}K`;
  }

  return `${sign}${formatBcValue(absValue)}`;
}

function getProjectedWorkerAllocation(planet: PlanetViewModel, moves: MovePopulationCommand[]) {
  const totals = {
    farmers: planet.food.farmers,
    workers: planet.industry.workers,
    scientists: planet.science.scientists,
  };

  for (const move of moves.filter((cmd) => cmd.planetId === planet.id)) {
    totals[move.from] -= move.count;
    totals[move.to] += move.count;
  }

  return {
    farmers: Math.max(0, totals.farmers),
    workers: Math.max(0, totals.workers),
    scientists: Math.max(0, totals.scientists),
  };
}

function getPendingBuild(planetId: number, builds: BuildCommand[]) {
  return builds.find((cmd) => cmd.planetId === planetId);
}

export function findStar(
  game: GameViewModel | null | undefined,
  starId: number,
) {
  return game?.galaxy.stars.find((star) => star.id === starId) ?? null;
}

export function findPlanet(
  game: GameViewModel | null | undefined,
  planetId: number,
) {
  return (
    game?.galaxy.stars
      .flatMap((star) => star.planets)
      .find((planet) => planet.id === planetId) ?? null
  );
}

export function defaultShipDesignForm(
  game?: GameViewModel | null,
): ShipDesignForm {
  const parts = game?.resources.shipParts;
  return {
    name: "",
    beam: "",
    drive: parts?.drives.find((d) => d.interstellarSpeed > 0)?.name ?? "",
    computer: parts?.computers[0]?.name ?? "",
    shield: parts?.shields[0]?.name ?? "",
    armor: parts?.armors[0]?.name ?? "",
    hullSize: parts?.hullSizes[0]?.name ?? "",
    fuelCell: parts?.fuelCells.find((f) => f.autonomy > 0)?.name ?? "",
  };
}

export function defaultShipDesignFormFromParts(parts: ApiShipParts | null | undefined): ShipDesignForm {
  return {
    name: "",
    beam: "",
    drive: parts?.drives.find((d) => d.interstellar_speed > 0)?.name ?? "",
    computer: parts?.computers[0]?.name ?? "",
    shield: parts?.shields[0]?.name ?? "",
    armor: parts?.armors[0]?.name ?? "",
    hullSize: parts?.hull_sizes[0]?.name ?? "",
    fuelCell: parts?.fuel_cells.find((f) => f.autonomy > 0)?.name ?? "",
  };
}

export function starOwnershipSummary(star: StarSummary) {
  if (!star.explored) {
    return "Unexplored system";
  }
  if (star.ownedPlanets === 0) {
    return "Unclaimed system";
  }
  return `${star.ownedPlanets} controlled world${star.ownedPlanets === 1 ? "" : "s"}`;
}

export function isPlanetSuitableForColonization(planet: PlanetViewModel): boolean {
  return planet.owner === 0 && planet.environment !== 1 && planet.environment !== 2;
}

export function getSuitablePlanetsForColonization(star: StarSummary): PlanetViewModel[] {
  return star.planets.filter(isPlanetSuitableForColonization);
}

export function isArtificialPlanetTarget(planet: PlanetViewModel): boolean {
  return planet.owner === 0 && (planet.environment === 1 || planet.environment === 2);
}

export function getArtificialPlanetTargets(star: StarSummary): PlanetViewModel[] {
  return star.planets.filter(isArtificialPlanetTarget);
}

export function hasEvolutionaryMutation(player: PlayerViewModel | null | undefined): boolean {
  if (!player) return false;
  return (player.evolutionaryMutationPoints ?? 0) > 0 && !player.hasUsedEvolutionaryMutation;
}

/** Population units (floor(pop/1000)) of free housing after current pop and committed inbound. */
export function destinationTransferRoomLeft(args: {
  destinationPopulation: number;
  destinationMaxPopulation: number;
  inFlightInboundUnits?: number;
  pendingInboundUnits?: number;
}): number {
  const capacityUnits = Math.floor(args.destinationMaxPopulation / 1000);
  const currentUnits = Math.floor(args.destinationPopulation / 1000);
  const committed =
    Math.max(0, args.inFlightInboundUnits ?? 0) + Math.max(0, args.pendingInboundUnits ?? 0);
  return Math.max(0, capacityUnits - currentUnits - committed);
}

/** Sum own in-flight transfer units headed to destinationPlanetId. */
export function inboundInFlightTransferUnits(
  transfers: { owner: number; destinationPlanetId: number; populationUnits: number }[] | undefined,
  playerId: number,
  destinationPlanetId: number,
): number {
  if (!transfers?.length) return 0;
  return transfers
    .filter(
      (transfer) =>
        transfer.owner === playerId && transfer.destinationPlanetId === destinationPlanetId,
    )
    .reduce((total, transfer) => total + transfer.populationUnits, 0);
}

// Match the full-save endpoint: unknown controllers must fail closed.
export function canDownloadGameSave(
  players: readonly { id?: number; controller?: string }[],
  playerId: number,
): boolean {
  return players.some((player) => player.id === playerId) &&
    players.every((player) => player.id === playerId || player.controller === "server_ai");
}
