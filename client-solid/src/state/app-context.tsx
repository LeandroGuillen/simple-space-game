import { createContext, createEffect, createMemo, onCleanup, onMount, useContext, type ParentProps } from 'solid-js';
import { createStore } from 'solid-js/store';

import { ApiClient, parseGameUpdate } from '../lib/api';
import { mapGame, mapPlayerColorOptions } from '../lib/mappers';
import type { ApiGameListItem, NewGameOptions } from '../lib/types/api';
import type { GameViewModel, PlayerColorOptions, ShipDesignForm } from '../lib/types/domain';
import type { ColonizeCommand, FleetMoveCommand, MovePopulationCommand, PendingCommands, SelectResearchCommand, TurnCommands } from '../lib/types/commands';
import { clearSessionLastSetupServerUrl } from '../lib/setup-session-storage';
import { findPlanet, getColonyRows, getPrimaryPlayer, normalizeServerUrl, toWebSocketUrl } from '../lib/utils';

type ConnectionStatus = 'idle' | 'checking' | 'connected' | 'error';

function gameActivityTime(game: ApiGameListItem) {
  const timestamp = game.last_turn_at ?? game.created_at;
  if (!timestamp) {
    return 0;
  }
  const parsed = Date.parse(timestamp);
  return Number.isNaN(parsed) ? 0 : parsed;
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
  debugMode: boolean;
  isBusy: boolean;
  error: string;
  info: string;
  pendingCommands: PendingCommands;
  pendingColonyBasePlanets: number[]; // planets where Colony Base completed
  showColonizeModal: boolean;
};

type AppContextValue = {
  state: AppState;
  currentPlayer: () => ReturnType<typeof getPrimaryPlayer>;
  colonies: () => ReturnType<typeof getColonyRows>;
  connect: (serverUrl: string) => Promise<void>;
  disconnect: () => void;
  refreshGames: () => Promise<void>;
  createGame: (options: NewGameOptions) => Promise<string>;
  selectGame: (gameId: string) => Promise<void>;
  refreshGame: () => Promise<void>;
  changeBuilding: (planetId: number, buildingId: number) => void;
  changeBuildQueue: (planetId: number, buildingNames: string[]) => void;
  addMovePopulationCommand: (command: MovePopulationCommand) => void;
  addFleetMoveCommand: (command: FleetMoveCommand) => void;
  addColonizeCommand: (command: ColonizeCommand) => void;
  selectColonizeTarget: (planetId: number) => void;
  closeColonizeModal: () => void;
  selectResearch: (command: SelectResearchCommand) => void;
  enableDebugMode: () => Promise<void>;
  colonize: (planetId: number) => Promise<void>;
  exploreStar: (starId: number) => Promise<void>;
  endTurn: () => Promise<void>;
  saveShipDesign: (form: ShipDesignForm, mode: 'create' | 'update') => Promise<void>;
  clearError: () => void;
};

const STORAGE_KEYS = {
  serverUrl: 'client-solid.serverUrl',
  selectedGameId: 'client-solid.selectedGameId',
};
const AppContext = createContext<AppContextValue>();

export function AppProvider(props: ParentProps) {
  const emptyPendingCommands = (): PendingCommands => ({
    build: [],
    movePopulation: [],
    fleetMove: [],
    selectResearch: undefined,
    colonize: undefined,
  });

  const [state, setState] = createStore<AppState>({
    session: {
      serverUrl: '',
      playerId: 1,
      selectedGameId: '',
      connectionStatus: 'idle',
    },
    games: [],
    playerColorOptions: {
      maxPlayers: 8,
      colors: ['#FFFFFF'],
    },
    game: null,
    debugMode: false,
    isBusy: false,
    error: '',
    info: '',
    pendingCommands: emptyPendingCommands(),
    pendingColonyBasePlanets: [],
    showColonizeModal: false,
  });

  const api = createMemo(() =>
    state.session.serverUrl ? new ApiClient(state.session.serverUrl, state.session.playerId) : null,
  );

  const currentPlayer = createMemo(() => getPrimaryPlayer(state.game));
  const colonies = createMemo(() => getColonyRows(state.game, state.pendingCommands));

  function clearError() {
    setState('error', '');
  }

  function setBusy(isBusy: boolean, info = state.info) {
    setState({
      isBusy,
      info,
    });
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
      info: 'Disconnected',
    });
    clearError();
    window.localStorage.removeItem(STORAGE_KEYS.serverUrl);
    window.localStorage.removeItem(STORAGE_KEYS.selectedGameId);
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
      const colorOptions = await client.getPlayerColorOptions();
      setState('playerColorOptions', mapPlayerColorOptions(colorOptions));
      setState('session', 'connectionStatus', 'connected');
      setBusy(false, 'Server connected');
      await refreshGames();
    } catch (error) {
      setState('session', 'connectionStatus', 'error');
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
      const gameId = await api()!.createGame(options);
      setState('session', 'selectedGameId', gameId);
      await refreshGames();
      setBusy(false, 'Game created');
      return gameId;
    } catch (error) {
      setBusy(false, '');
      setState('error', error instanceof Error ? error.message : 'Unable to create game.');
      throw error;
    }
  }

  async function selectGame(gameId: string) {
    setState('session', 'selectedGameId', gameId);
  }

  async function refreshGame() {
    if (!api() || !state.session.selectedGameId) {
      return;
    }
    setBusy(true, 'Loading galaxy state');
    clearError();
    try {
      const rawGame = await api()!.getGame(state.session.selectedGameId);
      const mapped = mapGame(rawGame);
      setState('game', mapped);
      setBusy(false, 'Game state refreshed');
    } catch (error) {
      setBusy(false, '');
      setState('error', error instanceof Error ? error.message : 'Unable to load game state.');
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
    const planet = findPlanet(state.game, planetId);
    const activeBuildingName = planet?.isBuilding && planet.currentProject
      ? planet.currentProject.name
      : buildingNames[0];
    nextBuildCommands.push({
      planetId,
      building: activeBuildingName ?? '',
      build_queue: buildingNames,
    });
    setState('pendingCommands', 'build', nextBuildCommands);
    setState('info', buildingNames.length > 0 ? 'Build queue updated' : 'Build queue cleared');
  }

  function addMovePopulationCommand(command: MovePopulationCommand) {
    setState('pendingCommands', 'movePopulation', (cmds) => [...cmds, command]);
  }

  function addFleetMoveCommand(command: FleetMoveCommand) {
    const fleet = state.game?.fleets.find((item) => item.id === command.fleet_id);
    // Server rejects destination === fleet.StarID; treat it as cancelling a queued move (or aborting travel planning).
    if (fleet && command.destination_star_id === fleet.starId) {
      const hadPending = state.pendingCommands.fleetMove.some((cmd) => cmd.fleet_id === command.fleet_id);
      const nextCommands = state.pendingCommands.fleetMove.filter((cmd) => cmd.fleet_id !== command.fleet_id);
      setState('pendingCommands', 'fleetMove', nextCommands);
      setState('info', hadPending ? 'Fleet move cancelled' : 'Fleet travel cancelled');
      return;
    }
    const nextCommands = state.pendingCommands.fleetMove.filter((cmd) => cmd.fleet_id !== command.fleet_id);
    nextCommands.push(command);
    setState('pendingCommands', 'fleetMove', nextCommands);
    setState('info', 'Fleet move queued for next turn');
  }

  function addColonizeCommand(command: ColonizeCommand) {
    setState('pendingCommands', 'colonize', command);
    setState('info', 'Colonize command queued for next turn');
  }

  function selectColonizeTarget(planetId: number) {
    addColonizeCommand({ planet_id: planetId });
    setState('showColonizeModal', false);
    setState('pendingColonyBasePlanets', []);
  }

  function closeColonizeModal() {
    setState('showColonizeModal', false);
    setState('pendingColonyBasePlanets', []);
  }

  function selectResearch(command: SelectResearchCommand) {
    setState('pendingCommands', 'selectResearch', command);
    setState('info', 'Research project queued for next turn');
  }

  async function enableDebugMode() {
    if (state.debugMode) {
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

  async function colonize(planetId: number) {
    if (!api() || !state.session.selectedGameId) {
      return;
    }
    setBusy(true, 'Dispatching colony ship');
    clearError();
    try {
      await api()!.colonize(state.session.selectedGameId, planetId);
      setState('info', 'Waiting for server update...');
    } catch (error) {
      setBusy(false, '');
      setState('error', error instanceof Error ? error.message : 'Unable to colonize planet.');
      throw error;
    }
  }

  async function endTurn() {
    if (!api() || !state.session.selectedGameId) {
      return;
    }
    setBusy(true, 'Ending turn');
    clearError();
    try {
      await api()!.endTurn(state.session.selectedGameId, toTurnCommands(state.pendingCommands, state.game));
      const rawGame = await api()!.getGame(state.session.selectedGameId);
      const mappedGame = mapGame(rawGame);
      setState('pendingCommands', emptyPendingCommands());
      setState('game', mappedGame);

      // Check for pending colony bases
      const playerId = state.session.playerId;
      const pendingBases = mappedGame.pendingColonyBases[playerId] ?? [];
      if (pendingBases.length > 0) {
        setState('pendingColonyBasePlanets', pendingBases);
        setState('showColonizeModal', true);
      }

      setBusy(false, 'Turn complete');
    } catch (error) {
      setBusy(false, '');
      setState('error', error instanceof Error ? error.message : 'Unable to complete turn.');
      throw error;
    }
  }

  async function exploreStar(starId: number) {
    if (!api() || !state.session.selectedGameId) {
      return;
    }
    setBusy(true, 'Exploring star system');
    clearError();
    try {
      await api()!.exploreStar(state.session.selectedGameId, starId);
      setState('info', 'Waiting for server update...');
    } catch (error) {
      setBusy(false, '');
      setState('error', error instanceof Error ? error.message : 'Unable to explore star system.');
      throw error;
    }
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
    if (state.session.selectedGameId) {
      window.localStorage.setItem(STORAGE_KEYS.selectedGameId, state.session.selectedGameId);
    } else {
      window.localStorage.removeItem(STORAGE_KEYS.selectedGameId);
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
          setState('game', mapGame(payload.game_update));
          setState('pendingCommands', emptyPendingCommands());
          setState({ isBusy: false, info: 'Live update received' });
        }
      } catch {
        setState('info', 'Ignored malformed websocket update');
      }
    };
    socket.onerror = () => {
      setState('info', 'Live updates unavailable; using refresh-after-action mode');
    };

    onCleanup(() => socket.close());
  });

  onMount(async () => {
    const savedServer = window.localStorage.getItem(STORAGE_KEYS.serverUrl) ?? '';
    const savedGame = window.localStorage.getItem(STORAGE_KEYS.selectedGameId) ?? '';
    if (!savedServer) {
      return;
    }

    setState('session', 'serverUrl', savedServer);
    setState('session', 'selectedGameId', savedGame);
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
    disconnect,
    refreshGames,
    createGame,
    selectGame,
    refreshGame,
    changeBuilding,
    changeBuildQueue,
    addMovePopulationCommand,
    addFleetMoveCommand,
    addColonizeCommand,
    selectColonizeTarget,
    closeColonizeModal,
    selectResearch,
    enableDebugMode,
    colonize,
    exploreStar,
    endTurn,
    saveShipDesign,
    clearError,
  };

  return <AppContext.Provider value={value}>{props.children}</AppContext.Provider>;
}

function toTurnCommands(commands: PendingCommands, game: GameViewModel | null): TurnCommands {
  const turnCommands: TurnCommands = {
    build: commands.build,
  };

  if (commands.selectResearch) {
    turnCommands.selectResearch = commands.selectResearch;
  }

  if (commands.colonize) {
    turnCommands.colonize = commands.colonize;
  }
  if (commands.fleetMove.length > 0) {
    turnCommands.fleet_move = commands.fleetMove;
  }

  const [firstMove] = commands.movePopulation;
  if (!firstMove || !game) {
    return turnCommands;
  }

  const planet = findPlanet(game, firstMove.planetId);
  if (!planet) {
    return turnCommands;
  }

  const totals = {
    farmers: planet.food.farmers,
    workers: planet.industry.workers,
    scientists: planet.science.scientists,
  };

  for (const move of commands.movePopulation.filter((command) => command.planetId === firstMove.planetId)) {
    totals[move.from] -= move.count;
    totals[move.to] += move.count;
  }

  turnCommands.movePopulation = {
    planetId: firstMove.planetId,
    farmers: Math.max(0, totals.farmers),
    workers: Math.max(0, totals.workers),
    scientists: Math.max(0, totals.scientists),
  };

  return turnCommands;
}

export function useAppContext() {
  const context = useContext(AppContext);
  if (!context) {
    throw new Error('App context is not available.');
  }
  return context;
}
