import { describe, expect, it } from "vitest";

import { submittedTurnMessage, toTurnCommands } from "../app-context";
import type { ApiTurnResponse } from "../../lib/types/api";
import type { PendingCommands } from "../../lib/types/commands";
import type { GameViewModel } from "../../lib/types/domain";

function pendingCommands(overrides: Partial<PendingCommands> = {}): PendingCommands {
  return {
    build: [],
    movePopulation: [],
    selectResearch: undefined,
    colonize: [],
    artificialPlanet: [],
    fleetMove: [],
    invade: [],
    buyProduction: [],
    transferPopulation: [],
    ...overrides,
  };
}

function gameWithFleets(fleetIds: number[]): GameViewModel {
  return {
    id: "g",
    currentTurn: 3500,
    galaxy: { stars: [], width: 600, height: 400 },
    players: [],
    fleets: fleetIds.map((id) => ({
      id,
      ownerId: 1,
      starId: 1,
      x: 0,
      y: 0,
      originStarId: null,
      destinationStarId: null,
      totalTurns: 0,
      turnsRemaining: 0,
      ships: [],
    })),
    resources: {
      shipParts: { hullSizes: [], drives: [], armors: [], computers: [], fuelCells: [], shields: [] },
      shipSystemCollection: { shipSystems: [] },
      buildingCollection: { buildings: [] },
      modifierCollection: { modifiers: [] },
      shipDesignCollection: { shipDesigns: [] },
      techTree: { subjects: [] },
      techCollection: { techs: [] },
    },
    pendingColonyBases: {},
    pendingColonyShips: {},
    pendingArtificialPlanets: {},
  };
}

describe("toTurnCommands", () => {
  it("assigns the remaining population after outbound transfers and job moves", () => {
    const game = gameWithFleets([]);
    game.galaxy.stars = [{ planets: [
      { id: 1, population: 8500, food: { farmers: 4 }, industry: { workers: 3 }, science: { scientists: 1 } },
      { id: 2, population: 5000, food: { farmers: 2 }, industry: { workers: 2 }, science: { scientists: 1 } },
    ] }] as unknown as GameViewModel["galaxy"]["stars"];
    const commands = toTurnCommands(pendingCommands({
      movePopulation: [
        { planetId: 1, from: "farmers", to: "workers", count: 1 },
        { planetId: 2, from: "workers", to: "scientists", count: 1 },
      ],
      transferPopulation: [
        { source_planet_id: 1, destination_planet_id: 2, population_units: 1 },
        { source_planet_id: 1, destination_planet_id: 2, population_units: 1 },
      ],
    }), game);
    expect(commands.movePopulation).toEqual([
      { planetId: 1, farmers: 1, workers: 4, scientists: 1 },
      { planetId: 2, farmers: 2, workers: 1, scientists: 2 },
    ]);
  });

  it("serializes multiple artificial planet commands", () => {
    const commands = toTurnCommands(
      pendingCommands({
        artificialPlanet: [
          { source_planet_id: 1, target_planet_id: 2 },
          { source_planet_id: 3, target_planet_id: 4 },
        ],
      }),
      null,
    );

    expect(commands.artificial_planet_many).toEqual([
      { source_planet_id: 1, target_planet_id: 2 },
      { source_planet_id: 3, target_planet_id: 4 },
    ]);
  });

  it("serializes population transfer commands", () => {
    const commands = toTurnCommands(
      pendingCommands({
        transferPopulation: [
          { source_planet_id: 1, destination_planet_id: 2, population_units: 1 },
        ],
      }),
      null,
    );

    expect(commands.transfer_population).toEqual([
      { source_planet_id: 1, destination_planet_id: 2, population_units: 1 },
    ]);
  });

  it("serializes Housing build commands unchanged", () => {
    const commands = toTurnCommands(
      pendingCommands({
        build: [{ planetId: 1, building: "Housing" }],
      }),
      null,
    );

    expect(commands.build).toEqual([{ planetId: 1, building: "Housing" }]);
  });

  it("drops fleet commands for fleets that no longer exist", () => {
    const commands = toTurnCommands(
      pendingCommands({
        colonize: [
          { planet_id: 2, fleet_id: 1 },
          { planet_id: 3, fleet_id: 9 },
        ],
        fleetMove: [
          { fleet_id: 1, destination_star_id: 4, ship_indexes: [0] },
          { fleet_id: 9, destination_star_id: 5, ship_indexes: [0] },
        ],
      }),
      gameWithFleets([9]),
    );

    expect(commands.colonize).toEqual({ planet_id: 3, fleet_id: 9 });
    expect(commands.fleet_move).toEqual([
      { fleet_id: 9, destination_star_id: 5, ship_indexes: [0] },
    ]);
  });
});


describe("submittedTurnMessage", () => {
  const response: ApiTurnResponse = {
    ok: true,
    advanced: false,
    turn_status: { mode: "simultaneous", ready_player_ids: [1], waiting_player_ids: [2], required_player_ids: [1, 2] },
  };

  it("shows the server message when another player must resubmit", () => {
    const game = gameWithFleets([]);
    game.turnStatus = { mode: "simultaneous", readyPlayerIds: [1], waitingPlayerIds: [2], requiredPlayerIds: [1, 2] };
    expect(submittedTurnMessage({ ...response, message: "B must resubmit" }, game, 1)).toBe("B must resubmit");
  });

  it("keeps the generic waiting message when no message or readiness is available", () => {
    expect(submittedTurnMessage(response, gameWithFleets([]), 1)).toBe("Orders submitted; waiting for other players");
  });
});
