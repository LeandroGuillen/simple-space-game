import {
  A,
  useLocation,
  useNavigate,
  type RouteSectionProps,
} from "@solidjs/router";
import { For, Show, createEffect, createMemo, createSignal, onCleanup, onMount } from "solid-js";
import { Portal } from "solid-js/web";

import { APP_VERSION } from "../lib/app-meta";
import type { MovePopulationCommand } from "../lib/types/commands";
import type { GameViewModel, PlayerViewModel, TechLevelViewModel, TechSubjectViewModel } from "../lib/types/domain";
import { formatSignedBc, gameIdShort, getEmpireIncome, isPlanetSuitableForColonization } from "../lib/utils";
import { useAppContext } from "../state/app-context";

const gameNavItems = [
  { href: "/galaxy", label: "Galaxy" },
  { href: "/colonies", label: "Colonies" },
  { href: "/fleets", label: "Fleets" },
  { href: "/technology", label: "Technology" },
] as const;

const REPO_PLACEHOLDER_HREF =
  "https://github.com/leandroguillen/simple-space-game";
const HYPER_ADVANCED_RESEARCH_PREFIX = "Hyper Advanced";
const HYPER_ADVANCED_RESEARCH_COMMAND_PREFIX = `${HYPER_ADVANCED_RESEARCH_PREFIX} `;

function isGameCommandPath(pathname: string) {
  return (
    pathname.startsWith("/galaxy") ||
    pathname.startsWith("/stars") ||
    pathname.startsWith("/colonies") ||
    pathname.startsWith("/fleets") ||
    pathname.startsWith("/technology")
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
  if (pathname.startsWith("/technology")) {
    return "/technology";
  }
  return null;
}

type ResearchChoice = {
  subject: TechSubjectViewModel;
  level: TechLevelViewModel;
  commandTechName: string;
  label: string;
  techNames: string[];
  forceAll: boolean;
  isHyperAdvanced: boolean;
};

type ResearchGroup = {
  subject: TechSubjectViewModel;
  level: TechLevelViewModel;
  choices: ResearchChoice[];
};

function projectedScientists(planetId: number, baseScientists: number, moves: MovePopulationCommand[]) {
  return Math.max(
    0,
    moves
      .filter((move) => move.planetId === planetId)
      .reduce((scientists, move) => {
        if (move.to === "scientists") {
          return scientists + move.count;
        }
        if (move.from === "scientists") {
          return scientists - move.count;
        }
        return scientists;
      }, baseScientists),
  );
}

function researchPerTurn(
  game: GameViewModel | null,
  player: PlayerViewModel | null,
  moves: MovePopulationCommand[],
) {
  if (!game || !player) {
    return 0;
  }
  return game.galaxy.stars
    .flatMap((star) => star.planets)
    .filter((planet) => planet.owner === player.id)
    .reduce(
      (total, planet) =>
        total +
        projectedScientists(planet.id, planet.science.scientists, moves) * planet.science.produced_per_scientist +
        planet.science.produced_per_building,
      0,
    );
}

function isLevelComplete(player: PlayerViewModel, level: TechLevelViewModel) {
  if (level.availableTechs.length === 0) {
    return false;
  }
  const knownTechs = new Set(player.researchedTechs.map((tech) => tech.toLowerCase()));
  if (!level.forceAll) {
    return level.availableTechs.some((tech) => knownTechs.has(tech.toLowerCase()));
  }
  return level.availableTechs.every((tech) => knownTechs.has(tech.toLowerCase()));
}

function researchChoices(
  game: GameViewModel | null,
  player: PlayerViewModel | null,
  hyperAdvancedLevels: Record<string, number>,
): ResearchChoice[] {
  if (!game || !player) {
    return [];
  }
  return game.resources.techTree.subjects.flatMap((subject): ResearchChoice[] => {
    const level = subject.levels.find((candidate) => !isLevelComplete(player, candidate));
    if (!level) {
      const completedHyperAdvancedLevels = hyperAdvancedLevels[subject.name] ?? 0;
      const nextHyperAdvancedLevel = completedHyperAdvancedLevels + 1;
      const lastRealLevelCost = subject.levels.at(-1)?.cost ?? 0;
      const cost = lastRealLevelCost * 2 ** nextHyperAdvancedLevel;
      const commandTechName = `${HYPER_ADVANCED_RESEARCH_COMMAND_PREFIX}${subject.name}`;
      const label = `${HYPER_ADVANCED_RESEARCH_PREFIX} ${nextHyperAdvancedLevel}`;
      return [
        {
          subject,
          level: {
            name: label,
            cost,
            forceAll: false,
            availableTechs: [label],
          },
          commandTechName,
          label,
          techNames: [label],
          forceAll: false,
          isHyperAdvanced: true,
        },
      ];
    }
    if (level.forceAll) {
      return [
        {
          subject,
          level,
          commandTechName: level.name,
          label: level.name,
          techNames: level.availableTechs,
          forceAll: true,
          isHyperAdvanced: false,
        },
      ];
    }
    const knownTechs = new Set(player.researchedTechs.map((tech) => tech.toLowerCase()));
    return level.availableTechs
      .filter((techName) => !knownTechs.has(techName.toLowerCase()))
      .map((techName) => ({
        subject,
        level,
        commandTechName: techName,
        label: techName,
        techNames: [techName],
        forceAll: false,
        isHyperAdvanced: false,
      }));
  });
}

function researchGroups(
  game: GameViewModel | null,
  player: PlayerViewModel | null,
  hyperAdvancedLevels: Record<string, number>,
): ResearchGroup[] {
  const groups = new Map<string, ResearchGroup>();
  for (const choice of researchChoices(game, player, hyperAdvancedLevels)) {
    const key = `${choice.subject.name}\n${choice.level.name}`;
    const group = groups.get(key);
    if (group) {
      group.choices.push(choice);
    } else {
      groups.set(key, {
        subject: choice.subject,
        level: choice.level,
        choices: [choice],
      });
    }
  }
  return [...groups.values()];
}

function findResearchLevel(game: GameViewModel | null, breakthroughName: string | undefined) {
  if (!game || !breakthroughName) {
    return null;
  }
  for (const subject of game.resources.techTree.subjects) {
    const exactLevel = subject.levels.find((level) => level.name === breakthroughName);
    if (exactLevel) {
      return exactLevel;
    }
    const techLevel = subject.levels.find((level) => level.availableTechs.includes(breakthroughName));
    if (techLevel) {
      return techLevel;
    }
  }
  return null;
}

function formatResearchTurns(level: TechLevelViewModel | null, accumulatedRp: number, rpPerTurn: number) {
  if (!level) {
    return "";
  }
  if (rpPerTurn <= 0) {
    return "no research";
  }
  if (accumulatedRp >= level.cost) {
    return "ready";
  }
  const remainingCost = Math.max(0, level.cost - accumulatedRp);
  return `~${Math.max(1, Math.ceil(remainingCost / rpPerTurn))} turns`;
}

export function AppShell(props: RouteSectionProps) {
  const location = useLocation();
  const navigate = useNavigate();
  const { state, currentPlayer, enableDebugMode, endTurn, selectResearch, selectColonizeTarget, closeColonizeModal } = useAppContext();
  const [isResearchOpen, setIsResearchOpen] = createSignal(false);
  const [recentBreakthroughs, setRecentBreakthroughs] = createSignal<string[]>([]);
  let previousKnownTechs: Set<string> | null = null;
  let previousBreakthroughGameId = "";
  const sessionStatusDotClass = createMemo(
    () => `status-dot status-dot--${state.session.connectionStatus}`,
  );

  const showGameChrome = createMemo(() => isGameCommandPath(location.pathname));

  const activeCampaignNavHref = createMemo(() =>
    campaignNavActiveHref(location.pathname),
  );
  const playerResearchPerTurn = createMemo(() =>
    researchPerTurn(state.game, currentPlayer(), state.pendingCommands.movePopulation),
  );
  const empireIncome = createMemo(() =>
    getEmpireIncome(state.game, currentPlayer()?.id ?? state.session.playerId, state.pendingCommands),
  );
  const queuedResearchName = createMemo(
    () => state.pendingCommands.selectResearch?.tech_name,
  );
  const activeResearchName = createMemo(() => {
    const currentResearch = currentPlayer()?.currentResearch;
    return (
      queuedResearchName() ||
      currentResearch?.techName ||
      currentResearch?.levelName ||
      currentResearch?.subject
    );
  });
  const hyperAdvancedLevels = createMemo(() => currentPlayer()?.hyperAdvanced ?? {});
  const colonizeModalPlanets = createMemo(() => {
    const game = state.game;
    if (!game) return [];
    return game.galaxy.stars.flatMap((star) => star.planets.filter(isPlanetSuitableForColonization));
  });
  const availableResearchGroups = createMemo(() =>
    researchGroups(state.game, currentPlayer(), hyperAdvancedLevels()),
  );
  const activeResearchLevel = createMemo(() =>
    findResearchLevel(state.game, activeResearchName()) ??
    availableResearchGroups()
      .flatMap((group) => group.choices)
      .find((choice) => choice.commandTechName === activeResearchName())?.level ??
    null,
  );
  const hasSelectedResearch = createMemo(() => Boolean(activeResearchName()));
  const researchTurnsText = createMemo(() =>
    formatResearchTurns(
      activeResearchLevel(),
      currentPlayer()?.researchPoints ?? 0,
      playerResearchPerTurn(),
    ),
  );

  createEffect(() => {
    if (isGameCommandPath(location.pathname) && !state.session.selectedGameId) {
      navigate("/setup", { replace: true });
    }
  });

  createEffect(() => {
    const gameId = state.game?.id ?? "";
    const player = currentPlayer();
    const knownTechs = player?.researchedTechs ?? [];
    if (!gameId || !player) {
      previousKnownTechs = null;
      previousBreakthroughGameId = "";
      return;
    }
    if (previousBreakthroughGameId !== gameId || previousKnownTechs === null) {
      previousBreakthroughGameId = gameId;
      previousKnownTechs = new Set(knownTechs);
      return;
    }

    const newTechs = knownTechs.filter((techName) => !previousKnownTechs?.has(techName));
    previousKnownTechs = new Set(knownTechs);
    if (newTechs.length === 0) {
      return;
    }

    setRecentBreakthroughs(newTechs);
    if (!player.currentResearch && !queuedResearchName()) {
      setIsResearchOpen(true);
    }
  });

  onMount(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setIsResearchOpen(false);
      }
    };
    document.addEventListener("keydown", handleKeyDown);
    onCleanup(() => document.removeEventListener("keydown", handleKeyDown));
  });

  function chooseResearch(choice: ResearchChoice) {
    selectResearch({ tech_name: choice.commandTechName });
    setRecentBreakthroughs([]);
    setIsResearchOpen(false);
  }

  function handleEndTurn() {
    if (!hasSelectedResearch()) {
      setIsResearchOpen(true);
      return;
    }
    void endTurn();
  }

  return (
    <div class="frame">
      <header class="app-topbar panel" aria-label="Application">
        <div class="app-topbar-brand">
          <span class="app-title">Simple Space Game</span>
          <span class="app-version" title="Client version">
            v{APP_VERSION}
          </span>
        </div>
        <Show when={showGameChrome()}>
          <A
            href="/setup"
            class="app-topbar-session"
            title="Open server and game setup"
            aria-label={`Session: ${state.session.serverUrl ?? "no server"}${state.session.selectedGameId ? `, game ${gameIdShort(state.session.selectedGameId)}` : ""}`}
          >
            <span class={sessionStatusDotClass()} aria-hidden="true" />
            <Show when={state.session.serverUrl}>
              <span class="app-topbar-session-url">
                {state.session.serverUrl}
              </span>
            </Show>
            <Show
              when={
                Boolean(state.session.serverUrl) &&
                Boolean(state.session.selectedGameId)
              }
            >
              <span class="app-topbar-session-sep" aria-hidden="true">
                ·
              </span>
            </Show>
            <Show when={state.session.selectedGameId}>
              <span class="app-topbar-session-game">
                {gameIdShort(state.session.selectedGameId)}
              </span>
            </Show>
          </A>
        </Show>
        <div class="app-topbar-end">
          <a
            class="app-repo-link"
            href={REPO_PLACEHOLDER_HREF}
            target="_blank"
            rel="noreferrer"
          >
            github
          </a>
          <button
            type="button"
            class="app-debug-link"
            onClick={() => void enableDebugMode()}
            disabled={state.debugMode || state.isBusy}
          >
            debug mode: {state.debugMode ? "on" : "off"}
          </button>
        </div>
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

                    <div
                      class="turn-stat"
                      classList={{ "turn-stat--negative": empireIncome() < 0 }}
                    >
                      <span>Income</span>
                      <strong>{formatSignedBc(empireIncome())}</strong>
                    </div>

                    <button
                      type="button"
                      class="turn-stat research-topbar-button"
                      onClick={() => setIsResearchOpen(true)}
                    >
                      <span>Research</span>
                      <strong>
                        {playerResearchPerTurn()}RP
                        <Show when={researchTurnsText()}>
                          {" "}{researchTurnsText()}
                        </Show>
                      </strong>
                    </button>

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
                      onClick={handleEndTurn}
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

      <Show when={isResearchOpen()}>
        <Portal>
          <div class="research-modal-backdrop" onClick={() => setIsResearchOpen(false)}>
            <section
              class="research-modal panel"
              role="dialog"
              aria-modal="true"
              aria-labelledby="research-modal-title"
              onClick={(event) => event.stopPropagation()}
            >
              <div class="research-modal-header">
                <div>
                  <p class="eyebrow">Research</p>
                  <h2 id="research-modal-title">Select Project</h2>
                </div>
                <button class="secondary-button research-close-button" type="button" onClick={() => setIsResearchOpen(false)}>
                  Close
                </button>
              </div>

              <Show when={recentBreakthroughs().length > 0}>
                <div class="research-breakthrough-notice">
                  <span>Breakthrough</span>
                  <strong>{recentBreakthroughs().join(", ")}</strong>
                </div>
              </Show>

              <div class="research-subject-grid">
                <For each={availableResearchGroups()}>
                  {(group) => (
                    <article
                      class="research-subject-card"
                      classList={{
                        "is-selected": group.choices.some((choice) => activeResearchName() === choice.commandTechName),
                      }}
                    >
                      <span class="research-subject-card-header">
                        <span>{group.subject.name}</span>
                        <strong>{group.level.cost}RP</strong>
                      </span>
                      <Show when={!group.choices.every((choice) => choice.isHyperAdvanced)}>
                        <span class="research-level-name">{group.level.name}</span>
                      </Show>
                      <div class="research-tech-list">
                        <For each={group.choices}>
                          {(choice) => (
                            <button
                              type="button"
                              class="research-tech-option"
                              classList={{
                                "is-selected": activeResearchName() === choice.commandTechName,
                                "is-bundle": choice.forceAll,
                                "is-hyper-advanced": choice.isHyperAdvanced,
                              }}
                              disabled={state.isBusy}
                              onClick={() => chooseResearch(choice)}
                            >
                              <For each={choice.techNames}>
                                {(techName) => <span class="research-tech-option-item">{techName}</span>}
                              </For>
                            </button>
                          )}
                        </For>
                      </div>
                    </article>
                  )}
                </For>
              </div>
            </section>
          </div>
        </Portal>
      </Show>

      <Show when={state.showColonizeModal}>
        <Portal>
          <div class="research-modal-backdrop" onClick={closeColonizeModal}>
            <section
              class="research-modal panel"
              role="dialog"
              aria-modal="true"
              aria-labelledby="colonize-modal-title"
              onClick={(event) => event.stopPropagation()}
            >
              <div class="research-modal-header">
                <div>
                  <p class="eyebrow">Colony Base Ready</p>
                  <h2 id="colonize-modal-title">Select Target Planet</h2>
                </div>
                <button class="secondary-button research-close-button" type="button" onClick={closeColonizeModal}>
                  Close
                </button>
              </div>

              <Show
                when={colonizeModalPlanets().length > 0}
                fallback={
                  <div class="research-breakthrough-notice">
                    <span>No suitable planets</span>
                    <strong>Colony Base lost</strong>
                  </div>
                }
              >
                <div class="research-subject-grid">
                  <For each={state.game?.galaxy.stars ?? []}>
                    {(star) => {
                      const starPlanets = () => star.planets.filter(isPlanetSuitableForColonization);
                      return (
                        <Show when={starPlanets().length > 0}>
                          <article class="research-subject-card">
                            <span class="research-subject-card-header">
                              <span>{star.name}</span>
                              <strong>{starPlanets().length} planet{starPlanets().length > 1 ? 's' : ''}</strong>
                            </span>
                            <div class="research-tech-list">
                              <For each={starPlanets()}>
                                {(planet) => (
                                  <button
                                    type="button"
                                    class="research-tech-option"
                                    onClick={() => selectColonizeTarget(planet.id)}
                                  >
                                    <span class="research-tech-option-item">{planet.name}</span>
                                  </button>
                                )}
                              </For>
                            </div>
                          </article>
                        </Show>
                      );
                    }}
                  </For>
                </div>
              </Show>
            </section>
          </div>
        </Portal>
      </Show>
    </div>
  );
}
