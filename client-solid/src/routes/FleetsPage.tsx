import { createMemo, createSignal, Show } from "solid-js";

import { InvasionReportPanel } from "../components/InvasionReportPanel";
import {
  ShipDesignModal,
  type ShipDesignModalMode,
} from "../components/ShipDesignModal";
import type { ShipDesignViewModel } from "../lib/types/domain";
import { useAppContext } from "../state/app-context";

export function FleetsPage() {
  const { state, currentPlayer } = useAppContext();

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
        <p class="eyebrow">Fleets</p>
        <h2>Fleet roster</h2>
        <div class="table-frame">
          <table class="data-table">
            <thead>
              <tr>
                <th>Fleet ID</th>
                <th>Commander</th>
              </tr>
            </thead>
            <tbody>
              {currentPlayer()?.fleets.map((fleetId) => (
                <tr>
                  <td>{fleetId}</td>
                  <td>{currentPlayer()?.name}</td>
                </tr>
              ))}
              <Show when={(currentPlayer()?.fleets.length ?? 0) === 0}>
                <tr>
                  <td colSpan={2}>
                    No fleet records available for the current commander.
                  </td>
                </tr>
              </Show>
            </tbody>
          </table>
        </div>

        <div class="fleet-note">
          <p class="eyebrow">Fleet Manifest</p>
          <p>
            The backend currently exposes fleet IDs but not a full fleet object
            collection. This screen focuses on ship design operations until
            fleet payloads are expanded.
          </p>
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
