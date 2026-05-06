import { createContext, createEffect, createMemo, onCleanup, onMount, useContext, type ParentProps } from 'solid-js';
import { createStore } from 'solid-js/store';

import { ApiClient, parseGameUpdate } from '../lib/api';
import { mapGame, mapPlayerColorOptions } from '../lib/mappers';
import type { ApiGameListItem } from '../lib/types/api';
import type { GameViewModel, PlayerColorOptions, ShipDesignForm } from '../lib/types/domain';
import type { MovePopulationCommand } from '../lib/types/commands';
import { getColonyRows, getPrimaryPlayer, normalizeServerUrl, toWebSocketUrl } from '../lib/utils';

type ConnectionStatus = 'idle' | 'checking' | 'connected' | 'error';

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
  isBusy: boolean;
  error: string;
  info: string;
  pendingCommands: {
    movePopulation: MovePopulationCommand[];
  };
};

type AppContextValue = {
  state: AppState;
  currentPlayer: () => ReturnType<typeof getPrimaryPlayer>;
  colonies: () => ReturnType<typeof getColonyRows>;
  connect: (serverUrl: string) => Promise<void>;
  disconnect: () => void;
  refreshGames: () => Promise<void>;
  createGame: (playerName: string, numStars: number, playerColor: string) => Promise<string>;
  selectGame: (gameId: string) => Promise<void>;
  refreshGame: () => Promise<void>;
  changeBuilding: (planetId: number, buildingId: number) => Promise<void>;
  addMovePopulationCommand: (command: MovePopulationCommand) => void;
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
    isBusy: false,
    error: '',
    info: '',
    pendingCommands: {
      movePopulation: [],
    },
  });

  const api = createMemo(() =>
    state.session.serverUrl ? new ApiClient(state.session.serverUrl, state.session.playerId) : null,
  );

  const currentPlayer = createMemo(() => getPrimaryPlayer(state.game));
  const colonies = createMemo(() => getColonyRows(state.game));

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
      list.sort((left, right) => left.game_id.localeCompare(right.game_id));
      setState('games', list);
      setBusy(false, `Loaded ${list.length} game${list.length === 1 ? '' : 's'}`);
    } catch (error) {
      setBusy(false, '');
      setState('error', error instanceof Error ? error.message : 'Unable to load games.');
      throw error;
    }
  }

  async function createGame(playerName: string, numStars: number, playerColor: string) {
    if (!api()) {
      throw new Error('Connect to a server first.');
    }
    setBusy(true, 'Creating game');
    clearError();
    try {
      const gameId = await api()!.createGame(playerName, numStars, playerColor);
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

  async function changeBuilding(planetId: number, buildingId: number) {
    if (!api() || !state.session.selectedGameId) {
      return;
    }
    setBusy(true, 'Queuing build project');
    clearError();
    try {
      await api()!.buildOnPlanet(state.session.selectedGameId, planetId, buildingId);
      setState('info', 'Waiting for server update...');
    } catch (error) {
      setBusy(false, '');
      setState('error', error instanceof Error ? error.message : 'Unable to change building.');
      throw error;
    }
  }

  function addMovePopulationCommand(command: MovePopulationCommand) {
    setState('pendingCommands', 'movePopulation', (cmds) => [...cmds, command]);
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
      await api()!.endTurn(state.session.selectedGameId, state.pendingCommands);
      setState('pendingCommands', { movePopulation: [] });
      setState('info', 'Waiting for server update...');
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
    addMovePopulationCommand,
    colonize,
    exploreStar,
    endTurn,
    saveShipDesign,
    clearError,
  };

  return <AppContext.Provider value={value}>{props.children}</AppContext.Provider>;
}

export function useAppContext() {
  const context = useContext(AppContext);
  if (!context) {
    throw new Error('App context is not available.');
  }
  return context;
}
