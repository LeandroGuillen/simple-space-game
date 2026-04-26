import { Component, OnInit } from '@angular/core';
import { Store } from '@ngrx/store';
import { EMPTY, Observable } from 'rxjs';
import { map } from 'rxjs/operators';

import { Server } from '../../model/model';
import { connectToServer, startGame } from '../../store/actions';
import { AppState } from '../../store/reducers';
import { selectCurrentGameID, selectServerName } from '../../store/selectors/game.selectors';
import { selectServer } from '../../store/selectors/server.selectors';

@Component({
  selector: 'ssg-top-bar',
  standalone: false,
  templateUrl: './top-bar.component.html',
  styleUrls: ['./top-bar.component.scss'],
})
export class TopBarComponent implements OnInit {
  gameID!: Observable<string>;
  serverName!: Observable<string>;

  constructor(private readonly store: Store<AppState>) {}

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

        return '';
      }),
      map(id => id.substring(0, 8))
    );

    this.store
      .select(selectServer)
      .pipe(
        map(server => {
          if (server.url) {
            return server;
          }

          // if there is no current game going on, try to load from storage
          const serverString =
            window.localStorage.getItem('currentServer') || '';

          if (serverString) {
            const server: Server = JSON.parse(serverString);
            this.store.dispatch(connectToServer({ server }));
            return server;
          }
          return EMPTY;
        })
      )
      .subscribe();

    this.serverName = this.store.select(selectServerName);
  }
}
