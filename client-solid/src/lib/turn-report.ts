import type { ApiTurnEvent } from "./types/api";

const CATEGORIES: { label: string; kinds: string[] }[] = [
  { label: "Population", kinds: ["population", "starvation", "colony"] },
  { label: "Construction", kinds: ["building", "ship"] },
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

/** The info toast stays hidden while the turn report is open; the report header says the turn is complete. */
export function visibleInfoBanner(info: string, error: string, reportOpen: boolean): string {
  return error || reportOpen ? "" : info;
}
