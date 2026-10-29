import { cleanup, render } from "@solidjs/testing-library";
import { afterEach, describe, expect, it } from "vitest";

import { InvasionReportPanel } from "../../components/InvasionReportPanel";

afterEach(() => {
  cleanup();
});

describe("InvasionReportPanel", () => {
  it("tolerates nullish marine/armored arrays from older saves", () => {
    const reports = [
      {
        id: 1,
        turn: 1,
        planet_id: 1,
        planet_name: "Grey Harbor",
        x: 0,
        y: 0,
        attacker_owner_id: 1,
        defender_owner_id: 2,
        success: false,
        planet_captured: false,
        rounds: [],
        initial_attacker: { owner_id: 1, marines: null, armored: null },
        initial_defender: { owner_id: 2, marines: undefined, armored: null },
        surviving_units: { owner_id: 0, marines: [], armored: [] },
      },
    ] as unknown as Parameters<typeof InvasionReportPanel>[0]["reports"];

    const view = render(() => (
      <InvasionReportPanel reports={reports} playerNames={new Map([[1, "A"], [2, "B"]])} />
    ));
    expect(view.getAllByText("Marines: 0").length).toBeGreaterThanOrEqual(2);
    expect(view.getAllByText("Armored: 0").length).toBeGreaterThanOrEqual(2);
  });
});
