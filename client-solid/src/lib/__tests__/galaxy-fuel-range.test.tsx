import { MemoryRouter, Route } from "@solidjs/router";
import { cleanup, fireEvent, render, screen } from "@solidjs/testing-library";
import { createStore } from "solid-js/store";
import { afterEach, describe, expect, it, vi } from "vitest";
import { GalaxyPage } from "../../routes/GalaxyPage";
import type { GameViewModel } from "../types/domain";
import type { useAppContext } from "../../state/app-context";

const mocked = vi.hoisted(() => ({ context: null as unknown as ReturnType<typeof useAppContext> }));
vi.mock("../../state/app-context", () => ({ useAppContext: () => mocked.context }));
afterEach(cleanup);

function setup(width = 600, height = 400, range = 2, unlimited = false) {
  const player = { id: 1, color: "#fff", available: { fuelCells: [] }, shipDesigns: [], researchedTechs: [] };
  const stars = [
    { id: 1, name: "Home", x: 0, y: 0, planets: [{ owner: 1 }] },
    { id: 2, name: "Boundary", x: 72, y: 96, planets: [] }, // exactly two parsecs
    { id: 3, name: "Distant", x: 180, y: 0, planets: [] },
    { id: 4, name: "Remote colony", x: width - 60, y: height - 10, planets: [{ owner: 1 }] },
    { id: 5, name: "Remote neighbor", x: width - 10, y: height - 10, planets: [] },
  ].map((star) => ({ ...star, color: 1, explored: false, ownedPlanets: 0, ownerColor: null }));
  const ship = { type: "ship_design", name: "Scout", fuelCell: "Test fuel", fuelRangeParsecs: range, unlimitedFuelRange: unlimited, interstellarSpeed: 1 };
  const game = {
    id: "fuel-test", galaxy: { width, height, stars }, players: [player],
    fleets: [{ id: 11, ownerId: 1, starId: 1, x: 0, y: 0, turnsRemaining: 0, ships: [ship, { ...ship, name: "Long range", fuelCell: "Long fuel", fuelRangeParsecs: 4, unlimitedFuelRange: false }] }],
    resources: { shipParts: { fuelCells: [{ name: "Test fuel", autonomy: range }, { name: "Long fuel", autonomy: 4 }] } },
  } as unknown as GameViewModel;
  const [state, setState] = createStore({ game, session: { selectedGameId: "fuel-test" }, pendingCommands: { fleetMove: [] as { fleet_id: number; destination_star_id: number; ship_indexes: number[] }[], colonize: [], invade: [] }, isBusy: false });
  const move = vi.fn((command) => setState("pendingCommands", "fleetMove", (commands) => [...commands, command]));
  mocked.context = { state, currentPlayer: () => state.game.players[0], addFleetMoveCommand: move, setInfo: vi.fn() } as unknown as ReturnType<typeof useAppContext>;
  const view = render(() => <MemoryRouter><Route path="/" component={GalaxyPage} /></MemoryRouter>);
  const selectFleet = () => fireEvent.click(view.container.querySelector(".star-fleet-link")!);
  const star = (name: string) => screen.getByText(name, { selector: ".galaxy-star text" }).closest(".galaxy-star")!;
  return { ...view, selectFleet, star, move, setState };
}

function expectRange(star: Element, reachable: boolean) {
  expect(star.classList.contains(reachable ? "is-in-fuel-range" : "is-out-of-fuel-range")).toBe(true);
  expect(star.querySelector("title")?.textContent).toContain(reachable ? "In fuel range" : "Out of fuel range");
}

describe("galaxy fuel range selection", () => {
  it.each([[240, 180], [600, 400], [1000, 800]])("highlights legal destinations at %s × %s and uses the minimum selected range", (width, height) => {
    const { selectFleet, star } = setup(width, height);
    expect(document.querySelector(".is-in-fuel-range")).toBeNull();
    selectFleet();
    expect(screen.getByText("Range 2 parsecs")).toBeTruthy();
    expectRange(star("Boundary"), true);
    expectRange(star("Distant"), false);
    expectRange(star("Remote neighbor"), true); // supply range follows server rules
    fireEvent.click(screen.getByText("Scout"));
    expect(screen.getByText("Range 4 parsecs")).toBeTruthy();
    expectRange(star("Distant"), true);
  });

  it("clears on deselection, viewing the system, and submitting a move", () => {
    const { selectFleet, star, move } = setup();
    selectFleet();
    expectRange(star("Boundary"), true);
    fireEvent.click(screen.getByRole("button", { name: "Unselect All" }));
    expect(document.querySelector(".is-in-fuel-range, .is-out-of-fuel-range")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Select All" }));
    expectRange(star("Boundary"), true);
    fireEvent.click(screen.getByRole("button", { name: "View System" }));
    expect(document.querySelector(".is-in-fuel-range, .is-out-of-fuel-range")).toBeNull();
    selectFleet();
    fireEvent.click(star("Boundary"));
    expect(move).toHaveBeenCalledWith({ fleet_id: 11, destination_star_id: 2, ship_indexes: [0, 1] });
    expect(document.querySelector(".is-in-fuel-range, .is-out-of-fuel-range")).toBeNull();
  });

  it("keeps the overlay after rejecting an out-of-range destination and clears on cancel", () => {
    const { selectFleet, star, move } = setup();
    selectFleet();
    fireEvent.click(star("Distant"));
    expect(move).not.toHaveBeenCalled();
    expect(mocked.context.setInfo).toHaveBeenCalledWith("Destination is outside fleet fuel range");
    expectRange(star("Distant"), false);
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(document.querySelector(".is-in-fuel-range, .is-out-of-fuel-range")).toBeNull();
  });

  it("handles zero range without invalid geometry", () => {
    const { selectFleet, star } = setup(420, 300, 0);
    selectFleet();
    expect(screen.getByText("Range 0 parsecs")).toBeTruthy();
    expectRange(star("Home"), true);
    expectRange(star("Boundary"), false);
    expect(document.querySelector("svg")?.outerHTML).not.toMatch(/NaN|Infinity/);
  });

  it("treats unlimited ships as unrestricted only when all selected ships are unlimited", () => {
    const { selectFleet, star } = setup(2400, 1600, 0, true);
    selectFleet();
    expect(screen.getByText("Range 4 parsecs")).toBeTruthy();
    fireEvent.click(screen.getByText("Long range"));
    expect(screen.getByText("Range Unlimited")).toBeTruthy();
    expectRange(star("Distant"), true);
    expectRange(star("Remote neighbor"), true);
  });
});
