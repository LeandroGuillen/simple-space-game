import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Store } from '@ngrx/store';
import { Observable } from 'rxjs';
import { filter, map } from 'rxjs/operators';

import { BEGame, BEPlanet } from '../model/be-model';
import { mapGame, mapPlanet } from '../model/mapper';
import { Game, Planet, Server } from '../model/model';
import { NewGameRequest } from '../store/actions';
import { AppState } from '../store/reducers';
import { selectCurrentGameID } from '../store/selectors/game.selectors';

type NewGameResponse = {
  ok: boolean;
  gameID: string;
};

export type GameListResponseItem = {
  game_id: string;
  num_stars: number;
  current_turn: number;
};
export type GameListResponse = {
  game_list: GameListResponseItem[];
};

@Injectable({ providedIn: 'root' })
export class GameService {
  private currentGameID: string;
  #host: string;

  constructor(
    private readonly http: HttpClient,
    private readonly store: Store<AppState>
  ) {
    this.#host = window.localStorage.getItem('currentServerHost') || '';

    this.currentGameID = window.localStorage.getItem('currentGameID') || '';

    this.store.select(selectCurrentGameID).subscribe(gameID => {
      if (gameID) {
        window.localStorage.setItem('currentGameID', gameID);
        this.currentGameID = gameID;
      } else {
        this.currentGameID = window.localStorage.getItem('currentGameID') || '';
      }
    });
  }

  establishConnection(server: Server): Observable<Server> {
    if (!server.url.startsWith('http')) {
      this.#host = 'http://' + server.url;
    }
    this.#host = server.url;

    console.log('trying', server.url);
    return this.http.get<{ ok: boolean }>(`${this.#host}/status`).pipe(
      map(res => {
        if (res.ok) {
          window.localStorage.setItem('currentServerHost', this.#host);
          window.localStorage.setItem('currentServer', JSON.stringify(server));
          window.localStorage.setItem('currentGameID', '');
          this.currentGameID = '';
          return server;
        }
        window.localStorage.setItem('currentServerHost', '');
        throw new Error(`cannot connect to '${server.url}'`);
      })
    );
  }

  checkServerHealth(server: Server): Observable<boolean> {
    return this.http
      .get<{ ok: boolean }>(`${server.url}/status`)
      .pipe(map(res => res.ok));
  }

  getGalaxy(gameID: string): Observable<Game> {
    return this.http.get<BEGame>(`${this.#host}/games/${gameID}`).pipe(
      map(g => {
        window.localStorage.setItem('currentGameID', gameID);
        return mapGame(g);
      })
    );
  }

  colonize(ownerID: number, planetID: number): Observable<Game> {
    const url = `${this.#host}/action/${ownerID}/colonize/${planetID}`;
    return this.http
      .post<{ game: Game }>(url, { game_id: this.currentGameID })
      .pipe(map(({ game }) => game));
  }

  turn(): Observable<boolean> {
    return this.http
      .get<{ ok: boolean }>(`${this.#host}/turn`, {
        headers: {
          'game-id': this.currentGameID,
          'player-id': '1',
        },
      })
      .pipe(
        map(res => {
          if (res.ok) {
            return true;
          }
          throw Error('cannot complete turn');
        })
      );
  }

  newGame(newGameRequest: NewGameRequest): Observable<string> {
    return this.http
      .post<NewGameResponse>(`${this.#host}/games`, newGameRequest)
      .pipe(
        map(res => {
          if (!res.ok) {
            throw new Error('could not create game');
          }
          return res.gameID;
        })
      );
  }

  getGameList(): Observable<GameListResponseItem[]> {
    return this.http.get<GameListResponse>(`${this.#host}/games`).pipe(
      map(res =>
        res.game_list.sort((a, b) => {
          return a.game_id.localeCompare(b.game_id);
        })
      )
    );
  }

  getColonyList(): Observable<Planet[]> {
    const getColonyListRequest = {
      game_id: this.currentGameID,
      player_id: 1,
    };
    return this.http
      .post<BEPlanet[]>(`${this.#host}/colonies`, getColonyListRequest)
      .pipe(map(planets => planets.map(mapPlanet)));
  }

  changeBuilding(
    planetID: number,
    buildingID: number
  ): Observable<{ planetID: number; buildingID: number }> {
    const changeBuildingRequest = {
      game_id: this.currentGameID,
      owner_id: 1,
      planet_id: planetID,
      building_id: buildingID,
    };
    return this.http
      .post<{ ok: boolean }>(
        `${this.#host}/colonies/build`,
        changeBuildingRequest
      )
      .pipe(
        filter(res => res.ok),
        map(() => ({ planetID, buildingID }))
      );
  }
}
