import { expect, it } from "vitest";
import * as utils from "../utils";

it("matches full-save permission for the requesting slot and AI opponents", () => {
  expect(utils).toHaveProperty("canDownloadGameSave");
  const allowed = utils.canDownloadGameSave;
  expect(allowed([{ id: 1 }], 1)).toBe(true);
  expect(allowed([{ id: 1, controller: "human" }, { id: 2, controller: "server_ai" }, { id: 3, controller: "server_ai" }], 1)).toBe(true);
  for (const controller of ["human", "external", undefined, "unknown"]) {
    expect(allowed([{ id: 1 }, { id: 2, controller }], 1)).toBe(false);
  }
  expect(allowed([{ id: 1 }, { id: 2, controller: "server_ai" }], 2)).toBe(false);
  expect(allowed([{ id: 1, controller: "server_ai" }], 99)).toBe(false);
  expect(allowed([], 1)).toBe(false);
});
