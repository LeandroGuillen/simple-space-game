import { createReducer, on } from '@ngrx/store';

import { GameListResponseItem } from '../game.service';
import { emptyGame, Game } from '../model/model';
import {
  changeBuildingOK,
  getGameListOK,
  newGameOK,
  startGame,
  turnOK,
  updateGame,
} from './actions';

export type AppState = {
	game: GameState;
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
	on(updateGame, (state, { game }) => ({ ...state, currentGame: game })),
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
);
