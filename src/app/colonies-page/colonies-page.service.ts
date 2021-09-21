import {Injectable} from "@angular/core";
import {Observable} from "rxjs";
import {HttpClient} from "@angular/common/http";
import {map} from "rxjs/operators";
import {Planet} from "../model/model";
import {BEPlanet} from "../model/be-model";
import {mapPlanet} from "../model/mapper";

@Injectable({providedIn: "root"})
export class ColoniesPageService {
  private currentGameID: string;
  private _host: string;

  constructor(private readonly http: HttpClient) {
    this._host = window.localStorage.getItem('currentServerHost') || '';
    this.currentGameID = window.localStorage.getItem('currentGameID') || '';
  }

  getColonyList(): Observable<Planet[]> {
    const getColonyListRequest = {
      game_id: this.currentGameID,
      player_id: 1,
    }
    return this.http.post<BEPlanet[]>(`${this._host}/colonies`, getColonyListRequest)
      .pipe(map(planets => planets.map(mapPlanet)))
  }
}
