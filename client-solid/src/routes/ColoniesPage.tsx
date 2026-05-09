import { A } from "@solidjs/router";
import { createMemo, createSignal } from "solid-js";

import { formatSignedBc } from "../lib/utils";
import { useAppContext } from "../state/app-context";

type SortKey = "name" | "starName" | "population" | "industry" | "income";

export function ColoniesPage() {
  const { colonies, state, currentPlayer, changeBuilding } = useAppContext();
  const [filter, setFilter] = createSignal("");
  const [sortKey, setSortKey] = createSignal<SortKey>("name");
  const [sortAsc, setSortAsc] = createSignal(true);
  const buildingOptions = createMemo(
    () => {
      const availableBuildingNames = new Set(currentPlayer()?.available.buildings ?? []);
      return state.game?.resources.buildingCollection.buildings.filter((building) =>
        availableBuildingNames.has(building.name),
      ) ?? [];
    },
  );
  const planetForRow = (planetId: number) =>
    state.game?.galaxy.stars
      .flatMap((star) => star.planets)
      .find((planet) => planet.id === planetId);
  const uniqueBuildingsLockedElsewhere = (planetId: number) => {
    const playerId = currentPlayer()?.id;
    if (!playerId) return new Set<string>();
    const locked = new Set<string>();
    for (const star of state.game?.galaxy.stars ?? []) {
      for (const planet of star.planets) {
        if (planet.owner !== playerId || planet.id === planetId) {
          continue;
        }
        for (const building of planet.buildings) {
          if (building.unique) {
            locked.add(building.name);
          }
        }
        if (planet.currentProject?.unique) {
          locked.add(planet.currentProject.name);
        }
        for (const queued of planet.buildQueue) {
          if (queued.unique) {
            locked.add(queued.name);
          }
        }
      }
    }
    return locked;
  };
  const buildableOptions = (planetId: number) => {
    const builtBuildingNames = new Set(planetForRow(planetId)?.buildings.map((building) => building.name) ?? []);
    const lockedUniqueNames = uniqueBuildingsLockedElsewhere(planetId);
    return buildingOptions().filter(
      (building) =>
        !builtBuildingNames.has(building.name) &&
        (!building.unique || !lockedUniqueNames.has(building.name)),
    );
  };

  const rows = createMemo(() => {
    const needle = filter().trim().toLowerCase();
    const filtered = colonies().filter(
      (row) =>
        !needle ||
        row.name.toLowerCase().includes(needle) ||
        row.starName.toLowerCase().includes(needle),
    );
    const sorted = [...filtered].sort((left, right) => {
      const key = sortKey();
      const leftValue = left[key];
      const rightValue = right[key];
      const order = sortAsc() ? 1 : -1;
      if (typeof leftValue === "number" && typeof rightValue === "number") {
        return (leftValue - rightValue) * order;
      }
      return String(leftValue).localeCompare(String(rightValue)) * order;
    });
    return sorted;
  });

  const selectedProjectId = (planetId: number) => {
    const pendingBuild = state.pendingCommands.build.find((command) => command.planetId === planetId);
    if (pendingBuild) {
      return buildingOptions().find((building) => building.name === pendingBuild.building)?.id ?? "";
    }

    return (
      planetForRow(planetId)?.currentProject?.id ?? ""
    );
  };

  function toggleSort(nextKey: SortKey) {
    if (sortKey() === nextKey) {
      setSortAsc(!sortAsc());
      return;
    }
    setSortKey(nextKey);
    setSortAsc(true);
  }

  return (
    <section class="panel">
      <div class="section-header">
        <div>
          <p class="eyebrow">Colonies</p>
          <h2>Production ledger</h2>
        </div>
        <label class="field inline-field">
          <span>Filter</span>
          <input
            value={filter()}
            onInput={(event) => setFilter(event.currentTarget.value)}
          />
        </label>
      </div>

      <div class="table-frame">
        <table class="data-table">
          <thead>
            <tr>
              <th>
                <button class="sort-button" onClick={() => toggleSort("name")}>
                  Colony
                </button>
              </th>
              <th>
                <button
                  class="sort-button"
                  onClick={() => toggleSort("starName")}
                >
                  System
                </button>
              </th>
              <th>
                <button
                  class="sort-button"
                  onClick={() => toggleSort("population")}
                >
                  Population
                </button>
              </th>
              <th>Farmers</th>
              <th>Workers</th>
              <th>Scientists</th>
              <th>
                <button
                  class="sort-button"
                  onClick={() => toggleSort("industry")}
                >
                  Industry
                </button>
              </th>
              <th>
                <button
                  class="sort-button"
                  onClick={() => toggleSort("income")}
                >
                  Income
                </button>
              </th>
              <th>Project</th>
              <th>Action</th>
            </tr>
          </thead>
          <tbody>
            {rows().map((row) => (
              <tr>
                <td>{row.name}</td>
                <td>{row.starName}</td>
                <td>
                  {row.population}/{row.maxPopulation}
                </td>
                <td>{row.farmers}</td>
                <td>{row.workers}</td>
                <td>{row.scientists}</td>
                <td>{row.industry}</td>
                <td classList={{ "numeric-negative": row.income < 0 }}>
                  {formatSignedBc(row.income)}
                </td>
                <td>
                  <label class="sr-only" for={`project-${row.id}`}>
                    Set project for {row.name}
                  </label>
                  <select
                    id={`project-${row.id}`}
                    value={selectedProjectId(row.id)}
                    onChange={(event) =>
                      changeBuilding(row.id, Number(event.currentTarget.value))
                    }
                  >
                    <option value="">Select building...</option>
                    {buildableOptions(row.id).map((building) => (
                      <option value={building.id}>{building.name}</option>
                    ))}
                  </select>
                </td>
                <td>
                  <A
                    class="link-button secondary-button"
                    href={`/stars/${row.starId}/planets/${row.id}`}
                  >
                    Inspect
                  </A>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}
