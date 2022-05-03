import { createSelector } from '@ngrx/store';

import { AppState } from '../reducers';

const selectAllPlayers = (state: AppState) => state.game.currentGame?.players;
const selectPlayer = (playerID: number) => (state: AppState) =>
  selectAllPlayers(state)?.find(player => player.id === playerID);

export const selectCurrentPlayer = selectPlayer(1);
export const selectTreasury = createSelector(
  selectCurrentPlayer,
  player => player?.treasury || 0
);
export const selectTaxRate = createSelector(
  selectCurrentPlayer,
  player => player?.taxRate || 0
);
export const selectShipDesigns = createSelector(
  selectCurrentPlayer,
  player => player?.shipDesigns
);
