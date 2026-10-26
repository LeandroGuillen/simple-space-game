import { A } from "@solidjs/router";
import { createMemo, createSignal, For } from "solid-js";

import { ConstructionPanel } from "../components/ConstructionPanel";
import type { PlanetViewModel } from "../lib/types/domain";
import { formatInteger, formatSignedBc, getBuyProductionCost } from "../lib/utils";
import { useAppContext } from "../state/app-context";

const TRADE_GOODS_NAME = "Trade Goods";

function planetBuyContext(planet: PlanetViewModel | undefined) {
  if (
    !planet ||
    !planet.isBuilding ||
    !planet.currentProject ||
    planet.currentProject.name === TRADE_GOODS_NAME ||
    planet.buildBucket <= 0
  ) {
    return null;
  }
  const baseCost = planet.currentProject.cost;
  if (baseCost <= 0) {
    return null;
  }
  return {
    cost: getBuyProductionCost(
      baseCost,
      Math.max(0, planet.buildBucket - planet.industryStack),
    ),
    projectName: planet.currentProject.name,
  };
}

type SortKey =
  | "name"
  | "starName"
  | "population"
  | "industry"
  | "income"
  | "morale";

export function ColoniesPage() {
  const {
    colonies,
    state,
    currentPlayer,
    setPlanetAutobuild,
    buyProduction,
    cancelBuyProduction,
  } = useAppContext();
  const [filter, setFilter] = createSignal("");
  const [sortKey, setSortKey] = createSignal<SortKey>("name");
  const [sortAsc, setSortAsc] = createSignal(true);
  const planetForRow = (planetId: number) =>
    state.game?.galaxy.stars
      .flatMap((star) => star.planets)
      .find((planet) => planet.id === planetId);
  const populationUnits = (population: number) => Math.floor(population / 1000);

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
  const rowById = createMemo(
    () => new Map(rows().map((row) => [row.id, row])),
  );
  const rowIds = createMemo(() => rows().map((row) => row.id));

  function toggleSort(nextKey: SortKey) {
    if (sortKey() === nextKey) {
      setSortAsc(!sortAsc());
      return;
    }
    setSortKey(nextKey);
    setSortAsc(true);
  }
  const moraleIcons = (morale: number) =>
    Array.from({ length: Math.max(0, Math.floor(morale * 10 + 0.000001)) });

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
                  onClick={() => toggleSort("morale")}
                >
                  Morale
                </button>
              </th>
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
              <th class="colonies-project-heading">Project</th>
              <th class="colonies-autobuild-heading">Autobuild</th>
            </tr>
          </thead>
          <tbody>
            <For each={rowIds()}>
              {(rowId) => {
                const row = () => rowById().get(rowId);
                const planet = () => planetForRow(rowId);
                return (
              <tr>
                <td>
                  <A href={`/stars/${row()?.starId}/planets/${rowId}`}>
                    {row()?.name}
                  </A>
                </td>
                <td>{row()?.starName}</td>
                <td>
                  {populationUnits(row()?.population ?? 0)}/{populationUnits(row()?.maxPopulation ?? 0)}
                </td>
                <td>{row()?.farmers}</td>
                <td>{row()?.workers}</td>
                <td>{row()?.scientists}</td>
                <td>
                  {moraleIcons(row()?.morale ?? 0).map(() => ":)").join(" ")}
                </td>
                <td>{row()?.industry}</td>
                <td classList={{ "numeric-negative": (row()?.income ?? 0) < 0 }}>
                  {formatSignedBc(row()?.income ?? 0)}
                </td>
                <td class="colonies-project-cell">
                  {planet() ? (
                      <ConstructionPanel planet={planet()!} compact />
                    ) : (
                      row()?.currentProject
                    )}
                </td>
                <td class="colonies-autobuild-cell">
                  <div class="colonies-autobuild-controls">
                    <label class="construction-autobuild colonies-autobuild-toggle">
                      <span class="sr-only">Autobuild {row()?.name}</span>
                      <span class="construction-autobuild-switch">
                        <input
                          type="checkbox"
                          checked={planet()?.autobuild ?? false}
                          aria-label={`Autobuild ${row()?.name}`}
                          onChange={(event) =>
                            setPlanetAutobuild(
                              rowId,
                              event.currentTarget.checked,
                            )
                          }
                        />
                        <span aria-hidden="true" />
                      </span>
                    </label>
                    {(() => {
                      const buy = () => planetBuyContext(planet());
                      const isPending = () =>
                        state.pendingCommands.buyProduction.some(
                          (command) => command.planetId === rowId,
                        );
                      const queuedBuyTotal = () =>
                        state.pendingCommands.buyProduction.reduce(
                          (total, command) => {
                            if (command.planetId === rowId) return total;
                            const context = planetBuyContext(
                              planetForRow(command.planetId),
                            );
                            return total + (context?.cost ?? 0);
                          },
                          0,
                        );
                      const remainingBudget = () =>
                        Math.max(
                          0,
                          (currentPlayer()?.treasury ?? 0) - queuedBuyTotal(),
                        );
                      const canAfford = () =>
                        Boolean(buy()) && remainingBudget() >= (buy()?.cost ?? 0);
                      const onClick = () => {
                        const ctx = buy();
                        if (!ctx) return;
                        if (isPending()) {
                          cancelBuyProduction(rowId);
                          return;
                        }
                        if (!canAfford()) return;
                        buyProduction(rowId);
                      };
                      const title = () => {
                        const ctx = buy();
                        if (!ctx) return "Nothing to buy";
                        if (isPending()) return "Buy queued · click to cancel";
                        if (!canAfford())
                          return `Need ${formatInteger(ctx.cost)} BC (available after queued buys: ${formatInteger(remainingBudget())} BC)`;
                        return `Pay ${formatInteger(ctx.cost)} BC to rush ${ctx.projectName}`;
                      };
                      return (
                        <button
                          type="button"
                          class="colonies-buy-button"
                          classList={{ "is-pending": isPending() }}
                          onClick={onClick}
                          disabled={
                            state.isBusy ||
                            !buy() ||
                            (!isPending() && !canAfford())
                          }
                          title={title()}
                          aria-label={title()}
                        >
                          {isPending() ? "Buying" : "Buy"}
                        </button>
                      );
                    })()}
                  </div>
                </td>
              </tr>
                );
              }}
            </For>
          </tbody>
        </table>
      </div>
    </section>
  );
}
