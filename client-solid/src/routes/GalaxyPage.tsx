import { A } from "@solidjs/router";
import { createMemo, createSignal, Show } from "solid-js";

import { FleetInspectorPanel } from "../components/FleetInspectorPanel";
import { GalaxyMap } from "../components/GalaxyMap";
import { PlanetVisual } from "../components/PlanetVisual";
import {
  calculateFleetTravelTurnsFromCoordinates,
  findPlayerFleetsAtStar,
  findStar,
  labelForEnvironment,
  labelForGravity,
  labelForMinerals,
  labelForSize,
  sortFleetsForStarTabs,
} from "../lib/utils";
import { useAppContext } from "../state/app-context";

export function GalaxyPage() {
  const { state, currentPlayer, addFleetMoveCommand } = useAppContext();
  const [selectedStarId, setSelectedStarId] = createSignal<number>();
  const [selectedFleetStarId, setSelectedFleetStarId] = createSignal<number | null>(null);
  const [selectedFleetId, setSelectedFleetId] = createSignal<number | null>(null);
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
  const fleetsAtSelectedStar = createMemo(() => {
    const starId = selectedStar()?.id;
    if (!starId) {
      return [];
    }
    return sortFleetsForStarTabs(findPlayerFleetsAtStar(state.game, currentPlayer()?.id ?? 0, starId));
  });
  const fleetShipCountAtSelectedStar = createMemo(() =>
    fleetsAtSelectedStar().reduce((total, f) => total + f.ships.length, 0),
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
      setSelectedFleetStarId(null);
      setSelectedFleetId(null);
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

  function selectFleetForInspector(fleetId: number | null, starId: number | null) {
    setSelectedFleetStarId(starId);
    setSelectedFleetId(fleetId);
    if (starId) {
      setSelectedStarId(starId);
    }
  }

  function handleFleetSelectionChange(fleetId: number | null, shipIndexes: number[]) {
    if (fleetId == null || shipIndexes.length === 0) {
      if (travelMode()?.fleetId === fleetId) {
        exitFleetTravelMode();
      }
      return;
    }
    const f = ownedFleets().find((item) => item.id === fleetId);
    const originStarId = f?.starId ?? selectedFleetStarId();
    if (!originStarId) {
      return;
    }
    setTravelMode({ fleetId, originStarId, shipIndexes });
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
            onFleetClick={(fleetId) => {
              const f = ownedFleets().find((item) => item.id === fleetId);
              selectFleetForInspector(fleetId, f?.starId ?? null);
            }}
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
          <Show when={selectedFleetStarId()} fallback={
            <Show
              when={selectedStar()}
              fallback={<p>Select a star to inspect the system.</p>}
            >
              {(star) => (
                <>
                  <div class="inspector-header">
                    <div>
                      <p class="eyebrow">System Inspector</p>
                      <h2>{star().name}</h2>
                      <p class="inspector-subtitle">
                        {star().explored ? `${star().planets.length} planets catalogued` : "Unexplored system"}
                      </p>
                    </div>
                    <Show when={fleetsAtSelectedStar().length > 0}>
                      <button
                        class="secondary-button inspector-header-action"
                        type="button"
                        onClick={() => selectFleetForInspector(null, selectedStar()?.id ?? null)}
                      >
                        Fleets ({fleetShipCountAtSelectedStar()})
                      </button>
                    </Show>
                  </div>

                  <div class="planet-list">
                    <Show
                      when={star().explored}
                      fallback={
                        <article class="planet-card inspector-empty-card">
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
          }>
            {(fleetStarId) => (
              <FleetInspectorPanel
                starId={fleetStarId()}
                fleetId={selectedFleetId()}
                onViewSystem={(starId) => {
                  setSelectedStarId(starId);
                  setSelectedFleetStarId(null);
                  setSelectedFleetId(null);
                }}
                onSelectionChange={handleFleetSelectionChange}
                isMoveTargeting={travelMode() != null}
                onCancelMove={() => {
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
              />
            )}
          </Show>
        </section>
      </section>
    </Show>
  );
}
