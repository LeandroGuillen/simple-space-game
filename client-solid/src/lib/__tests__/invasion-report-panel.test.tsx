import { cleanup, render } from "@solidjs/testing-library";
import { afterEach, describe, expect, it } from "vitest";

import { InvasionReportPanel } from "../../components/InvasionReportPanel";

afterEach(() => {
  cleanup();
});

describe("InvasionReportPanel", () => {
  it("shows conquest consequences for participants", () => {
    const report = {
      id: 1, turn: 2, planet_id: 3, planet_name: "Praar", x: 0, y: 0,
      attacker_owner_id: 1, defender_owner_id: 2, success: true, planet_captured: true,
      rounds: [], initial_attacker: { owner_id: 1, marines: [], armored: [] },
      initial_defender: { owner_id: 2, marines: [], armored: [] },
      surviving_units: { owner_id: 1, marines: [], armored: [] },
      capture_effects: { population_race: "Klackon", unrest_penalty: 0.2,
        destroyed_buildings: ["Automated Factory"], removed_buildings: ["Capitol"] },
    };
    const view = render(() => <InvasionReportPanel reports={[report]} playerId={2} playerNames={new Map()} />);
    expect(view.getByText("Population: Klackon remains on the planet.")).toBeTruthy();
    expect(view.getByText("Conquest unrest: 20% morale penalty.")).toBeTruthy();
    expect(view.getByText("Buildings destroyed: Automated Factory.")).toBeTruthy();
    expect(view.getByText("Former empire buildings removed: Capitol.")).toBeTruthy();
    expect(view.getByText("Construction project and queue cleared.")).toBeTruthy();
  });
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
      <InvasionReportPanel reports={reports} playerId={1} playerNames={new Map([[1, "A"], [2, "B"]])} />
    ));
    expect(view.getAllByText("Marines: 0").length).toBeGreaterThanOrEqual(2);
    expect(view.getAllByText("Armored: 0").length).toBeGreaterThanOrEqual(2);
  });

  it("shows only reports involving the current player", () => {
    const base = { turn: 1, planet_id: 1, x: 0, y: 0, success: false, planet_captured: false, rounds: null,
      initial_attacker: { owner_id: 1, marines: null, armored: null },
      initial_defender: { owner_id: 2, marines: null, armored: null },
      surviving_units: { owner_id: 1, marines: null, armored: null } };
    const reports = [
      { ...base, id: 1, planet_name: "Visible", attacker_owner_id: 1, defender_owner_id: 2 },
      { ...base, id: 2, planet_name: "Hidden", attacker_owner_id: 2, defender_owner_id: 3 },
    ];
    const view = render(() => <InvasionReportPanel reports={reports} playerId={1} playerNames={new Map()} />);
    expect(view.getByText(/Invasion of Visible/)).toBeTruthy();
    expect(view.queryByText(/Invasion of Hidden/)).toBeNull();
  });

  it("tolerates null rounds from cancelled older saves", () => {
    const reports = [
      {
        id: 2,
        turn: 2,
        planet_id: 2,
        planet_name: "Praar",
        x: 0,
        y: 0,
        attacker_owner_id: 1,
        defender_owner_id: 2,
        success: false,
        planet_captured: false,
        cancelled: true,
        rounds: null,
        initial_attacker: { owner_id: 1, marines: [], armored: [] },
        initial_defender: { owner_id: 2, marines: [], armored: [] },
        surviving_units: { owner_id: 0, marines: [], armored: [] },
      },
    ] as unknown as Parameters<typeof InvasionReportPanel>[0]["reports"];

    const view = render(() => (
      <InvasionReportPanel reports={reports} playerId={1} playerNames={new Map([[1, "A"], [2, "B"]])} />
    ));
    expect(view.getByText(/Invasion cancelled/i)).toBeTruthy();
    expect(view.queryByText(/Battle Summary/i)).toBeNull();
  });
});
