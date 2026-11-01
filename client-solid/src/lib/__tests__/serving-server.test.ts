import { describe, expect, it } from "vitest";
import { servingServerUrl } from "../setup-session-storage";
describe("serving server default", () => {
  it("uses the HTTP origin in production, including HTTPS and custom ports", () => {
    expect(servingServerUrl({ origin: "https://game.example:8443", protocol: "https:" }, true)).toBe("https://game.example:8443");
    expect(servingServerUrl({ origin: "http://localhost:8080", protocol: "http:" }, true)).toBe("http://localhost:8080");
  });
  it("does not point development or file URLs at an unrelated API", () => {
    expect(servingServerUrl({ origin: "http://localhost:5173", protocol: "http:" }, false)).toBe("");
    expect(servingServerUrl({ origin: "null", protocol: "file:" }, true)).toBe("");
  });
});
