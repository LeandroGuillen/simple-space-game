import { AppState } from '../reducers';

export const selectCurrentGame = (state: AppState) => state.game.currentGame;
export const selectCurrentGameID = (state: AppState) => state.game.currentGame.id;
export const selectTurn = (state: AppState) => state.game.currentGame.currentTurn;
export const selectGameList = (state: AppState) => state.game.gameList;
export const selectServerName = (state: AppState) => state.game.currentServer.name;
