import {Injectable} from "@angular/core";
import {Observable} from "rxjs";
import {HttpClient} from "@angular/common/http";
import {map, switchMap} from "rxjs/operators";
import {Planet} from "../model/model";
import {BEPlanet} from "../model/be-model";
import {mapPlanet} from "../model/mapper";
import {GameService} from "../game.service";

@Injectable({providedIn: "root"})
export class ColoniesPageService {
  private _host: string;

  constructor(private readonly http: HttpClient,
              private readonly gameService: GameService) {
    this._host = window.localStorage.getItem('currentServerHost') || '';
  }

  getColonyList(): Observable<Planet[]> {
    return this.gameService.currentGameID.pipe(
      switchMap(gameID => {
        const getColonyListRequest = {
          game_id: gameID,
          player_id: 1,
        }
        return this.http.post<BEPlanet[]>(`${this._host}/colonies`, getColonyListRequest)
      }),
      map(planets => planets.map(mapPlanet))
    )
  }

  changeBuilding(planetId: number, buildingId: number): Observable<boolean> {
    return this.gameService.currentGameID.pipe(
      switchMap(gameID => {
        const changeBuildingRequest = {
          game_id: gameID,
          owner_id: 1,
          planet_id: planetId,
          building_id: buildingId,
        }
        return this.http.post<{ ok: boolean }>(`${this._host}/colonies/build`, changeBuildingRequest);
      }),
      map(res => res.ok)
    )
  }

}
