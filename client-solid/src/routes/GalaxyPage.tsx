import { useNavigate } from "@solidjs/router";
import { createMemo, createSignal, onMount, Show } from "solid-js";

import { GalaxyMap } from "../components/GalaxyMap";
import {
  findStar,
  labelForEnvironment,
  starOwnershipSummary,
} from "../lib/utils";
import { useAppContext } from "../state/app-context";

export function GalaxyPage() {
  const navigate = useNavigate();
  const { state, refreshGame } = useAppContext();
  const [selectedStarId, setSelectedStarId] = createSignal<number>();

  onMount(() => {
    if (state.session.selectedGameId) {
      void refreshGame();
    }
  });

  const stars = createMemo(() => state.game?.galaxy.stars ?? []);
  const selectedStar = createMemo(() => {
    const firstId = stars()[0]?.id;
    const chosenId = selectedStarId() ?? firstId;
    return chosenId ? findStar(state.game, chosenId) : null;
  });

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
                <p>{starOwnershipSummary(star())}</p>
                <div class="metric-grid">
                  <article class="metric-card">
                    <span>Planets</span>
                    <strong>{star().planets.length}</strong>
                  </article>
                  <article class="metric-card">
                    <span>Best World</span>
                    <strong>
                      {labelForEnvironment(star().planets[0]?.environment ?? 0)}
                    </strong>
                  </article>
                </div>

                <ul class="compact-list">
                  {star().planets.map((planet) => (
                    <li>
                      <strong>{planet.name}</strong>
                      <span>
                        {planet.owner === 1 ? "Controlled" : "Unclaimed"}
                      </span>
                    </li>
                  ))}
                </ul>

                <button
                  class="primary-button"
                  onClick={() => navigate(`/stars/${star().id}`)}
                >
                  Open Star System
                </button>
              </>
            )}
          </Show>
        </section>
      </section>
    </Show>
  );
}
