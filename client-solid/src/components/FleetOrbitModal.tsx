import { Portal } from "solid-js/web";
import { For, Show, createEffect, createMemo, createSignal } from "solid-js";

import { findStar, findFleetInOrbit } from "../lib/utils";
import { useAppContext } from "../state/app-context";

type FleetOrbitModalProps = {
  starId: number | null;
  fleetId?: number | null;
  isOpen: boolean;
  onClose: () => void;
  onMove?: (fleetId: number, shipIndexes: number[]) => void;
};

export function FleetOrbitModal(props: FleetOrbitModalProps) {
  const { state, currentPlayer } = useAppContext();
  const star = createMemo(() =>
    props.starId ? findStar(state.game, props.starId) : null,
  );
  const fleet = createMemo(() => {
    if (props.fleetId != null) {
      return (state.game?.fleets ?? []).find((item) => item.id === props.fleetId) ?? null;
    }
    if (!props.starId) return null;
    return findFleetInOrbit(state.game, currentPlayer()?.id ?? 0, props.starId);
  });
  const [selectedIndexes, setSelectedIndexes] = createSignal<number[]>([]);
  const allSelected = createMemo(() => {
    const shipCount = fleet()?.ships.length ?? 0;
    return shipCount > 0 && selectedIndexes().length === shipCount;
  });
  const canRerouteMovingFleet = createMemo(() =>
    (currentPlayer()?.researchedTechs ?? []).includes("Hyperspace Communications"),
  );
  const moveDisabled = createMemo(() => {
    if (!props.onMove || selectedIndexes().length === 0) {
      return true;
    }
    const inTransit = (fleet()?.turnsRemaining ?? 0) > 0;
    return inTransit && !canRerouteMovingFleet();
  });

  createEffect(() => {
    if (!props.isOpen || !fleet()) {
      return;
    }
    setSelectedIndexes(fleet()!.ships.map((_, index) => index));
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
    setSelectedIndexes(fleet()?.ships.map((_, index) => index) ?? []);
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
            onClick={(event) => event.stopPropagation()}
          >
            <div class="research-modal-header">
              <div>
                <p class="eyebrow">Fleet in Orbit</p>
                <h2 id="fleet-orbit-modal-title">{star()?.name ?? "Unknown system"}</h2>
              </div>
              <button class="secondary-button research-close-button" type="button" onClick={props.onClose}>
                Close
              </button>
            </div>

            <Show
              when={fleet()}
              fallback={
                <div class="research-breakthrough-notice">
                  <span>No fleet in orbit</span>
                  <strong>Build a Colony Ship to create one</strong>
                </div>
              }
            >
              {(orbitFleet) => (
                <>
                  <div class="fleet-orbit-actions">
                    <button
                      class="primary-button"
                      type="button"
                      disabled={moveDisabled()}
                      onClick={() => props.onMove?.(orbitFleet().id, selectedIndexes())}
                    >
                      Move
                    </button>
                    <button class="secondary-button" type="button" onClick={toggleSelectAll}>
                      {allSelected() ? "Deselect All" : "Select All"}
                    </button>
                  </div>
                  <Show when={(orbitFleet().turnsRemaining ?? 0) > 0 && !canRerouteMovingFleet()}>
                    <p class="eyebrow">Hyperspace Communications is required to redirect moving fleets</p>
                  </Show>
                  <div class="fleet-orbit-grid" role="list" aria-label="Ships in orbit fleet">
                    <For each={orbitFleet().ships}>
                      {(ship, index) => (
                        <article
                          class="fleet-orbit-ship"
                          classList={{ "is-selected": selectedIndexes().includes(index()) }}
                          role="listitem"
                          onClick={() => toggleShipSelection(index())}
                        >
                        <span class="fleet-orbit-ship-icon">{ship.icon || "🚀"}</span>
                        <strong>{ship.type === "colony_ship" ? "Colony Ship" : ship.name}</strong>
                        </article>
                      )}
                    </For>
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
