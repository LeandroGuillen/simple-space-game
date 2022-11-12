import { AppState } from '../reducers';

export const selectServer = (state: AppState) => state.game.currentServer;
