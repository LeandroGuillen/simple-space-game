import { describe, expect, it } from "vitest";

import { toTurnCommands } from "../app-context";
import type { PendingCommands } from "../../lib/types/commands";

function pendingCommands(overrides: Partial<PendingCommands> = {}): PendingCommands {
  return {
    build: [],
    movePopulation: [],
    selectResearch: undefined,
    colonize: [],
    artificialPlanet: [],
    fleetMove: [],
    buyProduction: [],
    ...overrides,
  };
}

describe("toTurnCommands", () => {
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
});
