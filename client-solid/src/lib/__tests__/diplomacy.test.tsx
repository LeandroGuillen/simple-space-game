import { MemoryRouter, Route } from "@solidjs/router";
import { cleanup, render } from "@solidjs/testing-library";
import { afterEach, describe, expect, it } from "vitest";

import { DiplomacyContacts } from "../../components/DiplomacyContacts";
import { screenTitleForPath } from "../document-title";
import { diplomacyContacts, newContactsThisTurn } from "../diplomacy";
import { groupTurnEvents, turnEventHref } from "../turn-report";
import type { ApiTurnEvent } from "../types/api";
import type { GameViewModel } from "../types/domain";

afterEach(cleanup);

function sampleGame(): GameViewModel {
  return {
    id: "g",
    currentTurn: 3501.2,
    galaxy: { stars: [{ id: 4, name: "Vega", planets: [] }, { id: 9, name: "Rigel", planets: [] }] },
    players: [
      {
        id: 1, name: "Ada", color: "#fff", raceName: "Human", raceGovernment: "Dictatorship",
        contacts: [
          { playerId: 3, metTurn: 3500.4, starId: 9, relation: 0 },
          { playerId: 2, metTurn: 3501.1, starId: 4, relation: 0 },
        ],
      },
      { id: 2, name: "Zorg", color: "#f00", raceName: "Mrrshan", raceGovernment: "Feudal", eliminated: true, contacts: [] },
      { id: 3, name: "Kree", color: "#0f0", raceName: "Psilon", raceGovernment: "Democracy", contacts: [] },
    ],
    fleets: [],
  } as unknown as GameViewModel;
}

function renderContacts(game: GameViewModel) {
  const rows = diplomacyContacts(game, 1);
  return render(() => (
    <MemoryRouter>
      <Route path="/" component={() => <DiplomacyContacts rows={rows} />} />
    </MemoryRouter>
  ));
}

describe("diplomacy contacts", () => {
  it("lists contacted empires in first-contact order with relation 0", () => {
    const rows = diplomacyContacts(sampleGame(), 1);
    expect(rows.map((row) => [row.playerId, row.name, row.relation, row.starName])).toEqual([
      [3, "Kree", 0, "Rigel"],
      [2, "Zorg", 0, "Vega"],
    ]);
    expect(rows[1]?.eliminated).toBe(true);
  });

  it("lists nobody when the player has no contacts (old saves)", () => {
    const game = sampleGame();
    expect(diplomacyContacts(game, 2)).toEqual([]);
    expect(diplomacyContacts(null, 1)).toEqual([]);
  });

  it("renders a read-only table without war or peace actions", () => {
    const view = renderContacts(sampleGame());
    const table = view.getByRole("table", { name: "Contacts" });
    expect(table.textContent).toContain("Kree");
    expect(table.textContent).toContain("Zorg");
    expect(table.textContent).toContain("Eliminated");
    const relationCells = view.getAllByRole("row").slice(1).map((row) => row.querySelectorAll("td")[4]?.textContent);
    expect(relationCells).toEqual(["0", "0"]);
    expect(view.queryAllByRole("button")).toHaveLength(0);
    expect(view.queryByText(/declare war|offer peace|treaty/i)).toBeNull();
  });

  it("shows an empty state before first contact", () => {
    const game = sampleGame();
    game.players[0]!.contacts = [];
    const view = renderContacts(game);
    expect(view.queryByRole("table")).toBeNull();
    expect(view.getByText(/not met any other empire/)).toBeTruthy();
  });
});

describe("first contact events", () => {
  const events: ApiTurnEvent[] = [
    { owner: 1, kind: "first_contact", text: "First contact with Zorg at Vega", star_id: 4 },
    { owner: 1, kind: "arrival", text: "Fleet 3 arrived at Vega", star_id: 4, fleet_id: 3 },
    { owner: 2, kind: "first_contact", text: "First contact with Ada at Vega", star_id: 4 },
  ];

  it("counts the current player's new contacts for the nav badge", () => {
    expect(newContactsThisTurn(events, 1)).toBe(1);
    expect(newContactsThisTurn(events, 3)).toBe(0);
    expect(newContactsThisTurn(undefined, 1)).toBe(0);
  });

  it("groups first contact under Diplomacy and links to the Diplomacy screen", () => {
    const groups = groupTurnEvents(events.filter((event) => event.owner === 1));
    expect(groups.map((group) => group.label)).toEqual(["Fleets & combat", "Diplomacy"]);
    expect(turnEventHref(events[0]!)).toBe("/diplomacy");
  });

  it("titles the Diplomacy screen", () => {
    expect(screenTitleForPath("/diplomacy")).toBe("Diplomacy");
  });
});
