import {
  A,
  useLocation,
  useNavigate,
  type RouteSectionProps,
} from "@solidjs/router";
import {
  For,
  Show,
  createEffect,
  createMemo,
  createSignal,
  onCleanup,
  onMount,
} from "solid-js";
import { Portal } from "solid-js/web";

import { APP_VERSION } from "../lib/app-meta";
import { recordPlanetReturnCandidate } from "../lib/navigation-history";
import { createShortcut, withShortcutHint } from "../lib/shortcuts";
import type { MovePopulationCommand } from "../lib/types/commands";
import type {
  GameViewModel,
  PlayerViewModel,
  TechLevelViewModel,
  TechSubjectViewModel,
} from "../lib/types/domain";
import {
  formatCommandCapacityTooltip,
  formatCommandPointsDisplay,
  formatEmpireFoodTooltip,
  formatEmpireIncomeTooltip,
  formatFreighterDisplay,
  formatFreighterTooltip,
  formatInteger,
  formatSigned,
  formatSignedBc,
  formatStardate,
  formatTreasury,
  getEmpireFoodBalance,
  getEmpireIncome,
  getProjectedFoodFreightersUsed,
  getPlanetScienceForScientists,
  getArtificialPlanetTargets,
  findPlanet,
  isPlanetSuitableForColonization,
  isArtificialPlanetTarget,
  labelForEnvironment,
  labelForGravity,
  labelForMinerals,
  labelForSize,
} from "../lib/utils";
import { useAppContext } from "../state/app-context";
import { PlanetVisual } from "./PlanetVisual";

const gameNavItems = [
  { href: "/galaxy", label: "Galaxy", shortcutKey: "g" },
  { href: "/colonies", label: "Colonies", shortcutKey: "c" },
  { href: "/fleets", label: "Fleets", shortcutKey: "f" },
  { href: "/technology", label: "Technology", shortcutKey: "r" },
] as const;

const END_TURN_SHORTCUT = "t";

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

/** Which campaign navigation href should show as active (star/planet views belong to Galaxy). */
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

function projectedScientists(
  planetId: number,
  baseScientists: number,
  moves: MovePopulationCommand[],
) {
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
        getPlanetScienceForScientists(
          planet,
          projectedScientists(planet.id, planet.science.scientists, moves),
        ),
      0,
    );
}

function isLevelComplete(player: PlayerViewModel, level: TechLevelViewModel) {
  if (level.availableTechs.length === 0) {
    return false;
  }
  const knownTechs = new Set(
    player.researchedTechs.map((tech) => tech.toLowerCase()),
  );
  if (!level.forceAll) {
    return level.availableTechs.some((tech) =>
      knownTechs.has(tech.toLowerCase()),
    );
  }
  return level.availableTechs.every((tech) =>
    knownTechs.has(tech.toLowerCase()),
  );
}

function researchChoices(
  game: GameViewModel | null,
  player: PlayerViewModel | null,
  hyperAdvancedLevels: Record<string, number>,
): ResearchChoice[] {
  if (!game || !player) {
    return [];
  }
  return game.resources.techTree.subjects.flatMap(
    (subject): ResearchChoice[] => {
      const level = subject.levels.find(
        (candidate) => !isLevelComplete(player, candidate),
      );
      if (!level) {
        const completedHyperAdvancedLevels =
          hyperAdvancedLevels[subject.name] ?? 0;
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
      const knownTechs = new Set(
        player.researchedTechs.map((tech) => tech.toLowerCase()),
      );
      return level.availableTechs
        .filter((techName) => !knownTechs.has(techName.toLowerCase()))
        .filter((techName) => isResearchChoiceAllowedForGovernment(game, player, techName))
        .map((techName) => ({
          subject,
          level,
          commandTechName: techName,
          label: techName,
          techNames: [techName],
          forceAll: false,
          isHyperAdvanced: false,
        }));
    },
  );
}

function isResearchChoiceAllowedForGovernment(
  game: GameViewModel,
  player: PlayerViewModel,
  techName: string,
) {
  const tech = game.resources.techCollection.techs.find((item) => item.name === techName);
  if (!tech) return true;
  const active = activeGovernmentForPlayer(game, player);
  if (tech.type === "government") {
    return Boolean(active && tech.upgrades.includes(active.name));
  }
  if (active?.grants.modifiers.some((modifier) => modifier.name === "morale_neutral")) {
    return !tech.grants.unlocks.buildings.some((buildingName) => {
      const building = game.resources.buildingCollection.buildings.find((item) => item.name === buildingName);
      return building?.grants.modifiers.some((modifier) => modifier.name === "morale_bonus");
    });
  }
  return true;
}

function activeGovernmentForPlayer(game: GameViewModel, player: PlayerViewModel) {
  let active =
    game.resources.techCollection.techs.find((tech) => tech.type === "government" && tech.name === player.raceGovernment) ??
    game.resources.techCollection.techs.find((tech) => tech.type === "government" && tech.default);
  if (!active) return null;
  const researched = new Set(player.researchedTechs);
  const seen = new Set([active.name]);
  let changed = true;
  while (changed) {
    changed = false;
    for (const tech of game.resources.techCollection.techs) {
      if (tech.type !== "government" || tech.upgrades.length === 0 || !researched.has(tech.name)) {
        continue;
      }
      if (!tech.upgrades.includes(active.name)) {
        continue;
      }
      if (seen.has(tech.name)) {
        return active;
      }
      seen.add(tech.name);
      active = tech;
      changed = true;
      break;
    }
  }
  return active;
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

function findResearchLevel(
  game: GameViewModel | null,
  breakthroughName: string | undefined,
) {
  if (!game || !breakthroughName) {
    return null;
  }
  for (const subject of game.resources.techTree.subjects) {
    const exactLevel = subject.levels.find(
      (level) => level.name === breakthroughName,
    );
    if (exactLevel) {
      return exactLevel;
    }
    const techLevel = subject.levels.find((level) =>
      level.availableTechs.includes(breakthroughName),
    );
    if (techLevel) {
      return techLevel;
    }
  }
  return null;
}

function formatResearchTurns(
  level: TechLevelViewModel | null,
  accumulatedRp: number,
  rpPerTurn: number,
) {
  if (!level) {
    return "";
  }
  if (rpPerTurn <= 0) {
    return "";
  }
  if (accumulatedRp >= level.cost) {
    return "ready";
  }
  const remainingCost = Math.max(0, level.cost - accumulatedRp);
  return `~${Math.max(1, Math.ceil(remainingCost / rpPerTurn))} turns`;
}

function gameSwitcherPlayerLabel(
  game: { players: { name: string; race_name?: string }[] },
  fallbackRaceName?: string,
) {
  const player = game.players[0];
  if (!player) {
    return "Unassigned";
  }
  const raceName = player.race_name || fallbackRaceName;
  return raceName ? `${player.name} · ${raceName}` : player.name;
}

export function AppShell(props: RouteSectionProps) {
  const location = useLocation();
  const navigate = useNavigate();
  const {
    state,
    currentPlayer,
    enableDebugMode,
    endTurn,
    selectGame,
    selectResearch,
    selectColonizeTarget,
    closeColonizeModal,
    selectArtificialPlanetTarget,
    closeArtificialPlanetModal,
  } = useAppContext();
  const [isResearchOpen, setIsResearchOpen] = createSignal(false);
  const [recentBreakthroughs, setRecentBreakthroughs] = createSignal<string[]>(
    [],
  );
  let previousKnownTechs: Set<string> | null = null;
  let previousBreakthroughGameId = "";
  const sessionStatusDotClass = createMemo(
    () => `status-dot status-dot--${state.session.connectionStatus}`,
  );
  const quickSwitchGames = createMemo(() => state.games.slice(0, 3));
  const quickSwitchLabel = (game: {
    game_id: string;
    players: { name: string; race_name?: string }[];
  }) =>
    gameSwitcherPlayerLabel(
      game,
      game.game_id === state.game?.id ? currentPlayer()?.raceName : undefined,
    );

  const showGameChrome = createMemo(() => isGameCommandPath(location.pathname));

  const activeCampaignNavHref = createMemo(() =>
    campaignNavActiveHref(location.pathname),
  );
  const playerResearchPerTurn = createMemo(() =>
    researchPerTurn(
      state.game,
      currentPlayer(),
      state.pendingCommands.movePopulation,
    ),
  );
  const empireIncome = createMemo(() =>
    getEmpireIncome(
      state.game,
      currentPlayer()?.id ?? state.session.playerId,
      state.pendingCommands,
      currentPlayer() ?? null,
    ),
  );
  const empireIncomeTitle = createMemo(() =>
    formatEmpireIncomeTooltip(
      state.game,
      currentPlayer() ?? null,
      state.pendingCommands,
    ),
  );
  const empireFoodBalance = createMemo(() =>
    getEmpireFoodBalance(
      state.game,
      currentPlayer()?.id ?? state.session.playerId,
      state.pendingCommands,
    ),
  );
  const empireFoodTitle = createMemo(() =>
    formatEmpireFoodTooltip(
      state.game,
      currentPlayer() ?? null,
      state.pendingCommands,
    ),
  );
  const commandCapacityTitle = createMemo(() =>
    formatCommandCapacityTooltip(state.game, currentPlayer() ?? null),
  );
  const projectedFoodFreightersUsed = createMemo(() =>
    getProjectedFoodFreightersUsed(
      state.game,
      currentPlayer()?.id ?? state.session.playerId,
      state.pendingCommands,
      currentPlayer() ?? null,
    ),
  );
  const freighterTitle = createMemo(() =>
    formatFreighterTooltip(currentPlayer() ?? null, projectedFoodFreightersUsed()),
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
  const hyperAdvancedLevels = createMemo(
    () => currentPlayer()?.hyperAdvanced ?? {},
  );
  const playerNameById = createMemo(
    () =>
      new Map(
        (state.game?.players ?? []).map((player) => [player.id, player.name]),
      ),
  );
  const playerColorById = createMemo(
    () =>
      new Map(
        (state.game?.players ?? []).map((player) => [player.id, player.color]),
      ),
  );
  const colonizeModalPlanets = createMemo(() => {
    const game = state.game;
    if (!game) return [];
    const sourceStarIds = new Set([
      ...state.pendingColonyBasePlanets
        .map((planetId) => findPlanet(game, planetId)?.parentStar)
        .filter((starId): starId is number => typeof starId === "number"),
      ...state.pendingColonyShipFleets
        .map(
          (fleetId) =>
            game.fleets.find((fleet) => fleet.id === fleetId)?.starId,
        )
        .filter((starId): starId is number => typeof starId === "number"),
    ]);
    return game.galaxy.stars
      .filter((star) => sourceStarIds.has(star.id))
      .flatMap((star) => star.planets.filter(isPlanetSuitableForColonization));
  });
  const colonizeModalStars = createMemo(() => {
    const game = state.game;
    if (!game) return [];
    const sourceStarIds = new Set([
      ...state.pendingColonyBasePlanets
        .map((planetId) => findPlanet(game, planetId)?.parentStar)
        .filter((starId): starId is number => typeof starId === "number"),
      ...state.pendingColonyShipFleets
        .map(
          (fleetId) =>
            game.fleets.find((fleet) => fleet.id === fleetId)?.starId,
        )
        .filter((starId): starId is number => typeof starId === "number"),
    ]);
    return game.galaxy.stars.filter(
      (star) =>
        sourceStarIds.has(star.id) &&
        star.planets.some(isPlanetSuitableForColonization),
    );
  });
  const artificialPlanetModalStars = createMemo(() => {
    const game = state.game;
    if (!game) return [];
    const sourceStarIds = new Set(
      state.pendingArtificialPlanetSourcePlanets
        .map((planetId) => findPlanet(game, planetId)?.parentStar)
        .filter((starId): starId is number => typeof starId === "number"),
    );
    return game.galaxy.stars.filter(
      (star) =>
        sourceStarIds.has(star.id) &&
        getArtificialPlanetTargets(star).length > 0,
    );
  });
  const artificialPlanetModalTargets = createMemo(() =>
    artificialPlanetModalStars().flatMap((star) =>
      star.planets.filter(isArtificialPlanetTarget),
    ),
  );
  const artificialPlanetSourceForTarget = (targetParentStar: number) => {
    const game = state.game;
    if (!game) return null;
    return (
      state.pendingArtificialPlanetSourcePlanets.find(
        (sourcePlanetId) =>
          findPlanet(game, sourcePlanetId)?.parentStar === targetParentStar,
      ) ?? null
    );
  };
  const availableResearchGroups = createMemo(() =>
    researchGroups(state.game, currentPlayer(), hyperAdvancedLevels()),
  );
  const activeResearchLevel = createMemo(
    () =>
      findResearchLevel(state.game, activeResearchName()) ??
      availableResearchGroups()
        .flatMap((group) => group.choices)
        .find((choice) => choice.commandTechName === activeResearchName())
        ?.level ??
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
      navigate("/", { replace: true });
    }
  });

  createEffect(() => {
    recordPlanetReturnCandidate(location.pathname);
  });

  createShortcut(
    END_TURN_SHORTCUT,
    () => {
      if (!showGameChrome() || state.isBusy) return;
      handleEndTurn();
    },
    { when: () => !isResearchOpen() },
  );

  for (const item of gameNavItems) {
    createShortcut(
      item.shortcutKey,
      () => {
        if (!showGameChrome()) return;
        if (location.pathname === item.href) return;
        navigate(item.href);
      },
      { when: () => !isResearchOpen() },
    );
  }

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

    const newTechs = knownTechs.filter(
      (techName) => !previousKnownTechs?.has(techName),
    );
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

  async function handleQuickSwitchGame(gameId: string) {
    if (gameId === state.session.selectedGameId) {
      return;
    }
    await selectGame(gameId);
    navigate("/galaxy");
  }

  return (
    <div class="frame">
      <header class="app-topbar panel" aria-label="Application">
        <A
          href="/"
          class="app-topbar-brand"
          title="Home"
          aria-label="Home"
        >
          <span class="app-title">Simple Space Game</span>
          <span class="app-version" title="Client version">
            v{APP_VERSION}
          </span>
        </A>
        <Show when={showGameChrome()}>
          <div
            class="app-topbar-session"
            aria-label={`Server: ${state.session.serverUrl ?? "no server"}`}
          >
            <span class={sessionStatusDotClass()} aria-hidden="true" />
            <Show when={state.session.serverUrl}>
              <span class="app-topbar-session-url">
                {state.session.serverUrl}
              </span>
            </Show>
          </div>
        </Show>
        <Show when={showGameChrome() && quickSwitchGames().length > 1}>
          <div class="app-game-switcher" aria-label="Recent games">
            <For each={quickSwitchGames()}>
              {(game) => (
                <button
                  type="button"
                  class="app-game-switcher-button"
                  classList={{
                    "app-game-switcher-button--active":
                      game.game_id === state.session.selectedGameId,
                  }}
                  title={`${quickSwitchLabel(game)} · T${formatStardate(game.current_turn)} · ${game.game_id}`}
                  aria-label={`Switch to ${quickSwitchLabel(game)} at stardate ${formatStardate(game.current_turn)}`}
                  aria-current={
                    game.game_id === state.session.selectedGameId
                      ? "true"
                      : undefined
                  }
                  disabled={state.isBusy}
                  onClick={() => void handleQuickSwitchGame(game.game_id)}
                >
                  <span class="app-game-switcher-player">
                    {quickSwitchLabel(game)}
                  </span>
                  <span class="app-game-switcher-turn">
                    T{formatStardate(game.current_turn)}
                  </span>
                </button>
              )}
            </For>
          </div>
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
          <nav class="campaign-nav panel" aria-label="Campaign">
            <div class="campaign-nav-inner">
              <div class="campaign-nav-list" aria-label="Game commands">
                <For each={gameNavItems}>
                  {(item) => (
                    <A
                      href={item.href}
                      class="campaign-nav-link"
                      classList={{
                        active: activeCampaignNavHref() === item.href,
                      }}
                      aria-current={
                        activeCampaignNavHref() === item.href
                          ? "page"
                          : undefined
                      }
                      title={withShortcutHint(item.label, item.shortcutKey)}
                    >
                      {item.label}
                      <span class="campaign-nav-link-shortcut" aria-hidden="true">
                        {item.shortcutKey.toUpperCase()}
                      </span>
                    </A>
                  )}
                </For>
              </div>
            </div>
          </nav>

          <div class="game-main">
            <Show when={state.game}>
              <header class="game-topbar panel" aria-label="Current game">
                <div class="turn-panel">
                  <div class="turn-meta">
                    <div
                      class="turn-stat"
                      classList={{ "turn-stat--negative": empireIncome() < 0 }}
                      title={empireIncomeTitle()}
                    >
                      <span>Income</span>
                      <strong>{formatSignedBc(empireIncome())}</strong>
                    </div>

                    <div class="turn-stat turn-stat--accent">
                      <span>Treasury</span>
                      <strong>
                        {formatTreasury(currentPlayer()?.treasury ?? 0)} BC
                      </strong>
                    </div>

                    <div
                      class="turn-stat"
                      classList={{
                        "turn-stat--negative": empireFoodBalance() < 0,
                      }}
                      title={empireFoodTitle()}
                    >
                      <span>Food</span>
                      <strong>{formatSigned(empireFoodBalance())}</strong>
                    </div>

                    <div
                      class="turn-stat"
                      classList={{
                        "turn-stat--negative":
                          projectedFoodFreightersUsed() >
                          Math.max(
                            0,
                            (currentPlayer()?.freighters ?? 0) -
                              (currentPlayer()?.freightersUsed ?? 0),
                          ),
                      }}
                      title={freighterTitle()}
                    >
                      <span>Freighters</span>
                      <strong>
                        {formatFreighterDisplay(
                          currentPlayer() ?? null,
                          projectedFoodFreightersUsed(),
                        )}
                      </strong>
                    </div>

                    <button
                      type="button"
                      class="turn-stat research-topbar-button"
                      onClick={() => setIsResearchOpen(true)}
                    >
                      <span>Research</span>
                      <strong>
                        {formatInteger(playerResearchPerTurn())}RP
                        <Show when={researchTurnsText()}>
                          {" "}
                          {researchTurnsText()}
                        </Show>
                      </strong>
                    </button>

                    <div
                      class="turn-stat"
                      classList={{
                        "turn-stat--negative":
                          (currentPlayer()?.commandPointsRemaining ?? 0) < 0,
                      }}
                      title={commandCapacityTitle()}
                    >
                      <span>COMMAND</span>
                      <strong>
                        {formatCommandPointsDisplay(currentPlayer() ?? null)}
                      </strong>
                    </div>

                    <div class="turn-stat">
                      <span>Stardate</span>
                      <strong>{formatStardate(state.game?.currentTurn)}</strong>
                    </div>
                  </div>
                  <div class="turn-actions">
                    <button
                      type="button"
                      class="primary-button game-turn-button"
                      onClick={handleEndTurn}
                      disabled={state.isBusy}
                      aria-busy={state.isBusy}
                      title={withShortcutHint("End Turn", END_TURN_SHORTCUT)}
                    >
                      End Turn
                      <span class="game-turn-button-shortcut" aria-hidden="true">
                        {END_TURN_SHORTCUT.toUpperCase()}
                      </span>
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
          <div
            class="research-modal-backdrop"
            onClick={() => setIsResearchOpen(false)}
          >
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
                <button
                  class="secondary-button research-close-button"
                  type="button"
                  onClick={() => setIsResearchOpen(false)}
                >
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
                        "is-selected": group.choices.some(
                          (choice) =>
                            activeResearchName() === choice.commandTechName,
                        ),
                      }}
                    >
                      <span class="research-subject-card-header">
                        <span>{group.subject.name}</span>
                        <strong>{group.level.cost}RP</strong>
                      </span>
                      <Show
                        when={
                          !group.choices.every(
                            (choice) => choice.isHyperAdvanced,
                          )
                        }
                      >
                        <span class="research-level-name">
                          {group.level.name}
                        </span>
                      </Show>
                      <div class="research-tech-list">
                        <For each={group.choices}>
                          {(choice) => (
                            <button
                              type="button"
                              class="research-tech-option"
                              classList={{
                                "is-selected":
                                  activeResearchName() ===
                                  choice.commandTechName,
                                "is-bundle": choice.forceAll,
                                "is-hyper-advanced": choice.isHyperAdvanced,
                              }}
                              disabled={state.isBusy}
                              onClick={() => chooseResearch(choice)}
                            >
                              <For each={choice.techNames}>
                                {(techName) => (
                                  <span class="research-tech-option-item">
                                    {techName}
                                  </span>
                                )}
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
                  <p class="eyebrow">
                    {state.pendingColonyShipFleets.length > 0
                      ? "Colony Ship Ready"
                      : "Colony Base Ready"}
                  </p>
                  <h2 id="colonize-modal-title">Select Target Planet</h2>
                </div>
                <button
                  class="secondary-button research-close-button"
                  type="button"
                  onClick={closeColonizeModal}
                >
                  Close
                </button>
              </div>

              <Show
                when={colonizeModalPlanets().length > 0}
                fallback={
                  <div class="research-breakthrough-notice">
                    <span>No suitable planets for colonization</span>
                    <strong>
                      {state.pendingColonyShipFleets.length > 0
                        ? "Colony Ship retained"
                        : "Colony Base lost"}
                    </strong>
                  </div>
                }
              >
                <div class="colonize-system-grid">
                  <For each={colonizeModalStars()}>
                    {(star) => {
                      return (
                        <article
                          class="colonize-system-panel"
                          aria-label={star.name}
                        >
                          <div class="planet-list">
                            <For each={star.planets}>
                              {(planet) => {
                                const eligible = () =>
                                  isPlanetSuitableForColonization(planet);
                                return (
                                  <button
                                    type="button"
                                    class="planet-card colonize-planet-card"
                                    classList={{ "is-eligible": eligible() }}
                                    disabled={!eligible()}
                                    onClick={() =>
                                      selectColonizeTarget(planet.id)
                                    }
                                  >
                                    <div class="planet-card-visual">
                                      <PlanetVisual
                                        planetId={planet.id}
                                        environment={planet.environment}
                                        size="small"
                                      />
                                    </div>
                                    <div class="planet-card-info">
                                      <p class="planet-title">
                                        <span
                                          style={{
                                            color:
                                              planet.owner > 0
                                                ? (playerColorById().get(
                                                    planet.owner,
                                                  ) ?? "var(--text)")
                                                : "var(--text)",
                                          }}
                                        >
                                          {planet.name}
                                        </span>
                                      </p>
                                      <p>
                                        {labelForEnvironment(
                                          planet.environment,
                                        )}{" "}
                                        · {labelForSize(planet.size)} ·{" "}
                                        {labelForMinerals(planet.mineralClass)}
                                      </p>
                                      <p>
                                        Gravity:{" "}
                                        {labelForGravity(planet.gravity)}
                                        <Show when={planet.owner > 0}>
                                          {" "}
                                          · Population {planet.population}/
                                          {planet.maxPopulation}
                                        </Show>
                                      </p>
                                      <p>
                                        {planet.owner > 0
                                          ? `Controlled by ${playerNameById().get(planet.owner) ?? `Player ${planet.owner}`}`
                                          : eligible()
                                            ? "Eligible colonization target"
                                            : "Not suitable for colonization"}
                                      </p>
                                    </div>
                                    <Show when={eligible()}>
                                      <span class="game-card-cue">Select</span>
                                    </Show>
                                  </button>
                                );
                              }}
                            </For>
                          </div>
                        </article>
                      );
                    }}
                  </For>
                </div>
              </Show>
            </section>
          </div>
        </Portal>
      </Show>

      <Show when={state.showArtificialPlanetModal}>
        <Portal>
          <div class="research-modal-backdrop" onClick={closeArtificialPlanetModal}>
            <section
              class="research-modal panel"
              role="dialog"
              aria-modal="true"
              aria-labelledby="artificial-planet-modal-title"
              onClick={(event) => event.stopPropagation()}
            >
              <div class="research-modal-header">
                <div>
                  <p class="eyebrow">System Inspector</p>
                  <h2 id="artificial-planet-modal-title">
                    Select Artificial Planet Target
                  </h2>
                </div>
                <button
                  class="secondary-button research-close-button"
                  type="button"
                  onClick={closeArtificialPlanetModal}
                >
                  Close
                </button>
              </div>

              <Show
                when={artificialPlanetModalTargets().length > 0}
                fallback={
                  <div class="research-breakthrough-notice">
                    <span>No valid artificial planet targets</span>
                    <strong>Production lost</strong>
                  </div>
                }
              >
                <div class="colonize-system-grid">
                  <For each={artificialPlanetModalStars()}>
                    {(star) => (
                      <article
                        class="colonize-system-panel"
                        aria-label={star.name}
                      >
                        <div class="planet-list">
                          <For each={star.planets.filter(isArtificialPlanetTarget)}>
                            {(planet) => {
                              const sourcePlanetId = () =>
                                artificialPlanetSourceForTarget(planet.parentStar);
                              const resultLabel = () =>
                                planet.environment === 1
                                  ? "Creates Huge Barren"
                                  : "Creates Large Barren";
                              return (
                                <button
                                  type="button"
                                  class="planet-card colonize-planet-card is-eligible"
                                  disabled={!sourcePlanetId()}
                                  onClick={() => {
                                    const sourceId = sourcePlanetId();
                                    if (sourceId) {
                                      selectArtificialPlanetTarget(
                                        sourceId,
                                        planet.id,
                                      );
                                    }
                                  }}
                                >
                                  <div class="planet-card-visual">
                                    <PlanetVisual
                                      planetId={planet.id}
                                      environment={planet.environment}
                                      size="small"
                                    />
                                  </div>
                                  <div class="planet-card-info">
                                    <p class="planet-title">
                                      <span>{planet.name}</span>
                                    </p>
                                    <p>
                                      {labelForEnvironment(planet.environment)}{" "}
                                      · {labelForSize(planet.size)} ·{" "}
                                      {labelForMinerals(planet.mineralClass)}
                                    </p>
                                    <p>{resultLabel()}</p>
                                  </div>
                                  <span class="game-card-cue">Select</span>
                                </button>
                              );
                            }}
                          </For>
                        </div>
                      </article>
                    )}
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
