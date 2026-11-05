import { cleanup, fireEvent, render, screen, waitFor } from "@solidjs/testing-library";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import App from "../../App";
import type { ApiFleet, ApiGame, ApiPlanet, ApiStar, ApiTurnEvent } from "../../lib/types/api";
import { gameSessionPath } from "../../lib/utils";
import { useAppContext } from "../app-context";

const captured = vi.hoisted(() => ({
  app: undefined as ReturnType<typeof useAppContext> | undefined,
}));

vi.mock("../../routes/GalaxyPage", async () => {
  const { useAppContext } = await import("../app-context");
  return {
    GalaxyPage: () => {
      captured.app = useAppContext();
      return null;
    },
  };
});

vi.mock("../../routes/SetupPage", async () => {
  const { useAppContext } = await import("../app-context");
  return {
    SetupPage: () => {
      captured.app = useAppContext();
      return null;
    },
  };
});

const REPORT_MODE_KEY = "client-solid.turnReportMode";
const SERVER_URL = "http://example.test";

function planet(id: number, starId: number, environment = 0, owner = 0): ApiPlanet {
  return {
    id,
    name: `Planet ${id}`,
    owner,
    parent_star: starId,
    mineral_class: 2,
    size: 4,
    gravity: 2,
    environment,
    population: 0,
    last_turn_pop_growth: 0,
    food: { farmers: 0, produced_per_farmer: 1, produced_per_building: 0 },
    industry: { workers: 0, produced_per_worker: 1, produced_per_building: 0 },
    science: { scientists: 0, produced_per_scientist: 1, produced_per_building: 0 },
    pollution: 0,
    marines: 0,
    armor: 0,
    build_queue: null,
    buildings: null,
    industry_stack: 0,
    build_bucket: 0,
    is_building: false,
  };
}

function star(id: number, name: string, environments: Array<number | { environment: number; owner?: number }> = [0]): ApiStar {
  return {
    id,
    name,
    color: 1,
    x: id * 50,
    y: id * 40,
    planets: environments.map((entry, index) => {
      const environment = typeof entry === "number" ? entry : entry.environment;
      const owner = typeof entry === "number" ? 0 : entry.owner ?? 0;
      const planetId = id * 100 + index + 1;
      const result = planet(planetId, id, environment, owner);
      result.name = `${name} ${index + 1}`;
      return result;
    }),
  };
}

function fleet(
  id: number,
  starId: number,
  options: { turnsRemaining?: number; destinationStarId?: number; ships?: string[] } = {},
): ApiFleet {
  return {
    id,
    owner: 1,
    star_id: starId,
    x: 0,
    y: 0,
    turns_remaining: options.turnsRemaining ?? 0,
    destination_star_id: options.destinationStarId ?? 0,
    total_turns: options.turnsRemaining ? 1 : 0,
    ships: (options.ships ?? ["colony_ship"]).map((type) => ({ type, name: type })),
  };
}

function arrival(fleetId: number, starId: number): ApiTurnEvent {
  return { owner: 1, kind: "arrival", text: `Fleet ${fleetId} arrived`, fleet_id: fleetId, star_id: starId };
}

function game(options: {
  id?: string;
  turn?: number;
  fleets?: ApiFleet[];
  stars?: ApiStar[];
  events?: ApiTurnEvent[];
  pendingColonyBases?: Record<number, number[]>;
  pendingColonyShips?: Record<number, number[]>;
  pendingArtificialPlanets?: Record<number, number[]>;
} = {}): ApiGame {
  return {
    id: options.id ?? "g",
    current_turn: options.turn ?? 1,
    galaxy: {
      width: 600,
      height: 400,
      stars: options.stars ?? [star(1, "Alpha"), star(2, "Beta"), star(3, "Gamma"), star(4, "Delta")],
    },
    players: [{ id: 1, name: "Ada", color: "#FFFFFF", race: { id: 1, name: "Human" }, available: { buildings: [] } }],
    fleets: options.fleets ?? [],
    turn_events: options.events ?? [],
    pending_colony_bases: options.pendingColonyBases ?? {},
    pending_colony_ships: options.pendingColonyShips ?? {},
    pending_artificial_planets: options.pendingArtificialPlanets ?? {},
    resources: {
      ship_parts: { hull_sizes: [], drives: [], armors: [], computers: [], fuel_cells: [], shields: [] },
    },
  };
}

class FakeWebSocket {
  onmessage: ((event: MessageEvent) => void) | null = null;
  onerror: (() => void) | null = null;
  url: string;
  constructor(url: string) {
    this.url = url;
  }
  close() {}
}

async function setup(initialGame: ApiGame) {
  const games = new Map<string, ApiGame>([[initialGame.id, initialGame], ["other", game({ id: "other" })]]);
  vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);
    let body: unknown;
    if (url.endsWith("/status")) {
      body = { ok: true };
    } else if (url.endsWith("/games/player-colors")) {
      body = { max_players: 8, colors: ["#FFFFFF"] };
    } else if (url.endsWith("/games")) {
      body = { game_list: [] };
    } else if (url.endsWith("/turn") && init?.method === "POST") {
      body = {
        ok: true,
        advanced: true,
        turn_status: { mode: "simultaneous", ready_player_ids: [], waiting_player_ids: [], required_player_ids: [1] },
      };
    } else {
      const match = url.match(/\/games\/([^/]+)$/);
      if (!match) throw new Error(`Unexpected request: ${init?.method ?? "GET"} ${url}`);
      const selected = games.get(match[1]);
      if (!selected) throw new Error(`Unknown test game: ${match[1]}`);
      body = selected;
    }
    return new Response(JSON.stringify(body), { status: 200, headers: { "content-type": "application/json" } });
  }));

  captured.app = undefined;
  render(() => <App />);
  await waitFor(() => expect(captured.app).toBeDefined());
  const context = captured.app!;
  await context.connect(SERVER_URL);
  await context.joinGame(initialGame.id, 1);
  await waitFor(() => expect(context.state.game?.id).toBe(initialGame.id));
  window.history.pushState(null, "", gameSessionPath(initialGame.id, 1));
  window.dispatchEvent(new PopStateEvent("popstate"));
  await new Promise((resolve) => setTimeout(resolve, 0));

  return {
    context,
    games,
    async advance(nextGame: ApiGame) {
      games.set(nextGame.id, nextGame);
      await context.endTurn();
      await Promise.resolve();
    },
  };
}

beforeEach(() => {
  const values = new Map<string, string>();
  const storage: Storage = {
    get length() { return values.size; },
    clear: () => values.clear(),
    getItem: (key) => values.get(key) ?? null,
    key: (index) => [...values.keys()][index] ?? null,
    removeItem: (key) => { values.delete(key); },
    setItem: (key, value) => { values.set(key, String(value)); },
  };
  vi.stubGlobal("localStorage", storage);
  Object.defineProperty(window, "localStorage", { value: storage, configurable: true });
  window.localStorage.setItem(REPORT_MODE_KEY, "off");
  window.history.replaceState({}, "", "/");
  vi.stubGlobal("WebSocket", FakeWebSocket);
  vi.spyOn(HTMLElement.prototype, "getClientRects").mockReturnValue([{}] as unknown as DOMRectList);
});

afterEach(() => {
  cleanup();
  window.localStorage.clear();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("colony ship arrival prompts", () => {
  it("opens for an arrival, lists every eligible planet, and queues the right fleet command", async () => {
    const initial = game({ fleets: [fleet(11, 1, { turnsRemaining: 1, destinationStarId: 2 })] });
    const { context, advance } = await setup(initial);
    await advance(game({
      turn: 2,
      fleets: [fleet(11, 2)],
      stars: [star(1, "Alpha"), star(2, "Beta", [0, 3, 1, { environment: 0, owner: 2 }])],
      events: [arrival(11, 2)],
    }));

    expect(context.state.showColonizeModal).toBe(true);
    expect(context.state.pendingColonyShipFleets).toEqual([11]);
    await waitFor(() => expect(document.querySelectorAll(".colonize-planet-card.is-eligible")).toHaveLength(2));
    expect(screen.getByRole("heading", { name: "Colony ship arrived at Beta" })).toBeTruthy();
    expect(document.querySelectorAll(".colonize-planet-card")).toHaveLength(4);

    const target = screen.getByText("Beta 2").closest("button");
    expect(target).not.toBeNull();
    fireEvent.click(target!);
    expect(context.state.pendingCommands.colonize).toEqual([{ planet_id: 202, fleet_id: 11 }]);
    expect(context.state.showColonizeModal).toBe(false);
  });

  it("does not prompt without an eligible planet, while travelling, or with a queued colonize command", async () => {
    const initial = game({ fleets: [
      fleet(11, 1, { turnsRemaining: 1, destinationStarId: 2 }),
      fleet(12, 1, { turnsRemaining: 1, destinationStarId: 3 }),
      fleet(13, 1, { turnsRemaining: 1, destinationStarId: 4 }),
    ] });
    const { context, advance } = await setup(initial);
    context.addColonizeCommand({ fleet_id: 13, planet_id: 401 });
    await advance(game({
      turn: 2,
      fleets: [
        fleet(11, 2),
        fleet(12, 1, { turnsRemaining: 1, destinationStarId: 3 }),
        fleet(13, 4),
      ],
      stars: [
        star(1, "Alpha"),
        star(2, "Beta", [1, { environment: 0, owner: 2 }]),
        star(3, "Gamma"),
        star(4, "Delta"),
      ],
      events: [arrival(11, 2), arrival(12, 3), arrival(13, 4)],
    }));

    expect(context.state.showColonizeModal).toBe(false);
    expect(context.state.pendingColonyShipFleets).toEqual([]);
    expect(screen.queryByRole("dialog")).toBeNull();
  });

  it.each(["close button", "Escape", "backdrop"])("cancel via %s suppresses later prompts at the same star", async (method) => {
    const initial = game({ fleets: [fleet(11, 1, { turnsRemaining: 1, destinationStarId: 2 })] });
    const { context, advance } = await setup(initial);
    const arrived = game({ turn: 2, fleets: [fleet(11, 2)], events: [arrival(11, 2)] });
    await advance(arrived);
    expect(context.state.showColonizeModal).toBe(true);
    await waitFor(() => expect(screen.getByRole("dialog")).toBeTruthy());

    if (method === "close button") {
      fireEvent.click(screen.getByRole("button", { name: "Close" }));
    } else if (method === "Escape") {
      fireEvent.keyDown(screen.getByRole("dialog"), { key: "Escape" });
    } else {
      const backdrop = document.querySelector(".research-modal-backdrop");
      expect(backdrop).not.toBeNull();
      fireEvent.click(backdrop!);
    }
    expect(context.state.showColonizeModal).toBe(false);
    expect(context.state.pendingCommands.colonize).toEqual([]);
    expect(context.state.game?.fleets[0]?.turnsRemaining).toBe(0);

    await advance(game({
      turn: 3,
      fleets: [fleet(11, 2)],
      stars: [star(1, "Alpha"), star(2, "Beta", [1, 0])],
      events: [],
    }));
    expect(context.state.showColonizeModal).toBe(false);
  });

  it("manual Colonize always reopens and clears cancellation suppression", async () => {
    const initial = game({ fleets: [fleet(11, 1, { turnsRemaining: 1, destinationStarId: 2 })] });
    const { context, advance } = await setup(initial);
    await advance(game({ turn: 2, fleets: [fleet(11, 2)], events: [arrival(11, 2)] }));
    expect(context.state.showColonizeModal).toBe(true);
    context.closeColonizeModal();
    expect(context.state.colonizePromptDismissed).toEqual({ 11: 2 });

    context.beginColonyShipColonization(11);
    expect(context.state.showColonizeModal).toBe(true);
    expect(context.state.pendingColonyShipFleets).toEqual([11]);
    expect(context.state.colonizePromptDismissed).toEqual({});
    context.closeColonizeModal();
    expect(context.state.showColonizeModal).toBe(false);
    expect(context.state.colonizePromptDismissed).toEqual({});
  });

  it("prompts again after travelling to a different eligible system", async () => {
    const initial = game({ fleets: [fleet(11, 1, { turnsRemaining: 1, destinationStarId: 2 })] });
    const { context, advance } = await setup(initial);
    await advance(game({ turn: 2, fleets: [fleet(11, 2)], events: [arrival(11, 2)] }));
    expect(context.state.showColonizeModal).toBe(true);
    context.closeColonizeModal();

    await advance(game({ turn: 3, fleets: [fleet(11, 2, { turnsRemaining: 1, destinationStarId: 3 })] }));
    expect(context.state.showColonizeModal).toBe(false);
    await advance(game({ turn: 4, fleets: [fleet(11, 3)], events: [arrival(11, 3)] }));
    expect(context.state.showColonizeModal).toBe(true);
    expect(context.state.pendingColonyShipFleets).toEqual([11]);
  });

  it("queues several arrivals and advances after both cancellation and selection", async () => {
    const initial = game({ fleets: [
      fleet(11, 1, { turnsRemaining: 1, destinationStarId: 2 }),
      fleet(12, 1, { turnsRemaining: 1, destinationStarId: 3 }),
      fleet(13, 1, { turnsRemaining: 1, destinationStarId: 4 }),
    ] });
    const { context, advance } = await setup(initial);
    await advance(game({
      turn: 2,
      fleets: [fleet(11, 2), fleet(12, 3), fleet(13, 4)],
      events: [arrival(11, 2), arrival(12, 3), arrival(13, 4)],
    }));

    expect(context.state.showColonizeModal).toBe(true);
    expect(context.state.pendingColonyShipFleets).toEqual([11]);
    context.closeColonizeModal();
    expect(context.state.showColonizeModal).toBe(true);
    expect(context.state.pendingColonyShipFleets).toEqual([12]);
    context.selectColonizeTarget(301);
    expect(context.state.pendingCommands.colonize).toEqual([{ planet_id: 301, fleet_id: 12 }]);
    expect(context.state.showColonizeModal).toBe(true);
    expect(context.state.pendingColonyShipFleets).toEqual([13]);
    context.closeColonizeModal();
    expect(context.state.showColonizeModal).toBe(false);
  });

  it("resolves a merged arrival to the surviving owner-and-star orbit fleet", async () => {
    const initial = game({ fleets: [
      fleet(99, 2, { ships: ["scout"] }),
      fleet(11, 1, { turnsRemaining: 1, destinationStarId: 2 }),
    ] });
    const { context, advance } = await setup(initial);
    await advance(game({
      turn: 2,
      fleets: [fleet(99, 2, { ships: ["scout", "colony_ship"] })],
      events: [arrival(11, 2)],
    }));

    expect(context.state.showColonizeModal).toBe(true);
    expect(context.state.pendingColonyShipFleets).toEqual([99]);
    context.selectColonizeTarget(201);
    expect(context.state.pendingCommands.colonize).toEqual([{ planet_id: 201, fleet_id: 99 }]);
  });

  it("does not prompt on load, refresh, or game switch for an already parked ship", async () => {
    const parked = game({ fleets: [fleet(11, 2)], events: [arrival(11, 2)] });
    const { context, games } = await setup(parked);
    expect(context.state.showColonizeModal).toBe(false);
    await context.refreshGame();
    expect(context.state.showColonizeModal).toBe(false);

    games.set("other", game({ id: "other", fleets: [fleet(21, 3)], events: [arrival(21, 3)] }));
    await context.joinGame("other", 1);
    expect(context.state.showColonizeModal).toBe(false);
    expect(context.state.pendingColonyShipFleets).toEqual([]);
  });

  it("waits for the Turn report and never displays both dialogs", async () => {
    window.localStorage.setItem(REPORT_MODE_KEY, "popup");
    const initial = game({ fleets: [fleet(11, 1, { turnsRemaining: 1, destinationStarId: 2 })] });
    const { context, advance } = await setup(initial);
    await advance(game({
      turn: 2,
      fleets: [fleet(11, 2)],
      events: [arrival(11, 2)],
      pendingColonyShips: { 1: [11] },
    }));

    expect(context.state.showColonizeModal).toBe(true);
    await waitFor(() => expect(screen.getByRole("heading", { name: "Turn report" })).toBeTruthy());
    expect(screen.getAllByRole("dialog")).toHaveLength(1);
    fireEvent.click(screen.getByRole("button", { name: "OK" }));
    expect(context.state.showColonizeModal).toBe(true);
    await waitFor(() => expect(screen.getByRole("heading", { name: "Colony ship arrived at Beta" })).toBeTruthy());
    expect(screen.getAllByRole("dialog")).toHaveLength(1);
  });

  it("chains colony-base, arrival, and artificial-planet modals without overlap", async () => {
    const initial = game({ fleets: [fleet(11, 1, { turnsRemaining: 1, destinationStarId: 2 })] });
    const { context, advance } = await setup(initial);
    await advance(game({
      turn: 2,
      fleets: [fleet(11, 2)],
      events: [arrival(11, 2)],
      pendingColonyBases: { 1: [101] },
      pendingArtificialPlanets: { 1: [101] },
    }));

    expect(context.state.showColonizeModal).toBe(true);
    await waitFor(() => expect(screen.getByText("Colony Base Ready")).toBeTruthy());
    expect(screen.getAllByRole("dialog")).toHaveLength(1);
    context.closeColonizeModal();
    expect(context.state.showColonizeModal).toBe(true);
    expect(context.state.showArtificialPlanetModal).toBe(false);
    await waitFor(() => expect(screen.getByRole("heading", { name: "Colony ship arrived at Beta" })).toBeTruthy());
    expect(screen.getAllByRole("dialog")).toHaveLength(1);
    context.closeColonizeModal();
    expect(context.state.showColonizeModal).toBe(false);
    expect(context.state.showArtificialPlanetModal).toBe(true);
    await waitFor(() => expect(screen.getByRole("heading", { name: "Select Artificial Planet Target" })).toBeTruthy());
    expect(screen.getAllByRole("dialog")).toHaveLength(1);
  });
});
