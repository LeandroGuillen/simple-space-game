import { useNavigate } from "@solidjs/router";
import { createSignal, onMount, Show } from "solid-js";

import { gameIdShort } from "../lib/utils";
import { useAppContext } from "../state/app-context";

export function SetupPage() {
  const navigate = useNavigate();
  const { state, connect, createGame, refreshGames, selectGame } =
    useAppContext();
  const [serverUrl, setServerUrl] = createSignal(
    state.session.serverUrl || "http://127.0.0.1:8080",
  );
  const [playerName, setPlayerName] = createSignal("Admiral");
  const [numStars, setNumStars] = createSignal(24);

  onMount(() => {
    if (state.session.serverUrl) {
      void refreshGames();
    }
  });

  const handleConnect = async (event: SubmitEvent) => {
    event.preventDefault();
    try {
      await connect(serverUrl());
    } catch {
      // error surfaced in state
    }
  };

  const handleCreate = async (event: SubmitEvent) => {
    event.preventDefault();
    try {
      const gameId = await createGame(playerName(), Number(numStars()));
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
        <p class="eyebrow">Server</p>
        <h2>Connect</h2>
        <p class="lede">
          Point this client at the Go backend, then pick or create a game below.
        </p>

        <form class="stack-form" onSubmit={handleConnect}>
          <label class="field">
            <span>Server URL</span>
            <input
              value={serverUrl()}
              onInput={(event) => setServerUrl(event.currentTarget.value)}
              placeholder="http://127.0.0.1:8080"
            />
          </label>
          <button class="primary-button" type="submit" disabled={state.isBusy}>
            {state.session.connectionStatus === "connected"
              ? "Reconnect"
              : "Connect"}
          </button>
        </form>

        <div class="metric-grid setup-metrics">
          <article class="metric-card">
            <span>State</span>
            <strong>{state.session.connectionStatus}</strong>
          </article>
          <article class="metric-card">
            <span>Known games</span>
            <strong>{state.games.length}</strong>
          </article>
          <article class="metric-card">
            <span>Selected game</span>
            <strong>
              {state.session.selectedGameId
                ? gameIdShort(state.session.selectedGameId)
                : "None"}
            </strong>
          </article>
        </div>

        <Show when={state.games.length > 0}>
          <div class="list-block">
            <p class="eyebrow">Detected games</p>
            <ul class="compact-list">
              {state.games.map((game) => (
                <li>
                  <strong>{gameIdShort(game.game_id)}</strong>
                  <span>{game.num_stars} stars</span>
                  <span>Turn {game.current_turn}</span>
                </li>
              ))}
            </ul>
          </div>
        </Show>
      </section>

      <section class="panel setup-panel">
        <p class="eyebrow">Game</p>
        <h2>Join or create</h2>
        <p class="lede">
          Enter an existing campaign or provision a new sector once the server is
          connected.
        </p>

        <Show
          when={state.session.connectionStatus === "connected"}
          fallback={
            <p class="empty-copy">
              Connect to a server using the form above before selecting a game.
            </p>
          }
        >
          <button
            class="secondary-button"
            type="button"
            onClick={() => void refreshGames()}
            disabled={state.isBusy}
          >
            Refresh game list
          </button>

          <div class="game-list">
            {state.games.map((game) => (
              <article class="game-card">
                <div>
                  <strong>{gameIdShort(game.game_id)}</strong>
                  <p>{game.num_stars} stars</p>
                  <p>Current turn {game.current_turn}</p>
                </div>
                <button
                  class="primary-button"
                  type="button"
                  onClick={() => void handleSelect(game.game_id)}
                >
                  Enter
                </button>
              </article>
            ))}
            <Show when={state.games.length === 0}>
              <p class="empty-copy">No games on this server yet.</p>
            </Show>
          </div>

          <p class="eyebrow setup-subform-label">New game</p>
          <form class="stack-form" onSubmit={handleCreate}>
            <label class="field">
              <span>Commander name</span>
              <input
                value={playerName()}
                onInput={(event) => setPlayerName(event.currentTarget.value)}
              />
            </label>
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
            <button class="primary-button" type="submit" disabled={state.isBusy}>
              Create sector
            </button>
          </form>
        </Show>
      </section>
    </section>
  );
}
