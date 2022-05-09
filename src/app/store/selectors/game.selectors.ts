import { AppState } from '../reducers';

export const selectCurrentGame = (state: AppState) => state.game.currentGame;
export const selectCurrentGameID = (state: AppState) =>
  selectCurrentGame(state).id;
export const selectTurn = (state: AppState) =>
  selectCurrentGame(state).currentTurn;
export const selectGameList = (state: AppState) => state.game.gameList;
export const selectServerName = (state: AppState) =>
  state.game.currentServer.name;
export const selectResources = (state: AppState) =>
  selectCurrentGame(state).resources;
export const selectShipParts = (state: AppState) =>
  selectResources(state).shipParts;
export const selectHullSizes = (state: AppState) =>
  selectShipParts(state).hullSizes;
export const selectDrives = (state: AppState) => selectShipParts(state).drives;
export const selectShield = (state: AppState) => selectShipParts(state).shields;
export const selectComputers = (state: AppState) =>
  selectShipParts(state).computers;
export const selectArmors = (state: AppState) => selectShipParts(state).armors;
export const selectCommonShipDesigns = (state: AppState) =>
  selectResources(state).shipDesignCollection.shipDesigns;
