import { A, useLocation, type RouteSectionProps } from "@solidjs/router";
import { For, Show, createMemo } from "solid-js";

import { APP_VERSION } from "../lib/app-meta";
import { gameIdShort } from "../lib/utils";
import { useAppContext } from "../state/app-context";

const gameNavItems = [
  { href: "/galaxy", label: "Galaxy" },
  { href: "/colonies", label: "Colonies" },
  { href: "/fleets", label: "Fleets" },
] as const;

const REPO_PLACEHOLDER_HREF =
  "https://github.com/placeholder/simple-space-game";

function isGameCommandPath(pathname: string) {
  return (
    pathname.startsWith("/galaxy") ||
    pathname.startsWith("/stars") ||
    pathname.startsWith("/colonies") ||
    pathname.startsWith("/fleets")
  );
}

type ConnectionStatus = "idle" | "checking" | "connected" | "error";

function connectionStatusMeta(status: ConnectionStatus): {
  label: string;
  pillClass: string;
} {
  switch (status) {
    case "idle":
      return {
        label: "Not connected",
        pillClass: "status-pill status-pill--idle",
      };
    case "checking":
      return {
        label: "Connecting…",
        pillClass: "status-pill status-pill--checking",
      };
    case "connected":
      return {
        label: "Connected",
        pillClass: "status-pill status-pill--connected",
      };
    case "error":
      return {
        label: "Connection error",
        pillClass: "status-pill status-pill--error",
      };
  }
}

export function AppShell(props: RouteSectionProps) {
  const location = useLocation();
  const { state, currentPlayer, endTurn } = useAppContext();
  const connectionMeta = createMemo(() =>
    connectionStatusMeta(state.session.connectionStatus),
  );

  const showGameChrome = createMemo(() =>
    isGameCommandPath(location.pathname),
  );

  return (
    <div class="frame">
      <header class="app-topbar panel" aria-label="Application">
        <div class="app-topbar-brand">
          <span class="app-title">Simple Space Game</span>
          <span class="app-version" title="Client version">
            v{APP_VERSION}
          </span>
        </div>
        <a
          class="app-repo-link"
          href={REPO_PLACEHOLDER_HREF}
          target="_blank"
          rel="noreferrer"
        >
          Repository
        </a>
      </header>

      <Show
        when={showGameChrome()}
        fallback={<main class="content content--route">{props.children}</main>}
      >
        <div class="workspace">
          <aside class="sidebar panel" aria-label="Campaign">
            <div class="sidebar-block sidebar-block--campaign">
              <p class="eyebrow">Campaign</p>
              <nav class="nav-list" aria-label="Game commands">
                <For each={gameNavItems}>
                  {(item) => {
                    const active = location.pathname.startsWith(item.href);
                    return (
                      <A
                        href={item.href}
                        class="nav-link"
                        classList={{ active }}
                        aria-current={active ? "page" : undefined}
                      >
                        {item.label}
                      </A>
                    );
                  }}
                </For>
              </nav>
            </div>

            <section class="sidebar-section">
              <p class="eyebrow">Session</p>
              <dl class="session-rows">
                <dt>Server</dt>
                <dd>
                  <span
                    class="session-value"
                    title={state.session.serverUrl || undefined}
                  >
                    {state.session.serverUrl || "—"}
                  </span>
                </dd>
                <dt>Status</dt>
                <dd>
                  <span class={connectionMeta().pillClass}>
                    {connectionMeta().label}
                  </span>
                </dd>
                <Show when={state.session.selectedGameId}>
                  <dt>Game</dt>
                  <dd>{gameIdShort(state.session.selectedGameId)}</dd>
                </Show>
              </dl>
              <p class="sidebar-setup-link-wrap">
                <A href="/setup" class="link-button sidebar-setup-link">
                  Server & game…
                </A>
              </p>
            </section>

            <Show when={state.info}>
              <section class="sidebar-section sidebar-note">
                <p>{state.info}</p>
              </section>
            </Show>
          </aside>

          <div class="game-main">
            <Show when={state.game}>
              <header class="game-topbar panel" aria-label="Current game">
                <div class="turn-panel">
                  <div class="turn-chip">
                    <span>Stardate</span>
                    <strong>{state.game?.currentTurn}</strong>
                  </div>
                  <div class="turn-chip">
                    <span>Treasury</span>
                    <strong>{currentPlayer()?.treasury ?? 0} BC</strong>
                  </div>
                  <div class="turn-chip">
                    <span>Game</span>
                    <strong>{gameIdShort(state.game?.id ?? "")}</strong>
                  </div>
                  <button
                    type="button"
                    class="primary-button game-turn-button"
                    onClick={() => void endTurn()}
                    disabled={state.isBusy}
                    aria-busy={state.isBusy}
                  >
                    Turn
                  </button>
                </div>
              </header>
            </Show>

            <main class="content content--route">{props.children}</main>
          </div>
        </div>
      </Show>

      <Show when={state.error}>
        <div class="status-banner error">{state.error}</div>
      </Show>
    </div>
  );
}
