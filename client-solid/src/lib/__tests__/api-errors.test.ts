import { afterEach, describe, expect, it, vi } from "vitest";

import { ApiClient } from "../api";

describe("ApiClient upload error surfacing", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("surfaces server error message from 400 upload responses", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        new Response(JSON.stringify({ error: "planet 1 has negative population" }), {
          status: 400,
          statusText: "Bad Request",
          headers: { "content-type": "application/json" },
        }),
      ),
    );

    const client = new ApiClient("http://example.test");
    await expect(client.loadUploadedSave(new File(["{}"], "bad.json"))).rejects.toThrow(
      "planet 1 has negative population",
    );
  });

  it("surfaces server error message from 413 upload responses", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        new Response(JSON.stringify({ error: "uploaded save exceeds 20971520 byte limit" }), {
          status: 413,
          statusText: "Request Entity Too Large",
          headers: { "content-type": "application/json" },
        }),
      ),
    );

    const client = new ApiClient("http://example.test");
    await expect(client.loadUploadedSave(new File(["x"], "huge.json"))).rejects.toThrow(
      "uploaded save exceeds 20971520 byte limit",
    );
  });
});
