import {Injectable} from "@angular/core";
import {Observable, of, throwError} from "rxjs";
import {HttpClient} from "@angular/common/http";
import {map, switchMap} from "rxjs/operators";

export type GameListResponseItem = {
  game_id: string;
  num_stars: number;
}
type GameListResponse = {
  game_list: GameListResponseItem[];
}

@Injectable({providedIn: "root"})
export class GamePageService {

  private currentGameID: string;
  private _host: string;

  constructor(private readonly http: HttpClient) {
    this._host = window.localStorage.getItem('currentServerHost') || '';
    this.currentGameID = window.localStorage.getItem('currentGameID') || '';
  }

  getGameList(): Observable<GameListResponseItem[]> {
    return this.http.get<GameListResponse>(`${this._host}/games`).pipe(
      map(res => res.game_list.sort((a, b)=>{
        return a.game_id.localeCompare(b.game_id)
      })),
    );
  }


  newGame(numStars: number): Observable<string> {
    return this.http.post<{ ok: boolean, gameID: string }>(`${this._host}/games`, {numStars}).pipe(
      switchMap(response => {
        if (!response.ok) {
          return throwError('could not create game');
        }
        window.localStorage.setItem('currentGameID', response.gameID)
        return of(response.gameID);
      })
    )
  }

}
