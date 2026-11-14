import { cleanup, fireEvent, render, screen } from "@solidjs/testing-library";
import { createStore } from "solid-js/store";
import { afterEach, describe, expect, it, vi } from "vitest";

import { FleetInspectorPanel } from "../../components/FleetInspectorPanel";
import { mapGame } from "../mappers";
import type { ApiGame } from "../types/api";
import type { GameViewModel } from "../types/domain";
import type { useAppContext } from "../../state/app-context";

const mocked = vi.hoisted(() => ({ context: null as unknown as ReturnType<typeof useAppContext> }));
vi.mock("../../state/app-context", () => ({ useAppContext: () => mocked.context }));
afterEach(cleanup);

type ContactState = { atWar?: boolean; peaceTreaty?: boolean };

/** A transport of player 1 orbits a star with a colony of player 2. */
function setup(contact: ContactState | null) {
  const contacts = contact
    ? [{ playerId: 2, metTurn: 3500.1, starId: 5, relation: 0, atWar: false, warSince: 0, peaceTreaty: false, ...contact }]
    : [];
  const game = {
    id: "invade-gate",
    currentTurn: 3501,
    galaxy: {
      width: 100, height: 100,
      stars: [{ id: 5, name: "Vega", x: 0, y: 0, explored: true, planets: [{ id: 50, name: "Vega I", owner: 2 }] }],
    },
    players: [
      { id: 1, name: "Ada", color: "#fff", available: { fuelCells: [] }, shipDesigns: [], researchedTechs: [], contacts },
      { id: 2, name: "Zorg", color: "#f00", contacts: [] },
    ],
    fleets: [{ id: 11, ownerId: 1, starId: 5, x: 0, y: 0, turnsRemaining: 0, ships: [{ type: "transport", name: "Transport 1" }] }],
    resources: { shipParts: { fuelCells: [] } },
  } as unknown as GameViewModel;
  const [state] = createStore({
    game,
    pendingCommands: { fleetMove: [], colonize: [], invade: [] },
    isBusy: false,
  });
  const beginInvasion = vi.fn();
  mocked.context = {
    state,
    currentPlayer: () => state.game.players[0],
    beginColonyShipColonization: vi.fn(),
    cancelColonyShipColonization: vi.fn(),
    beginInvasion,
    cancelInvasion: vi.fn(),
  } as unknown as ReturnType<typeof useAppContext>;
  render(() => <FleetInspectorPanel starId={5} fleetId={11} onViewSystem={() => {}} onSelectionChange={() => {}} />);
  const button = screen.getByRole("button", { name: "Invade Planet" }) as HTMLButtonElement;
  return { button, beginInvasion };
}

describe("Invade gate", () => {
  it("is disabled with an explanation under a peace treaty", () => {
    const { button, beginInvasion } = setup({ peaceTreaty: true });
    expect(button.disabled).toBe(true);
    expect(button.title).toMatch(/peace treaty with Zorg/i);
    expect(screen.getByText(/Peace treaty with Zorg: declare war before invading/)).toBeTruthy();
    fireEvent.click(button);
    expect(beginInvasion).not.toHaveBeenCalled();
  });

  it.each<[string, ContactState | null]>([
    ["at war", { atWar: true }],
    ["in the default no-treaty state (auto-declares)", {}],
    ["without a contact entry", null],
  ])("stays available %s", (_label, contact) => {
    const { button, beginInvasion } = setup(contact);
    expect(button.disabled).toBe(false);
    expect(screen.queryByText(/declare war before invading/)).toBeNull();
    fireEvent.click(button);
    expect(beginInvasion).toHaveBeenCalledWith(11, 50);
  });

  it("maps peace_treaty from the API contact", () => {
    const api = {
      id: "g", current_turn: 3501,
      galaxy: { width: 10, height: 10, stars: [] },
      players: [{ id: 1, name: "Ada", color: "#fff", contacts: [{ player_id: 2, peace_treaty: true }, { player_id: 3 }] }],
      fleets: [],
      resources: { ship_parts: { hull_sizes: [], drives: [], armors: [], computers: [], fuel_cells: [], shields: [] } },
    } as unknown as ApiGame;
    const contacts = mapGame(api).players[0]?.contacts ?? [];
    expect(contacts.map((c) => [c.playerId, c.peaceTreaty])).toEqual([[2, true], [3, false]]);
  });
});
