import { describe, expect, it } from "vitest";
import { fleetRoster } from "../fleet-roster";
import type { PendingCommands } from "../types/commands";
import type { FleetViewModel, GameViewModel, PlayerViewModel } from "../types/domain";
const ship = { type: "ship_design", name: "Warship", designName: "Frigate A", fuelRangeParsecs: 2, unlimitedFuelRange: false, fuelCell: "" };
const fleet = (patch: Partial<FleetViewModel>): FleetViewModel => ({ id: 1, ownerId: 1, starId: 1, x: 0, y: 0, originStarId: null, destinationStarId: null, totalTurns: 0, turnsRemaining: 0, ships: [ship, ship] as FleetViewModel["ships"], ...patch });
const player = { id: 1, available: { fuelCells: [] }, shipDesigns: [] } as unknown as PlayerViewModel;
const commands = (): PendingCommands => ({ build: [], movePopulation: [], colonize: [], artificialPlanet: [], fleetMove: [], invade: [], buyProduction: [] });
const game = (fleets: FleetViewModel[]) => ({ fleets, pendingColonyShips: {}, galaxy: { stars: [{ id: 1, name: "Sol", explored: true }, { id: 2, name: "Rigel", explored: true }] }, resources: { shipParts: { fuelCells: [] }, shipDesignCollection: { shipDesigns: [] } } }) as unknown as GameViewModel;
describe("fleetRoster", () => {
  it("filters rivals and maps orbit and transit without placing moving ships at origin", () => {
    const rows = fleetRoster(game([fleet({}), fleet({ id: 2, starId: 1, originStarId: 1, destinationStarId: 2, turnsRemaining: 3 }), fleet({ id: 3, ownerId: 2 })]), player, commands());
    expect(rows).toHaveLength(2);
    expect(rows[0]).toMatchObject({ location: "Sol", ships: "Frigate A ×2", fuel: "2 parsecs", status: "Idle" });
    expect(rows[1]).toMatchObject({ location: "In transit Sol → Rigel", destination: "Rigel · 3 turns", status: "Moving" });
  });
  it("recognizes pending actions and keeps unknown destinations private", () => {
    const pending = commands(); pending.colonize.push({ planet_id: 3, fleet_id: 1 }); pending.invade.push({ fleet_id: 2, target_planet_id: 5, ship_indexes: [0] }); pending.fleetMove.push({ fleet_id: 3, destination_star_id: 99, ship_indexes: [0] });
    const rows = fleetRoster(game([fleet({}), fleet({ id: 2 }), fleet({ id: 3 })]), player, pending);
    expect(rows.map((row) => row.status)).toEqual(["Pending colonize", "Pending invade", "Pending move"]);
    expect(rows[2].destination).toBe("Unknown system · queued");
  });
  it("uses a design fuel cell for design-only ships", () => {
    const designPlayer = { ...player, shipDesigns: [{ name: "Frigate A", fuelCell: { name: "Standard", autonomy: 4 } }] } as PlayerViewModel;
    const designOnly = { ...ship, fuelRangeParsecs: 0 };
    expect(fleetRoster(game([fleet({ ships: [designOnly] as FleetViewModel["ships"] })]), designPlayer, commands())[0].fuel).toBe("4 parsecs");
  });
});
