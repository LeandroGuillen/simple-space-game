import {
  formatGameLastTurnAt,
  getEmpireFoodBalance,
  getColonyRows,
  getPlanetGrossIndustryForWorkers,
  getPlanetProjectedIncome,
  getPlanetProjectedPopulationGrowth,
} from "../utils";
import type { BuildingViewModel, GameViewModel, PlanetViewModel } from "../types/domain";

function building(name: string, modifiers: BuildingViewModel["modifiers"] = []): BuildingViewModel {
  return {
    id: 1,
    name,
    description: "",
    icon: "🏗️",
    cost: 0,
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
      modifiers: modifiers.map((modifier) => ({
        name: modifier.name,
        value: modifier.bonus,
      })),
    },
    modifiers,
  };
}

function planet(overrides: Partial<PlanetViewModel> = {}): PlanetViewModel {
  return {
    id: 1,
    name: "Test",
    owner: 1,
    parentStar: 1,
    mineralClass: 3,
    size: 5,
    gravity: 2,
    environment: 11,
    population: 10000,
    maxPopulation: 20000,
    last_turn_pop_growth: 0,
    marines: 0,
    armor: 0,
    food: {
      farmers: 10,
      produced_per_farmer: 2,
      produced_per_building: 0,
    },
    industry: {
      workers: 0,
      produced_per_worker: 3,
      produced_per_building: 0,
      produced_per_colonist: 0,
    },
    science: {
      scientists: 0,
      produced_per_scientist: 3,
      produced_per_building: 0,
    },
    income: {
      taxes: 10,
      buildings: 0,
      tradeGoods: 0,
      maintenance: 0,
      total: 10,
      bonus: 0,
    },
    morale: {
      bonus: 0,
      multiplier: 1,
      sources: [],
    },
    pollution: 0,
    currentProject: null,
    buildQueue: [],
    buildings: [],
    industryStack: 0,
    buildBucket: 0,
    isBuilding: false,
    autobuild: false,
    ...overrides,
  };
}

describe("formatGameLastTurnAt", () => {
  it("returns em dash for missing or invalid input", () => {
    expect(formatGameLastTurnAt(undefined)).toBe("—");
    expect(formatGameLastTurnAt(null)).toBe("—");
    expect(formatGameLastTurnAt("")).toBe("—");
    expect(formatGameLastTurnAt("not-a-date")).toBe("—");
  });

  it("formats valid ISO timestamps", () => {
    const out = formatGameLastTurnAt("2026-05-08T14:30:00.000Z");
    expect(out).not.toBe("—");
    expect(out.length).toBeGreaterThan(4);
  });
});

describe("getPlanetProjectedPopulationGrowth", () => {
  it("includes flat population growth from built buildings", () => {
    const colony = planet({
      buildings: [
        building("Cloning Center", [
          { name: "population_growth", description: "", bonus: 100 },
        ]),
      ],
    });

    expect(getPlanetProjectedPopulationGrowth(colony)).toBe(200);
  });

  it("uses morale-boosted farmer food for starvation checks", () => {
    const colony = planet({
      population: 10000,
      maxPopulation: 10000,
      food: {
        farmers: 4,
        produced_per_farmer: 2,
        produced_per_building: 0,
      },
      morale: {
        bonus: 0.25,
        multiplier: 1.25,
        sources: [],
      },
    });

    expect(getPlanetProjectedPopulationGrowth(colony)).toBe(0);
  });
});

describe("morale projections", () => {
  it("boosts taxes and worker industry without boosting colonist industry", () => {
    const colony = planet({
      population: 10000,
      morale: {
        bonus: 0.25,
        multiplier: 1.25,
        sources: [],
      },
      industry: {
        workers: 3,
        produced_per_worker: 3,
        produced_per_building: 5,
        produced_per_colonist: 1,
      },
      income: {
        taxes: 10,
        buildings: 2,
        tradeGoods: 0,
        maintenance: 1,
        total: 11,
        bonus: 0,
      },
      isBuilding: true,
    });

    expect(getPlanetGrossIndustryForWorkers(colony, 3)).toBe(26);
    expect(getPlanetProjectedIncome(colony).taxes).toBe(12);
    expect(getPlanetProjectedIncome(colony).total).toBe(13);
  });
});

describe("getColonyRows", () => {
  it("shows pending population moves before the turn is submitted", () => {
    const colony = planet({
      food: {
        farmers: 4,
        produced_per_farmer: 2,
        produced_per_building: 0,
      },
      industry: {
        workers: 3,
        produced_per_worker: 3,
        produced_per_building: 0,
        produced_per_colonist: 0,
      },
      science: {
        scientists: 3,
        produced_per_scientist: 3,
        produced_per_building: 0,
      },
    });
    const game: GameViewModel = {
      id: "test",
      currentTurn: 1,
      players: [],
      fleets: [],
      pendingColonyBases: {},
      pendingColonyShips: {},
      resources: {
        shipParts: {
          hullSizes: [],
          drives: [],
          armors: [],
          computers: [],
          fuelCells: [],
          shields: [],
        },
        shipSystemCollection: { shipSystems: [] },
        buildingCollection: { buildings: [] },
        modifierCollection: { modifiers: [] },
        techCollection: { techs: [] },
        shipDesignCollection: { shipDesigns: [] },
        techTree: { subjects: [] },
      },
      galaxy: {
        width: 700,
        height: 400,
        stars: [
          {
            id: 1,
            name: "Sol",
            color: 0,
            x: 0,
            y: 0,
            explored: true,
            ownerColor: null,
            ownedPlanets: 1,
            planets: [colony],
          },
        ],
      },
    };

    const [row] = getColonyRows(
      game,
      {
        build: [],
        fleetMove: [],
        colonize: [],
        selectResearch: undefined,
        movePopulation: [
          { planetId: colony.id, from: "workers", to: "scientists", count: 2 },
        ],
      },
      1,
    );

    expect(row?.farmers).toBe(4);
    expect(row?.workers).toBe(1);
    expect(row?.scientists).toBe(5);
    expect(row?.industry).toBe(3);
  });
});

describe("getEmpireFoodBalance", () => {
  it("sums projected colony food production minus population consumption", () => {
    const colony = planet({
      population: 6000,
      food: {
        farmers: 2,
        produced_per_farmer: 3,
        produced_per_building: 1,
      },
      industry: {
        workers: 2,
        produced_per_worker: 3,
        produced_per_building: 0,
        produced_per_colonist: 0,
      },
    });
    const game: GameViewModel = {
      id: "test",
      currentTurn: 1,
      players: [],
      fleets: [],
      pendingColonyBases: {},
      pendingColonyShips: {},
      resources: {
        shipParts: {
          hullSizes: [],
          drives: [],
          armors: [],
          computers: [],
          fuelCells: [],
          shields: [],
        },
        shipSystemCollection: { shipSystems: [] },
        buildingCollection: { buildings: [] },
        modifierCollection: { modifiers: [] },
        techCollection: { techs: [] },
        shipDesignCollection: { shipDesigns: [] },
        techTree: { subjects: [] },
      },
      galaxy: {
        width: 700,
        height: 400,
        stars: [
          {
            id: 1,
            name: "Sol",
            color: 0,
            x: 0,
            y: 0,
            explored: true,
            ownerColor: null,
            ownedPlanets: 1,
            planets: [colony],
          },
        ],
      },
    };

    expect(getEmpireFoodBalance(game, 1)).toBe(1);
    expect(
      getEmpireFoodBalance(game, 1, {
        build: [],
        fleetMove: [],
        colonize: [],
        selectResearch: undefined,
        movePopulation: [
          { planetId: colony.id, from: "workers", to: "farmers", count: 2 },
        ],
      }),
    ).toBe(7);
  });
});
