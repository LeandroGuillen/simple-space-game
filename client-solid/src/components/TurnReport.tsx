import { A } from "@solidjs/router";
import { For, onMount, Show } from "solid-js";
import { Portal } from "solid-js/web";

import { attachModalBehavior } from "../lib/modal";
import { groupTurnEvents, TURN_REPORT_MODES, turnEventHref, type TurnReportMode } from "../lib/turn-report";
import type { ApiTurnEvent } from "../lib/types/api";
import { formatStardate } from "../lib/utils";

type ModeProps = { mode: TurnReportMode; onModeChange: (mode: TurnReportMode) => void };

/** Pop up / Dock left / Hide switch shown in both the popup and the dock. */
export function TurnReportModeControl(props: ModeProps) {
  return (
    <div class="turn-report-mode" role="radiogroup" aria-label="Show turn reports">
      <For each={TURN_REPORT_MODES}>
        {(option) => (
          <button
            type="button"
            role="radio"
            class="turn-report-mode-option"
            aria-checked={props.mode === option.value}
            title={option.title}
            onClick={() => props.onModeChange(option.value)}
          >
            {option.label}
          </button>
        )}
      </For>
    </div>
  );
}

function TurnReportEvents(props: { events: ApiTurnEvent[]; onNavigate?: () => void }) {
  const groups = () => groupTurnEvents(props.events);
  return (
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
                    <A class="turn-report-row" href={turnEventHref(event)} onClick={() => props.onNavigate?.()}>
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
  );
}

export function TurnReport(props: { turn: number; events: ApiTurnEvent[]; onClose: () => void } & ModeProps) {
  let okButton: HTMLButtonElement | undefined;
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
            <TurnReportEvents events={props.events} onNavigate={() => props.onClose()} />
          </div>
          <footer class="turn-report-actions">
            <TurnReportModeControl mode={props.mode} onModeChange={props.onModeChange} />
            <button ref={okButton} type="button" class="primary-button" onClick={() => props.onClose()}>
              OK
            </button>
          </footer>
        </section>
      </div>
    </Portal>
  );
}

/** Left-hand dock: always visible, refreshed in place after every turn. */
export function TurnReportDock(props: { turn: number; events: ApiTurnEvent[] } & ModeProps) {
  return (
    <aside class="turn-report-dock" aria-labelledby="turn-report-dock-title">
      <header class="turn-report-header">
        <div>
          <p class="eyebrow">Stardate {formatStardate(props.turn)}</p>
          <h2 id="turn-report-dock-title">Turn report</h2>
        </div>
      </header>
      <div class="turn-report-body">
        <TurnReportEvents events={props.events} />
      </div>
      <footer class="turn-report-actions">
        <TurnReportModeControl mode={props.mode} onModeChange={props.onModeChange} />
      </footer>
    </aside>
  );
}
