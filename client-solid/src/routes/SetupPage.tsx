import { useNavigate } from "@solidjs/router";
import {
  createEffect,
  createMemo,
  createSignal,
  For,
  Show,
} from "solid-js";

import type { ApiGameListItem } from "../lib/types/api";
import { useAppContext } from "../state/app-context";

type ServerInfo = {
  url: string;
  status: "idle" | "checking" | "connected" | "error";
  games: ApiGameListItem[];
};

function PanelHeader(props: { title: string; count?: number }) {
  return (
    <header class="panel-header">
      <h2>{props.title}</h2>
      <Show when={props.count !== undefined}>
        <span class="panel-count">{props.count}</span>
      </Show>
    </header>
  );
}

function StatusDot(props: { status: string }) {
  const label = () => {
    switch (props.status) {
      case "idle": return "Not connected";
      case "checking": return "Connecting...";
      case "connected": return "OK";
      case "error": return "Failed";
      default: return props.status;
    }
  };
  return (
    <span
      class={`status-dot status-dot--${props.status}`}
      title={label()}
    />
  );
}

function PlusIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
      <path d="M7 2v10M2 7h10" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
    </svg>
  );
}

function ArrowIcon() {
  return (
    <svg width="10" height="10" viewBox="0 0 10 10" fill="none">
      <path d="M2 5h6M6 2l4 3-4 3" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"/>
    </svg>
  );
}

export function SetupPage() {
  const navigate = useNavigate();
  const { state, connect, createGame, refreshGames, selectGame } =
    useAppContext();

  const [servers, setServers] = createSignal<ServerInfo[]>([]);
  const [selectedServerUrl, setSelectedServerUrl] = createSignal("");
  const [showServerForm, setShowServerForm] = createSignal(false);
  const [newServerUrl, setNewServerUrl] = createSignal("http://127.0.0.1:8080");
  const [playerName, setPlayerName] = createSignal("Admiral");
  const [numStars, setNumStars] = createSignal(24);
  const [playerColor, setPlayerColor] = createSignal("#FFFFFF");
  const [showNewGameForm, setShowNewGameForm] = createSignal(false);
  const [isConnecting, setIsConnecting] = createSignal(false);
  const [confirmRemove, setConfirmRemove] = createSignal<string | null>(null);

  const activeServer = createMemo(() =>
    servers().find((s) => s.url === selectedServerUrl())
  );

  const isConnected = createMemo(() => activeServer()?.status === "connected");
  const gameCount = createMemo(() => activeServer()?.games.length ?? 0);

  createEffect(() => {
    const status = state.session.connectionStatus;
    const url = state.session.serverUrl;
    if (url && (status === "connected" || status === "error" || status === "checking")) {
      setServers((prev) => {
        const existing = prev.find((s) => s.url === url);
        if (existing) {
          return prev.map((s) =>
            s.url === url
              ? { ...s, status: status as ServerInfo["status"], games: status === "connected" ? state.games : s.games }
              : s
          );
        }
        return [
          ...prev,
          {
            url,
            status: status as ServerInfo["status"],
            games: status === "connected" ? state.games : [],
          },
        ];
      });
    }
  });

  createEffect(() => {
    const colors = state.playerColorOptions.colors;
    if (colors.length === 0) {
      return;
    }
    if (!colors.includes(playerColor())) {
      setPlayerColor(colors[0]!);
    }
  });

  const handleAddServer = async (event: SubmitEvent) => {
    event.preventDefault();
    const url = newServerUrl();
    setServers((prev) => [
      ...prev,
      { url, status: "checking" as const, games: [] },
    ]);
    setSelectedServerUrl(url);
    setShowServerForm(false);
    setNewServerUrl("http://127.0.0.1:8080");
    setIsConnecting(true);
    try {
      await connect(url);
    } catch {
      // error handled in effect
    } finally {
      setIsConnecting(false);
    }
  };

  const handleSelectServer = async (url: string) => {
    setSelectedServerUrl(url);
    const server = servers().find((s) => s.url === url);
    if (server && server.status === "connected") {
      await refreshGames();
    }
  };

  const handleRemoveServer = (url: string) => {
    if (confirmRemove() === url) {
      setServers((prev) => prev.filter((s) => s.url !== url));
      if (selectedServerUrl() === url) {
        setSelectedServerUrl("");
      }
      setShowNewGameForm(false);
      setConfirmRemove(null);
    } else {
      setConfirmRemove(url);
      setTimeout(() => setConfirmRemove(null), 3000);
    }
  };

  const handleCreate = async (event: SubmitEvent) => {
    event.preventDefault();
    try {
      const gameId = await createGame(
        playerName(),
        Number(numStars()),
        playerColor(),
      );
      await selectGame(gameId);
      navigate("/galaxy");
    } catch {
      // error shown globally
    }
  };

  const handleSelect = async (gameId: string) => {
    await selectGame(gameId);
    navigate("/galaxy");
  };

  return (
    <section class="setup-page">
      <aside class="panel server-panel">
        <PanelHeader title="Servers" />

        <div class="server-list">
          <For each={servers()}>
            {(server) => (
              <li class="server-item-wrap">
                <button
                  type="button"
                  class="server-item"
                  classList={{ selected: selectedServerUrl() === server.url }}
                  onClick={() => void handleSelectServer(server.url)}
                  disabled={isConnecting()}
                >
                  <StatusDot status={server.status} />
                  <span class="server-item-url">{server.url}</span>
                  <span class="server-item-status">
                    {server.status === "connected"
                      ? `${server.games.length}`
                      : server.status}
                  </span>
                </button>
                <button
                  type="button"
                  class="server-remove"
                  classList={{ confirming: confirmRemove() === server.url }}
                  onClick={() => handleRemoveServer(server.url)}
                  disabled={isConnecting()}
                  title="Remove server"
                >
                  {confirmRemove() === server.url ? "?" : "×"}
                </button>
              </li>
            )}
          </For>
        </div>

        <Show when={!showServerForm()}>
          <button
            class="add-button"
            type="button"
            onClick={() => setShowServerForm(true)}
          >
            <PlusIcon />
            Add server
          </button>
        </Show>

        <Show when={showServerForm()}>
          <form class="server-form" onSubmit={handleAddServer}>
            <input
              value={newServerUrl()}
              onInput={(event) => setNewServerUrl(event.currentTarget.value)}
              placeholder="http://127.0.0.1:8080"
              autofocus
            />
            <div class="form-actions">
              <button
                class="link-button"
                type="button"
                onClick={() => setShowServerForm(false)}
                disabled={isConnecting()}
              >
                Cancel
              </button>
              <button
                class="primary-button"
                type="submit"
                disabled={isConnecting()}
              >
                Connect
              </button>
            </div>
          </form>
        </Show>
      </aside>

      <section class="panel game-panel">
        <PanelHeader title="Games" count={gameCount()} />

        <Show
          when={isConnected()}
          fallback={
            <div class="empty-state">
              <span class="empty-text">Select a server to view games</span>
            </div>
          }
        >
          <Show
            when={activeServer()?.games.length}
            fallback={
              <div class="empty-state">
                <span class="empty-text">No games yet</span>
                <button
                  class="secondary-button create-first"
                  type="button"
                  onClick={() => setShowNewGameForm(true)}
                >
                  Create one
                </button>
              </div>
            }
          >
            <ul class="game-list">
              <For each={activeServer()?.games}>
                {(game) => (
                  <li>
                    <button
                      type="button"
                      class="game-item"
                      onClick={() => void handleSelect(game.game_id)}
                      disabled={state.isBusy}
                    >
                      <div class="game-item-players">
                        <For each={game.players}>
                          {(player) => (
                            <span
                              class="game-player"
                              style={{ "background-color": player.color }}
                              title={`${player.name}: ${player.num_stars} stars`}
                            >
                              {player.name}
                              <span class="player-stars">
                                {player.num_stars}
                              </span>
                            </span>
                          )}
                        </For>
                      </div>
                      <div class="game-item-info">
                        <span class="game-id">{game.game_id.slice(0, 8)}</span>
                        <span class="game-meta">
                          {game.num_stars} stars · T{game.current_turn}
                        </span>
                      </div>
                      <span class="game-item-action">
                        <ArrowIcon />
                      </span>
                    </button>
                  </li>
                )}
              </For>
            </ul>
          </Show>

          <Show
            when={showNewGameForm()}
            fallback={
              <Show when={activeServer()?.games.length}>
                <button
                  class="secondary-button new-game-button"
                  type="button"
                  onClick={() => setShowNewGameForm(true)}
                >
                  <PlusIcon />
                  New Game
                </button>
              </Show>
            }
          >
            <form class="new-game-form" onSubmit={handleCreate}>
              <div class="new-game-row">
                <label class="field">
                  <span>Commander</span>
                  <input
                    value={playerName()}
                    onInput={(event) => setPlayerName(event.currentTarget.value)}
                    autofocus
                  />
                </label>
                <label class="field field-small">
                  <span>Stars</span>
                  <input
                    type="number"
                    min="12"
                    max="72"
                    value={numStars()}
                    onInput={(event) =>
                      setNumStars(Number(event.currentTarget.value))
                    }
                  />
                </label>
              </div>
              <label class="field">
                <span>Color</span>
                <div class="color-swatches">
                  <For each={state.playerColorOptions.colors}>
                    {(color) => (
                      <button
                        type="button"
                        class="color-swatch"
                        classList={{ selected: playerColor() === color }}
                        style={{ "background-color": color }}
                        onClick={() => setPlayerColor(color)}
                      />
                    )}
                  </For>
                </div>
              </label>
              <div class="form-actions">
                <button
                  class="link-button"
                  type="button"
                  onClick={() => setShowNewGameForm(false)}
                  disabled={state.isBusy}
                >
                  Cancel
                </button>
                <button
                  class="primary-button"
                  type="submit"
                  disabled={state.isBusy}
                >
                  Create
                </button>
              </div>
            </form>
          </Show>
        </Show>
      </section>
    </section>
  );
}