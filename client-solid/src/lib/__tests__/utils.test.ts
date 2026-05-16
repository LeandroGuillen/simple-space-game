import {
  formatCommandPointsDisplay,
  formatGameLastTurnAt,
  formatFreighterDisplay,
  formatInteger,
  formatSigned,
  formatSignedBc,
  formatStardate,
  formatTreasury,
  getBuyProductionCost,
  getArtificialPlanetTargets,
  getEmpireFoodBalance,
  getProjectedFoodFreightersUsed,
  getColonyRows,
  getPlanetGrossIndustryForWorkers,
  getPlanetProjectedIncome,
  getPlanetProjectedPopulationGrowth,
  isArtificialPlanetTarget,
  fleetSelectionFuelRange,
} from "../utils";
import type {
  BuildingViewModel,
  FleetShipViewModel,
  GameViewModel,
  PlanetViewModel,
  PlayerViewModel,
  ResourcesViewModel,
} from "../types/domain";

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
    gravityMultiplier: 1,
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

describe("formatStardate", () => {
  it("keeps one fractional digit", () => {
    expect(formatStardate(3500)).toBe("3500.0");
    expect(formatStardate(3500.9)).toBe("3500.9");
  });
});

describe("fleetSelectionFuelRange", () => {
  it("prefers current fuel cell data over stale ship range snapshots", () => {
    const player = {
      id: 1,
      available: { fuelCells: ["Standard"] },
    } as PlayerViewModel;
    const resources = {
      shipParts: {
        hullSizes: [],
        drives: [],
        armors: [],
        computers: [],
        fuelCells: [{ name: "Standard", autonomy: 2 }],
        shields: [],
      },
      shipSystemCollection: { shipSystems: [] },
      buildingCollection: { buildings: [] },
      modifierCollection: { modifiers: [] },
      shipDesignCollection: { shipDesigns: [] },
      techTree: { subjects: [] },
      techCollection: { techs: [] },
    } satisfies ResourcesViewModel;
    const ship = {
      type: "ship_design",
      name: "Scout",
      icon: "🚀",
      designName: "Scout",
      hullSize: "Frigate",
      drive: "Nuclear",
      interstellarSpeed: 1,
      fuelCell: "Standard",
      fuelRangeParsecs: 4,
      unlimitedFuelRange: false,
    } as FleetShipViewModel;

    expect(fleetSelectionFuelRange([ship], player, resources)).toEqual({
      rangeParsecs: 2,
      unlimited: false,
    });
  });
});

describe("getBuyProductionCost", () => {
  it("charges 4 BC per remaining PP when invested progress is below 50%", () => {
    expect(getBuyProductionCost(100, 100)).toBe(400);
    expect(getBuyProductionCost(100, 51)).toBe(204);
  });

  it("charges 2 BC per remaining PP at or above 50% invested progress", () => {
    expect(getBuyProductionCost(100, 50)).toBe(100);
    expect(getBuyProductionCost(100, 40)).toBe(80);
    expect(getBuyProductionCost(100, 1)).toBe(2);
  });

  it("returns zero for finished projects and non-positive inputs", () => {
    expect(getBuyProductionCost(100, 0)).toBe(0);
    expect(getBuyProductionCost(100, -10)).toBe(0);
    expect(getBuyProductionCost(0, 50)).toBe(0);
  });
});

describe("artificial planet targets", () => {
  it("matches unowned gas giants and asteroid belts only", () => {
    expect(isArtificialPlanetTarget(planet({ owner: 0, environment: 1 }))).toBe(true);
    expect(isArtificialPlanetTarget(planet({ owner: 0, environment: 2 }))).toBe(true);
    expect(isArtificialPlanetTarget(planet({ owner: 1, environment: 1 }))).toBe(false);
    expect(isArtificialPlanetTarget(planet({ owner: 0, environment: 5 }))).toBe(false);
  });

  it("returns artificial planet targets in a star", () => {
    const gasGiant = planet({ id: 2, owner: 0, environment: 1 });
    const asteroid = planet({ id: 3, owner: 0, environment: 2 });
    const barren = planet({ id: 4, owner: 0, environment: 5 });

    expect(
      getArtificialPlanetTargets({
        id: 1,
        name: "Ada",
        planets: [gasGiant, asteroid, barren],
        color: 0,
        x: 0,
        y: 0,
        ownedPlanets: 1,
        explored: true,
        ownerColor: null,
      }),
    ).toEqual([gasGiant, asteroid]);
  });
});

describe("top bar number formatting", () => {
  it("adds thousands separators to integer and signed values", () => {
    expect(formatInteger(1234567)).toBe("1,234,567");
    expect(formatSigned(1234)).toBe("+1,234");
    expect(formatSigned(-1234)).toBe("-1,234");
    expect(formatSignedBc(1234)).toBe("+1,234 BC");
  });

  it("compacts large treasury values", () => {
    expect(formatTreasury(99999)).toBe("99,999");
    expect(formatTreasury(100000)).toBe("100K");
    expect(formatTreasury(999999)).toBe("999K");
    expect(formatTreasury(1000000)).toBe("1M");
    expect(formatTreasury(1100000)).toBe("1.1M");
  });

  it("adds thousands separators to command point display", () => {
    expect(
      formatCommandPointsDisplay({
        commandPointsRemaining: 1200,
        commandPointsCapacity: 3000,
      } as Parameters<typeof formatCommandPointsDisplay>[0]),
    ).toBe("+1,200 (3,000)");
  });

  it("formats freighter availability", () => {
    expect(
      formatFreighterDisplay(
        { freighters: 15, freightersUsed: 1 } as Parameters<typeof formatFreighterDisplay>[0],
        0,
      ),
    ).toBe("+14 (15)");
  });
});

describe("freighter food projection", () => {
  it("caps projected food freighter use by available capacity", () => {
    const game: GameViewModel = {
      id: "game",
      currentTurn: 3500,
      players: [
        {
          id: 1,
          freighters: 2,
          freightersUsed: 1,
        } as GameViewModel["players"][number],
      ],
      fleets: [],
      resources: {
        shipParts: { hullSizes: [], drives: [], armors: [], computers: [], fuelCells: [], shields: [] },
        shipSystemCollection: { shipSystems: [] },
        buildingCollection: { buildings: [] },
        modifierCollection: { modifiers: [] },
        shipDesignCollection: { shipDesigns: [] },
        techCollection: { techs: [] },
        techTree: { subjects: [] },
      },
      pendingColonyBases: {},
      pendingColonyShips: {},
      pendingArtificialPlanets: {},
      galaxy: {
        width: 100,
        height: 100,
        stars: [
          {
            id: 1,
            name: "Sol",
            color: 1,
            x: 0,
            y: 0,
            explored: true,
            ownedPlanets: 2,
            ownerColor: null,
            planets: [
              planet({ id: 1, population: 2000, food: { farmers: 2, produced_per_farmer: 2, produced_per_building: 0 } }),
              planet({ id: 2, population: 2000, food: { farmers: 0, produced_per_farmer: 2, produced_per_building: 0 } }),
            ],
          },
        ],
      },
    };

    expect(getProjectedFoodFreightersUsed(game, 1, undefined, game.players[0])).toBe(1);
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

  it("includes Housing growth from projected net industry", () => {
    const colony = planet({
      currentProject: building("Housing"),
      industry: {
        workers: 5,
        produced_per_worker: 2,
        produced_per_building: 0,
        produced_per_colonist: 0,
      },
    });

    expect(getPlanetProjectedPopulationGrowth(colony)).toBe(140);
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

describe("getPlanetProjectedIncome", () => {
  it("does not produce Trade Goods while Housing", () => {
    const colony = planet({
      currentProject: building("Housing"),
      isBuilding: false,
    });

    expect(getPlanetProjectedIncome(colony).tradeGoods).toBe(0);
  });

  it("produces Trade Goods only while Trade Goods is active", () => {
    const colony = planet({
      currentProject: building("Trade Goods"),
      isBuilding: false,
    });

    expect(getPlanetProjectedIncome(colony).tradeGoods).toBe(5);
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
      pendingArtificialPlanets: {},
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
        artificialPlanet: [],
        buyProduction: [],
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
      pendingArtificialPlanets: {},
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
        artificialPlanet: [],
        buyProduction: [],
        selectResearch: undefined,
        movePopulation: [
          { planetId: colony.id, from: "workers", to: "farmers", count: 2 },
        ],
      }),
    ).toBe(7);
  });
});
