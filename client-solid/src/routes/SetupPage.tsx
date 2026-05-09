import { useNavigate } from "@solidjs/router";
import {
  createEffect,
  createMemo,
  createSignal,
  For,
  onMount,
  Show,
} from "solid-js";

import type { ApiGameListItem, NewGameOptions } from "../lib/types/api";
import {
  getInitialSetupSelectedServerUrl,
  setSessionLastSetupServerUrl,
} from "../lib/setup-session-storage";
import { formatGameLastTurnAt, normalizeServerUrl } from "../lib/utils";
import { useAppContext } from "../state/app-context";

type ServerInfo = {
  url: string;
  status: "idle" | "checking" | "connected" | "error";
  games: ApiGameListItem[];
};

type ToggleOption<T extends string | number> = {
  label: string;
  value: T;
};

const difficultyOptions = [
  { label: "Easy", value: "easy" },
  { label: "Medium", value: "medium" },
  { label: "Hard", value: "hard" },
] satisfies ToggleOption<NewGameOptions["difficulty"]>[];

const galaxySizeOptions = [
  { label: "Tiny 15", value: "tiny" },
  { label: "Medium 30", value: "medium" },
  { label: "Huge 50", value: "huge" },
] satisfies ToggleOption<NewGameOptions["galaxy_size"]>[];

const galaxyAgeOptions = [
  { label: "Organic Rich", value: "organic_rich" },
  { label: "Average", value: "average" },
  { label: "Mineral Rich", value: "mineral_rich" },
] satisfies ToggleOption<NewGameOptions["galaxy_age"]>[];

const playerOptions = Array.from({ length: 8 }, (_, index) => {
  const value = index + 1;
  return { label: String(value), value };
}) satisfies ToggleOption<number>[];

const techLevelOptions = [
  { label: "Low", value: "low" },
  { label: "Average", value: "average" },
  { label: "Advanced", value: "advanced" },
] satisfies ToggleOption<NewGameOptions["tech_level"]>[];

const galaxySizeStars: Record<NewGameOptions["galaxy_size"], number> = {
  tiny: 15,
  medium: 30,
  huge: 50,
};

function PanelHeader(props: { title: string; count?: number }) {
  return (
    <header class="setup-panel-header">
      <h2>{props.title}</h2>
      <Show when={props.count !== undefined}>
        <span class="setup-panel-count">{props.count}</span>
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

function OptionRow<T extends string | number>(props: {
  label: string;
  options: ToggleOption<T>[];
  selected: T;
  onSelect: (value: T) => void;
}) {
  return (
    <div class="option-row">
      <span class="option-label">{props.label}</span>
      <div class="option-toggle-group" role="group" aria-label={props.label}>
        <For each={props.options}>
          {(option) => (
            <button
              type="button"
              class="option-toggle"
              classList={{ selected: props.selected === option.value }}
              aria-pressed={props.selected === option.value}
              onClick={() => props.onSelect(option.value)}
            >
              {option.label}
            </button>
          )}
        </For>
      </div>
    </div>
  );
}

export function SetupPage() {
  const navigate = useNavigate();
  const { state, connect, createGame, refreshGames, selectGame } =
    useAppContext();

  const [servers, setServers] = createSignal<ServerInfo[]>([]);
  const [selectedServerUrl, setSelectedServerUrl] = createSignal(
    getInitialSetupSelectedServerUrl(),
  );
  const [showServerForm, setShowServerForm] = createSignal(false);
  const [newServerUrl, setNewServerUrl] = createSignal("http://127.0.0.1:8080");
  const [playerName, setPlayerName] = createSignal("Admiral");
  const [difficulty, setDifficulty] = createSignal<NewGameOptions["difficulty"]>("medium");
  const [galaxySize, setGalaxySize] = createSignal<NewGameOptions["galaxy_size"]>("medium");
  const [galaxyAge, setGalaxyAge] = createSignal<NewGameOptions["galaxy_age"]>("average");
  const [players, setPlayers] = createSignal(1);
  const [techLevel, setTechLevel] = createSignal<NewGameOptions["tech_level"]>("average");
  const [race, setRace] = createSignal<NewGameOptions["race"]>("human");
  const [customRaceName, setCustomRaceName] = createSignal("");
  const [customRaceDraft, setCustomRaceDraft] = createSignal("Custom");
  const [showCustomRaceModal, setShowCustomRaceModal] = createSignal(false);
  const [playerColor, setPlayerColor] = createSignal("#FFFFFF");
  const [showNewGameForm, setShowNewGameForm] = createSignal(false);
  const [isConnecting, setIsConnecting] = createSignal(false);
  const [confirmRemove, setConfirmRemove] = createSignal<string | null>(null);

  const activeServer = createMemo(() =>
    servers().find((s) => s.url === selectedServerUrl())
  );

  const isConnected = createMemo(() => activeServer()?.status === "connected");
  const gameCount = createMemo(() => activeServer()?.games.length ?? 0);
  const trimmedCustomRaceDraft = createMemo(() => customRaceDraft().trim());
  const raceOptions = createMemo(
    () =>
      [
        { label: "Human", value: "human" },
        { label: "Psilon", value: "psilon" },
        {
          label: customRaceName()
            ? `${customRaceName()} (Custom)`
            : "Custom",
          value: "custom",
        },
      ] satisfies ToggleOption<NewGameOptions["race"]>[],
  );

  createEffect(() => {
    setSessionLastSetupServerUrl(selectedServerUrl());
  });

  createEffect(() => {
    if (selectedServerUrl()) {
      return;
    }
    const ctx = normalizeServerUrl(state.session.serverUrl);
    if (ctx) {
      setSelectedServerUrl(ctx);
    }
  });

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

  onMount(() => {
    const sel = selectedServerUrl();
    if (!sel) {
      return;
    }
    const nSel = normalizeServerUrl(sel);
    const ctx = normalizeServerUrl(state.session.serverUrl);
    if (
      state.session.connectionStatus === "connected" &&
      ctx === nSel
    ) {
      void refreshGames();
      return;
    }
    if (state.session.connectionStatus === "checking") {
      return;
    }
    setIsConnecting(true);
    void (async () => {
      try {
        await connect(nSel);
      } finally {
        setIsConnecting(false);
      }
    })();
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
      const gameId = await createGame({
        player_name: playerName(),
        player_color: playerColor(),
        num_stars: galaxySizeStars[galaxySize()],
        difficulty: difficulty(),
        galaxy_size: galaxySize(),
        galaxy_age: galaxyAge(),
        players: players(),
        tech_level: techLevel(),
        race: race(),
        ...(race() === "custom" && customRaceName()
          ? { custom_race_name: customRaceName() }
          : {}),
      });
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

  const handleSelectRace = (nextRace: NewGameOptions["race"]) => {
    if (nextRace === "custom") {
      setCustomRaceDraft(customRaceName() || "Custom");
      setShowCustomRaceModal(true);
      return;
    }
    setRace(nextRace);
  };

  const handleSaveCustomRace = () => {
    const name = trimmedCustomRaceDraft();
    if (!name) {
      return;
    }
    setCustomRaceName(name);
    setRace("custom");
    setShowCustomRaceModal(false);
  };

  return (
    <section class="setup-page">
      <div class="setup-hub">
        <div class="setup-hub-grid">
          <aside class="setup-sidebar" aria-label="Servers">
            <PanelHeader title="Servers" />

            <ul class="server-list">
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
            </ul>

            <Show when={!showServerForm()}>
              <button
                class="link-button setup-add-server"
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

          <section class="setup-games" aria-label="Games">
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
                          <div class="game-item-body">
                            <div class="game-item-line-meta">
                              <span class="game-id">{game.game_id.slice(0, 8)}</span>
                              <span class="game-meta">
                                {game.num_stars} stars · T{game.current_turn}
                              </span>
                              <span
                                class="game-last-turn"
                                title={
                                  game.last_turn_at
                                    ? "Last completed turn (your local time)"
                                    : "Game created (your local time)"
                                }
                              >
                                {formatGameLastTurnAt(
                                  game.last_turn_at ?? game.created_at,
                                )}
                              </span>
                            </div>
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
                  </div>
                  <div class="new-game-options">
                    <OptionRow
                      label="Difficulty"
                      options={difficultyOptions}
                      selected={difficulty()}
                      onSelect={setDifficulty}
                    />
                    <OptionRow
                      label="Galaxy Size"
                      options={galaxySizeOptions}
                      selected={galaxySize()}
                      onSelect={setGalaxySize}
                    />
                    <OptionRow
                      label="Galaxy Age"
                      options={galaxyAgeOptions}
                      selected={galaxyAge()}
                      onSelect={setGalaxyAge}
                    />
                    <OptionRow
                      label="Players"
                      options={playerOptions}
                      selected={players()}
                      onSelect={setPlayers}
                    />
                    <OptionRow
                      label="Tech Level"
                      options={techLevelOptions}
                      selected={techLevel()}
                      onSelect={setTechLevel}
                    />
                    <OptionRow
                      label="Race"
                      options={raceOptions()}
                      selected={race()}
                      onSelect={handleSelectRace}
                    />
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
        </div>
      </div>
      <Show when={showCustomRaceModal()}>
        <div
          class="setup-modal-backdrop"
          onClick={() => setShowCustomRaceModal(false)}
        >
          <section
            class="setup-modal"
            role="dialog"
            aria-modal="true"
            aria-labelledby="custom-race-modal-title"
            onClick={(event) => event.stopPropagation()}
          >
            <header class="setup-modal-header">
              <h2 id="custom-race-modal-title">Custom Race</h2>
            </header>
            <label class="field">
              <span>Race Name</span>
              <input
                value={customRaceDraft()}
                onInput={(event) => setCustomRaceDraft(event.currentTarget.value)}
                autofocus
              />
            </label>
            <Show when={!trimmedCustomRaceDraft()}>
              <p class="field-error">Enter a race name.</p>
            </Show>
            <div class="form-actions">
              <button
                class="link-button"
                type="button"
                onClick={() => setShowCustomRaceModal(false)}
              >
                Cancel
              </button>
              <button
                class="primary-button"
                type="button"
                onClick={handleSaveCustomRace}
                disabled={!trimmedCustomRaceDraft()}
              >
                Save
              </button>
            </div>
          </section>
        </div>
      </Show>
    </section>
  );
}
