import { A } from "@solidjs/router";
import { For, Show } from "solid-js";

import type { DiplomacyContactRow } from "../lib/diplomacy";
import { formatStardate } from "../lib/utils";

/**
 * Read-only list of contacted empires. Diplomatic actions (war, peace,
 * treaties) are not available yet.
 */
export function DiplomacyContacts(props: { rows: DiplomacyContactRow[] }) {
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
                  <td>{row.eliminated ? "Eliminated" : "Peace"}</td>
                  <td>{row.relation}</td>
                  <td>
                    <Show when={row.starId} fallback={formatStardate(row.metTurn)}>
                      <A href={`/galaxy?star=${row.starId}`}>{row.starName || `Star ${row.starId}`}</A>
                      {` · ${formatStardate(row.metTurn)}`}
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
