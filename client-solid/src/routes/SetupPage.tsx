import { useNavigate } from "@solidjs/router";
import {
  createEffect,
  createMemo,
  createSignal,
  Show,
} from "solid-js";

import { gameIdShort } from "../lib/utils";
import { useAppContext } from "../state/app-context";

type ConnectionStatus = "idle" | "checking" | "connected" | "error";

function connectionStatusMeta(status: ConnectionStatus): {
  label: string;
  statusClass: string;
} {
  switch (status) {
    case "idle":
      return { label: "Not connected", statusClass: "status status--idle" };
    case "checking":
      return { label: "Connecting…", statusClass: "status status--checking" };
    case "connected":
      return { label: "Connected", statusClass: "status status--connected" };
    case "error":
      return {
        label: "Connection error",
        statusClass: "status status--error",
      };
  }
}

export function SetupPage() {
  const navigate = useNavigate();
  const { state, connect, disconnect, createGame, refreshGames, selectGame } =
    useAppContext();
  const [serverUrl, setServerUrl] = createSignal("http://127.0.0.1:8080");
  const [showServerForm, setShowServerForm] = createSignal(false);
  const [playerName, setPlayerName] = createSignal("Admiral");
  const [numStars, setNumStars] = createSignal(24);
  const [playerColor, setPlayerColor] = createSignal("#FFFFFF");
  const [showNewGameForm, setShowNewGameForm] = createSignal(false);

  const hasServer = createMemo(() => Boolean(state.session.serverUrl));
  const isConnected = createMemo(
    () => state.session.connectionStatus === "connected",
  );
  const showServerFormResolved = createMemo(
    () => !hasServer() || showServerForm(),
  );

  createEffect(() => {
    const colors = state.playerColorOptions.colors;
    if (colors.length === 0) {
      return;
    }
    if (!colors.includes(playerColor())) {
      setPlayerColor(colors[0]!);
    }
  });

  const handleConnect = async (event: SubmitEvent) => {
    event.preventDefault();
    try {
      await connect(serverUrl());
      setShowServerForm(false);
    } catch {
      // error surfaced in state
    }
  };

  const handleDisconnect = () => {
    disconnect();
    setShowServerForm(false);
    setShowNewGameForm(false);
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
      <section class="panel setup-panel">
        <header class="setup-header">
          <p class="eyebrow">Server</p>
          <h2>Connect</h2>
          <p class="lede">
            Point this client at the Go backend, then pick or create a game
            below.
          </p>
        </header>

        <Show when={hasServer()}>
          <ul class="server-list">
            <li class="server-row">
              <div class="server-info">
                <span class="server-url" title={state.session.serverUrl}>
                  {state.session.serverUrl}
                </span>
                <span
                  class={
                    connectionStatusMeta(state.session.connectionStatus)
                      .statusClass
                  }
                >
                  <span class="status-dot" aria-hidden="true" />
                  {connectionStatusMeta(state.session.connectionStatus).label}
                </span>
              </div>
              <button
                class="secondary-button"
                type="button"
                onClick={handleDisconnect}
                disabled={state.isBusy}
              >
                Disconnect
              </button>
            </li>
          </ul>
        </Show>

        <Show
          when={showServerFormResolved()}
          fallback={
            <div class="form-actions reveal-actions">
              <button
                class="secondary-button"
                type="button"
                onClick={() => setShowServerForm(true)}
              >
                Add server
              </button>
            </div>
          }
        >
          <form class="stack-form reveal-form" onSubmit={handleConnect}>
            <label class="field">
              <span>Server URL</span>
              <input
                value={serverUrl()}
                onInput={(event) => setServerUrl(event.currentTarget.value)}
                placeholder="http://127.0.0.1:8080"
                autofocus
              />
            </label>
            <div class="form-actions">
              <Show when={hasServer()}>
                <button
                  class="link-button"
                  type="button"
                  onClick={() => setShowServerForm(false)}
                  disabled={state.isBusy}
                >
                  Cancel
                </button>
              </Show>
              <button
                class="primary-button"
                type="submit"
                disabled={state.isBusy}
              >
                {isConnected() ? "Reconnect" : "Connect"}
              </button>
            </div>
          </form>
        </Show>

        <div class="setup-divider" role="separator" />

        <header class="setup-header">
          <p class="eyebrow">Game</p>
          <h2>Join or create</h2>
          <p class="lede">
            Enter an existing campaign or provision a new sector once the server
            is connected.
          </p>
          <p class="games-count">
            Max players per game: {state.playerColorOptions.maxPlayers}
          </p>
        </header>

        <Show
          when={isConnected()}
          fallback={
            <p class="empty-copy">
              Connect to a server using the form above before selecting a game.
            </p>
          }
        >
          <div class="game-list-header">
            <Show
              when={state.games.length > 0}
              fallback={<p class="empty-copy">No games on this server yet.</p>}
            >
              <p class="games-count">
                {state.games.length} game
                {state.games.length === 1 ? "" : "s"} available
              </p>
            </Show>
            <button
              class="link-button games-refresh"
              type="button"
              onClick={() => void refreshGames()}
              disabled={state.isBusy}
              title="Refresh game list"
            >
              Refresh
            </button>
          </div>

          <Show when={state.games.length > 0}>
            <ul class="game-list">
              {state.games.map((game) => (
                <li>
                  <button
                    type="button"
                    class="game-card"
                    onClick={() => void handleSelect(game.game_id)}
                    disabled={state.isBusy}
                  >
                    <div>
                      <strong>{gameIdShort(game.game_id)}</strong>
                      <p>
                        {game.num_stars} stars · Turn {game.current_turn}
                      </p>
                    </div>
                    <span class="game-card-cue" aria-hidden="true">
                      Enter →
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          </Show>

          <Show
            when={showNewGameForm()}
            fallback={
              <div class="form-actions reveal-actions">
                <button
                  class="secondary-button"
                  type="button"
                  onClick={() => setShowNewGameForm(true)}
                >
                  New Game
                </button>
              </div>
            }
          >
            <form class="stack-form reveal-form" onSubmit={handleCreate}>
              <p class="eyebrow setup-subform-label">New game</p>
              <label class="field">
                <span>Commander name</span>
                <input
                  value={playerName()}
                  onInput={(event) => setPlayerName(event.currentTarget.value)}
                  autofocus
                />
              </label>
              <label class="field">
                <span>Empire color</span>
              </label>
              <div class="color-swatches">
                {state.playerColorOptions.colors.map((color) => (
                  <button
                    type="button"
                    class="color-swatch-button"
                    classList={{ selected: playerColor() === color }}
                    style={{ "background-color": color }}
                    title={color}
                    onClick={() => setPlayerColor(color)}
                  />
                ))}
              </div>
              <label class="field">
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
