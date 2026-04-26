import type {
  ColonyRow,
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

export function toWebSocketUrl(serverUrl: string) {
  return `${serverUrl.replace(/^http/, "ws")}/gameupdates`;
}

export function gameIdShort(gameId: string) {
  return gameId.slice(0, 8);
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

export function getPrimaryPlayer(game?: GameViewModel | null) {
  return game?.players[0] ?? null;
}

export function getPlanetIndustry(planet: PlanetViewModel) {
  return planet.workers * industryPerMineralClass(planet.mineralClass);
}

function industryPerMineralClass(value: number) {
  switch (value) {
    case 1:
      return 1;
    case 2:
      return 2;
    case 3:
      return 3;
    case 4:
      return 5;
    case 5:
      return 8;
    default:
      return 0;
  }
}

export function getColonyRows(game?: GameViewModel | null): ColonyRow[] {
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
        farmers: planet.farmers,
        workers: planet.workers,
        scientists: planet.scientists,
        industry: getPlanetIndustry(planet),
        currentProject: planet.currentProject?.name || "Idle",
      })),
  );
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
  if (star.ownedPlanets === 0) {
    return "Unclaimed system";
  }
  return `${star.ownedPlanets} controlled world${star.ownedPlanets === 1 ? "" : "s"}`;
}
