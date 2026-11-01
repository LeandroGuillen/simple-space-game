import type { PendingCommands } from "./types/commands";
import type { FleetViewModel, GameViewModel, PlayerViewModel } from "./types/domain";
import { fleetSelectionFuelRange, formatFuelRange } from "./utils";

export type FleetRosterRow = {
  fleet: FleetViewModel;
  location: string;
  destination: string;
  ships: string;
  fuel: string;
  status: "Idle" | "Moving" | "Pending move" | "Pending colonize" | "Pending invade";
};

export function fleetRoster(game: GameViewModel | null, player: PlayerViewModel | null, commands: PendingCommands, preparing: readonly number[] = []): FleetRosterRow[] {
  if (!game || !player) return [];
  const starName = (id: number | null | undefined) => {
    const star = game.galaxy.stars.find((item) => item.id === id);
    return star?.explored && star.name ? star.name : "Unknown system";
  };
  return game.fleets.filter((fleet) => fleet.ownerId === player.id).map((fleet) => {
    const moving = Boolean(fleet.destinationStarId || fleet.turnsRemaining > 0);
    const colonize = preparing.includes(fleet.id) || (game.pendingColonyShips[player.id] ?? []).includes(fleet.id) || commands.colonize.some((command) => command.fleet_id === fleet.id);
    const invade = commands.invade.some((command) => command.fleet_id === fleet.id);
    const queuedMoves = commands.fleetMove.filter((command) => command.fleet_id === fleet.id);
    const status: FleetRosterRow["status"] = moving ? "Moving" : invade ? "Pending invade" : colonize ? "Pending colonize" : queuedMoves.length ? "Pending move" : "Idle";
    const breakdown = new Map<string, number>();
    for (const ship of fleet.ships) {
      const label = ship.designName || ship.name || ship.hullSize || (ship.type === "colony_ship" ? "Colony ship" : ship.type === "transport" ? "Transport" : "Ship");
      breakdown.set(label, (breakdown.get(label) ?? 0) + 1);
    }
    return {
      fleet,
      location: moving ? `In transit ${starName(fleet.originStarId || fleet.starId)} → ${starName(fleet.destinationStarId)}` : starName(fleet.starId),
      destination: moving ? `${starName(fleet.destinationStarId)} · ${fleet.turnsRemaining} turn${fleet.turnsRemaining === 1 ? "" : "s"}` : queuedMoves.length ? [...new Set(queuedMoves.map((move) => starName(move.destination_star_id)))].join(", ") + " · queued" : "—",
      ships: [...breakdown].map(([name, count]) => `${name} ×${count}`).join(", ") || "No ships",
      fuel: formatFuelRange(fleetSelectionFuelRange(fleet.ships, player, game.resources)),
      status,
    };
  });
}
