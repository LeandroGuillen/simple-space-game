import {Injectable} from "@angular/core";
import {Observable, of, throwError} from "rxjs";
import {HttpClient} from "@angular/common/http";
import {map, switchMap} from "rxjs/operators";
import {GameService} from "../game.service";

export type GameListResponseItem = {
  game_id: string;
  num_stars: number;
}
type GameListResponse = {
  game_list: GameListResponseItem[];
}

export type NewGameRequest = {
  num_stars: number;
  player_name: string;
}
type NewGameResponse = {
  ok: boolean;
  gameID: string
}

@Injectable({providedIn: "root"})
export class GamePageService {

  private _host: string;

  constructor(private readonly http: HttpClient,
              private readonly gameService: GameService) {
    this._host = window.localStorage.getItem('currentServerHost') || '';
  }

  getGameList(): Observable<GameListResponseItem[]> {
    return this.http.get<GameListResponse>(`${this._host}/games`).pipe(
      map(res => res.game_list.sort((a, b) => {
        return a.game_id.localeCompare(b.game_id)
      })),
    );
  }

  newGame(newGameRequest: NewGameRequest): Observable<string> {
    return this.http.post<NewGameResponse>(`${this._host}/games`, newGameRequest).pipe(
      switchMap(response => {
        if (!response.ok) {
          return throwError('could not create game');
        }
        this.gameService.setCurrentGame(response.gameID);
        return of(response.gameID);
      })
    )
  }

}
