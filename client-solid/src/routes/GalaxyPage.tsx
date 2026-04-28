import { A } from "@solidjs/router";
import { createMemo, createSignal, Show } from "solid-js";

import { GalaxyMap } from "../components/GalaxyMap";
import { PlanetVisual } from "../components/PlanetVisual";
import {
  findStar,
  labelForEnvironment,
  labelForGravity,
  labelForMinerals,
  labelForSize,
} from "../lib/utils";
import { useAppContext } from "../state/app-context";

export function GalaxyPage() {
  const { state, exploreStar } = useAppContext();
  const [selectedStarId, setSelectedStarId] = createSignal<number>();

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
            onSelect={(starId) => setSelectedStarId(starId)}
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

                <Show when={!star().explored}>
                  <button
                    class="primary-button"
                    onClick={() => void exploreStar(star().id)}
                  >
                    Explore System
                  </button>
                </Show>
              </>
            )}
          </Show>
        </section>
      </section>
    </Show>
  );
}
