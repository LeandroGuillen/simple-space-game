import { attachModalBehavior } from "../lib/modal";
import { Portal } from "solid-js/web";
import { For, Show, createEffect, createMemo, createSignal, on } from "solid-js";

import {
  findPlayerFleetsAtStar,
  findStar,
  fleetTabLabel,
  sortFleetsForStarTabs,
} from "../lib/utils";
import { useAppContext } from "../state/app-context";
import type { FleetShipViewModel } from "../lib/types/domain";

type FleetOrbitModalProps = {
  starId: number | null;
  fleetId?: number | null;
  isOpen: boolean;
  onClose: () => void;
  onMove?: (fleetId: number, shipIndexes: number[]) => void;
};

type FleetStackShip = FleetShipViewModel & {
  originalIndex: number;
};

type FleetStack = {
  key: string;
  fleetId: number;
  label: string;
  ships: FleetStackShip[];
};

export function FleetOrbitModal(props: FleetOrbitModalProps) {
  const { state, currentPlayer } = useAppContext();
  const fleetsAtStar = createMemo(() => {
    if (!props.starId) {
      return [];
    }
    return sortFleetsForStarTabs(
      findPlayerFleetsAtStar(state.game, currentPlayer()?.id ?? 0, props.starId),
    );
  });
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
  const fleet = createMemo(() => {
    const stacks = fleetStacks();
    if (stacks.length === 0) {
      return null;
    }
    const key = activeStackKey();
    if (key != null) {
      const found = stacks.find((f) => f.key === key);
      if (found) {
        return found;
      }
    }
    return stacks[0];
  });
  const [selectedIndexes, setSelectedIndexes] = createSignal<number[]>([]);
  const allSelected = createMemo(() => {
    const shipCount = fleet()?.ships.length ?? 0;
    return shipCount > 0 && selectedIndexes().length === shipCount;
  });
  const moveDisabled = createMemo(() => {
    if (!props.onMove || selectedIndexes().length === 0) {
      return true;
    }
    return false;
  });

  createEffect(
    on(
      [() => props.isOpen, () => props.fleetId, () => props.starId],
      ([open, pid]) => {
        if (!open) {
          return;
        }
        const stacks = fleetStacks();
        if (pid != null && stacks.some((f) => f.fleetId === pid)) {
          setActiveStackKey(stacks.find((stack) => stack.fleetId === pid)?.key ?? null);
          return;
        }
        setActiveStackKey(null);
      },
    ),
  );

  const activeFleetShipCount = createMemo(() => {
    if (!props.isOpen) {
      return 0;
    }
    return fleet()?.ships.length ?? 0;
  });

  // Only reset selection when the user opens the modal, switches tabs, or the server
  // changes how many ships are in that stack. Do NOT subscribe blindly to `fleet()`:
  // every WebSocket game snapshot would re-run an effect and wipe partial selection
  // before "End turn", so the client would always queue a full-fleet move.
  createEffect(
    on(
      [() => props.isOpen, activeStackKey, activeFleetShipCount],
      ([open, , shipCount]) => {
        if (!open || shipCount === 0) {
          return;
        }
        setSelectedIndexes(fleet()?.ships.map((ship) => ship.originalIndex) ?? []);
      },
    ),
  );

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
    setSelectedIndexes(fleet()?.ships.map((ship) => ship.originalIndex) ?? []);
  }

  return (
    <Show when={props.isOpen}>
      <Portal>
        <div class="research-modal-backdrop" onClick={props.onClose}>
          <section
            class="research-modal panel fleet-orbit-modal"
            role="dialog"
            aria-modal="true"
            aria-labelledby="fleet-orbit-modal-title"
            ref={(element) => attachModalBehavior(element, props.onClose)}
            onClick={(event) => event.stopPropagation()}
          >
            <div class="research-modal-header fleet-orbit-modal-header">
              <h2 id="fleet-orbit-modal-title">FLEET</h2>
            </div>

            <Show when={fleetStacks().length > 0}>
              <div class="fleet-orbit-tabs" role="tablist" aria-label="Fleets at this star">
                <For each={fleetStacks()}>
                  {(stack) => (
                    <button
                      type="button"
                      role="tab"
                      class="fleet-orbit-tab"
                      classList={{ "is-active": fleet()?.key === stack.key }}
                      aria-selected={fleet()?.key === stack.key}
                      onClick={() => setActiveStackKey(stack.key)}
                    >
                      {stack.label}
                    </button>
                  )}
                </For>
              </div>
            </Show>

            <Show
              when={fleet()}
              fallback={
                <>
                  <div class="research-breakthrough-notice">
                    <span>No fleet in orbit</span>
                    <strong>Build a Colony Ship to create one</strong>
                  </div>
                  <div class="fleet-orbit-modal-footer">
                    <div class="fleet-orbit-actions-left" />
                    <button class="secondary-button research-close-button" type="button" onClick={props.onClose}>
                      Close
                    </button>
                  </div>
                </>
              }
            >
              {(activeFleet) => (
                <>
                  <div class="fleet-orbit-grid" role="list" aria-label="Ships in fleet">
                    <For each={activeFleet().ships}>
                      {(ship) => (
                        <article
                          class="fleet-orbit-ship"
                          classList={{ "is-selected": selectedIndexes().includes(ship.originalIndex) }}
                          role="listitem"
                          onClick={() => toggleShipSelection(ship.originalIndex)}
                        >
                          <span class="fleet-orbit-ship-icon">{ship.icon || "🚀"}</span>
                          <strong>{ship.type === "colony_ship" ? "Colony Ship" : ship.name}</strong>
                        </article>
                      )}
                    </For>
                  </div>
                  <p class="fleet-orbit-selection-summary">
                    {selectedIndexes().length} of {activeFleet().ships.length} ships selected
                  </p>
                  <div class="fleet-orbit-modal-footer">
                    <div class="fleet-orbit-actions-left">
                      <button
                        class="primary-button"
                        type="button"
                        disabled={moveDisabled()}
                        onClick={() => props.onMove?.(activeFleet().fleetId, [...selectedIndexes()])}
                      >
                        Move
                      </button>
                      <button class="secondary-button" type="button" onClick={toggleSelectAll}>
                        {allSelected() ? "Deselect All" : "Select All"}
                      </button>
                    </div>
                    <button class="secondary-button research-close-button" type="button" onClick={props.onClose}>
                      Close
                    </button>
                  </div>
                </>
              )}
            </Show>
          </section>
        </div>
      </Portal>
    </Show>
  );
}
