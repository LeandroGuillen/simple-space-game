import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Store } from '@ngrx/store';
import { Observable, of, throwError } from 'rxjs';
import { filter, map, switchMap } from 'rxjs/operators';

import { BEPlanet } from '../model/be-model';
import { mapGalaxy, mapPlanet } from '../model/mapper';
import { Game, Planet } from '../model/model';
import { NewGameRequest } from '../store/actions';
import { AppState } from '../store/reducers';
import { selectCurrentGameID } from '../store/selectors';

type NewGameResponse = {
  ok: boolean;
  gameID: string;
}

export type GameListResponseItem = {
  game_id: string;
  num_stars: number;
}
export type GameListResponse = {
  game_list: GameListResponseItem[];
}

@Injectable({ providedIn: "root" })
export class GameService {
  private currentGameID: string;
  #host: string;

  constructor(private readonly http: HttpClient,
    private readonly store: Store<AppState>) {
    this.#host = window.localStorage.getItem('currentServerHost') || '';
    this.currentGameID = window.localStorage.getItem('currentGameID') || '';

    this.store.select(selectCurrentGameID).subscribe(gameID => {
      if (gameID) {
        window.localStorage.setItem('currentGameID', gameID);
      } else {
        this.currentGameID = window.localStorage.getItem('currentGameID') || '';
      }
      this.currentGameID = gameID;
    })
  }

  establishConnection(host: string): Observable<boolean> {
    if (!host.startsWith('http')) {
      this.#host = 'http://' + host;
    }
    this.#host = host;
    console.log('trying', host);
    return this.http.get<{ ok: boolean }>(`${this.#host}/status`).pipe(
      map(res => {
        if (res.ok) {
          window.localStorage.setItem('currentServerHost', this.#host);
          return true;
        }
        window.localStorage.setItem('currentServerHost', '');
        return false;
      })
    )
  }

  getGalaxy(gameID: string): Observable<Game> {
    return this.http.get<Game>(`${this.#host}/games/${gameID}`).pipe(
      map(g => {
        g.galaxy = mapGalaxy(g.galaxy);
        window.localStorage.setItem('currentGameID', gameID);
        return g;
      }))
  }

  colonize(ownerID: number, planetID: number): Observable<Game> {
    const url = `${this.#host}/action/${ownerID}/colonize/${planetID}`
    return this.http.post<{ game: Game }>(url, { game_id: this.currentGameID })
      .pipe(map(({ game }) => game));
  }

  turn(): Observable<Game> {
    return this.http.post<{ ok: boolean, game: Game }>(`${this.#host}/turn`, {
      game_id: this.currentGameID,
      player_id: 1
    }).pipe(map(res => res.game));
  }

  newGame(newGameRequest: NewGameRequest): Observable<string> {
    return this.http.post<NewGameResponse>(`${this.#host}/games`, newGameRequest).pipe(
      switchMap(res => {
        if (!res.ok) {
          return throwError('could not create game');
        }
        return of(res.gameID);
      })
    )
  }

  getGameList(): Observable<GameListResponseItem[]> {
    return this.http.get<GameListResponse>(`${this.#host}/games`).pipe(
      map(res => res.game_list.sort((a, b) => {
        return a.game_id.localeCompare(b.game_id)
      })),
    );
  }

  getColonyList(): Observable<Planet[]> {
    const getColonyListRequest = {
      game_id: this.currentGameID,
      player_id: 1,
    }
    return this.http.post<BEPlanet[]>(`${this.#host}/colonies`, getColonyListRequest)
      .pipe(map(planets => planets.map(mapPlanet)))
  }

  changeBuilding(planetID: number, buildingID: number): Observable<{ planetID: number, buildingID: number }> {
    const changeBuildingRequest = {
      game_id: this.currentGameID,
      owner_id: 1,
      planet_id: planetID,
      building_id: buildingID,
    }
    return this.http.post<{ ok: boolean }>(`${this.#host}/colonies/build`, changeBuildingRequest)
      .pipe(filter(res => res.ok), map(() => ({ planetID, buildingID })))
  }
}
