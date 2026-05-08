import { getPlanetProjectedPopulationGrowth } from "../utils";
import type { BuildingViewModel, PlanetViewModel } from "../types/domain";

function building(name: string, modifiers: BuildingViewModel["modifiers"] = []): BuildingViewModel {
  return {
    id: 1,
    name,
    description: "",
    icon: "🏗️",
    cost: 0,
    maintenance: 0,
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
    pollution: 0,
    currentProject: null,
    buildQueue: [],
    buildings: [],
    industryStack: 0,
    buildBucket: 0,
    isBuilding: false,
    ...overrides,
  };
}

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
});
