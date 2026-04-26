import { Component, OnInit } from '@angular/core';
import { MatDialog } from '@angular/material/dialog';
import { Store } from '@ngrx/store';
import { Observable } from 'rxjs';
import { filter, tap } from 'rxjs/operators';

import { GameListResponseItem } from '../../services/game.service';
import { newGameRequest, startGame, updateGameList } from '../../store/actions';
import { AppState } from '../../store/reducers';
import { selectCurrentGameID, selectGameList, selectServerName, selectTurn } from '../../store/selectors/game.selectors';
import { NewGameDialogComponent } from './new-game-dialog/new-game-dialog.component';

@Component({
  selector: 'ssg-game-page',
  standalone: false,
  templateUrl: './game-page.component.html',
  styleUrls: ['./game-page.component.scss'],
})
export class GamePageComponent implements OnInit {
  gameList!: Observable<GameListResponseItem[]>;
  currentGameID!: string;
  serverName!: Observable<string>;
  turn!: Observable<number>;
  numGames = 0;

  constructor(
    private readonly dialog: MatDialog,
    private readonly store: Store<AppState>
  ) {}

  ngOnInit() {
    this.store.dispatch(updateGameList());
    this.gameList = this.store.select(selectGameList).pipe(
      tap(gameList => {
        this.numGames = gameList.length;
      })
    );
    this.store.select(selectCurrentGameID).subscribe(gameID => {
      this.currentGameID = gameID;
    });
    this.serverName = this.store.select(selectServerName);
    this.turn = this.store.select(selectTurn);
  }

  openNewGameDialog(): void {
    this.dialog
      .open(NewGameDialogComponent)
      .afterClosed()
      .pipe(filter(request => !!request))
      .subscribe(request => {
        this.store.dispatch(newGameRequest(request));
      });
  }

  startGame(gameID: string) {
    this.store.dispatch(startGame({ gameID }));
  }
}
