import { Router, Route } from "@solidjs/router";
import { cleanup, fireEvent, render, screen } from "@solidjs/testing-library";
import { createSignal, Show } from "solid-js";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { TurnReport } from "../../components/TurnReport";
import { groupTurnEvents, visibleInfoBanner } from "../turn-report";
import type { ApiTurnEvent } from "../types/api";

beforeEach(() => {
  vi.spyOn(HTMLElement.prototype, "getClientRects").mockReturnValue([{}] as unknown as DOMRectList);
});
afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

const events: ApiTurnEvent[] = [
  { owner: 1, kind: "population", text: "Population grew by 40 on Alnitak II", planet_id: 2, star_id: 1 },
  { owner: 1, kind: "research", text: "Research completed: Nuclear Drive" },
  { owner: 1, kind: "building", text: "Automated Factory completed at Alnitak II", planet_id: 2, star_id: 1 },
];

function Harness(props: { events: ApiTurnEvent[]; onClose?: () => void }) {
  const [open, setOpen] = createSignal(false);
  return <Router root={() => <>
    <button onClick={() => setOpen(true)}>End Turn</button>
    <Show when={open()}>
      <TurnReport turn={12.4} events={props.events} onClose={() => { props.onClose?.(); setOpen(false); }} />
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
    expect(screen.getByText("Population grew by 40 on Alnitak II").closest("a")?.getAttribute("href")).toBe("/stars/1/planets/2");
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

describe("turn report helpers", () => {
  it("hides the info toast while the report is open", () => {
    expect(visibleInfoBanner("Turn complete", "", true)).toBe("");
    expect(visibleInfoBanner("Turn complete", "", false)).toBe("Turn complete");
    expect(visibleInfoBanner("Turn complete", "boom", false)).toBe("");
  });

  it("puts unknown kinds in Other", () => {
    expect(groupTurnEvents([{ owner: 1, kind: "mystery", text: "?" }]).map((g) => g.label)).toEqual(["Other"]);
  });
});
