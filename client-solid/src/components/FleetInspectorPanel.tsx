import { For, Show, createEffect, createMemo, createSignal, on } from "solid-js";

import type { FleetShipViewModel } from "../lib/types/domain";
import {
  findPlayerFleetsAtStar,
  findStar,
  fleetSelectionFuelRange,
  fleetTabLabel,
  formatFuelRange,
  isPlanetSuitableForColonization,
  sortFleetsForStarTabs,
} from "../lib/utils";
import { useAppContext } from "../state/app-context";

type FleetStackShip = FleetShipViewModel & {
  originalIndex: number;
};

type FleetStack = {
  key: string;
  fleetId: number;
  label: string;
  ships: FleetStackShip[];
};

type FleetInspectorPanelProps = {
  starId: number | null;
  fleetId?: number | null;
  onViewSystem: (starId: number) => void;
  onSelectionChange: (fleetId: number | null, shipIndexes: number[]) => void;
  isMoveTargeting?: boolean;
  onCancelMove?: () => void;
};

export function FleetInspectorPanel(props: FleetInspectorPanelProps) {
  const { state, currentPlayer, beginColonyShipColonization, cancelColonyShipColonization, beginInvasion, cancelInvasion } = useAppContext();
  const fleetsAtStar = createMemo(() => {
    if (!props.starId) {
      return [];
    }
    return sortFleetsForStarTabs(
      findPlayerFleetsAtStar(state.game, currentPlayer()?.id ?? 0, props.starId),
    );
  });
  const system = createMemo(() => (props.starId ? findStar(state.game, props.starId) : null));
  const fleetStacks = createMemo<FleetStack[]>(() => {
    const stacks: FleetStack[] = [];
    for (const f of fleetsAtStar()) {
      const pendingForFleet = state.pendingCommands.fleetMove.filter((cmd) => cmd.fleet_id === f.id);
      const pendingByDestination = new Map<number, Set<number>>();
      for (const cmd of pendingForFleet) {
        const indexes = pendingByDestination.get(cmd.destination_star_id) ?? new Set<number>();
        for (const index of cmd.ship_indexes) {
          indexes.add(index);
        }
        pendingByDestination.set(cmd.destination_star_id, indexes);
      }
      const pendingIndexes = new Set<number>();
      for (const indexes of pendingByDestination.values()) {
        for (const index of indexes) {
          pendingIndexes.add(index);
        }
      }
      const orbitingShips = f.ships
        .map((ship, index) => ({ ...ship, originalIndex: index }))
        .filter((ship) => !pendingIndexes.has(ship.originalIndex));
      if (orbitingShips.length > 0) {
        stacks.push({
          key: `orbit-${f.id}`,
          fleetId: f.id,
          label: fleetTabLabel(state.game, f),
          ships: orbitingShips,
        });
      }
      for (const [destinationStarId, indexes] of pendingByDestination) {
        const dest = findStar(state.game, destinationStarId);
        const ships = [...indexes]
          .sort((a, b) => a - b)
          .map((index) => {
            const ship = f.ships[index];
            return ship ? { ...ship, originalIndex: index } : null;
          })
          .filter((ship): ship is FleetStackShip => Boolean(ship));
        if (ships.length > 0) {
          stacks.push({
            key: `pending-${f.id}-${destinationStarId}`,
            fleetId: f.id,
            label: dest && !dest.explored ? "Exploring a new system" : `Traveling to ${dest?.name ?? "Unknown system"}`,
            ships,
          });
        }
      }
    }
    return stacks;
  });
  const [activeStackKey, setActiveStackKey] = createSignal<string | null>(null);
  const activeStack = createMemo(() => {
    const stacks = fleetStacks();
    if (stacks.length === 0) {
      return null;
    }
    const key = activeStackKey();
    if (key != null) {
      const found = stacks.find((stack) => stack.key === key);
      if (found) {
        return found;
      }
    }
    return stacks[0];
  });
  const [selectedIndexes, setSelectedIndexes] = createSignal<number[]>([]);
  const allSelected = createMemo(() => {
    const stack = activeStack();
    return stack != null && stack.ships.length > 0 && selectedIndexes().length === stack.ships.length;
  });
  const canColonizeWithShip = createMemo(() => {
    const stack = activeStack();
    const star = system();
    return Boolean(
      stack &&
        star &&
        stack.key.startsWith("orbit-") &&
        stack.ships.some((ship) => ship.type === "colony_ship") &&
        star.planets.some(isPlanetSuitableForColonization),
    );
  });
  const isColonizing = createMemo(() =>
    state.pendingCommands.colonize.some((command) => command.fleet_id === activeStack()?.fleetId),
  );
  
  const canInvadeWithShip = createMemo(() => {
    const stack = activeStack();
    const star = system();
    const player = currentPlayer();
    return Boolean(
      stack &&
        star &&
        player &&
        stack.key.startsWith("orbit-") &&
        stack.ships.some((ship) => ship.type === "transport") &&
        star.planets.some((planet) => planet.owner !== player.id && planet.owner !== 0),
    );
  });
  
  const isInvading = createMemo(() =>
    state.pendingCommands.invade.some((command) => command.fleet_id === activeStack()?.fleetId),
  );

  const invasionTarget = createMemo(() => {
    const star = system();
    const player = currentPlayer();
    if (!star || !player) {
      return null;
    }
    return star.planets.find((planet) => planet.owner !== player.id && planet.owner !== 0) || null;
  });
  const selectedFuelRange = createMemo(() => {
    const stack = activeStack();
    if (!stack) {
      return null;
    }
    const selected = stack.ships.filter((ship) => selectedIndexes().includes(ship.originalIndex));
    return fleetSelectionFuelRange(selected, currentPlayer(), state.game?.resources);
  });

  createEffect(
    on(
      [() => props.fleetId, () => props.starId],
      ([fleetId]) => {
        const stacks = fleetStacks();
        if (fleetId != null) {
          setActiveStackKey(stacks.find((stack) => stack.fleetId === fleetId)?.key ?? null);
          return;
        }
        setActiveStackKey(null);
      },
    ),
  );

  createEffect(
    on(
      [() => activeStack()?.key, () => activeStack()?.ships.length],
      () => {
        setSelectedIndexes(activeStack()?.ships.map((ship) => ship.originalIndex) ?? []);
      },
    ),
  );

  createEffect(() => {
    const stack = activeStack();
    props.onSelectionChange(stack?.fleetId ?? null, selectedIndexes());
  });

  function toggleShipSelection(index: number) {
    setSelectedIndexes((current) =>
      current.includes(index) ? current.filter((value) => value !== index) : [...current, index],
    );
  }

  function toggleSelectAll() {
    if (allSelected()) {
      setSelectedIndexes([]);
      return;
    }
    setSelectedIndexes(activeStack()?.ships.map((ship) => ship.originalIndex) ?? []);
  }

  return (
    <Show when={activeStack()} fallback={<p>No fleet in orbit.</p>}>
      {(stack) => (
        <div class="fleet-inspector-panel">
          <div class="inspector-header">
            <div>
              <p class="eyebrow">Fleet Command</p>
            </div>
            <Show when={system()}>
              {(star) => (
                <button
                  type="button"
                  class="secondary-button inspector-header-action"
                  onClick={() => props.onViewSystem(star().id)}
                >
                  View System
                </button>
              )}
            </Show>
          </div>

          <div class="fleet-orbit-tabs" role="tablist" aria-label="Fleets at this star">
            <For each={fleetStacks()}>
              {(item) => (
                <button
                  type="button"
                  role="tab"
                  class="fleet-orbit-tab"
                  classList={{ "is-active": stack().key === item.key }}
                  aria-selected={stack().key === item.key}
                  onClick={() => setActiveStackKey(item.key)}
                >
                  {item.label}
                </button>
              )}
            </For>
          </div>

          <div class="fleet-orbit-grid" role="list" aria-label="Ships in fleet">
            <For each={stack().ships}>
              {(ship) => (
                <article
                  class="fleet-orbit-ship"
                  classList={{ "is-selected": selectedIndexes().includes(ship.originalIndex) }}
                  role="listitem"
                  onClick={() => toggleShipSelection(ship.originalIndex)}
                >
                  <span class="fleet-orbit-ship-icon">{ship.icon || "🚀"}</span>
                  <strong>
                    {ship.type === "colony_ship" && isColonizing()
                      ? "Colonizing a planet"
                      : ship.type === "colony_ship"
                        ? "Colony Ship"
                        : ship.type === "transport" && isInvading()
                          ? "Preparing invasion"
                          : ship.type === "transport"
                            ? "Transport"
                            : ship.name}
                  </strong>
                </article>
              )}
            </For>
          </div>

          <div class="fleet-inspector-actions">
            <div class="fleet-inspector-selection-actions">
              <button class="secondary-button" type="button" onClick={toggleSelectAll}>
                {allSelected() ? "Unselect All" : "Select All"}
              </button>
              <span class="fleet-orbit-selection-summary">
                {selectedIndexes().length} of {stack().ships.length} ships selected
              </span>
              <span class="fleet-orbit-selection-summary">
                Range {formatFuelRange(selectedFuelRange())}
              </span>
            </div>
            <Show when={props.isMoveTargeting && props.onCancelMove}>
              <button class="secondary-button" type="button" onClick={props.onCancelMove}>
                Cancel
              </button>
            </Show>
            <Show when={canColonizeWithShip() || isColonizing()}>
              <button
                class={isColonizing() ? "secondary-button" : "primary-button"}
                type="button"
                disabled={state.isBusy}
                onClick={() =>
                  isColonizing()
                    ? cancelColonyShipColonization(stack().fleetId)
                    : beginColonyShipColonization(stack().fleetId)
                }
              >
                {isColonizing() ? "Cancel Colonization" : "Colonize Planet"}
              </button>
            </Show>
            <Show when={canInvadeWithShip() || isInvading()}>
              <button
                class={isInvading() ? "secondary-button" : "primary-button"}
                type="button"
                disabled={state.isBusy}
                onClick={() => {
                  const stack_val = stack();
                  const target = invasionTarget();
                  if (isInvading()) {
                    cancelInvasion(stack_val.fleetId);
                  } else if (target) {
                    beginInvasion(stack_val.fleetId, target.id);
                  }
                }}
              >
                {isInvading() ? "Cancel Invasion" : "Invade Planet"}
              </button>
            </Show>
          </div>
        </div>
      )}
    </Show>
  );
}
