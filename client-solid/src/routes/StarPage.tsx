import { A, useNavigate, useParams } from "@solidjs/router";
import { createMemo, onMount, Show } from "solid-js";

import {
  findStar,
  labelForEnvironment,
  labelForGravity,
  labelForMinerals,
  labelForSize,
} from "../lib/utils";
import { useAppContext } from "../state/app-context";

export function StarPage() {
  const params = useParams();
  const navigate = useNavigate();
  const { state, refreshGame, colonize, exploreStar } = useAppContext();

  onMount(() => {
    if (state.session.selectedGameId) {
      void refreshGame();
    }
  });

  const star = createMemo(() => findStar(state.game, Number(params.starId)));

  return (
    <Show
      when={star()}
      fallback={<section class="panel">Star system not found.</section>}
    >
      {(currentStar) => (
        <section class="page-grid">
          <section class="panel">
            <div class="section-header">
              <div>
                <p class="eyebrow">Star System</p>
                <h2>{currentStar().name}</h2>
              </div>
              <button
                class="secondary-button"
                onClick={() => navigate("/galaxy")}
              >
                Back to Galaxy
              </button>
            </div>

            <div class="planet-list">
              <Show
                when={currentStar().explored}
                fallback={
                  <article class="planet-card">
                    <div>
                      <p class="planet-title">System is unexplored</p>
                      <p>Planet details unlock after exploration.</p>
                    </div>
                    <div class="planet-actions">
                      <button
                        class="primary-button"
                        onClick={() => void exploreStar(currentStar().id)}
                      >
                        Explore System
                      </button>
                    </div>
                  </article>
                }
              >
                {currentStar().planets.map((planet) => (
                  <article class="planet-card">
                    <div>
                      <p class="planet-title">{planet.name}</p>
                      <p>
                        {labelForEnvironment(planet.environment)} ·{" "}
                        {labelForSize(planet.size)} ·{" "}
                        {labelForMinerals(planet.mineralClass)}
                      </p>
                      <p>
                        Gravity: {labelForGravity(planet.gravity)} · Population{" "}
                        {planet.population}/{planet.maxPopulation}
                      </p>
                    </div>
                    <div class="planet-actions">
                      <A
                        class="secondary-button link-button"
                        href={`/stars/${currentStar().id}/planets/${planet.id}`}
                      >
                        Manage
                      </A>
                      <Show when={planet.owner === 0}>
                        <button
                          class="secondary-button"
                          onClick={() => void colonize(planet.id)}
                        >
                          Colonize
                        </button>
                      </Show>
                    </div>
                  </article>
                ))}
              </Show>
            </div>
          </section>
        </section>
      )}
    </Show>
  );
}
