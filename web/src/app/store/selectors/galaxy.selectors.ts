import { createSelector } from '@ngrx/store';

import { AppState } from '../reducers';
import { selectCurrentPlayer } from './player.selectors';

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
	selectAllStars, selectCurrentPlayer, (allStars, player) => {
		if (!allStars || !player) {
			return [];
		}
		return allStars.map(star => {
			return star.planets.filter(planet => planet.owner === player.id);
		}).reduce((planets, newPlanets) => [ ...planets, ...newPlanets ])
	}
)

