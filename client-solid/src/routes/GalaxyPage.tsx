import { A } from "@solidjs/router";
import { createMemo, createSignal, Show } from "solid-js";

import { FleetOrbitModal } from "../components/FleetOrbitModal";
import { GalaxyMap } from "../components/GalaxyMap";
import { PlanetVisual } from "../components/PlanetVisual";
import {
  calculateFleetTravelTurnsFromCoordinates,
  findFleetInOrbit,
  findStar,
  labelForEnvironment,
  labelForGravity,
  labelForMinerals,
  labelForSize,
} from "../lib/utils";
import { useAppContext } from "../state/app-context";

export function GalaxyPage() {
  const { state, currentPlayer, addFleetMoveCommand } = useAppContext();
  const [selectedStarId, setSelectedStarId] = createSignal<number>();
  const [fleetModalFleetId, setFleetModalFleetId] = createSignal<number | null>(null);
  const [travelMode, setTravelMode] = createSignal<{ fleetId: number; originStarId: number; shipIndexes: number[] } | null>(null);
  const [snappedStarId, setSnappedStarId] = createSignal<number | null>(null);

  const stars = createMemo(() => state.game?.galaxy.stars ?? []);
  const selectedStar = createMemo(() => {
    const firstId = stars()[0]?.id;
    const chosenId = selectedStarId() ?? firstId;
    return chosenId ? findStar(state.game, chosenId) : null;
  });
  const playerNameById = createMemo(
    () =>
      new Map(
        (state.game?.players ?? []).map((player) => [player.id, player.name]),
      ),
  );
  const playerColorById = createMemo(
    () =>
      new Map(
        (state.game?.players ?? []).map((player) => [player.id, player.color]),
      ),
  );
  const ownedFleets = createMemo(() =>
    (state.game?.fleets ?? [])
      .filter((fleet) => fleet.ownerId === (currentPlayer()?.id ?? 0)),
  );
  const selectedStarFleet = createMemo(() => {
    const starId = selectedStar()?.id;
    if (!starId) {
      return null;
    }
    return findFleetInOrbit(state.game, currentPlayer()?.id ?? 0, starId);
  });
  const modalFleet = createMemo(() =>
    ownedFleets().find((fleet) => fleet.id === fleetModalFleetId()) ?? null,
  );
  const pendingFleetRoutes = createMemo(() =>
    state.pendingCommands.fleetMove.map((command) => {
      const fleet = state.game?.fleets.find((item) => item.id === command.fleet_id);
      if (!fleet) return null;
      const to = findStar(state.game, command.destination_star_id);
      if (!to) return null;
      return {
        fromX: fleet.x,
        fromY: fleet.y,
        toX: to.x,
        toY: to.y,
        eta: calculateFleetTravelTurnsFromCoordinates(fleet.x, fleet.y, to.x, to.y),
      };
    }).filter((route): route is { fromX: number; fromY: number; toX: number; toY: number; eta: number } => Boolean(route)),
  );
  const activeFleetRoutes = createMemo(() =>
    (state.game?.fleets ?? [])
      .filter((fleet) => fleet.ownerId === (currentPlayer()?.id ?? 0) && fleet.destinationStarId)
      .map((fleet) => {
        const to = findStar(state.game, fleet.destinationStarId!);
        if (!to) {
          return null;
        }
        return {
          fromX: fleet.x,
          fromY: fleet.y,
          toX: to.x,
          toY: to.y,
          eta: fleet.turnsRemaining,
        };
      })
      .filter((route): route is { fromX: number; fromY: number; toX: number; toY: number; eta: number } => Boolean(route)),
  );
  const previewFleetRoute = createMemo(() => {
    const mode = travelMode();
    const toStarId = snappedStarId();
    if (!mode || !toStarId || mode.originStarId === toStarId) {
      return null;
    }
    const fleet = ownedFleets().find((item) => item.id === mode.fleetId);
    const to = findStar(state.game, toStarId);
    if (!fleet || !to) {
      return null;
    }
    return {
      fromX: fleet.x,
      fromY: fleet.y,
      toX: to.x,
      toY: to.y,
      eta: calculateFleetTravelTurnsFromCoordinates(fleet.x, fleet.y, to.x, to.y),
      preview: true,
    };
  });
  const fleetRouteLines = createMemo(() => {
    const routes = [...activeFleetRoutes(), ...pendingFleetRoutes()];
    if (previewFleetRoute()) {
      routes.push(previewFleetRoute()!);
    }
    return routes;
  });

  function exitFleetTravelMode() {
    setTravelMode(null);
    setSnappedStarId(null);
  }

  function handleStarSelect(starId: number) {
    const mode = travelMode();
    if (!mode) {
      setSelectedStarId(starId);
      return;
    }
    if (starId === mode.originStarId) {
      addFleetMoveCommand({
        fleet_id: mode.fleetId,
        destination_star_id: starId,
        ship_indexes: mode.shipIndexes,
      });
      exitFleetTravelMode();
      setSelectedStarId(starId);
      return;
    }
    addFleetMoveCommand({
      fleet_id: mode.fleetId,
      destination_star_id: starId,
      ship_indexes: mode.shipIndexes,
    });
    setSelectedStarId(starId);
    exitFleetTravelMode();
  }

  return (
    <Show
      when={state.game}
      fallback={
        <section class="panel">
          Select a game to render the command map.
        </section>
      }
    >
      <section class="page-grid galaxy-layout">
        <section class="panel map-panel">
          <div class="section-header">
            <p class="eyebrow">Galaxy Map</p>
          </div>
          <GalaxyMap
            stars={stars()}
            selectedStarId={selectedStar()?.id}
            onSelect={handleStarSelect}
            fleets={ownedFleets().map((fleet) => ({
              id: fleet.id,
              starId: fleet.starId,
              x: fleet.x,
              y: fleet.y,
              isMoving: fleet.turnsRemaining > 0,
            }))}
            onFleetClick={(fleetId) => setFleetModalFleetId(fleetId)}
            routeLines={fleetRouteLines()}
            onHoverSnappedStar={(starId) => {
              if (!travelMode()) {
                return;
              }
              setSnappedStarId(starId);
            }}
          />
        </section>

        <section class="panel inspector-panel">
          <Show
            when={selectedStar()}
            fallback={<p>Select a star to inspect the system.</p>}
          >
            {(star) => (
              <>
                <p class="eyebrow">System Inspector</p>
                <h2>{star().name}</h2>
                <Show when={selectedStarFleet()}>
                  <button class="fleet-orbit-link-inline fleet-orbit-link" type="button" onClick={() => setFleetModalFleetId(selectedStarFleet()?.id ?? null)}>
                    🚀 Fleet in Orbit ({selectedStarFleet()?.ships.length ?? 0})
                  </button>
                </Show>
                <Show when={travelMode()}>
                  <div class="fleet-travel-mode-banner">
                    <p class="eyebrow">
                      Fleet travel mode: pick a destination star, or the star this fleet is at to cancel a queued move or
                      stop planning
                    </p>
                    <button
                      class="secondary-button"
                      type="button"
                      onClick={() => {
                        const m = travelMode();
                        if (m) {
                          addFleetMoveCommand({
                            fleet_id: m.fleetId,
                            destination_star_id: m.originStarId,
                            ship_indexes: m.shipIndexes,
                          });
                        }
                        exitFleetTravelMode();
                      }}
                    >
                      Cancel
                    </button>
                  </div>
                </Show>

                <div class="planet-list">
                  <Show
                    when={star().explored}
                    fallback={
                      <article class="planet-card">
                        <div>
                          <p>No planet telemetry. Explore this system first.</p>
                        </div>
                      </article>
                    }
                  >
                    {star().planets.map((planet) => (
                      <article class="planet-card">
                        <div class="planet-card-visual">
                          <A href={`/stars/${star().id}/planets/${planet.id}`}>
                            <PlanetVisual
                              environment={planet.environment}
                              size="small"
                            />
                          </A>
                        </div>
                        <div class="planet-card-info">
                          <p class="planet-title">
                            <Show
                              when={planet.owner > 0}
                              fallback={<span>{planet.name}</span>}
                            >
                              <A
                                class="planet-name-link"
                                href={`/stars/${star().id}/planets/${planet.id}`}
                                style={{
                                  color:
                                    playerColorById().get(planet.owner) ??
                                    "var(--text)",
                                }}
                              >
                                {planet.name}
                              </A>
                            </Show>
                          </p>
                          <p>
                            {labelForEnvironment(planet.environment)} ·{" "}
                            {labelForSize(planet.size)} ·{" "}
                            {labelForMinerals(planet.mineralClass)}
                          </p>
                          <p>
                            Gravity: {labelForGravity(planet.gravity)}
                            <Show when={planet.owner > 0}>
                              {" "}
                              · Population {planet.population}/
                              {planet.maxPopulation}
                            </Show>
                          </p>
                          <p>
                            {planet.owner > 0
                              ? `Controlled by ${playerNameById().get(planet.owner) ?? `Player ${planet.owner}`}`
                              : "Uncontrolled"}
                          </p>
                        </div>
                      </article>
                    ))}
                  </Show>
                </div>

              </>
            )}
          </Show>
          <FleetOrbitModal
            starId={modalFleet()?.starId ?? null}
            fleetId={fleetModalFleetId()}
            isOpen={fleetModalFleetId() !== null}
            onClose={() => setFleetModalFleetId(null)}
            onMove={(fleetId, shipIndexes) => {
              const originStarId = modalFleet()?.starId ?? null;
              if (!originStarId) {
                return;
              }
              setFleetModalFleetId(null);
              setTravelMode({ fleetId, originStarId, shipIndexes });
            }}
          />
        </section>
      </section>
    </Show>
  );
}
