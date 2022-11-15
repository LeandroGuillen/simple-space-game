import * as fromRouter from '@ngrx/router-store';
import { createReducer, on } from '@ngrx/store';

import { emptyGame, emptyServer, Game, Server } from '../model/model';
import { GameListResponseItem } from '../services/game.service';
import {
  changeBuildingOK,
  colonizeOK,
  getGameListOK,
  serverConnectionSuccess,
  updateGame,
  updateTaxRate,
} from './actions';

export type AppState = {
  game: GameState;
  router: fromRouter.RouterReducerState;
};

export type GameState = {
  currentGame: Game;
  currentServer: Server;
  gameList: GameListResponseItem[];
};

const initialState: GameState = {
  currentGame: emptyGame(),
  currentServer: emptyServer(),
  gameList: [],
};

export const gameReducer = createReducer(
  initialState,
  on(getGameListOK, (state, { game_list }) => ({
    ...state,
    gameList: game_list,
  })),
  on(updateGame, (state, { game }) => {
    return {
      ...state,
      currentGame: game,
    };
  }),
  on(changeBuildingOK, (state, foo) => {
    const game = state.currentGame;
    const stars = game.galaxy.stars.map(star => {
      return {
        ...star,
        planets: star.planets.map(planet =>
          planet.id === foo.planetID
            ? { ...planet, current_project: foo.buildingID }
            : { ...planet }
        ),
      };
    });

    return {
      ...state,
      currentGame: {
        ...state.currentGame,
        galaxy: {
          ...state.currentGame?.galaxy,
          stars,
        },
      },
    };
  }),
  on(colonizeOK, (state, { game }) => ({
    ...state,
    currentGame: {
      galaxy: game.galaxy,
      id: game.id,
      players: game.players,
      currentTurn: game.currentTurn,
      resources: game.resources,
    },
  })),
  on(serverConnectionSuccess, (state, { server }) => ({
    ...state,
    currentServer: {
      id: server.id,
      name: server.name,
      url: server.url,
      isOnline: true,
    },
  })),
  on(updateTaxRate, (state, { newTaxRate }) => ({
    ...state,
    currentGame: {
      ...state.currentGame,
      players: [
        {
          ...state.currentGame.players[0],
          taxRate: newTaxRate,
        },
        ...state.currentGame.players.slice(1),
      ],
    },
  }))
);
