import { createContext, createEffect, createMemo, onCleanup, onMount, useContext, type ParentProps } from 'solid-js';
import { createStore } from 'solid-js/store';

import { ApiClient, parseGameUpdate } from '../lib/api';
import { mapGame, mapPlayerColorOptions } from '../lib/mappers';
import type { ApiGameListItem, ApiTurnResponse, NewGameOptions } from '../lib/types/api';
import type { BuildingViewModel, GameViewModel, PlayerColorOptions, ShipDesignForm } from '../lib/types/domain';
import type { ArtificialPlanetCommand, ColonizeCommand, FleetMoveCommand, InvadeCommand, MovePopulationCommand, PendingCommands, SelectResearchCommand, TransferPopulationCommand, TurnCommands } from '../lib/types/commands';
import type { ApiTacticalBattle, BattleActionRequest } from '../lib/types/battle';
import {
  clearSessionLastSetupServerUrl,
  getInitialSetupSelectedServerUrl,
} from '../lib/setup-session-storage';
import { isAutobuildProject, isBuildingApplicableToPlanet } from '../lib/construction';
import { isServerDebugEnabled } from '../lib/server-status';
import {
  findPlanet,
  getColonyRows,
  getPlayerById,
  isGameCommandPath,
  isLobbyPath,
  normalizeServerUrl,
  toWebSocketUrl,
} from '../lib/utils';

type ConnectionStatus = 'idle' | 'checking' | 'connected' | 'error';

function gameActivityTime(game: ApiGameListItem) {
  const timestamp = game.last_turn_at ?? game.created_at;
  if (!timestamp) {
    return 0;
  }
  const parsed = Date.parse(timestamp);
  return Number.isNaN(parsed) ? 0 : parsed;
}

function playerNames(game: GameViewModel, playerIds: number[]) {
  const nameById = new Map(game.players.map((player) => [player.id, player.name]));
  return playerIds.map((id) => nameById.get(id) ?? `Player ${id}`);
}

function turnReadinessMessage(game: GameViewModel, playerId: number, fallback: string) {
  const status = game.turnStatus;
  if (!status) {
    return fallback;
  }
  const ready = new Set(status.readyPlayerIds);
  const waiting = new Set(status.waitingPlayerIds);
  if (ready.has(playerId) && status.waitingPlayerIds.length > 0) {
    return `Orders submitted; waiting for ${playerNames(game, status.waitingPlayerIds).join(', ')}`;
  }
  if (waiting.has(playerId) && status.readyPlayerIds.length > 0) {
    return `${playerNames(game, status.readyPlayerIds).join(', ')} ready; your orders are pending`;
  }
  if (ready.has(playerId) && status.waitingPlayerIds.length === 0) {
    return 'All players ready; resolving turn';
  }
  return fallback;
}

export function submittedTurnMessage(response: ApiTurnResponse, game: GameViewModel, playerId: number) {
  return response.message || turnReadinessMessage(game, playerId, 'Orders submitted; waiting for other players');
}

type AppState = {
  session: {
    serverUrl: string;
    playerId: number;
    selectedGameId: string;
    connectionStatus: ConnectionStatus;
  };
  games: ApiGameListItem[];
  playerColorOptions: PlayerColorOptions;
  game: GameViewModel | null;
  /** True when GET /status reports the server has /debug/* enabled. */
  serverDebugEnabled: boolean;
  debugMode: boolean;
  isBusy: boolean;
  error: string;
  info: string;
  pendingCommands: PendingCommands;
  pendingColonyBasePlanets: number[]; // planets where Colony Base completed
  pendingColonyShipFleets: number[]; // fleets where Colony Ship command completed
  pendingArtificialPlanetSourcePlanets: number[]; // planets where Artificial Planet Construction completed
  showColonizeModal: boolean;
  showArtificialPlanetModal: boolean;
};

type AppContextValue = {
  state: AppState;
  currentPlayer: () => ReturnType<typeof getPlayerById>;
  colonies: () => ReturnType<typeof getColonyRows>;
  connect: (serverUrl: string) => Promise<void>;
  ensureConnected: (serverUrl: string) => Promise<void>;
  disconnect: () => void;
  refreshGames: () => Promise<void>;
  createGame: (options: NewGameOptions) => Promise<string>;
  selectGame: (gameId: string) => Promise<void>;
  joinGame: (gameId: string, playerId: number) => Promise<GameViewModel>;
  clearSelectedGame: () => void;
  setPlayerId: (playerId: number) => void;
  refreshGame: (playerId?: number) => Promise<GameViewModel>;
  changeBuilding: (planetId: number, buildingId: number) => void;
  changeBuildQueue: (planetId: number, buildingNames: string[]) => void;
  setPlanetAutobuild: (planetId: number, enabled: boolean) => Promise<void>;
  addMovePopulationCommand: (command: MovePopulationCommand) => void;
  addTransferPopulationCommand: (command: TransferPopulationCommand) => void;
  addFleetMoveCommand: (command: FleetMoveCommand) => void;
  addColonizeCommand: (command: ColonizeCommand) => void;
  addInvadeCommand: (command: InvadeCommand) => void;
  beginInvasion: (fleetId: number, planetId: number) => void;
  cancelInvasion: (fleetId: number) => void;
  beginColonyShipColonization: (fleetId: number) => void;
  cancelColonyShipColonization: (fleetId: number) => void;
  selectColonizeTarget: (planetId: number) => void;
  closeColonizeModal: () => void;
  selectArtificialPlanetTarget: (sourcePlanetId: number, targetPlanetId: number) => void;
  closeArtificialPlanetModal: () => void;
  selectResearch: (command: SelectResearchCommand) => void;
  buyProduction: (planetId: number) => void;
  cancelBuyProduction: (planetId: number) => void;
  enableDebugMode: () => Promise<void>;
  endTurn: () => Promise<void>;
  saveGameNow: () => Promise<void>;
  getBattle: (battleId: number) => Promise<ApiTacticalBattle>;
  sendBattleAction: (battleId: number, action: BattleActionRequest) => Promise<ApiTacticalBattle>;
  saveShipDesign: (form: ShipDesignForm, mode: 'create' | 'update') => Promise<void>;
  setInfo: (info: string) => void;
  clearError: () => void;
  clearInfo: () => void;
  mutateRace: (traitIds: string[]) => void;
};

const STORAGE_KEYS = {
  serverUrl: 'client-solid.serverUrl',
  selectedGameId: 'client-solid.selectedGameId',
  playerId: 'client-solid.playerId',
};

// readBootstrapSession resolves the initial server/game/player triple from
// URL query params and localStorage. Saved game ids are only restored on
// campaign routes so the lobby does not auto-attach to a stale game.
function readBootstrapSession() {
  const fromStorage = {
    serverUrl: window.localStorage.getItem(STORAGE_KEYS.serverUrl) ?? '',
    gameId: window.localStorage.getItem(STORAGE_KEYS.selectedGameId) ?? '',
    playerId: parseInt(window.localStorage.getItem(STORAGE_KEYS.playerId) ?? '', 10) || 1,
  };
  if (typeof window === 'undefined' || !window.location) {
    return fromStorage;
  }
  const params = new URLSearchParams(window.location.search);
  const queryGame = params.get('game') ?? '';
  const rawQueryPlayer = params.get('player');
  const queryPlayer = rawQueryPlayer ? parseInt(rawQueryPlayer, 10) : NaN;
  const onCampaignRoute = isGameCommandPath(window.location.pathname);
  return {
    serverUrl: fromStorage.serverUrl,
    gameId: queryGame || (onCampaignRoute ? fromStorage.gameId : ''),
    playerId: Number.isFinite(queryPlayer) && queryPlayer > 0 ? queryPlayer : fromStorage.playerId,
  };
}
const AppContext = createContext<AppContextValue>();

export function AppProvider(props: ParentProps) {
  const autosaveWarnings = new Map<string, string>();
  const emptyPendingCommands = (): PendingCommands => ({
    build: [],
    movePopulation: [],
    transferPopulation: [],
    fleetMove: [],
    selectResearch: undefined,
    colonize: [],
    artificialPlanet: [],
    invade: [],
    buyProduction: [],
    mutateRace: undefined,
  });

  const bootstrap = readBootstrapSession();
  const [state, setState] = createStore<AppState>({
    session: {
      serverUrl: '',
      playerId: bootstrap.playerId,
      selectedGameId: bootstrap.gameId,
      connectionStatus: 'idle',
    },
    games: [],
    playerColorOptions: {
      maxPlayers: 8,
      colors: ['#FFFFFF'],
    },
    game: null,
    serverDebugEnabled: false,
    debugMode: false,
    isBusy: false,
    error: '',
    info: '',
    pendingCommands: emptyPendingCommands(),
    pendingColonyBasePlanets: [],
    pendingColonyShipFleets: [],
    pendingArtificialPlanetSourcePlanets: [],
    showColonizeModal: false,
    showArtificialPlanetModal: false,
  });

  const api = createMemo(() =>
    state.session.serverUrl ? new ApiClient(state.session.serverUrl, state.session.playerId) : null,
  );

  const currentPlayer = createMemo(() => getPlayerById(state.game, state.session.playerId));
  const colonies = createMemo(() =>
    getColonyRows(state.game, state.pendingCommands, state.session.playerId),
  );

  function clearError() {
    setState('error', '');
  }

  function clearInfo() {
    setState('info', '');
  }

  function setBusy(isBusy: boolean, info = state.info) {
    setState({
      isBusy,
      info,
    });
  }

  function resetPerGameUiState(info = '') {
    setState({
      pendingCommands: emptyPendingCommands(),
      pendingColonyBasePlanets: [],
      pendingColonyShipFleets: [],
      pendingArtificialPlanetSourcePlanets: [],
      showColonizeModal: false,
      showArtificialPlanetModal: false,
      error: '',
      info,
    });
  }

  function setInfo(info: string) {
    setState('info', info);
  }

  function disconnect() {
    setState({
      session: {
        serverUrl: '',
        playerId: state.session.playerId,
        selectedGameId: '',
        connectionStatus: 'idle',
      },
      games: [],
      game: null,
      serverDebugEnabled: false,
      debugMode: false,
      pendingCommands: emptyPendingCommands(),
      pendingColonyBasePlanets: [],
      pendingColonyShipFleets: [],
      pendingArtificialPlanetSourcePlanets: [],
      showColonizeModal: false,
      showArtificialPlanetModal: false,
      info: 'Disconnected',
    });
    clearError();
    window.localStorage.removeItem(STORAGE_KEYS.serverUrl);
    window.localStorage.removeItem(STORAGE_KEYS.selectedGameId);
    window.localStorage.removeItem(STORAGE_KEYS.playerId);
    clearSessionLastSetupServerUrl();
  }

  async function connect(serverUrl: string) {
    const normalized = normalizeServerUrl(serverUrl);
    if (!normalized) {
      setState('error', 'Enter a server URL.');
      return;
    }

    setState('session', 'serverUrl', normalized);
    setState('session', 'connectionStatus', 'checking');
    setBusy(true, 'Checking server health');
    clearError();

    try {
      const client = new ApiClient(normalized, state.session.playerId);
      const response = await client.status();
      if (!response.ok) {
        throw new Error('Server did not report healthy status.');
      }
      setState('serverDebugEnabled', isServerDebugEnabled(response));
      const colorOptions = await client.getPlayerColorOptions();
      setState('playerColorOptions', mapPlayerColorOptions(colorOptions));
      setState('session', 'connectionStatus', 'connected');
      setBusy(false, 'Server connected');
      await refreshGames();
    } catch (error) {
      setState('session', 'connectionStatus', 'error');
      setState('serverDebugEnabled', false);
      setBusy(false, '');
      setState('error', error instanceof Error ? error.message : 'Unable to connect to server.');
      throw error;
    }
  }

  async function refreshGames() {
    if (!api()) {
      return;
    }
    setBusy(true, 'Refreshing game list');
    clearError();
    try {
      const list = await api()!.getGames();
      list.sort((left, right) => {
        const activityDiff = gameActivityTime(right) - gameActivityTime(left);
        return activityDiff || left.game_id.localeCompare(right.game_id);
      });
      setState('games', list);
      setBusy(false, `Loaded ${list.length} game${list.length === 1 ? '' : 's'}`);
    } catch (error) {
      setBusy(false, '');
      setState('error', error instanceof Error ? error.message : 'Unable to load games.');
      throw error;
    }
  }

  async function createGame(options: NewGameOptions) {
    if (!api()) {
      throw new Error('Connect to a server first.');
    }
    setBusy(true, 'Creating game');
    clearError();
    try {
      const result = await api()!.createGame(options);
      const gameId = result.gameID;
      if (result.warning) autosaveWarnings.set(gameId, result.warning);
      setState('session', 'selectedGameId', gameId);
      setState('game', null);
      resetPerGameUiState();
      await refreshGames();
      setBusy(false, result.warning || 'Game created');
      return gameId;
    } catch (error) {
      setBusy(false, '');
      setState('error', error instanceof Error ? error.message : 'Unable to create game.');
      throw error;
    }
  }

  function clearSelectedGame() {
    setState('game', null);
    resetPerGameUiState('Left game');
    setState('session', 'selectedGameId', '');
  }

  function gameClient(playerId: number) {
    const serverUrl = state.session.serverUrl;
    if (!serverUrl) {
      return null;
    }
    return new ApiClient(serverUrl, playerId);
  }

  async function joinGame(gameId: string, playerId: number): Promise<GameViewModel> {
    if (!Number.isFinite(playerId) || playerId <= 0) {
      throw new Error(`Invalid player slot ${playerId}.`);
    }
    if (!state.session.serverUrl) {
      throw new Error('Connect to a server first.');
    }
    setState('session', 'playerId', playerId);
    setState('game', null);
    resetPerGameUiState();
    setState('session', 'selectedGameId', gameId);
    const mapped = await refreshGame(playerId);
    if (mapped.id !== gameId) {
      throw new Error('Game did not load. Check the error message below and try again.');
    }
    return mapped;
  }

  async function selectGame(gameId: string) {
    await joinGame(gameId, state.session.playerId);
  }

  async function ensureConnected(serverUrl: string) {
    const normalized = normalizeServerUrl(serverUrl);
    if (!normalized) {
      throw new Error('Enter a server URL.');
    }
    if (
      state.session.connectionStatus === 'connected' &&
      normalizeServerUrl(state.session.serverUrl) === normalized
    ) {
      return;
    }
    await connect(normalized);
    if (state.session.connectionStatus !== 'connected') {
      throw new Error('Could not connect to the server.');
    }
  }

  function setPlayerId(playerId: number) {
    if (!Number.isFinite(playerId) || playerId <= 0) {
      return;
    }
    if (playerId === state.session.playerId) {
      return;
    }
    // Switching slots invalidates per-game UI state (pending commands, modals).
    setState('game', null);
    resetPerGameUiState();
    setState('session', 'playerId', playerId);
  }

  async function refreshGame(playerId = state.session.playerId): Promise<GameViewModel> {
    const gameId = state.session.selectedGameId;
    if (!gameId) {
      throw new Error('Connect to a server and select a game first.');
    }
    const client = gameClient(playerId);
    if (!client) {
      throw new Error('Connect to a server first.');
    }
    setBusy(true, 'Loading galaxy state');
    clearError();
    try {
      const rawGame = await client.getGame(gameId);
      const mapped = mapGame(rawGame);
      setState('game', mapped);
      setState('debugMode', rawGame.debug === true);
      setBusy(false, 'Game state refreshed');
      const autosaveWarning = autosaveWarnings.get(gameId);
      if (autosaveWarning) setState('error', autosaveWarning);
      return mapped;
    } catch (error) {
      setBusy(false, '');
      const message = error instanceof Error ? error.message : 'Unable to load game state.';
      if (/not found|404/i.test(message)) {
        clearSelectedGame();
        setState('error', 'That game no longer exists on this server. Start a new game or pick another from the list.');
      } else {
        setState('error', message);
      }
      throw error;
    }
  }

  function changeBuilding(planetId: number, buildingId: number) {
    if (!state.game) {
      return;
    }
    clearError();

    const selectedBuilding = state.game.resources.buildingCollection.buildings.find(
      (building) => building.id === buildingId,
    );
    const nextBuildCommands = state.pendingCommands.build.filter((command) => command.planetId !== planetId);

    if (buildingId === 0) {
      nextBuildCommands.push({
        planetId,
        building: 'Trade Goods',
      });
    } else if (selectedBuilding) {
      nextBuildCommands.push({
        planetId,
        building: selectedBuilding.name,
      });
    }

    setState('pendingCommands', 'build', nextBuildCommands);
    setState('info', buildingId === 0 ? 'Trade Goods queued for next turn' : 'Build project queued for next turn');
  }

  function changeBuildQueue(planetId: number, buildingNames: string[]) {
    const nextBuildCommands = state.pendingCommands.build.filter((command) => command.planetId !== planetId);
    const activeBuildingName = buildingNames[0];
    nextBuildCommands.push({
      planetId,
      building: activeBuildingName ?? '',
      build_queue: buildingNames.slice(1),
    });
    setState('pendingCommands', 'build', nextBuildCommands);
    setState('info', buildingNames.length > 0 ? 'Build queue updated' : 'Build queue cleared');
  }

  function setPlanetAutobuildLocal(planetId: number, enabled: boolean) {
    for (let starIndex = 0; starIndex < (state.game?.galaxy.stars.length ?? 0); starIndex += 1) {
      const planetIndex = state.game!.galaxy.stars[starIndex].planets.findIndex((planet) => planet.id === planetId);
      if (planetIndex >= 0) {
        setState('game', 'galaxy', 'stars', starIndex, 'planets', planetIndex, 'autobuild', enabled);
        return;
      }
    }
  }

  async function setPlanetAutobuild(planetId: number, enabled: boolean) {
    setPlanetAutobuildLocal(planetId, enabled);
    if (!api() || !state.session.selectedGameId) {
      return;
    }
    clearError();
    try {
      const mapped = mapGame(await api()!.setPlanetAutobuild(state.session.selectedGameId, planetId, enabled));
      setState('game', mapped);
    } catch (error) {
      setPlanetAutobuildLocal(planetId, !enabled);
      setState('error', error instanceof Error ? error.message : 'Unable to update autobuild.');
      throw error;
    }
  }

  function addMovePopulationCommand(command: MovePopulationCommand) {
    setState('pendingCommands', 'movePopulation', (cmds) => [...cmds, command]);
  }

  function addTransferPopulationCommand(command: TransferPopulationCommand) {
    setState('pendingCommands', 'transferPopulation', (commands) => [
      ...(commands ?? []).filter(
        (item) =>
          item.source_planet_id !== command.source_planet_id ||
          item.destination_planet_id !== command.destination_planet_id,
      ),
      command,
    ]);
    setState('info', 'Population transfer queued for next turn');
  }

  function addFleetMoveCommand(command: FleetMoveCommand) {
    const fleet = state.game?.fleets.find((item) => item.id === command.fleet_id);
    const inTransit =
      fleet != null &&
      (fleet.turnsRemaining ?? 0) > 0 &&
      (fleet.destinationStarId ?? 0) > 0;
    // Stationary fleet: destination === current star clears a queued move without ending the turn.
    // In-transit fleet: same destination issues a recall and must be sent to the server.
    if (fleet && command.destination_star_id === fleet.starId && !inTransit) {
      const selected = new Set(command.ship_indexes);
      const nextCommands = state.pendingCommands.fleetMove
        .map((cmd) =>
          cmd.fleet_id === command.fleet_id
            ? { ...cmd, ship_indexes: cmd.ship_indexes.filter((index) => !selected.has(index)) }
            : cmd,
        )
        .filter((cmd) => cmd.ship_indexes.length > 0);
      const hadPending = nextCommands.length !== state.pendingCommands.fleetMove.length ||
        state.pendingCommands.fleetMove.some((cmd) =>
          cmd.fleet_id === command.fleet_id && cmd.ship_indexes.some((index) => selected.has(index)),
        );
      setState('pendingCommands', 'fleetMove', nextCommands);
      setState('info', hadPending ? 'Fleet move cancelled' : 'Fleet travel cancelled');
      return;
    }
    const selected = new Set(command.ship_indexes);
    const nextCommands = state.pendingCommands.fleetMove
      .map((cmd) =>
        cmd.fleet_id === command.fleet_id
          ? { ...cmd, ship_indexes: cmd.ship_indexes.filter((index) => !selected.has(index)) }
          : cmd,
      )
      .filter((cmd) => cmd.ship_indexes.length > 0);
    const sameDestination = nextCommands.find((cmd) =>
      cmd.fleet_id === command.fleet_id && cmd.destination_star_id === command.destination_star_id
    );
    if (sameDestination) {
      sameDestination.ship_indexes = [...sameDestination.ship_indexes, ...command.ship_indexes]
        .filter((index, position, indexes) => indexes.indexOf(index) === position)
        .sort((a, b) => a - b);
    } else {
      nextCommands.push({
      fleet_id: command.fleet_id,
      destination_star_id: command.destination_star_id,
      ship_indexes: [...command.ship_indexes],
      });
    }
    setState('pendingCommands', 'colonize', (commands) =>
      commands.filter((item) => item.fleet_id !== command.fleet_id),
    );
    setState('pendingCommands', 'fleetMove', nextCommands);
    setState('info', 'Fleet move queued for next turn');
  }

  function addColonizeCommand(command: ColonizeCommand) {
    if (command.fleet_id != null) {
      setState('pendingCommands', 'fleetMove', (commands) =>
        commands.filter((item) => item.fleet_id !== command.fleet_id),
      );
    }
    setState('pendingCommands', 'colonize', (commands) => [
      ...commands.filter((item) => item.fleet_id !== command.fleet_id && item.planet_id !== command.planet_id),
      command,
    ]);
    setState('info', 'Colonize command queued for next turn');
  }

  function addInvadeCommand(command: InvadeCommand) {
    // Clear any existing fleet move commands for this fleet
    setState('pendingCommands', 'fleetMove', (commands) =>
      commands.filter((item) => item.fleet_id !== command.fleet_id),
    );
    // Add invasion command, replacing any existing invasion by this fleet or to this planet
    setState('pendingCommands', 'invade', (commands) => [
      ...commands.filter((item) => item.fleet_id !== command.fleet_id && item.target_planet_id !== command.target_planet_id),
      command,
    ]);
    setState('info', 'Invasion command queued for next turn');
  }

  function beginInvasion(fleetId: number, planetId: number) {
    const game = state.game;
    const fleet = game?.fleets.find((f) => f.id === fleetId);
    if (!fleet) {
      return;
    }
    
    // Find transport ships in the fleet
    const transportIndexes = fleet.ships
      .map((ship, index) => ({ ship, index }))
      .filter(({ ship }) => ship.type === 'transport')
      .map(({ index }) => index);
    
    if (transportIndexes.length === 0) {
      setState('error', 'No transport ships available for invasion');
      return;
    }

    addInvadeCommand({
      fleet_id: fleetId,
      target_planet_id: planetId,
      ship_indexes: transportIndexes,
    });
  }

  function cancelInvasion(fleetId: number) {
    if (!state.pendingCommands.invade.some((command) => command.fleet_id === fleetId)) {
      return;
    }
    setState('pendingCommands', 'invade', (commands) => commands.filter((command) => command.fleet_id !== fleetId));
    setState('info', 'Invasion cancelled');
  }

  function selectColonizeTarget(planetId: number) {
    const game = state.game;
    const target = game ? findPlanet(game, planetId) : null;
    const fleetId =
      target && game
        ? state.pendingColonyShipFleets
            .map((fleetId) => game.fleets.find((fleet) => fleet.id === fleetId))
            .find((fleet) =>
              fleet != null &&
              fleet.starId === target.parentStar &&
              (fleet.turnsRemaining ?? 0) === 0 &&
              fleet.ships.some((ship) => ship.type === 'colony_ship')
            )?.id ??
          game.fleets.find((fleet) =>
            fleet.ownerId === state.session.playerId &&
            fleet.starId === target.parentStar &&
            (fleet.turnsRemaining ?? 0) === 0 &&
            fleet.ships.some((ship) => ship.type === 'colony_ship')
          )?.id
        : undefined;
    addColonizeCommand(fleetId ? { planet_id: planetId, fleet_id: fleetId } : { planet_id: planetId });
    setState('showColonizeModal', false);
    setState('pendingColonyBasePlanets', []);
    setState('pendingColonyShipFleets', []);
    if (state.pendingArtificialPlanetSourcePlanets.length > 0) {
      setState('showArtificialPlanetModal', true);
    }
  }

  function closeColonizeModal() {
    setState('showColonizeModal', false);
    setState('pendingColonyBasePlanets', []);
    setState('pendingColonyShipFleets', []);
    if (state.pendingArtificialPlanetSourcePlanets.length > 0) {
      setState('showArtificialPlanetModal', true);
    }
  }

  function addArtificialPlanetCommand(command: ArtificialPlanetCommand) {
    setState('pendingCommands', 'artificialPlanet', (commands) => [
      ...commands.filter((item) => item.source_planet_id !== command.source_planet_id && item.target_planet_id !== command.target_planet_id),
      command,
    ]);
    setState('info', 'Artificial planet target queued for next turn');
  }

  function selectArtificialPlanetTarget(sourcePlanetId: number, targetPlanetId: number) {
    addArtificialPlanetCommand({
      source_planet_id: sourcePlanetId,
      target_planet_id: targetPlanetId,
    });
    setState('pendingArtificialPlanetSourcePlanets', (sources) =>
      sources.filter((sourceId) => sourceId !== sourcePlanetId),
    );
    if (state.pendingArtificialPlanetSourcePlanets.length <= 1) {
      setState('showArtificialPlanetModal', false);
    }
  }

  function closeArtificialPlanetModal() {
    setState('showArtificialPlanetModal', false);
    setState('pendingArtificialPlanetSourcePlanets', []);
  }

  function beginColonyShipColonization(fleetId: number) {
    setState('pendingColonyBasePlanets', []);
    setState('pendingColonyShipFleets', [fleetId]);
    setState('showColonizeModal', true);
  }

  function cancelColonyShipColonization(fleetId: number) {
    if (!state.pendingCommands.colonize.some((command) => command.fleet_id === fleetId)) {
      return;
    }
    setState('pendingCommands', 'colonize', (commands) => commands.filter((command) => command.fleet_id !== fleetId));
    setState('info', 'Colony ship colonization cancelled');
  }

  function selectResearch(command: SelectResearchCommand) {
    setState('pendingCommands', 'selectResearch', command);
    setState('info', 'Research project queued for next turn');
  }

  function buyProduction(planetId: number) {
    const existing = state.pendingCommands.buyProduction.some(
      (command) => command.planetId === planetId,
    );
    if (existing) {
      return;
    }
    setState('pendingCommands', 'buyProduction', (commands) => [
      ...commands,
      { planetId },
    ]);
    setState('info', 'Buy production queued for next turn');
  }

  function cancelBuyProduction(planetId: number) {
    const hadPending = state.pendingCommands.buyProduction.some(
      (command) => command.planetId === planetId,
    );
    if (!hadPending) {
      return;
    }
    setState('pendingCommands', 'buyProduction', (commands) =>
      commands.filter((command) => command.planetId !== planetId),
    );
    setState('info', 'Buy production cancelled');
  }

  async function enableDebugMode() {
    if (!state.serverDebugEnabled || state.debugMode) {
      return;
    }
    if (!api() || !state.session.selectedGameId) {
      return;
    }
    setBusy(true, 'Enabling debug mode');
    clearError();
    setState('pendingCommands', 'selectResearch', undefined);
    try {
      const response = await api()!.debugResearchAllTechs(state.session.selectedGameId);
      setState('debugMode', true);
      setState('game', mapGame(response.game));
      setBusy(false, 'Debug mode enabled');
    } catch (error) {
      setBusy(false, '');
      setState('error', error instanceof Error ? error.message : 'Unable to enable debug mode.');
      throw error;
    }
  }

  async function saveGameNow() {
    const client = api();
    const gameId = state.session.selectedGameId;
    if (!client || !gameId) return;
    setBusy(true, 'Saving game');
    clearError();
    try {
      await client.saveGameNow(gameId);
      autosaveWarnings.delete(gameId);
      setBusy(false, 'Game saved. Queued orders are not included.');
    } catch (error) {
      setBusy(false, '');
      const message = error instanceof Error ? error.message : 'Unable to save game.';
      autosaveWarnings.set(gameId, message);
      setState('error', message);
    }
  }

  async function endTurn() {
    if (!api() || !state.session.selectedGameId) {
      return;
    }
    setBusy(true, 'Ending turn');
    clearError();
    try {
      const turnResponse = await api()!.endTurn(state.session.selectedGameId, toTurnCommands(state.pendingCommands, state.game));
      if (turnResponse.warning) autosaveWarnings.set(state.session.selectedGameId, turnResponse.warning);
      else if (turnResponse.advanced) autosaveWarnings.delete(state.session.selectedGameId);
      const rawGame = await api()!.getGame(state.session.selectedGameId);
      const mappedGame = mapGame(rawGame);
      setState('pendingCommands', emptyPendingCommands());
      setState('game', mappedGame);

      if (!turnResponse.advanced) {
        setState('pendingColonyBasePlanets', []);
        setState('pendingColonyShipFleets', []);
        setState('pendingArtificialPlanetSourcePlanets', []);
        setState('showColonizeModal', false);
        setState('showArtificialPlanetModal', false);
        setBusy(false, submittedTurnMessage(turnResponse, mappedGame, state.session.playerId));
        const autosaveWarning = autosaveWarnings.get(state.session.selectedGameId);
        if (autosaveWarning) setState('error', autosaveWarning);
        return;
      }

      // Check for pending colony bases
      const playerId = state.session.playerId;
      const pendingBases = mappedGame.pendingColonyBases[playerId] ?? [];
      const pendingShips = mappedGame.pendingColonyShips[playerId] ?? [];
      const pendingArtificialPlanets = mappedGame.pendingArtificialPlanets[playerId] ?? [];
      let completionInfo = 'Turn complete';
      if (pendingBases.length > 0 || pendingShips.length > 0) {
        setState('pendingColonyBasePlanets', pendingBases);
        setState('pendingColonyShipFleets', pendingShips);
        setState('pendingArtificialPlanetSourcePlanets', pendingArtificialPlanets);
        setState('showColonizeModal', true);
        setState('showArtificialPlanetModal', false);
      } else if (pendingArtificialPlanets.length > 0) {
        setState('pendingColonyBasePlanets', []);
        setState('pendingColonyShipFleets', []);
        setState('pendingArtificialPlanetSourcePlanets', pendingArtificialPlanets);
        setState('showColonizeModal', false);
        setState('showArtificialPlanetModal', true);
      } else {
        setState('pendingColonyBasePlanets', []);
        setState('pendingColonyShipFleets', []);
        setState('pendingArtificialPlanetSourcePlanets', []);
        setState('showColonizeModal', false);
        setState('showArtificialPlanetModal', false);
      }

      setBusy(false, turnResponse.warning || completionInfo);
      if (turnResponse.warning) setState('error', turnResponse.warning);
    } catch (error) {
      setBusy(false, '');
      setState('error', error instanceof Error ? error.message : 'Unable to complete turn.');
      throw error;
    }
  }

  function requireBattleApi() {
    const client = api();
    const gameId = state.session.selectedGameId;
    if (!client || !gameId) {
      throw new Error('Connect to a server and select a game first.');
    }
    return { client, gameId };
  }

  async function getBattle(battleId: number) {
    const { client, gameId } = requireBattleApi();
    return client.getBattle(gameId, battleId);
  }

  async function sendBattleAction(battleId: number, action: BattleActionRequest) {
    const { client, gameId } = requireBattleApi();
    const result = await client.postBattleAction(gameId, battleId, action);
    if (result.warning) {
      autosaveWarnings.set(gameId, result.warning);
      setState('error', result.warning);
    } else if (result.battle.status === 'finished') {
      autosaveWarnings.delete(gameId);
    }
    return result.battle;
  }

  async function saveShipDesign(form: ShipDesignForm, mode: 'create' | 'update') {
    if (!api() || !state.session.selectedGameId) {
      return;
    }
    setBusy(true, mode === 'create' ? 'Creating ship design' : 'Updating ship design');
    clearError();
    try {
      await api()!.saveShipDesign(state.session.selectedGameId, form, mode);
      setState('info', 'Waiting for server update...');
    } catch (error) {
      setBusy(false, '');
      setState('error', error instanceof Error ? error.message : 'Unable to save ship design.');
      throw error;
    }
  }

  createEffect(() => {
    if (state.session.serverUrl) {
      window.localStorage.setItem(STORAGE_KEYS.serverUrl, state.session.serverUrl);
    }
  });

  createEffect(() => {
    if (typeof window === 'undefined') {
      return;
    }
    const onCampaignRoute = isGameCommandPath(window.location.pathname);
    if (state.session.selectedGameId && onCampaignRoute) {
      window.localStorage.setItem(STORAGE_KEYS.selectedGameId, state.session.selectedGameId);
    } else {
      window.localStorage.removeItem(STORAGE_KEYS.selectedGameId);
    }
  });

  createEffect(() => {
    window.localStorage.setItem(STORAGE_KEYS.playerId, String(state.session.playerId));
  });

  // Mirror game/player into the URL only on campaign routes (not on the lobby).
  createEffect(() => {
    if (typeof window === 'undefined' || !window.history) {
      return;
    }
    const url = new URL(window.location.href);
    const gameId = state.session.selectedGameId;
    if (gameId && isGameCommandPath(url.pathname)) {
      url.searchParams.set('game', gameId);
      url.searchParams.set('player', String(state.session.playerId));
    } else if (isLobbyPath(url.pathname)) {
      url.searchParams.delete('game');
      url.searchParams.delete('player');
    }
    const next = url.pathname + (url.search ? url.search : '');
    if (next !== window.location.pathname + window.location.search) {
      window.history.replaceState(window.history.state, '', next);
    }
  });

  createEffect(() => {
    if (!state.game || state.isBusy) {
      return;
    }

    const player = currentPlayer();
    if (!player) {
      return;
    }

    const availableBuildingNames = new Set(player.available.buildings);
    for (const star of state.game.galaxy.stars) {
      for (const planet of star.planets) {
        if (
          planet.owner !== player.id ||
          !planet.autobuild ||
          planet.isBuilding ||
          state.pendingCommands.build.some((command) => command.planetId === planet.id)
        ) {
          continue;
        }

        const builtBuildingIds = new Set(
          planet.buildings
            .filter((building) => building.name !== 'Colony Base')
            .map((building) => building.id),
        );
        const lockedUniqueNames = uniqueBuildingsLockedElsewhere(
          state.game,
          player.id,
          planet.id,
        );
        const projects = state.game.resources.buildingCollection.buildings.filter(
          (project) =>
            availableBuildingNames.has(project.name) &&
            !builtBuildingIds.has(project.id) &&
            !hasSuperiorBuilding(project, planet.buildings) &&
            (!project.unique || !lockedUniqueNames.has(project.name)) &&
            isAutobuildProject(project) &&
            isBuildingApplicableToPlanet(project, planet, player.raceTraitIds),
        );
        const project = projects[Math.floor(Math.random() * projects.length)];
        if (project) {
          changeBuilding(planet.id, project.id);
        }
      }
    }
  });

  createEffect(() => {
    const serverUrl = state.session.serverUrl;
    const gameId = state.session.selectedGameId;
    if (!serverUrl || !gameId) {
      return;
    }

    const socket = new WebSocket(toWebSocketUrl(serverUrl, gameId, state.session.playerId));
    socket.onmessage = (event) => {
      try {
        const payload = parseGameUpdate(event.data);
        if (payload.game_update.id === state.session.selectedGameId) {
          const previousTurn = state.game?.currentTurn;
          const mappedGame = mapGame(payload.game_update);
          const info =
            previousTurn != null && mappedGame.currentTurn !== previousTurn
              ? 'Turn complete'
              : turnReadinessMessage(mappedGame, state.session.playerId, 'Live update received');
          setState('game', mappedGame);
          setState({ isBusy: false, info });
        }
      } catch (error) {
        const message =
          error instanceof Error ? error.message : 'Failed to apply live game update';
        setState('error', message);
        setState('info', '');
      }
    };
    socket.onerror = () => {
      setState('info', 'Live updates unavailable; using refresh-after-action mode');
    };

    onCleanup(() => socket.close());
  });

  onMount(async () => {
    const savedServer =
      getInitialSetupSelectedServerUrl() ||
      window.localStorage.getItem(STORAGE_KEYS.serverUrl) ||
      '';
    if (!savedServer) {
      return;
    }

    try {
      await connect(savedServer);
    } catch {
      // surface error in state only
    }
  });

  const value: AppContextValue = {
    state,
    currentPlayer,
    colonies,
    connect,
    ensureConnected,
    disconnect,
    refreshGames,
    createGame,
    selectGame,
    joinGame,
    clearSelectedGame,
    setPlayerId,
    refreshGame,
    changeBuilding,
    changeBuildQueue,
    setPlanetAutobuild,
    addMovePopulationCommand,
    addTransferPopulationCommand,
    addFleetMoveCommand,
    addColonizeCommand,
    addInvadeCommand,
    beginInvasion,
    cancelInvasion,
    beginColonyShipColonization,
    cancelColonyShipColonization,
    selectColonizeTarget,
    closeColonizeModal,
    selectArtificialPlanetTarget,
    closeArtificialPlanetModal,
    selectResearch,
    buyProduction,
    cancelBuyProduction,
    enableDebugMode,
    endTurn,
    saveGameNow,
    getBattle,
    sendBattleAction,
    saveShipDesign,
    setInfo,
    clearError,
    clearInfo,
    mutateRace: (traitIds: string[]) => {
      setState('pendingCommands', 'mutateRace', { trait_ids: traitIds });
      setState('info', 'Race mutation queued for next turn');
    },
  };

  return <AppContext.Provider value={value}>{props.children}</AppContext.Provider>;
}

export function toTurnCommands(commands: PendingCommands, game: GameViewModel | null): TurnCommands {
  const turnCommands: TurnCommands = {
    build: commands.build,
  };
  const existingFleetIds = game ? new Set(game.fleets.map((fleet) => fleet.id)) : null;

  if (commands.selectResearch) {
    turnCommands.selectResearch = commands.selectResearch;
  }

  const colonizeCommands = commands.colonize.filter(
    (command) => command.fleet_id == null || existingFleetIds == null || existingFleetIds.has(command.fleet_id),
  );
  if (colonizeCommands.length === 1) {
    turnCommands.colonize = colonizeCommands[0];
  } else if (colonizeCommands.length > 1) {
    turnCommands.colonize_many = colonizeCommands;
  }
  if (commands.artificialPlanet.length === 1) {
    turnCommands.artificial_planet = commands.artificialPlanet[0];
  } else if (commands.artificialPlanet.length > 1) {
    turnCommands.artificial_planet_many = commands.artificialPlanet;
  }
  const fleetMoveCommands = commands.fleetMove.filter(
    (command) => existingFleetIds == null || existingFleetIds.has(command.fleet_id),
  );
  if (fleetMoveCommands.length > 0) {
    turnCommands.fleet_move = fleetMoveCommands.map((cmd) => ({
      fleet_id: cmd.fleet_id,
      destination_star_id: cmd.destination_star_id,
      ship_indexes: [...cmd.ship_indexes],
    }));
  }

  const invadeCommands = commands.invade.filter(
    (command) => existingFleetIds == null || existingFleetIds.has(command.fleet_id),
  );
  if (invadeCommands.length > 0) {
    turnCommands.invade = invadeCommands.map((cmd) => ({
      fleet_id: cmd.fleet_id,
      target_planet_id: cmd.target_planet_id,
      ship_indexes: [...cmd.ship_indexes],
    }));
  }

  if (commands.buyProduction.length > 0) {
    turnCommands.buy_production = commands.buyProduction.map((cmd) => ({
      planetId: cmd.planetId,
    }));
  }

  if (commands.mutateRace) {
    turnCommands.mutate_race = {
      trait_ids: commands.mutateRace.trait_ids,
    };
  }

  if ((commands.transferPopulation ?? []).length > 0) {
    turnCommands.transfer_population = (commands.transferPopulation ?? []).map((cmd) => ({
      source_planet_id: cmd.source_planet_id,
      destination_planet_id: cmd.destination_planet_id,
      population_units: cmd.population_units,
    }));
  }

  if (commands.movePopulation.length === 0 || !game) {
    return turnCommands;
  }

  const movePlanetIds = [...new Set(commands.movePopulation.map((move) => move.planetId))];
  const movePopulation = [];
  for (const planetId of movePlanetIds) {
    const planet = findPlanet(game, planetId);
    if (!planet) {
      continue;
    }

    const totals = {
      farmers: planet.food.farmers,
      workers: planet.industry.workers,
      scientists: planet.science.scientists,
    };

    for (const move of commands.movePopulation.filter((command) => command.planetId === planetId)) {
      totals[move.from] -= move.count;
      totals[move.to] += move.count;
    }

    // The server removes outbound colonists before applying job assignments.
    const outbound = (commands.transferPopulation ?? [])
      .filter((transfer) => transfer.source_planet_id === planetId)
      .reduce((sum, transfer) => sum + transfer.population_units, 0);
    let excess = Math.max(0, totals.farmers + totals.workers + totals.scientists
      - (Math.floor(planet.population / 1000) - outbound));
    for (const job of ['farmers', 'workers', 'scientists'] as const) {
      const removed = Math.min(Math.max(0, totals[job]), excess);
      totals[job] -= removed;
      excess -= removed;
    }

    movePopulation.push({
      planetId,
      farmers: Math.max(0, totals.farmers),
      workers: Math.max(0, totals.workers),
      scientists: Math.max(0, totals.scientists),
    });
  }

  if (movePopulation.length > 0) {
    turnCommands.movePopulation = movePopulation;
  }

  return turnCommands;
}

function hasSuperiorBuilding(
  project: BuildingViewModel,
  builtBuildings: BuildingViewModel[],
) {
  return builtBuildings.some((builtBuilding) =>
    builtBuilding.upgrades.includes(project.name),
  );
}

function uniqueBuildingsLockedElsewhere(
  game: GameViewModel,
  playerId: number,
  planetId: number,
) {
  const locked = new Set<string>();
  for (const star of game.galaxy.stars) {
    for (const planet of star.planets) {
      if (planet.owner !== playerId || planet.id === planetId) {
        continue;
      }
      for (const building of planet.buildings) {
        if (building.unique) {
          locked.add(building.name);
        }
      }
      if (planet.currentProject?.unique) {
        locked.add(planet.currentProject.name);
      }
      for (const queued of planet.buildQueue) {
        if (queued.unique) {
          locked.add(queued.name);
        }
      }
    }
  }
  return locked;
}

export function useAppContext() {
  const context = useContext(AppContext);
  if (!context) {
    throw new Error('App context is not available.');
  }
  return context;
}
