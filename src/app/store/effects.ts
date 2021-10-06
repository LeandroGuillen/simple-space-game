import { Injectable } from '@angular/core';
import { Actions, createEffect, ofType } from '@ngrx/effects';
import { EMPTY } from 'rxjs';
import { catchError, map, mergeMap, switchMap } from 'rxjs/operators';

import { GameService } from '../services/game.service';
import * as actions from './actions';


@Injectable()
export class GameEffects {
  newGame$ = createEffect(() => this.actions$.pipe(
    ofType(actions.newGameRequest),
    mergeMap(req => this.gameService.newGame(req).pipe(
      switchMap(gameID => [
        actions.newGameOK({ gameID }),
        actions.startGame({ gameID })
      ]),
      catchError(() => EMPTY)
    ))
  ));

  getGameList$ = createEffect(() => this.actions$.pipe(
    ofType(actions.updateGameList),
    mergeMap(() => this.gameService.getGameList().pipe(
      map(response => actions.getGameListOK({ game_list: response })),
      catchError(() => EMPTY)
    ))
  ));

  turn$ = createEffect(() => this.actions$.pipe(
    ofType(actions.turnRequest),
    mergeMap(() => this.gameService.turn().pipe(
      map(game => actions.turnOK({ game })),
      catchError(() => EMPTY)
    ))
  ));

  startGame$ = createEffect(() => this.actions$.pipe(
    ofType(actions.startGame),
    mergeMap(({ gameID }) => this.gameService.getGalaxy(gameID).pipe(
      map(game => actions.updateGame({ game })),
      catchError(() => EMPTY)
    ))
  ));

  changeBuilding$ = createEffect(() => this.actions$.pipe(
    ofType(actions.changeBuildingRequest),
    mergeMap(({ planetID, buildingID }) => this.gameService.changeBuilding(planetID, buildingID).pipe(
      map(({ planetID, buildingID }) => actions.changeBuildingOK({ planetID, buildingID })),
      catchError(() => EMPTY)
    ))
  ));

  colonize$ = createEffect(() => this.actions$.pipe(
    ofType(actions.colonizeRequest),
    mergeMap(({ planetID, playerID }) => this.gameService.colonize(playerID, planetID).pipe(
      map(game => actions.colonizeOK({ game })),
      catchError(() => EMPTY)
    ))
  ));

  constructor(
    private readonly actions$: Actions,
    private readonly gameService: GameService
  ) {
  }

}
