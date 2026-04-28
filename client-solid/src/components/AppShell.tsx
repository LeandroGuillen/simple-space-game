import {
  A,
  useLocation,
  useNavigate,
  type RouteSectionProps,
} from "@solidjs/router";
import { For, Show, createEffect, createMemo } from "solid-js";

import { APP_VERSION } from "../lib/app-meta";
import { gameIdShort } from "../lib/utils";
import { useAppContext } from "../state/app-context";

const gameNavItems = [
  { href: "/galaxy", label: "Galaxy" },
  { href: "/colonies", label: "Colonies" },
  { href: "/fleets", label: "Fleets" },
] as const;

const REPO_PLACEHOLDER_HREF =
  "https://github.com/leandroguillen/simple-space-game";

function isGameCommandPath(pathname: string) {
  return (
    pathname.startsWith("/galaxy") ||
    pathname.startsWith("/stars") ||
    pathname.startsWith("/colonies") ||
    pathname.startsWith("/fleets")
  );
}

/** Which campaign sidebar href should show as active (star/planet views belong to Galaxy). */
function campaignNavActiveHref(
  pathname: string,
): (typeof gameNavItems)[number]["href"] | null {
  if (pathname.startsWith("/galaxy") || pathname.startsWith("/stars")) {
    return "/galaxy";
  }
  if (pathname.startsWith("/colonies")) {
    return "/colonies";
  }
  if (pathname.startsWith("/fleets")) {
    return "/fleets";
  }
  return null;
}

type ConnectionStatus = "idle" | "checking" | "connected" | "error";

function connectionStatusMeta(status: ConnectionStatus): {
  label: string;
  statusClass: string;
} {
  switch (status) {
    case "idle":
      return {
        label: "Not connected",
        statusClass: "status status--idle",
      };
    case "checking":
      return {
        label: "Connecting…",
        statusClass: "status status--checking",
      };
    case "connected":
      return {
        label: "Connected",
        statusClass: "status status--connected",
      };
    case "error":
      return {
        label: "Connection error",
        statusClass: "status status--error",
      };
  }
}

export function AppShell(props: RouteSectionProps) {
  const location = useLocation();
  const navigate = useNavigate();
  const { state, currentPlayer, endTurn } = useAppContext();
  const connectionMeta = createMemo(() =>
    connectionStatusMeta(state.session.connectionStatus),
  );

  const showGameChrome = createMemo(() => isGameCommandPath(location.pathname));

  const activeCampaignNavHref = createMemo(() =>
    campaignNavActiveHref(location.pathname),
  );

  createEffect(() => {
    if (isGameCommandPath(location.pathname) && !state.session.selectedGameId) {
      navigate("/setup", { replace: true });
    }
  });

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
          github
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
                  {(item) => (
                    <A
                      href={item.href}
                      class="nav-link"
                      classList={{
                        active: activeCampaignNavHref() === item.href,
                      }}
                      aria-current={
                        activeCampaignNavHref() === item.href
                          ? "page"
                          : undefined
                      }
                    >
                      {item.label}
                    </A>
                  )}
                </For>
              </nav>
            </div>

            <section class="sidebar-session" aria-label="Session">
              <span class={connectionMeta().statusClass}>
                <span class="status-dot" aria-hidden="true" />
                {connectionMeta().label}
              </span>
              <Show when={state.session.serverUrl}>
                <span class="session-server" title={state.session.serverUrl}>
                  {state.session.serverUrl}
                </span>
              </Show>
              <Show when={state.session.selectedGameId}>
                <span class="session-game">
                  Game · {gameIdShort(state.session.selectedGameId)}
                </span>
              </Show>
              <A href="/setup" class="session-manage">
                Manage
              </A>
            </section>
          </aside>

          <div class="game-main">
            <Show when={state.game}>
              <header class="game-topbar panel" aria-label="Current game">
                <div class="turn-panel">
                  <div class="turn-meta">
                    <div class="turn-stat turn-stat--accent">
                      <span>Treasury</span>
                      <strong>{currentPlayer()?.treasury ?? 0} BC</strong>
                    </div>

                    <div class="turn-stat">
                      <span>Command</span>
                      <strong>0</strong>
                    </div>

                    <div class="turn-stat">
                      <span>Stardate</span>
                      <strong>{state.game?.currentTurn}</strong>
                    </div>
                  </div>
                  <div class="turn-actions">
                    <button
                      type="button"
                      class="primary-button game-turn-button"
                      onClick={() => void endTurn()}
                      disabled={state.isBusy}
                      aria-busy={state.isBusy}
                    >
                      End Turn
                    </button>
                  </div>
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
