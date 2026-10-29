import { describe, expect, it } from "vitest";

import { isServerDebugEnabled } from "../server-status";

describe("isServerDebugEnabled", () => {
  it("is false when status omits debug or sets it false", () => {
    expect(isServerDebugEnabled(undefined)).toBe(false);
    expect(isServerDebugEnabled(null)).toBe(false);
    expect(isServerDebugEnabled({ ok: true })).toBe(false);
    expect(isServerDebugEnabled({ ok: true, debug: false })).toBe(false);
  });

  it("is true only when status.debug is true", () => {
    expect(isServerDebugEnabled({ ok: true, debug: true })).toBe(true);
  });
});
