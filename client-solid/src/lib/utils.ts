import type {
  PendingCommands,
  MovePopulationCommand,
  BuildCommand,
} from "./types/commands";
import type {
  ColonyRow,
  FleetViewModel,
  GameViewModel,
  PlanetViewModel,
  ShipDesignForm,
  StarSummary,
} from "./types/domain";

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
};

export function cssClassForEnvironment(value: number) {
  return environmentCssClassMap[value] ?? "barren";
}

export function getPrimaryPlayer(game?: GameViewModel | null) {
  return game?.players[0] ?? null;
}

export function findFleetInOrbit(game: GameViewModel | null | undefined, playerId: number, starId: number): FleetViewModel | null {
  if (!game) {
    return null;
  }
  return game.fleets.find((fleet) => fleet.ownerId === playerId && fleet.starId === starId && fleet.turnsRemaining === 0) ?? null;
}

export function calculateFleetTravelTurns(from: StarSummary, to: StarSummary) {
  return calculateFleetTravelTurnsFromCoordinates(from.x, from.y, to.x, to.y);
}

export function calculateFleetTravelTurnsFromCoordinates(fromX: number, fromY: number, toX: number, toY: number) {
  const dx = toX - fromX;
  const dy = toY - fromY;
  const distance = Math.sqrt(dx * dx + dy * dy);
  return Math.max(1, Math.ceil(distance / 90));
}

export function getPlanetIndustry(planet: PlanetViewModel) {
  return getPlanetIndustryForWorkers(planet, planet.industry.workers);
}

export function getPlanetGrossIndustryForWorkers(planet: PlanetViewModel, workers: number) {
  return (
    Math.max(0, workers) * planet.industry.produced_per_worker +
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

export function getPlanetPollutionForWorkers(planet: PlanetViewModel, workers: number) {
  const dirtyProduction = Math.max(0, workers) * planet.industry.produced_per_worker;
  const raw = (dirtyProduction / getPlanetPollutionDivisor(planet) - getPlanetPollutionSize(planet)) / 2;
  return Math.max(0, Math.ceil(raw));
}

export function getPlanetIndustryForWorkers(planet: PlanetViewModel, workers: number) {
  const grossIndustry = getPlanetGrossIndustryForWorkers(planet, workers);
  const pollution = getPlanetPollutionForWorkers(planet, workers);
  return Math.max(0, grossIndustry - pollution);
}

export function getPlanetProjectedIncome(planet: PlanetViewModel, pendingCommands?: PendingCommands) {
  const population = getProjectedWorkerAllocation(planet, pendingCommands?.movePopulation ?? []);
  const pendingBuild = getPendingBuild(planet.id, pendingCommands?.build ?? []);
  const food =
    population.farmers * planet.food.produced_per_farmer +
    planet.food.produced_per_building;
  const foodSurplus = Math.max(0, food - Math.floor(planet.population / 1000));
  const industry = getPlanetIndustryForWorkers(planet, population.workers);
  const producesTradeGoods =
    pendingBuild?.building === "Trade Goods" ||
    (!planet.isBuilding && !pendingBuild);
  const tradeGoods = producesTradeGoods
    ? Math.floor(foodSurplus / 2) + Math.floor(industry / 2)
    : 0;
  const gross = planet.income.taxes + planet.income.buildings + tradeGoods;
  const total = Math.trunc(gross * (1 + planet.income.bonus)) - planet.income.maintenance;

  return {
    taxes: planet.income.taxes,
    buildings: planet.income.buildings,
    tradeGoods,
    maintenance: planet.income.maintenance,
    total,
    bonus: planet.income.bonus,
  };
}

export function getPlanetProjectedPopulationGrowth(planet: PlanetViewModel, pendingCommands?: PendingCommands) {
  if (planet.population <= 0) {
    return 0;
  }

  const allocation = getProjectedWorkerAllocation(planet, pendingCommands?.movePopulation ?? []);
  const planetCapacity = Math.floor(planet.maxPopulation / 1000);
  const popUnits = Math.floor(Math.min(planet.population, planet.maxPopulation) / 1000);
  const freeSpace = planetCapacity - popUnits;
  const basicIncrement = planetCapacity > 0 && freeSpace > 0
    ? Math.floor(Math.sqrt((2000 * popUnits * freeSpace) / planetCapacity))
    : 0;
  const foodProduced =
    allocation.farmers * planet.food.produced_per_farmer +
    planet.food.produced_per_building;
  const foodLack = popUnits - foodProduced;
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

  return Math.floor(basicIncrement + flatGrowthBonus - foodLackPenalty);
}

export function getColonyRows(game?: GameViewModel | null, pendingCommands?: PendingCommands): ColonyRow[] {
  if (!game) {
    return [];
  }
  return game.galaxy.stars.flatMap((star) =>
    star.planets
      .filter((planet) => planet.owner === 1)
      .map((planet) => ({
        id: planet.id,
        name: planet.name,
        starId: star.id,
        starName: star.name,
        population: planet.population,
        maxPopulation: planet.maxPopulation,
        farmers: planet.food.farmers,
        workers: planet.industry.workers,
        scientists: planet.science.scientists,
        industry: getPlanetIndustry(planet),
        income: getPlanetProjectedIncome(planet, pendingCommands).total,
        currentProject: planet.currentProject?.name || "Idle",
      })),
  );
}

export function getEmpireIncome(game?: GameViewModel | null, playerId = 1, pendingCommands?: PendingCommands) {
  if (!game) {
    return 0;
  }
  return game.galaxy.stars
    .flatMap((star) => star.planets)
    .filter((planet) => planet.owner === playerId)
    .reduce((total, planet) => total + getPlanetProjectedIncome(planet, pendingCommands).total, 0);
}

export function formatSignedBc(value: number) {
  return `${value > 0 ? "+" : ""}${value} BC`;
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
    drive: parts?.drives[0]?.name ?? "",
    computer: parts?.computers[0]?.name ?? "",
    shield: parts?.shields[0]?.name ?? "",
    armor: parts?.armors[0]?.name ?? "",
    hullSize: parts?.hullSizes[0]?.name ?? "",
    fuelCell: parts?.fuelCells[0]?.name ?? "",
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
