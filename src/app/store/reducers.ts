import * as fromRouter from '@ngrx/router-store';
import { createReducer, on } from '@ngrx/store';

import { emptyGame, Game } from '../model/model';
import { GameListResponseItem } from '../services/game.service';
import {
  changeBuildingOK,
  colonizeOK,
  getGameListOK,
  newGameOK,
  startGame,
  turnOK,
  updateGame,
} from './actions';

export type AppState = {
	game: GameState;
	router: fromRouter.RouterReducerState;
}

export type GameState = {
	currentGameID: string;
	currentGame: Game;
	gameList: GameListResponseItem[];
}

const initialState: GameState = {
	currentGameID: '',
	currentGame: emptyGame(),
	gameList: [],
};

export const gameReducer = createReducer(
	initialState,
	on(newGameOK, (state, { gameID }) => ({ ...state, currentGameID: gameID })),
	on(getGameListOK, (state, { game_list }) => ({ ...state, gameList: game_list })),
	on(startGame, (state, { gameID }) => ({ ...state, currentGameID: gameID })),
	on(turnOK, (state, { game }) => ({ ...state, currentGame: game })),
	on(updateGame, (state, { game }) => {
		const newGame = JSON.parse(JSON.stringify(game)) as Game;
		return {
			...state,
			currentGameID: newGame.id,
			currentGame: { ...newGame }
		}
	}),
	on(changeBuildingOK, (state, foo) => {
		const game = state.currentGame;
		const stars = game.galaxy.stars.map(star => {
			return {
				...star,
				planets: star.planets.map(planet =>
					planet.id === foo.planetID ?
						{ ...planet, current_project: foo.buildingID }
						: { ...planet })
			}
		});

		return {
			...state,
			currentGameID: state.currentGameID,
			gameList: state.gameList,
			currentGame: {
				...state.currentGame,
				galaxy: {
					...state.currentGame?.galaxy,
					stars
				}
			}
		}
	}),
	on(colonizeOK, (state, { game }) => ({
		...state, currentGame: {
			galaxy: game.galaxy,
			id: game.id,
			players: game.players
		}
	})),
);
