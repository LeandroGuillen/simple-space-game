import { createAction, props } from '@ngrx/store';

import { Game, Server } from '../model/model';
import { GameListResponseItem } from '../services/game.service';

export type NewGameRequest = {
  num_stars: number;
  player_name: string;
};

export const colonize = createAction(
  '[Star View] Colonize',
  props<{ playerID: number; planetID: number }>()
);

export const connectToServer = createAction(
  '[Server Page] Connect to server',
  props<{ server: Server }>()
);
export const serverConnectionSuccess = createAction(
  '[Server Page] Server connected succesfully',
  props<{ server: Server }>()
);
export const checkServerHealth = createAction(
  '[Server Page] Check server health',
  props<{ server: Server }>()
);
export const serverHealthOk = createAction(
  '[Server Page] Server health OK',
  props<{ ok: boolean }>()
);

export const newGameRequest = createAction(
  '[Game Page] New game request',
  props<NewGameRequest>()
);

export const newGameOK = createAction(
  '[Game Page] New game ok',
  props<{ gameID: string }>()
);

export const startGame = createAction(
  '[Game Page] Start game',
  props<{ gameID: string }>()
);

export const updateGame = createAction(
  '[Game API] Updated game',
  props<{ game: Game }>()
);

export const updateGameList = createAction('[Game Page] Update game list');

export const getGameListOK = createAction(
  '[Game API] Get game list ok',
  props<{ game_list: GameListResponseItem[] }>()
);

export const turnRequest = createAction('[Top bar] Turn request');
export const turnOK = createAction(
  '[Top bar] Turn ok',
  props<{ game: Game }>()
);

export const changeBuildingRequest = createAction(
  '[Colonies Page] Change building request',
  props<{ planetID: number; buildingID: number }>()
);
export const changeBuildingOK = createAction(
  '[Colonies Page] Change building OK',
  props<{ planetID: number; buildingID: number }>()
);

export const colonizeRequest = createAction(
  '[Star View] Colonize request',
  props<{ planetID: number; playerID: number }>()
);
export const colonizeOK = createAction(
  '[Star View] Colonize ok',
  props<{ game: Game }>()
);

export const updateTaxRate = createAction(
  '[KPI bar] Update tax rate',
  props<{ newTaxRate: number }>()
);
