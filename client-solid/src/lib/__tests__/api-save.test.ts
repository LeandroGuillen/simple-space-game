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
    expect(fetchMock).toHaveBeenCalledWith("http://example.test/games/game-1/save");
  });
});
