import { A } from "@solidjs/router";
import { For, onMount, Show } from "solid-js";
import { Portal } from "solid-js/web";

import { attachModalBehavior } from "../lib/modal";
import { groupTurnEvents, turnEventHref } from "../lib/turn-report";
import type { ApiTurnEvent } from "../lib/types/api";
import { formatStardate } from "../lib/utils";

export function TurnReport(props: { turn: number; events: ApiTurnEvent[]; onClose: () => void }) {
  let okButton: HTMLButtonElement | undefined;
  const groups = () => groupTurnEvents(props.events);
  const handleKeyDown = (event: KeyboardEvent) => {
    if (event.key !== "Enter" || event.defaultPrevented) return;
    // Links and buttons handle Enter themselves (navigate or click).
    if ((event.target as HTMLElement).closest("a, button")) return;
    event.preventDefault();
    props.onClose();
  };

  return (
    <Portal>
      <div class="research-modal-backdrop turn-report-backdrop" onClick={() => props.onClose()}>
        <section
          class="turn-report"
          role="dialog"
          aria-modal="true"
          aria-labelledby="turn-report-title"
          ref={(element) => {
            attachModalBehavior(element, () => props.onClose());
            // Registered after the modal's own mount hook so OK, not the close X, gets initial focus.
            onMount(() => okButton?.focus());
          }}
          onClick={(event) => event.stopPropagation()}
          onKeyDown={handleKeyDown}
        >
          <header class="turn-report-header">
            <div>
              <p class="eyebrow">Stardate {formatStardate(props.turn)} complete</p>
              <h2 id="turn-report-title">Turn report</h2>
            </div>
            <button type="button" class="status-banner-close turn-report-close" aria-label="Close turn report" onClick={() => props.onClose()}>
              ×
            </button>
          </header>
          <div class="turn-report-body">
            <Show when={props.events.length > 0} fallback={<p class="turn-report-empty">No notable events this turn.</p>}>
              <For each={groups()}>
                {(group) => (
                  <section class="turn-report-group" aria-label={group.label}>
                    <Show when={groups().length > 1}>
                      <h3 class="turn-report-group-title">{group.label}</h3>
                    </Show>
                    <ul class="turn-report-list">
                      <For each={group.events}>
                        {(event) => (
                          <li>
                            <A class="turn-report-row" href={turnEventHref(event)} onClick={() => props.onClose()}>
                              <span class="turn-report-row-text">{event.text}</span>
                              <span class="turn-report-row-go" aria-hidden="true">View ›</span>
                            </A>
                          </li>
                        )}
                      </For>
                    </ul>
                  </section>
                )}
              </For>
            </Show>
          </div>
          <footer class="turn-report-actions">
            <button ref={okButton} type="button" class="primary-button" onClick={() => props.onClose()}>
              OK
            </button>
          </footer>
        </section>
      </div>
    </Portal>
  );
}
