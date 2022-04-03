import * as fromRouter from '@ngrx/router-store';
import { createSelector } from '@ngrx/store';

import { AppState } from './reducers';

// game info
export const selectCurrentGame = (state: AppState) => state.game.currentGame;
export const selectCurrentGameID = (state: AppState) => state.game.currentGameID;
export const selectGameList = (state: AppState) => state.game.gameList;
export const selectServerName = (state: AppState) => state.game.currentServer.name;
export const selectServer = (state: AppState) => state.game.currentServer;

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
export const selectAllPlanets = (state: AppState) => {
	const stars = selectAllStars(state);
	if (!stars || !stars.length) {
		return [];
	}
	return stars.map(star => star.planets)
		.reduce((planets, newPlanets) => [ ...planets, ...newPlanets ])
}

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



export const getRouterState = (state: AppState) => state.router;

export const getCurrentRouteState = createSelector(
	getRouterState,
	(router: fromRouter.RouterReducerState) => router.state
);


