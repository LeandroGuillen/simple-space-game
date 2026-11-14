import { A } from "@solidjs/router";
import { For, Show } from "solid-js";

import { canDeclareWar, type DiplomacyContactRow } from "../lib/diplomacy";
import { formatStardate } from "../lib/utils";

type DiplomacyContactsProps = {
  rows: DiplomacyContactRow[];
  /** Empires with a declaration of war queued for this turn. */
  queuedWarIds?: readonly number[];
  onDeclareWar?: (playerId: number) => void;
  onCancelDeclareWar?: (playerId: number) => void;
};

/**
 * Contacted empires with their war/peace status. Declare War queues a turn
 * order that resolves with the next turn; peace offers are not available yet.
 */
export function DiplomacyContacts(props: DiplomacyContactsProps) {
  const queued = () => props.queuedWarIds ?? [];
  const isQueued = (row: DiplomacyContactRow) => queued().includes(row.playerId);
  return (
    <Show
      when={props.rows.length > 0}
      fallback={<p class="empty-copy">You have not met any other empire yet. Send fleets out to explore.</p>}
    >
      <div class="table-frame">
        <table class="data-table" aria-label="Contacts">
          <thead>
            <tr>
              <th>Empire</th>
              <th>Race</th>
              <th>Government</th>
              <th>Status</th>
              <th>Relation</th>
              <th>First contact</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            <For each={props.rows}>
              {(row) => (
                <tr data-player-id={row.playerId}>
                  <td>
                    <span class="diplomacy-swatch" style={{ background: row.color }} aria-hidden="true" />
                    {row.name}
                  </td>
                  <td>{row.raceName}</td>
                  <td>{row.government}</td>
                  <td class={row.atWar ? "diplomacy-status-war" : undefined}>{row.status}</td>
                  <td>{row.relation}</td>
                  <td>
                    <Show when={row.starId} fallback={formatStardate(row.metTurn)}>
                      <A href={`/galaxy?star=${row.starId}`}>{row.starName || `Star ${row.starId}`}</A>
                      {` · ${formatStardate(row.metTurn)}`}
                    </Show>
                  </td>
                  <td>
                    <button
                      type="button"
                      class="secondary-button"
                      disabled={!canDeclareWar(row, queued()) || !props.onDeclareWar}
                      title={
                        row.eliminated
                          ? "This empire has been eliminated"
                          : row.atWar
                            ? "Already at war"
                            : isQueued(row)
                              ? "War will be declared when the turn ends"
                              : "Declare war when the turn ends"
                      }
                      onClick={() => props.onDeclareWar?.(row.playerId)}
                    >
                      {isQueued(row) ? "War queued" : "Declare War"}
                    </button>
                    <Show when={isQueued(row) && props.onCancelDeclareWar}>
                      {" "}
                      <button type="button" class="secondary-button" onClick={() => props.onCancelDeclareWar?.(row.playerId)}>
                        Cancel
                      </button>
                    </Show>
                  </td>
                </tr>
              )}
            </For>
          </tbody>
        </table>
      </div>
    </Show>
  );
}
