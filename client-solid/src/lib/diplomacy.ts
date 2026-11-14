import type { ApiTurnEvent } from "./types/api";
import type { GameViewModel } from "./types/domain";

export const FIRST_CONTACT_EVENT = "first_contact";

export type DiplomacyContactRow = {
  playerId: number;
  name: string;
  color: string;
  raceName: string;
  government: string;
  eliminated: boolean;
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
    return {
      playerId: contact.playerId,
      name: rival?.name ?? `Empire ${contact.playerId}`,
      color: rival?.color ?? "var(--text-mute)",
      raceName: rival?.raceName ?? "Unknown",
      government: rival?.raceGovernment ?? "Unknown",
      eliminated: rival?.eliminated ?? false,
      relation: contact.relation ?? 0,
      metTurn: contact.metTurn,
      starId: contact.starId,
      starName: (contact.starId && starNames.get(contact.starId)) || "",
    };
  });
}

/** Number of empires first met in the most recently resolved turn. */
export function newContactsThisTurn(events: ApiTurnEvent[] | null | undefined, playerId: number): number {
  return (events ?? []).filter((event) => event.owner === playerId && event.kind === FIRST_CONTACT_EVENT).length;
}
