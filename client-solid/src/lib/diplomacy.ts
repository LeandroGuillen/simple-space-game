import type { ApiTurnEvent } from "./types/api";
import type { GameViewModel } from "./types/domain";

export const FIRST_CONTACT_EVENT = "first_contact";
export const DECLARED_WAR_EVENT = "declared_war";

/** Label for an empire the viewer has not met (never its real name or id). */
export const UNKNOWN_EMPIRE = "Unknown empire";

export type DiplomacyStatus = "At war" | "Peace" | "Eliminated";

export type DiplomacyContactRow = {
  playerId: number;
  name: string;
  color: string;
  raceName: string;
  government: string;
  eliminated: boolean;
  atWar: boolean;
  status: DiplomacyStatus;
  relation: number;
  metTurn: number;
  starId: number;
  starName: string;
};

/**
 * Rows for the empires the viewing player has met, in first-contact order.
 * Uncontacted empires are never listed (the server also omits them).
 */
export function diplomacyContacts(game: GameViewModel | null | undefined, playerId: number): DiplomacyContactRow[] {
  const me = game?.players.find((player) => player.id === playerId);
  if (!game || !me) return [];
  const starNames = new Map(game.galaxy.stars.map((star) => [star.id, star.name]));
  return (me.contacts ?? []).map((contact) => {
    const rival = game.players.find((player) => player.id === contact.playerId);
    const eliminated = rival?.eliminated ?? false;
    const atWar = !eliminated && (contact.atWar ?? false);
    return {
      playerId: contact.playerId,
      name: rival?.name || UNKNOWN_EMPIRE,
      color: rival?.color ?? "var(--text-mute)",
      raceName: rival?.raceName ?? "Unknown",
      government: rival?.raceGovernment ?? "Unknown",
      eliminated,
      atWar,
      status: eliminated ? "Eliminated" : atWar ? "At war" : "Peace",
      relation: contact.relation ?? 0,
      metTurn: contact.metTurn,
      starId: contact.starId,
      starName: (contact.starId && starNames.get(contact.starId)) || "",
    };
  });
}

/**
 * Whether the viewer can queue a declaration of war on this contact: not
 * eliminated, not already at war, and not already queued this turn.
 */
export function canDeclareWar(row: DiplomacyContactRow, queuedPlayerIds: readonly number[]): boolean {
  return !row.eliminated && !row.atWar && !queuedPlayerIds.includes(row.playerId);
}

/**
 * Player-facing name for an owner id. Owners missing from the visible player
 * list are empires the viewer has not met, shown neutrally.
 */
export function empireName(players: readonly { id: number; name: string }[] | null | undefined, ownerId: number): string {
  return (players ?? []).find((player) => player.id === ownerId)?.name || UNKNOWN_EMPIRE;
}

/** Number of empires first met in the most recently resolved turn. */
export function newContactsThisTurn(events: ApiTurnEvent[] | null | undefined, playerId: number): number {
  return (events ?? []).filter((event) => event.owner === playerId && event.kind === FIRST_CONTACT_EVENT).length;
}
