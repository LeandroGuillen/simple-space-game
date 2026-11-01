import { afterEach, describe, expect, it, vi } from "vitest";

import { ApiClient } from "../api";

describe("ApiClient game saves", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("downloads the full state from the save endpoint", async () => {
    const fetchMock = vi.fn(async () =>
      new Response(JSON.stringify({ id: "game-1", players: [] }), {
        status: 200,
        headers: { "content-type": "application/json" },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    const client = new ApiClient("http://example.test");
    expect(client).toHaveProperty("getGameSave");
    await expect(client.getGameSave("game-1")).resolves.toEqual({
      id: "game-1",
      players: [],
    });
    expect(fetchMock).toHaveBeenCalledWith("http://example.test/games/game-1/save", { headers: { "content-type": "application/json", "Game-Id": "game-1", "Player-Id": "1" } });
  });
  it("saves server state with game and player headers", async () => {
    const fetchMock = vi.fn(async () => new Response(JSON.stringify({ ok: true }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("http://example.test", 2);
    await expect(client.saveGameNow("game-1")).resolves.toEqual({ ok: true });
    expect(fetchMock).toHaveBeenCalledWith("http://example.test/games/game-1/save", {
      method: "POST", headers: { "content-type": "application/json", "Game-Id": "game-1", "Player-Id": "2" },
    });
  });

  it("surfaces save failures", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => new Response(JSON.stringify({ error: "could not save game" }), { status: 500 })));
    await expect(new ApiClient("http://example.test").saveGameNow("game-1")).rejects.toThrow("could not save game");
  });
});
