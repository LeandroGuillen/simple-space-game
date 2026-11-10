import type { ApiTurnEvent } from "./types/api";

const CATEGORIES: { label: string; kinds: string[] }[] = [
  { label: "Population", kinds: ["population", "starvation", "colony"] },
  { label: "Construction", kinds: ["building", "ship", "production_blocked"] },
  { label: "Research", kinds: ["research"] },
  { label: "Fleets & combat", kinds: ["arrival", "battle"] },
  { label: "Invasions", kinds: ["invasion"] },
];

export type TurnEventGroup = { label: string; events: ApiTurnEvent[] };

/** Groups events by category in a fixed order; unknown kinds go to "Other". */
export function groupTurnEvents(events: ApiTurnEvent[]): TurnEventGroup[] {
  const groups = [...CATEGORIES.map((category) => ({ ...category, events: [] as ApiTurnEvent[] })),
    { label: "Other", kinds: [], events: [] as ApiTurnEvent[] }];
  for (const event of events) {
    const group = groups.find((candidate) => candidate.kinds.includes(event.kind)) ?? groups.at(-1)!;
    group.events.push(event);
  }
  return groups.filter((group) => group.events.length > 0).map(({ label, events }) => ({ label, events }));
}

export function turnEventHref(event: ApiTurnEvent): string {
  if (event.kind === "research") return "/technology";
  if (event.kind === "invasion") return "/fleets";
  if (event.planet_id && event.star_id) return `/stars/${event.star_id}/planets/${event.planet_id}`;
  if (event.star_id) return `/galaxy?star=${event.star_id}`;
  return "/galaxy";
}

/**
 * The info toast stays hidden while the popup is open, and the plain
 * "Turn complete" toast is dropped while docked: the report header says it.
 */
export function visibleInfoBanner(info: string, error: string, reportOpen: boolean, docked = false): string {
  if (error || reportOpen) return "";
  return docked && info === "Turn complete" ? "" : info;
}

export type TurnReportMode = "popup" | "docked" | "off";

export const TURN_REPORT_MODES: { value: TurnReportMode; label: string; title: string }[] = [
  { value: "popup", label: "Pop up", title: "Open the report after every turn" },
  { value: "docked", label: "Dock left", title: "Keep the report docked on the left; it updates each turn" },
  { value: "off", label: "Hide", title: "Hide the report; use the Report button to look it up" },
];

const TURN_REPORT_MODE_KEY = "client-solid.turnReportMode";

export function readTurnReportMode(storage: Pick<Storage, "getItem"> | undefined = globalThis.localStorage): TurnReportMode {
  const stored = storage?.getItem(TURN_REPORT_MODE_KEY);
  return TURN_REPORT_MODES.some((mode) => mode.value === stored) ? (stored as TurnReportMode) : "popup";
}

export function saveTurnReportMode(mode: TurnReportMode, storage: Pick<Storage, "setItem"> | undefined = globalThis.localStorage) {
  storage?.setItem(TURN_REPORT_MODE_KEY, mode);
}
