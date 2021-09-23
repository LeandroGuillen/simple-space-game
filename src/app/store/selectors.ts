import { createSelector } from '@ngrx/store';

import { AppState } from './reducers';

// game info
export const selectCurrentGame = (state: AppState) => state.game.currentGame;
export const selectCurrentGameID = (state: AppState) => state.game.currentGameID;
export const selectGameList = (state: AppState) => state.game.gameList;

// player data
export const selectAllPlayers = (state: AppState) => state.game.currentGame?.players;
export const selectPlayer = (playerID: number) => (state: AppState) =>
	selectAllPlayers(state)?.find(player => player.id === playerID);
export const selectPlayerTreasury = (playerID: number) => (state: AppState) =>
	selectPlayer(playerID)(state)?.treasury;

// galaxy
export const selectGalaxy = (state: AppState) => state.game.currentGame?.galaxy;

// stars
export const selectAllStars = (state: AppState) => state.game.currentGame?.galaxy.stars;
export const selectStar = (starID: number) => (state: AppState) =>
	selectAllStars(state)?.find(star => star.id === starID)

// planets
export const selectAllPlanets = (state: AppState) =>
	selectAllStars(state)?.map(star => star.planets)
		.reduce((planets, newPlanets) => [ ...planets, ...newPlanets ])
export const selectPlanet = (planetID: number) => (state: AppState) =>
	selectAllPlanets(state)?.find(planet => planet.id === planetID)

export const selectColonies = createSelector(
	selectAllStars, selectPlayer(1), (allStars, player) => {
		if (!allStars || !player) {
			return [];
		}
		return allStars.map(star => {
			return star.planets.filter(planet => planet.owner === player.id);
		}).reduce((planets, newPlanets) => [ ...planets, ...newPlanets ])
	}
)

