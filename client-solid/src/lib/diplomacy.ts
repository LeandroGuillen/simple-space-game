import type { ApiTurnEvent } from "./types/api";
import type { GameViewModel } from "./types/domain";

export const FIRST_CONTACT_EVENT = "first_contact";
export const DECLARED_WAR_EVENT = "declared_war";
/** A declaration of war that was valid at submit but could not be applied when the turn resolved. */
export const DECLARE_WAR_FAILED_EVENT = "declare_war_failed";

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
 * Whether viewer has a peace treaty in effect with otherId (which blocks
 * battles and invasions until war is declared again).
 */
export function hasPeaceTreaty(game: GameViewModel | null | undefined, viewerId: number, otherId: number): boolean {
  const me = game?.players.find((player) => player.id === viewerId);
  const contact = me?.contacts?.find((entry) => entry.playerId === otherId);
  return Boolean(contact?.peaceTreaty && !contact.atWar);
}

export type InvasionChoice<P> = {
  /** The rival colony an Invade order would target, or null when none may be invaded. */
  target: P | null;
  /** Why Invade is disabled (a peace treaty), or null when it is available. */
  blockedReason: string | null;
};

/**
 * Picks the rival colony at a star that Invade would target. Colonies of an
 * empire with a peace treaty are skipped; when they are the only rival
 * colonies, Invade is blocked with an explanation. Without a treaty Invade is
 * available even when not at war: the landing declares war automatically.
 */
export function invasionChoice<P extends { owner: number }>(
  game: GameViewModel | null | undefined,
  viewerId: number,
  planets: readonly P[],
): InvasionChoice<P> {
  const rivals = planets.filter((planet) => planet.owner !== 0 && planet.owner !== viewerId);
  const target = rivals.find((planet) => !hasPeaceTreaty(game, viewerId, planet.owner)) ?? null;
  if (target || rivals.length === 0) {
    return { target, blockedReason: null };
  }
  const name = empireName(game?.players, rivals[0]!.owner);
  return { target: null, blockedReason: `Peace treaty with ${name}: declare war before invading.` };
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

/**
 * Number of declarations of war against the viewer in the most recently
 * resolved turn: "X declared war on you", mutual declarations and wars
 * declared automatically when a fight or invasion started. The viewer's own
 * "You declared war on …" is not counted.
 */
export function incomingWarDeclarationsThisTurn(events: ApiTurnEvent[] | null | undefined, playerId: number): number {
  return (events ?? []).filter(
    (event) => event.owner === playerId && event.kind === DECLARED_WAR_EVENT && !event.text.startsWith("You declared war on "),
  ).length;
}

/** Diplomacy nav badge: new contacts plus incoming declarations of war this turn. */
export function diplomacyBadgeCount(events: ApiTurnEvent[] | null | undefined, playerId: number): number {
  return newContactsThisTurn(events, playerId) + incomingWarDeclarationsThisTurn(events, playerId);
}
