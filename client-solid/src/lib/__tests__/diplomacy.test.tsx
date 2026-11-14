import { MemoryRouter, Route } from "@solidjs/router";
import { cleanup, render } from "@solidjs/testing-library";
import { afterEach, describe, expect, it } from "vitest";

import { DiplomacyContacts } from "../../components/DiplomacyContacts";
import { InvasionReportPanel } from "../../components/InvasionReportPanel";
import { screenTitleForPath } from "../document-title";
import { canDeclareWar, diplomacyContacts, empireName, newContactsThisTurn } from "../diplomacy";
import { groupTurnEvents, turnEventHref } from "../turn-report";
import type { ApiInvasionReport, ApiTurnEvent } from "../types/api";
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
          { playerId: 4, metTurn: 3501.1, starId: 4, relation: 0, atWar: true, warSince: 3501.2 },
        ],
      },
      { id: 2, name: "Zorg", color: "#f00", raceName: "Mrrshan", raceGovernment: "Feudal", eliminated: true, contacts: [] },
      { id: 3, name: "Kree", color: "#0f0", raceName: "Psilon", raceGovernment: "Democracy", contacts: [] },
      { id: 4, name: "Vosk", color: "#00f", raceName: "Klackon", raceGovernment: "Unification", contacts: [] },
    ],
    fleets: [],
  } as unknown as GameViewModel;
}

function renderContacts(
  game: GameViewModel,
  options: { queued?: number[]; onDeclareWar?: (id: number) => void; onCancelDeclareWar?: (id: number) => void } = {},
) {
  const rows = diplomacyContacts(game, 1);
  const extra = {
    queuedWarIds: options.queued ?? [],
    onDeclareWar: options.onDeclareWar ?? (() => {}),
    onCancelDeclareWar: options.onCancelDeclareWar ?? (() => {}),
  };
  return render(() => (
    <MemoryRouter>
      <Route path="/" component={() => <DiplomacyContacts rows={rows} {...extra} />} />
    </MemoryRouter>
  ));
}

function cells(view: ReturnType<typeof render>, column: number) {
  return view.getAllByRole("row").slice(1).map((row) => row.querySelectorAll("td")[column]?.textContent);
}

function declareButtons(view: ReturnType<typeof render>) {
  return view.getAllByRole("row").slice(1).map((row) => {
    const button = Array.from(row.querySelectorAll("button")).find((b) => /declare war|war queued/i.test(b.textContent ?? ""));
    return button as HTMLButtonElement | undefined;
  });
}

describe("diplomacy contacts", () => {
  it("lists contacted empires in first-contact order with relation 0", () => {
    const rows = diplomacyContacts(sampleGame(), 1);
    expect(rows.map((row) => [row.playerId, row.name, row.relation, row.starName])).toEqual([
      [3, "Kree", 0, "Rigel"],
      [2, "Zorg", 0, "Vega"],
      [4, "Vosk", 0, "Vega"],
    ]);
    expect(rows[1]?.eliminated).toBe(true);
  });

  it("lists nobody when the player has no contacts (old saves)", () => {
    const game = sampleGame();
    expect(diplomacyContacts(game, 2)).toEqual([]);
    expect(diplomacyContacts(null, 1)).toEqual([]);
  });

  it("shows the real status: peace, at war, or eliminated", () => {
    const view = renderContacts(sampleGame());
    const table = view.getByRole("table", { name: "Contacts" });
    expect(table.textContent).toContain("Kree");
    expect(cells(view, 3)).toEqual(["Peace", "Eliminated", "At war"]);
    expect(cells(view, 4)).toEqual(["0", "0", "0"]);
    expect(view.queryByText(/offer peace|treaty/i)).toBeNull();
  });

  it("shows Eliminated, not At war, for an eliminated empire still flagged at war", () => {
    const game = sampleGame();
    game.players[0]!.contacts[1] = { ...game.players[0]!.contacts[1]!, atWar: true };
    const view = renderContacts(game);
    expect(cells(view, 3)).toEqual(["Peace", "Eliminated", "At war"]);
    expect(declareButtons(view)[1]?.disabled).toBe(true);
  });

  it("offers Declare War only for a contacted empire at peace", () => {
    const declared: number[] = [];
    const view = renderContacts(sampleGame(), { onDeclareWar: (id) => declared.push(id) });
    const buttons = declareButtons(view);
    expect(buttons.map((button) => button?.textContent)).toEqual(["Declare War", "Declare War", "Declare War"]);
    expect(buttons.map((button) => button?.disabled)).toEqual([false, true, true]);
    buttons[0]!.click();
    expect(declared).toEqual([3]);
  });

  it("disables a second declaration once war is queued this turn and lets it be cancelled", () => {
    const cancelled: number[] = [];
    const view = renderContacts(sampleGame(), { queued: [3], onCancelDeclareWar: (id) => cancelled.push(id) });
    const button = declareButtons(view)[0]!;
    expect(button.disabled).toBe(true);
    expect(button.textContent).toBe("War queued");
    view.getByRole("button", { name: "Cancel" }).click();
    expect(cancelled).toEqual([3]);
  });

  it("can declare only on contacted, non-eliminated empires not already at war", () => {
    const rows = diplomacyContacts(sampleGame(), 1);
    expect(rows.map((row) => row.status)).toEqual(["Peace", "Eliminated", "At war"]);
    expect(rows.map((row) => canDeclareWar(row, []))).toEqual([true, false, false]);
    expect(canDeclareWar(rows[0]!, [3])).toBe(false);
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

  it("groups declarations of war under Diplomacy and links to the Diplomacy screen", () => {
    const war: ApiTurnEvent = { owner: 1, kind: "declared_war", text: "Zorg declared war on you" };
    const groups = groupTurnEvents([war, events[1]!]);
    expect(groups.map((group) => [group.label, group.events.length])).toEqual([["Fleets & combat", 1], ["Diplomacy", 1]]);
    expect(turnEventHref(war)).toBe("/diplomacy");
  });

  it("titles the Diplomacy screen", () => {
    expect(screenTitleForPath("/diplomacy")).toBe("Diplomacy");
  });
});

describe("unmet empires", () => {
  const players = [{ id: 1, name: "Ada" }, { id: 3, name: "Kree" }];

  it("labels an owner missing from the visible players as an unknown empire", () => {
    expect(empireName(players, 3)).toBe("Kree");
    expect(empireName(players, 7)).toBe("Unknown empire");
    expect(empireName(undefined, 7)).toBe("Unknown empire");
  });

  it("does not show an unmet empire's id in invasion reports", () => {
    const report = {
      id: 1, turn: 3501, planet_id: 5, planet_name: "Terra", x: 0, y: 0,
      attacker_owner_id: 7, defender_owner_id: 1, success: false, planet_captured: false,
      rounds: [], initial_attacker: {}, initial_defender: {}, surviving_units: {},
    } as unknown as ApiInvasionReport;
    const view = render(() => (
      <InvasionReportPanel reports={[report]} playerId={1} playerNames={new Map([[1, "Ada"]])} />
    ));
    expect(view.container.textContent).toContain("Unknown empire attacked");
    expect(view.container.textContent).not.toContain("Player 7");
  });
});
