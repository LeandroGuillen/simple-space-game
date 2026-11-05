import type { FleetViewModel, GameViewModel } from "./types/domain";
import { isPlanetSuitableForColonization } from "./utils";

export type ColonizePrompt = {
  fleetId: number;
  starId: number;
  automatic: boolean;
};

function hasColonyShip(fleet: FleetViewModel) {
  return fleet.ships.some((ship) => ship.type === "colony_ship");
}

function isOrbitingColonyShipFleet(fleet: FleetViewModel) {
  return (
    (fleet.turnsRemaining ?? 0) === 0 &&
    !fleet.destinationStarId &&
    hasColonyShip(fleet)
  );
}

function wasTravelling(fleet: FleetViewModel) {
  return (fleet.turnsRemaining ?? 0) > 0 || Boolean(fleet.destinationStarId);
}

export function hasEligibleColonizationPlanet(game: GameViewModel, starId: number) {
  return game.galaxy.stars.some(
    (star) => star.id === starId && star.planets.some(isPlanetSuitableForColonization),
  );
}

/** Find only arrivals observed while advancing a turn, never parked ships on load. */
export function colonyShipArrivalPrompts(
  previousGame: GameViewModel | null,
  nextGame: GameViewModel,
  playerId: number,
): ColonizePrompt[] {
  if (!previousGame) return [];

  const prompts = new Map<number, ColonizePrompt>();
  const addPrompt = (fleet: FleetViewModel | undefined) => {
    if (
      !fleet ||
      fleet.ownerId !== playerId ||
      !isOrbitingColonyShipFleet(fleet) ||
      !hasEligibleColonizationPlanet(nextGame, fleet.starId)
    ) {
      return;
    }
    if (!prompts.has(fleet.id)) {
      prompts.set(fleet.id, { fleetId: fleet.id, starId: fleet.starId, automatic: true });
    }
  };

  for (const event of nextGame.turnEvents ?? []) {
    if (event.owner !== playerId || event.kind !== "arrival" || event.star_id == null) {
      continue;
    }
    const previousFleet = event.fleet_id == null
      ? undefined
      : previousGame.fleets.find((fleet) => fleet.id === event.fleet_id);
    const eventFleet = event.fleet_id == null
      ? undefined
      : nextGame.fleets.find((fleet) => fleet.id === event.fleet_id);

    // A non-colony arrival must not prompt merely because it merged with a
    // colony ship already in orbit. If the arriving fleet is unknown, only a
    // surviving event fleet provides enough ship information to be certain.
    if (previousFleet) {
      if (previousFleet.ownerId !== playerId || !hasColonyShip(previousFleet)) continue;
    } else if (!eventFleet || !hasColonyShip(eventFleet)) {
      continue;
    }

    // The event fleet may have merged away. Resolve the surviving orbit fleet
    // by owner and star, which is also the fleet that must receive the order.
    addPrompt(
      eventFleet && isOrbitingColonyShipFleet(eventFleet)
        ? eventFleet
        : nextGame.fleets.find(
            (fleet) => fleet.ownerId === playerId && fleet.starId === event.star_id,
          ),
    );
  }

  // Fallback for servers or fixtures without arrival events. Requiring a
  // previously travelling fleet avoids prompting for newly built or parked ships.
  for (const fleet of nextGame.fleets) {
    if (fleet.ownerId !== playerId || !isOrbitingColonyShipFleet(fleet)) continue;
    const previousFleet = previousGame.fleets.find((candidate) => candidate.id === fleet.id);
    if (previousFleet && hasColonyShip(previousFleet) && wasTravelling(previousFleet)) {
      addPrompt(fleet);
    }
  }

  return [...prompts.values()];
}
