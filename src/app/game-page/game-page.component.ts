import { Component, OnInit } from '@angular/core';
import { MatDialog } from '@angular/material/dialog';
import { Router } from '@angular/router';
import { Store } from '@ngrx/store';
import { Observable } from 'rxjs';

import { GameListResponseItem } from '../game.service';
import * as Actions from '../store/actions';
import { AppState } from '../store/reducers';
import { selectCurrentGameID, selectGameList } from '../store/selectors';
import { NewGameDialogComponent } from './new-game-dialog/new-game-dialog.component';

@Component({
  selector: 'crn-game-page',
  templateUrl: './game-page.component.html',
  styleUrls: [ './game-page.component.scss' ]
})
export class GamePageComponent implements OnInit {
  gameList!: Observable<GameListResponseItem[]>;
  gameID!: Observable<string>;

  constructor(
    private readonly dialog: MatDialog,
    private readonly route: Router,
    private readonly store: Store<AppState>) {
  }

  ngOnInit() {
    this.store.dispatch(Actions.updateGameList());
    this.gameList = this.store.select(selectGameList);
    this.gameID = this.store.select(selectCurrentGameID);
  }

  openNewGameDialog(): void {
    this.dialog.open(NewGameDialogComponent).afterClosed()
      .subscribe(request => {
        this.store.dispatch(Actions.newGameRequest(request));
        this.store.dispatch(Actions.updateGameList());
      });
  }

  startGame(gameID: string) {
    this.store.dispatch(Actions.startGame({ gameID }));
    this.route.navigate([ 'stars' ]);
  }

}
