import { Router, Route } from "@solidjs/router";
import { cleanup, fireEvent, render, screen } from "@solidjs/testing-library";
import { createSignal, Show } from "solid-js";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { TurnReport, TurnReportDock } from "../../components/TurnReport";
import { groupTurnEvents, readTurnReportMode, saveTurnReportMode, visibleInfoBanner, type TurnReportMode } from "../turn-report";
import type { ApiTurnEvent } from "../types/api";

beforeEach(() => {
  vi.spyOn(HTMLElement.prototype, "getClientRects").mockReturnValue([{}] as unknown as DOMRectList);
});
afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

const events: ApiTurnEvent[] = [
  { owner: 1, kind: "starvation", text: "Food shortage on Alnitak II: 12 population lost", planet_id: 2, star_id: 1 },
  { owner: 1, kind: "research", text: "Research completed: Nuclear Drive" },
  { owner: 1, kind: "building", text: "Automated Factory completed at Alnitak II", planet_id: 2, star_id: 1 },
];

function Harness(props: { events: ApiTurnEvent[]; onClose?: () => void; onModeChange?: (mode: TurnReportMode) => void }) {
  const [open, setOpen] = createSignal(false);
  return <Router root={() => <>
    <button onClick={() => setOpen(true)}>End Turn</button>
    <Show when={open()}>
      <TurnReport turn={12.4} events={props.events} onClose={() => { props.onClose?.(); setOpen(false); }}
        mode="popup" onModeChange={(mode) => props.onModeChange?.(mode)} />
    </Show>
  </>}><Route path="*" component={() => null} /></Router>;
}

async function openReport(view: ReturnType<typeof render>) {
  const opener = view.getByText("End Turn");
  opener.focus();
  fireEvent.click(opener);
  await Promise.resolve();
  return opener;
}

describe("TurnReport", () => {
  it("opens as a modal dialog with the turn in the header and focuses OK", async () => {
    const view = render(() => <Harness events={events} />);
    await openReport(view);
    const dialog = screen.getByRole("dialog");
    expect(dialog.getAttribute("aria-modal")).toBe("true");
    expect(dialog.closest(".research-modal-backdrop")).toBeTruthy();
    expect(screen.getByText("Stardate 12.4 complete")).toBeTruthy();
    expect(document.activeElement).toBe(screen.getByRole("button", { name: "OK" }));
  });

  it("groups events under category headings when there is more than one category", async () => {
    const view = render(() => <Harness events={events} />);
    await openReport(view);
    expect(screen.getAllByRole("heading", { level: 3 }).map((h) => h.textContent)).toEqual(["Population", "Construction", "Research"]);
    expect(screen.getByText("Food shortage on Alnitak II: 12 population lost").closest("a")?.getAttribute("href")).toBe("/stars/1/planets/2");
  });

  it("omits category headings for a single category", async () => {
    const view = render(() => <Harness events={[events[0]]} />);
    await openReport(view);
    expect(screen.queryAllByRole("heading", { level: 3 })).toHaveLength(0);
  });

  it.each(["Escape", "Enter"])("dismisses with %s and restores focus", async (key) => {
    const onClose = vi.fn();
    const view = render(() => <Harness events={events} onClose={onClose} />);
    const opener = await openReport(view);
    fireEvent.keyDown(screen.getByRole("dialog"), { key });
    expect(onClose).toHaveBeenCalledOnce();
    expect(screen.queryByRole("dialog")).toBeNull();
    expect(document.activeElement).toBe(opener);
  });

  it("dismisses with the visible OK and close buttons", async () => {
    const onClose = vi.fn();
    const view = render(() => <Harness events={events} onClose={onClose} />);
    await openReport(view);
    fireEvent.click(screen.getByRole("button", { name: "OK" }));
    await openReport(view);
    fireEvent.click(screen.getByRole("button", { name: "Close turn report" }));
    expect(onClose).toHaveBeenCalledTimes(2);
  });

  it("shows an empty state", async () => {
    const view = render(() => <Harness events={[]} />);
    await openReport(view);
    expect(screen.getByText("No notable events this turn.")).toBeTruthy();
  });
});

describe("turn report modes", () => {
  it("offers Pop up / Dock left / Hide from inside the popup", async () => {
    const onModeChange = vi.fn();
    const view = render(() => <Harness events={events} onModeChange={onModeChange} />);
    await openReport(view);
    expect(screen.getByRole("radio", { name: "Pop up" }).getAttribute("aria-checked")).toBe("true");
    fireEvent.click(screen.getByRole("radio", { name: "Hide" }));
    fireEvent.click(screen.getByRole("radio", { name: "Dock left" }));
    expect(onModeChange.mock.calls).toEqual([["off"], ["docked"]]);
  });

  it("renders the dock inline (not as a dialog) and updates it in place for a new turn", () => {
    const [turn, setTurn] = createSignal(3);
    const [list, setList] = createSignal<ApiTurnEvent[]>([events[0]]);
    const view = render(() => <Router root={() =>
      <TurnReportDock turn={turn()} events={list()} mode="docked" onModeChange={() => {}} />}>
      <Route path="*" component={() => null} />
    </Router>);
    expect(view.queryByRole("dialog")).toBeNull();
    expect(view.getByText("Stardate 3.0")).toBeTruthy();
    expect(view.getByRole("radio", { name: "Dock left" }).getAttribute("aria-checked")).toBe("true");
    setTurn(4);
    setList([events[1]]);
    expect(view.getByText("Stardate 4.0")).toBeTruthy();
    expect(view.getByText("Research completed: Nuclear Drive")).toBeTruthy();
    expect(view.queryByText("Food shortage on Alnitak II: 12 population lost")).toBeNull();
  });

  it("persists the mode and falls back to popup for missing or unknown values", () => {
    const store = new Map<string, string>();
    const storage = { getItem: (k: string) => store.get(k) ?? null, setItem: (k: string, v: string) => void store.set(k, v) };
    expect(readTurnReportMode(storage)).toBe("popup");
    saveTurnReportMode("docked", storage);
    expect(readTurnReportMode(storage)).toBe("docked");
    store.set("client-solid.turnReportMode", "sideways");
    expect(readTurnReportMode(storage)).toBe("popup");
  });
});

describe("turn report helpers", () => {
  it("drops only the plain Turn complete toast while docked", () => {
    expect(visibleInfoBanner("Turn complete", "", false, true)).toBe("");
    expect(visibleInfoBanner("Waiting for 1 player", "", false, true)).toBe("Waiting for 1 player");
  });

  it("hides the info toast while the report is open", () => {
    expect(visibleInfoBanner("Turn complete", "", true)).toBe("");
    expect(visibleInfoBanner("Turn complete", "", false)).toBe("Turn complete");
    expect(visibleInfoBanner("Turn complete", "boom", false)).toBe("");
  });

  it("puts unknown kinds in Other", () => {
    expect(groupTurnEvents([{ owner: 1, kind: "mystery", text: "?" }]).map((g) => g.label)).toEqual(["Other"]);
  });
});
