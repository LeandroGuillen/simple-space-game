import { cleanup, render, screen, waitFor } from "@solidjs/testing-library";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import App from "../../App";
import type { ApiGame, ApiPlanet } from "../../lib/types/api";
import { gameSessionPath, getPlanetProjectedIncome } from "../../lib/utils";
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

const SERVER_URL = "http://example.test";
const HYDRATED_TOTAL = 42;

function colony(): ApiPlanet {
  return {
    id: 101,
    name: "Home",
    owner: 1,
    parent_star: 1,
    mineral_class: 2,
    size: 4,
    gravity: 2,
    environment: 0,
    population: 5000,
    last_turn_pop_growth: 0,
    food: { farmers: 2, produced_per_farmer: 2, produced_per_building: 0 },
    industry: { workers: 3, produced_per_worker: 3, produced_per_building: 0 },
    science: { scientists: 0, produced_per_scientist: 1, produced_per_building: 0 },
    pollution: 0,
    marines: 0,
    armor: 0,
    build_queue: null,
    buildings: null,
    industry_stack: 0,
    build_bucket: 0,
    is_building: false,
    income: { taxes: 5, buildings: 0, industry_tax: 0, trade_goods: 0, maintenance: 0, total: HYDRATED_TOTAL },
  };
}

function game(turn: number, taxRate: number): ApiGame {
  return {
    id: "g",
    current_turn: turn,
    galaxy: {
      width: 600,
      height: 400,
      stars: [{ id: 1, name: "Alpha", color: 1, x: 50, y: 40, planets: [colony()] }],
    },
    players: [{
      id: 1,
      name: "Ada",
      color: "#FFFFFF",
      race: { id: 1, name: "Human" },
      available: { buildings: [] },
      tax_rate: taxRate,
    }],
    fleets: [],
    turn_events: [],
    pending_colony_bases: {},
    pending_colony_ships: {},
    pending_artificial_planets: {},
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
  let current = initialGame;
  const turnBodies: Record<string, unknown>[] = [];
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
      turnBodies.push(JSON.parse(String(init.body)).commands);
      body = {
        ok: true,
        advanced: true,
        turn_status: { mode: "simultaneous", ready_player_ids: [], waiting_player_ids: [], required_player_ids: [1] },
      };
    } else if (/\/games\/g$/.test(url)) {
      body = current;
    } else {
      throw new Error(`Unexpected request: ${init?.method ?? "GET"} ${url}`);
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
    turnBodies,
    async advance(nextGame: ApiGame) {
      current = nextGame;
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
  window.localStorage.setItem("client-solid.turnReportMode", "off");
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

describe("tax order", () => {
  it("is one-shot: cleared after End Turn and never re-sent over a rate changed elsewhere", async () => {
    const { context, turnBodies, advance } = await setup(game(1, 0));

    context.setTaxRate(0.3);
    context.mutateRace(["x"]);
    await advance(game(2, 0.3));

    expect(turnBodies[0]).toMatchObject({ set_tax_rate: { rate: 0.3 }, mutate_race: { trait_ids: ["x"] } });
    expect(context.state.pendingCommands.setTaxRate).toBeUndefined();
    expect(context.state.pendingCommands.mutateRace).toBeUndefined();

    // The rate is changed elsewhere (another tab, an uploaded save, ...).
    await advance(game(3, 0));
    await advance(game(4, 0));

    expect(turnBodies).toHaveLength(3);
    expect(turnBodies[1]).not.toHaveProperty("set_tax_rate");
    expect(turnBodies[2]).not.toHaveProperty("set_tax_rate");
    expect(context.state.game?.players[0]?.taxRate).toBe(0);

    const select = await waitFor(() => screen.getByLabelText("Empire tax rate") as HTMLSelectElement);
    await waitFor(() => expect(select.value).toBe("0"));
  });

  it("re-sends only when the player changes the Tax control again", async () => {
    const { context, turnBodies, advance } = await setup(game(1, 0));
    context.setTaxRate(0.2);
    await advance(game(2, 0.2));
    await advance(game(3, 0.2));
    context.setTaxRate(0.4);
    await advance(game(4, 0.4));

    expect(turnBodies.map((body) => body.set_tax_rate)).toEqual([{ rate: 0.2 }, undefined, { rate: 0.4 }]);
  });

  it("lets projected income use hydrated server income again once the order clears", async () => {
    const { context, advance } = await setup(game(1, 0));
    context.setTaxRate(0.5);
    await advance(game(2, 0.5));

    const planet = context.state.game!.galaxy.stars[0]!.planets[0]!;
    expect(planet.income.total).toBe(HYDRATED_TOTAL);
    expect(getPlanetProjectedIncome(planet, context.state.pendingCommands).total).toBe(HYDRATED_TOTAL);
  });
});
