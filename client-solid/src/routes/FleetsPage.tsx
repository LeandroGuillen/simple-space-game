import { useNavigate } from "@solidjs/router";
import { createMemo, createSignal, For, Show } from "solid-js";
import { fleetRoster } from "../lib/fleet-roster";

import { InvasionReportPanel } from "../components/InvasionReportPanel";
import {
  ShipDesignModal,
  type ShipDesignModalMode,
} from "../components/ShipDesignModal";
import type { ShipDesignViewModel } from "../lib/types/domain";
import { useAppContext } from "../state/app-context";

export function FleetsPage() {
  const { state, currentPlayer } = useAppContext();
  const navigate = useNavigate();
  const [idleOnly, setIdleOnly] = createSignal(false);
  const rows = createMemo(() => fleetRoster(state.game, currentPlayer(), state.pendingCommands, state.pendingColonyShipFleets));
  const visibleRows = createMemo(() => idleOnly() ? rows().filter((row) => row.status === "Idle") : rows());

  const commonDesigns = createMemo(
    () => state.game?.resources.shipDesignCollection.shipDesigns ?? [],
  );
  const playerDesigns = createMemo(() => currentPlayer()?.shipDesigns ?? []);

  const [modalOpen, setModalOpen] = createSignal(false);
  const [modalMode, setModalMode] =
    createSignal<ShipDesignModalMode>("create");
  const [modalInitial, setModalInitial] =
    createSignal<ShipDesignViewModel | null>(null);

  function openNewDesign() {
    setModalMode("create");
    setModalInitial(null);
    setModalOpen(true);
  }

  function openDesign(design: ShipDesignViewModel) {
    setModalMode(design.isCommon ? "duplicate" : "edit");
    setModalInitial(design);
    setModalOpen(true);
  }

  return (
    <section class="page-grid">
      <section class="panel">
        <div class="fleet-list-header">
          <div>
            <p class="eyebrow">Ship Designs</p>
            <h2>Fleet architecture</h2>
          </div>
          <button
            class="primary-button"
            type="button"
            onClick={openNewDesign}
            disabled={!state.game}
          >
            New Design
          </button>
        </div>
        <div class="table-frame">
          <table class="data-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Hull</th>
                <th>Drive</th>
                <th>Computer</th>
                <th>Shield</th>
                <th>Fuel</th>
                <th>Type</th>
              </tr>
            </thead>
            <tbody>
              {[...playerDesigns(), ...commonDesigns()].map((design) => (
                <tr onClick={() => openDesign(design)}>
                  <td>{design.name}</td>
                  <td>{design.hullSize.name}</td>
                  <td>{design.drive.name}</td>
                  <td>{design.computer.name}</td>
                  <td>{design.shield.name}</td>
                  <td>{design.fuelCell.name}</td>
                  <td>{design.isCommon ? "Common" : "Custom"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      <section class="panel">
        <div class="fleet-list-header">
          <div><p class="eyebrow">Fleets</p><h2>Fleet roster</h2></div>
          <label class="fleet-idle-filter"><input type="checkbox" checked={idleOnly()} onChange={(event) => setIdleOnly(event.currentTarget.checked)} /> Idle only</label>
        </div>
        <div class="table-frame">
          <table class="data-table fleet-roster-table">
            <thead><tr><th>Fleet</th><th>Location</th><th>Destination / ETA</th><th>Ships</th><th>Fuel range</th><th>Status</th></tr></thead>
            <tbody>
              <For each={visibleRows()}>{(row) => <tr onClick={() => navigate(`/galaxy?fleet=${row.fleet.id}`)}>
                <td><button class="link-button fleet-roster-link" type="button" onClick={(event) => { event.stopPropagation(); navigate(`/galaxy?fleet=${row.fleet.id}`); }}>Fleet {row.fleet.id}</button></td>
                <td>{row.location}</td><td>{row.destination}</td><td>{row.fleet.ships.length} · {row.ships}</td><td>{row.fuel}</td><td>{row.status}</td>
              </tr>}</For>
              <Show when={visibleRows().length === 0}><tr><td colSpan={6}>{idleOnly() ? "No idle fleets." : "No fleets available for the current commander."}</td></tr></Show>
            </tbody>
          </table>
        </div>
      </section>

      <InvasionReportPanel
        reports={state.game?.invasionReports ?? []}
        playerId={currentPlayer()?.id ?? 0}
        playerNames={new Map((state.game?.players ?? []).map((player) => [player.id, player.name]))}
      />

      <ShipDesignModal
        isOpen={modalOpen()}
        mode={modalMode()}
        initial={modalInitial()}
        onClose={() => setModalOpen(false)}
      />
    </section>
  );
}
