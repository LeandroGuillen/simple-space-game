import { useParams } from "@solidjs/router";
import { createMemo, onMount, Show } from "solid-js";

import {
  findPlanet,
  labelForEnvironment,
  labelForGravity,
  labelForMinerals,
  labelForSize,
} from "../lib/utils";
import { useAppContext } from "../state/app-context";

export function PlanetPage() {
  const params = useParams();
  const { state, refreshGame, changeBuilding } = useAppContext();

  onMount(() => {
    if (state.session.selectedGameId) {
      void refreshGame();
    }
  });

  const planet = createMemo(() =>
    findPlanet(state.game, Number(params.planetId)),
  );
  const buildings = createMemo(
    () => state.game?.resources.buildingCollection.buildings ?? [],
  );

  return (
    <Show
      when={planet()}
      fallback={<section class="panel">Planet not found.</section>}
    >
      {(currentPlanet) => (
        <section class="page-grid">
          <section class="panel feature-panel">
            <p class="eyebrow">Planet Management</p>
            <h2>{currentPlanet().name}</h2>
            <p class="lede">
              {labelForEnvironment(currentPlanet().environment)} ·{" "}
              {labelForSize(currentPlanet().size)} ·{" "}
              {labelForGravity(currentPlanet().gravity)} gravity ·{" "}
              {labelForMinerals(currentPlanet().mineralClass)}
            </p>

            <div class="metric-grid">
              <article class="metric-card">
                <span>Population</span>
                <strong>
                  {currentPlanet().population}/{currentPlanet().maxPopulation}
                </strong>
              </article>
              <article class="metric-card">
                <span>Workers</span>
                <strong>{currentPlanet().workers}</strong>
              </article>
              <article class="metric-card">
                <span>Industry Stack</span>
                <strong>{currentPlanet().industryStack}</strong>
              </article>
              <article class="metric-card">
                <span>Build Bucket</span>
                <strong>{currentPlanet().buildBucket}</strong>
              </article>
            </div>
          </section>

          <section class="panel">
            <p class="eyebrow">Command Queue</p>
            <div class="stack-form">
              <Show
                when={currentPlanet().owner === 1}
                fallback={<p>This world is not yet under your control.</p>}
              >
                <label class="field">
                  <span>Current project</span>
                  <select
                    value={currentPlanet().currentProject?.id ?? ""}
                    onChange={(event) =>
                      void changeBuilding(
                        currentPlanet().id,
                        Number(event.currentTarget.value),
                      )
                    }
                  >
                    <option value="">Select building…</option>
                    {buildings().map((building) => (
                      <option value={building.id}>{building.name}</option>
                    ))}
                  </select>
                </label>
              </Show>
            </div>

            <div class="list-block">
              <p class="eyebrow">Population Allocation</p>
              <ul class="compact-list">
                <li>
                  <strong>Farmers</strong>
                  <span>{currentPlanet().farmers}</span>
                </li>
                <li>
                  <strong>Workers</strong>
                  <span>{currentPlanet().workers}</span>
                </li>
                <li>
                  <strong>Scientists</strong>
                  <span>{currentPlanet().scientists}</span>
                </li>
                <li>
                  <strong>Marines</strong>
                  <span>{currentPlanet().marines}</span>
                </li>
              </ul>
            </div>

            <div class="list-block">
              <p class="eyebrow">Build Queue</p>
              <ul class="compact-list">
                {currentPlanet().buildQueue.map((building) => (
                  <li>
                    <strong>{building.name}</strong>
                    <span>{building.cost} cost</span>
                  </li>
                ))}
                <Show when={currentPlanet().buildQueue.length === 0}>
                  <li>No queued projects.</li>
                </Show>
              </ul>
            </div>

            <div class="list-block">
              <p class="eyebrow">Installed Buildings</p>
              <ul class="compact-list">
                {currentPlanet().buildings.map((building) => (
                  <li>
                    <strong>{building.name}</strong>
                    <span>{building.maintenance} maintenance</span>
                  </li>
                ))}
                <Show when={currentPlanet().buildings.length === 0}>
                  <li>No installed structures yet.</li>
                </Show>
              </ul>
            </div>
          </section>
        </section>
      )}
    </Show>
  );
}
