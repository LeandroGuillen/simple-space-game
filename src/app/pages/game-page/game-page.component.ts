import { Component, OnInit } from '@angular/core';
import { MatDialog } from '@angular/material/dialog';
import { Store } from '@ngrx/store';
import { Observable } from 'rxjs';
import { filter, tap } from 'rxjs/operators';

import { GameListResponseItem } from '../../services/game.service';
import * as Actions from '../../store/actions';
import { AppState } from '../../store/reducers';
import { selectCurrentGameID, selectGameList, selectServerName, selectTurn } from '../../store/selectors/game.selectors';
import { NewGameDialogComponent } from './new-game-dialog/new-game-dialog.component';

@Component({
  selector: 'ssg-game-page',
  templateUrl: './game-page.component.html',
  styleUrls: ['./game-page.component.scss'],
})
export class GamePageComponent implements OnInit {
  gameList!: Observable<GameListResponseItem[]>;
  currentGameID!: Observable<string>;
  serverName!: Observable<string>;
  turn!: Observable<number>;
  numGames = 0;

  constructor(
    private readonly dialog: MatDialog,
    private readonly store: Store<AppState>
  ) {}

  ngOnInit() {
    this.store.dispatch(Actions.updateGameList());
    this.gameList = this.store.select(selectGameList).pipe(
      tap(gameList => {
        this.numGames = gameList.length;
      })
    );
    this.currentGameID = this.store.select(selectCurrentGameID);
    this.serverName = this.store.select(selectServerName);
    this.turn = this.store.select(selectTurn);
  }

  openNewGameDialog(): void {
    this.dialog
      .open(NewGameDialogComponent)
      .afterClosed()
      .pipe(filter(request => !!request))
      .subscribe(request => {
        this.store.dispatch(Actions.newGameRequest(request));
      });
  }

  startGame(gameID: string) {
    this.store.dispatch(Actions.startGame({ gameID }));
  }
}
