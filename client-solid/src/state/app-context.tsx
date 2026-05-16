import { createContext, createEffect, createMemo, onCleanup, onMount, useContext, type ParentProps } from 'solid-js';
import { createStore } from 'solid-js/store';

import { ApiClient, parseGameUpdate } from '../lib/api';
import { mapGame, mapPlayerColorOptions } from '../lib/mappers';
import type { ApiGameListItem, NewGameOptions } from '../lib/types/api';
import type { BuildingViewModel, GameViewModel, PlayerColorOptions, ShipDesignForm } from '../lib/types/domain';
import type { ArtificialPlanetCommand, ColonizeCommand, FleetMoveCommand, MovePopulationCommand, PendingCommands, SelectResearchCommand, TurnCommands } from '../lib/types/commands';
import { clearSessionLastSetupServerUrl } from '../lib/setup-session-storage';
import { isAutobuildProject } from '../lib/construction';
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
  pendingColonyShipFleets: number[]; // fleets where Colony Ship command completed
  pendingArtificialPlanetSourcePlanets: number[]; // planets where Artificial Planet Construction completed
  showColonizeModal: boolean;
  showArtificialPlanetModal: boolean;
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
  setPlanetAutobuild: (planetId: number, enabled: boolean) => Promise<void>;
  addMovePopulationCommand: (command: MovePopulationCommand) => void;
  addFleetMoveCommand: (command: FleetMoveCommand) => void;
  addColonizeCommand: (command: ColonizeCommand) => void;
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
    colonize: [],
    artificialPlanet: [],
    buyProduction: [],
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
    pendingColonyShipFleets: [],
    pendingArtificialPlanetSourcePlanets: [],
    showColonizeModal: false,
    showArtificialPlanetModal: false,
  });

  const api = createMemo(() =>
    state.session.serverUrl ? new ApiClient(state.session.serverUrl, state.session.playerId) : null,
  );

  const currentPlayer = createMemo(() => getPrimaryPlayer(state.game));
  const colonies = createMemo(() =>
    getColonyRows(state.game, state.pendingCommands, state.session.playerId),
  );

  function clearError() {
    setState('error', '');
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
      setState('game', null);
      resetPerGameUiState();
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
    if (gameId !== state.session.selectedGameId) {
      setState('game', null);
      resetPerGameUiState();
    }
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

      setBusy(false, completionInfo);
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
            isAutobuildProject(project),
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
    changeBuildQueue,
    setPlanetAutobuild,
    addMovePopulationCommand,
    addFleetMoveCommand,
    addColonizeCommand,
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
    colonize,
    exploreStar,
    endTurn,
    saveShipDesign,
    clearError,
  };

  return <AppContext.Provider value={value}>{props.children}</AppContext.Provider>;
}

export function toTurnCommands(commands: PendingCommands, game: GameViewModel | null): TurnCommands {
  const turnCommands: TurnCommands = {
    build: commands.build,
  };

  if (commands.selectResearch) {
    turnCommands.selectResearch = commands.selectResearch;
  }

  if (commands.colonize.length === 1) {
    turnCommands.colonize = commands.colonize[0];
  } else if (commands.colonize.length > 1) {
    turnCommands.colonize_many = commands.colonize;
  }
  if (commands.artificialPlanet.length === 1) {
    turnCommands.artificial_planet = commands.artificialPlanet[0];
  } else if (commands.artificialPlanet.length > 1) {
    turnCommands.artificial_planet_many = commands.artificialPlanet;
  }
  if (commands.fleetMove.length > 0) {
    turnCommands.fleet_move = commands.fleetMove.map((cmd) => ({
      fleet_id: cmd.fleet_id,
      destination_star_id: cmd.destination_star_id,
      ship_indexes: [...cmd.ship_indexes],
    }));
  }

  if (commands.buyProduction.length > 0) {
    turnCommands.buy_production = commands.buyProduction.map((cmd) => ({
      planetId: cmd.planetId,
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
