import { Component, OnInit } from '@angular/core';
import { Router } from '@angular/router';
import { Store } from '@ngrx/store';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';

import { startGame, turnRequest } from '../store/actions';
import { AppState } from '../store/reducers';
import { selectCurrentGameID, selectPlayerTreasury } from '../store/selectors';

@Component({
  selector: 'crn-top-bar',
  templateUrl: './top-bar.component.html',
  styleUrls: [ './top-bar.component.scss' ]
})
export class TopBarComponent implements OnInit {
  gameID!: Observable<string>;
  treasury!: Observable<number | undefined>;

  constructor(
    private readonly router: Router,
    private readonly store: Store<AppState>) {
  }

  ngOnInit(): void {
    this.gameID = this.store.select(selectCurrentGameID).pipe(
      map(id => {
        if (id) {
          return id;
        }

        // if there is no current game going on, try to load from storage
        const gameID = window.localStorage.getItem('currentGameID') || '';

        if (gameID) {
          this.store.dispatch(startGame({ gameID }));
          return gameID;
        }

        // if still no game, go to Game Screen
        this.router.navigate([ 'game' ])
        return ''
      }),
      map(id => id.substring(0, 8))
    );
    this.treasury = this.store.select(selectPlayerTreasury(1));

  }

  turn() {
    this.store.dispatch(turnRequest());
  }
}
