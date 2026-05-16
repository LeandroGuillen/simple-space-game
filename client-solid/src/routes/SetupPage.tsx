import { useNavigate } from "@solidjs/router";
import {
  createEffect,
  createMemo,
  createSignal,
  For,
  onMount,
  Show,
} from "solid-js";
import { Portal } from "solid-js/web";

import type {
  ApiGameListItem,
  NewGameOptions,
  RaceDesign,
  RaceTraitId,
} from "../lib/types/api";
import {
  getCustomRaceDraft,
  setCustomRaceDraft,
} from "../lib/custom-race-storage";
import { ApiClient } from "../lib/api";
import { isPlayableCustomRaceDesign, raceSlugForNewGame, newGameRaceFieldsFromEncodedSelection } from "../lib/race-design";
import {
  getInitialSetupSelectedServerUrl,
  setSessionLastSetupServerUrl,
} from "../lib/setup-session-storage";
import { formatGameLastTurnAt, formatStardate, normalizeServerUrl } from "../lib/utils";
import { useAppContext } from "../state/app-context";

type ServerInfo = {
  url: string;
  status: "idle" | "checking" | "connected" | "error";
  games: ApiGameListItem[];
};

type ToggleOption<T extends string | number> = {
  label: string;
  value: T;
  disabled?: boolean;
  title?: string;
};

const RACE_EDITOR_VALUE = "__race_editor__" as const;

type RaceSelection = `b:${string}` | `c:${string}` | typeof RACE_EDITOR_VALUE;

const difficultyOptions = [
  { label: "Easy", value: "easy" },
  { label: "Medium", value: "medium" },
  { label: "Hard", value: "hard" },
] satisfies ToggleOption<NewGameOptions["difficulty"]>[];

const galaxySizeOptions = [
  { label: "Tiny", value: "tiny" },
  { label: "Medium", value: "medium" },
  { label: "Huge", value: "huge" },
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
  medium: 40,
  huge: 100,
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
      case "idle":
        return "Not connected";
      case "checking":
        return "Connecting...";
      case "connected":
        return "OK";
      case "error":
        return "Failed";
      default:
        return props.status;
    }
  };
  return (
    <span class={`status-dot status-dot--${props.status}`} title={label()} />
  );
}

function PlusIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
      <path
        d="M7 2v10M2 7h10"
        stroke="currentColor"
        stroke-width="1.5"
        stroke-linecap="round"
      />
    </svg>
  );
}

function ArrowIcon() {
  return (
    <svg width="10" height="10" viewBox="0 0 10 10" fill="none">
      <path
        d="M2 5h6M6 2l4 3-4 3"
        stroke="currentColor"
        stroke-width="1.3"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
    </svg>
  );
}

function OptionRow<T extends string | number>(props: {
  label: string;
  options: ToggleOption<T>[];
  selected: T | null;
  onSelect: (value: T) => void;
  linkValues?: readonly T[];
  error?: string;
}) {
  return (
    <div class="ng-row">
      <span class="ng-row-label">{props.label}</span>
      <div class="ng-row-control">
        <div class="ng-toggle-group" role="group" aria-label={props.label}>
          <For each={props.options}>
            {(option) => (
              <button
                type="button"
                class="ng-toggle"
                classList={{
                  selected: props.selected === option.value,
                  "ng-toggle--link": props.linkValues?.includes(option.value),
                }}
                disabled={option.disabled}
                title={option.title}
                aria-pressed={props.selected === option.value}
                onClick={() => props.onSelect(option.value)}
              >
                {option.label}
              </button>
            )}
          </For>
        </div>
        <Show when={props.error}>
          <p class="ng-row-error">{props.error}</p>
        </Show>
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
  const [playerName, setPlayerName] = createSignal("Player");
  const [difficulty, setDifficulty] =
    createSignal<NewGameOptions["difficulty"]>("medium");
  const [galaxySize, setGalaxySize] =
    createSignal<NewGameOptions["galaxy_size"]>("medium");
  const [galaxyAge, setGalaxyAge] =
    createSignal<NewGameOptions["galaxy_age"]>("average");
  const [players, setPlayers] = createSignal(1);
  const [techLevel, setTechLevel] =
    createSignal<NewGameOptions["tech_level"]>("average");
  const [allRaces, setAllRaces] = createSignal<RaceDesign[]>([]);
  const [race, setRace] = createSignal<RaceSelection | null>(null);
  const [customRace, setCustomRace] = createSignal<RaceDesign | null>(null);
  const [playerColor, setPlayerColor] = createSignal("");
  const [isNewGameModalOpen, setIsNewGameModalOpen] = createSignal(false);
  const [isConnecting, setIsConnecting] = createSignal(false);
  const [confirmRemove, setConfirmRemove] = createSignal<string | null>(null);
  const [validationAttempted, setValidationAttempted] = createSignal(false);

  const activeServer = createMemo(() =>
    servers().find((s) => s.url === selectedServerUrl()),
  );

  const isConnected = createMemo(() => activeServer()?.status === "connected");
  const gameCount = createMemo(() => activeServer()?.games.length ?? 0);
  const raceOptions = createMemo(
    () =>
      [
        ...allRaces()
          .filter((r) => r.built_in)
          .map((r) => ({
            label: r.playable === false ? `${r.name} (Unplayable)` : r.name,
            value: `b:${raceSlugForNewGame(r.name)}` as const,
            disabled: r.playable === false,
            title:
              r.playable === false ? "This built-in race is marked unplayable." : undefined,
          })),
        ...allRaces()
          .filter((r) => !r.built_in)
          .map((r) => ({
            label: isPlayableCustomRaceDesign(r)
              ? r.name
              : `${r.name} (Unplayable)`,
            value: `c:${r.name}` as const,
            disabled: !isPlayableCustomRaceDesign(r),
            title: !isPlayableCustomRaceDesign(r)
              ? "Edit this race before using it in a game."
              : undefined,
          })),
        {
          label: "Race Editor...",
          value: RACE_EDITOR_VALUE,
        },
      ] satisfies ToggleOption<RaceSelection>[],
  );

  const trimmedPlayerName = createMemo(() => playerName().trim());
  const isFormValid = createMemo(
    () =>
      trimmedPlayerName().length > 0 &&
      race() !== null &&
      race() !== RACE_EDITOR_VALUE &&
      playerColor().length > 0,
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
    if (
      url &&
      (status === "connected" || status === "error" || status === "checking")
    ) {
      setServers((prev) => {
        const existing = prev.find((s) => s.url === url);
        if (existing) {
          return prev.map((s) =>
            s.url === url
              ? {
                  ...s,
                  status: status as ServerInfo["status"],
                  games: status === "connected" ? state.games : s.games,
                }
              : s,
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
    const draft = getCustomRaceDraft();
    if (draft?.name.trim()) {
      setCustomRace(draft);
      setRace(`c:${draft.name}`);
    }
    const sel = selectedServerUrl();
    if (!sel) {
      return;
    }
    const nSel = normalizeServerUrl(sel);
    const ctx = normalizeServerUrl(state.session.serverUrl);
    if (state.session.connectionStatus === "connected" && ctx === nSel) {
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
    const url = state.session.serverUrl;
    if (!url || state.session.connectionStatus !== "connected") {
      return;
    }
    void (async () => {
      try {
        const client = new ApiClient(url, state.session.playerId);
        setAllRaces(await client.getRaces());
      } catch {
        // global connection flow handles server errors; custom races are optional on setup
      }
    })();
  });

  createEffect(() => {
    const colors = state.playerColorOptions.colors;
    if (colors.length === 0) {
      return;
    }
    if (!playerColor() || !colors.includes(playerColor())) {
      const red = colors.find((c) => c.toLowerCase() === "#ff3b30");
      setPlayerColor(red ?? colors[0]!);
    }
  });

  createEffect(() => {
    if (!isNewGameModalOpen()) return;
    const handler = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        closeModal();
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  });

  const closeModal = () => {
    setIsNewGameModalOpen(false);
    setValidationAttempted(false);
  };

  const openModal = () => {
    setValidationAttempted(false);
    setIsNewGameModalOpen(true);
  };

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
      setIsNewGameModalOpen(false);
      setConfirmRemove(null);
    } else {
      setConfirmRemove(url);
      setTimeout(() => setConfirmRemove(null), 3000);
    }
  };

  const handleCreate = async (event: SubmitEvent) => {
    event.preventDefault();
    setValidationAttempted(true);
    if (!isFormValid()) {
      return;
    }
    try {
      const selected = race();
      if (!selected || selected === RACE_EDITOR_VALUE) return;
      const selectedCustom =
        selected.startsWith("c:") ? allRaces().find((r) => !r.built_in && r.name === selected.slice(2)) ?? customRace() : null;
      if (
        selectedCustom &&
        !isPlayableCustomRaceDesign(selectedCustom)
      ) {
        return;
      }
      const raceFields = newGameRaceFieldsFromEncodedSelection(selected);
      const gameId = await createGame({
        player_name: trimmedPlayerName(),
        player_color: playerColor(),
        num_stars: galaxySizeStars[galaxySize()],
        difficulty: difficulty(),
        galaxy_size: galaxySize(),
        galaxy_age: galaxyAge(),
        players: players(),
        tech_level: techLevel(),
        ...raceFields,
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

  const handleSelectRace = (nextRace: RaceSelection) => {
    if (nextRace === RACE_EDITOR_VALUE) {
      navigate("/race-editor");
      return;
    }
    if (nextRace.startsWith("c:")) {
      const saved = allRaces().find((r) => `c:${r.name}` === nextRace);
      if (saved) {
        setCustomRace(saved);
        setCustomRaceDraft({
          name: saved.name,
          government: saved.government || "Dictatorship",
          trait_ids: [...(saved.trait_ids ?? [])] as RaceTraitId[],
          playable: saved.playable,
        });
      }
    }
    setRace(nextRace);
  };

  return (
    <section class="setup-page">
      <div class="setup-hub">
        <div class="setup-hub-grid">
          <aside class="setup-sidebar" aria-label="Servers">
            <PanelHeader title="Servers" />

            <ul class="server-list" classList={{ "server-list--empty": servers().length === 0 }}>
              <For each={servers()}>
                {(server) => (
                  <li class="server-item-wrap">
                    <button
                      type="button"
                      class="server-item"
                      classList={{
                        selected: selectedServerUrl() === server.url,
                      }}
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
                class="setup-add-server"
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
                  onInput={(event) =>
                    setNewServerUrl(event.currentTarget.value)
                  }
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
            <div class="setup-games-header">
              <PanelHeader title="Games" count={isConnected() ? gameCount() : undefined} />
              <Show when={isConnected() && gameCount() > 0}>
                <button
                  class="primary-button setup-new-game-btn"
                  type="button"
                  onClick={openModal}
                >
                  <PlusIcon />
                  New Game
                </button>
              </Show>
            </div>

            <Show
              when={isConnected()}
              fallback={
                <div class="empty-state">
                  <span class="empty-text">Select a server to view games</span>
                </div>
              }
            >
              <Show
                when={gameCount() > 0}
                fallback={
                  <div class="empty-state">
                    <span class="empty-text">No games on this server yet</span>
                    <button
                      class="primary-button"
                      type="button"
                      onClick={openModal}
                    >
                      <PlusIcon />
                      Create your first game
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
                              <span class="game-id">
                                {game.game_id.slice(0, 8)}
                              </span>
                              <span class="game-meta">
                                {game.num_stars} stars · T{formatStardate(game.current_turn)}
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
            </Show>
          </section>
        </div>
      </div>

      <div class="setup-landing-tools" role="group" aria-label="Design tools">
        <button
          type="button"
          class="setup-landing-tool setup-landing-tool--race"
          disabled={!isConnected()}
          onClick={() => navigate("/race-editor")}
        >
          <span class="setup-landing-tool-label">Race Editor</span>
          <span class="setup-landing-tool-hint">Design or edit custom races</span>
        </button>
        <button
          type="button"
          class="setup-landing-tool setup-landing-tool--ship"
          disabled={!isConnected()}
          onClick={() => navigate("/ship-editor")}
        >
          <span class="setup-landing-tool-label">Ship Editor</span>
          <span class="setup-landing-tool-hint">Design or edit custom ship templates</span>
        </button>
      </div>

      <Show when={isNewGameModalOpen()}>
        <Portal>
          <div
            class="ng-backdrop"
            onClick={closeModal}
            role="presentation"
          >
            <div
              class="ng-modal"
              role="dialog"
              aria-modal="true"
              aria-labelledby="ng-title"
              onClick={(event) => event.stopPropagation()}
            >
              <header class="ng-header">
                <div>
                  <h2 id="ng-title">New Game</h2>
                  <p class="ng-subtitle">Configure a fresh galaxy.</p>
                </div>
                <button
                  type="button"
                  class="ng-close"
                  onClick={closeModal}
                  aria-label="Close"
                >
                  ×
                </button>
              </header>

              <form class="ng-form" onSubmit={handleCreate}>
                <div class="ng-row">
                  <span class="ng-row-label">Commander</span>
                  <div class="ng-row-control">
                    <input
                      class="ng-input"
                      value={playerName()}
                      onInput={(event) =>
                        setPlayerName(event.currentTarget.value)
                      }
                      placeholder="Your name"
                      autofocus
                    />
                    <Show
                      when={
                        validationAttempted() &&
                        trimmedPlayerName().length === 0
                      }
                    >
                      <p class="ng-row-error">Enter a commander name.</p>
                    </Show>
                  </div>
                </div>

                <OptionRow
                  label="Race"
                  options={raceOptions()}
                  selected={race()}
                  onSelect={handleSelectRace}
                  linkValues={[RACE_EDITOR_VALUE]}
                  error={
                    validationAttempted() && race() === null
                      ? "Select a race to continue."
                      : undefined
                  }
                />
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

                <div class="ng-row">
                  <span class="ng-row-label">Color</span>
                  <div class="ng-row-control">
                    <div class="color-swatches">
                      <For each={state.playerColorOptions.colors}>
                        {(color) => (
                          <button
                            type="button"
                            class="color-swatch"
                            classList={{ selected: playerColor() === color }}
                            style={{ "background-color": color }}
                            onClick={() => setPlayerColor(color)}
                            aria-label={`Pick color ${color}`}
                          />
                        )}
                      </For>
                    </div>
                  </div>
                </div>

                <footer class="ng-footer">
                  <button
                    class="link-button"
                    type="button"
                    onClick={closeModal}
                    disabled={state.isBusy}
                  >
                    Cancel
                  </button>
                  <button
                    class="primary-button"
                    type="submit"
                    disabled={state.isBusy || !isFormValid()}
                  >
                    Create Game
                  </button>
                </footer>
              </form>
            </div>
          </div>
        </Portal>
      </Show>
    </section>
  );
}
