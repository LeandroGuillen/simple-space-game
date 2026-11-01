import { cleanup, render } from "@solidjs/testing-library";
import { MemoryRouter, Route } from "@solidjs/router";
import { afterEach, describe, expect, it } from "vitest";
import { GameOver } from "../../components/GameOver";
import type { GameViewModel } from "../types/domain";

afterEach(cleanup);

function sampleGame(winnerId: number, eliminated: boolean): GameViewModel {
  return {
    id: "g", status: "finished", winnerId, turnsPlayed: 12, currentTurn: 3501.2,
    players: [{ id: 1, name: "Ada", eliminated, researchedTechs: ["Tech"] }, { id: 2, name: "Rival", eliminated: false }],
    galaxy: { stars: [{ planets: [{ owner: 1, population: 8000 }] }] },
    fleets: [{ ownerId: 1, ships: [{}, {}] }],
  } as unknown as GameViewModel;
}

describe("game over", () => {
  for (const [winner, eliminated, title] of [[1, false, "Victory"], [2, true, "Defeat"]] as const) {
    it(`shows ${title} and the player's summary`, () => {
      const game = sampleGame(winner, eliminated);
      const view = render(() => <MemoryRouter><Route path="/" component={() => <GameOver game={game} playerId={1} />} /></MemoryRouter>);
      expect(view.getByRole("heading", { name: title })).toBeTruthy();
      expect(view.getByText("Turns played")).toBeTruthy();
      expect(view.getByText("12")).toBeTruthy();
      expect(view.getByText("8,000")).toBeTruthy();
      expect(view.getByText("Return to setup")).toBeTruthy();
    });
  }
});
