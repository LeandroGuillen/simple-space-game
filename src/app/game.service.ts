import {Injectable} from "@angular/core";
import {BehaviorSubject, Observable} from "rxjs";
import {HttpClient} from "@angular/common/http";
import {filter, map, take} from "rxjs/operators";
import {Galaxy, Planet, Star} from "./model/model";
import {mapGalaxy} from "./model/mapper";
import {BEGalaxy} from "./model/be-model";

@Injectable({providedIn: "root"})
export class GameService {
  private _galaxy = new BehaviorSubject<Galaxy>({} as Galaxy);
  private _galaxyIDSub = new BehaviorSubject('');

  constructor(private readonly http: HttpClient) {
    this._host = window.localStorage.getItem('currentServerHost') || '';
    this._currentGameID = window.localStorage.getItem('currentGameID') || '';
    this._galaxyIDSub.next(this._currentGameID);
  }

  private _currentGameID: string;

  get currentGameID(): Observable<string> {
    return this._galaxyIDSub.asObservable();
  }

  private _host: string;

  set host(host: string) {
    if (!host.startsWith('http')) {
      this._host = 'http://' + host;
    }
    this._host = host;
  }

  get currentGalaxyUpdates(): Observable<Galaxy> {
    const galaxyWatcher = this._galaxy.asObservable().pipe(filter(g => !!g.stars))
    if (this._currentGameID !== '') {
      return this.getCurrentGalaxy(this._currentGameID);
    }
    return galaxyWatcher;
  }

  establishConnection(): Observable<boolean> {
    return this.http.get<{ ok: boolean }>(`${this._host}/status`).pipe(
      map(res => {
        if (res.ok) {
          window.localStorage.setItem('currentServerHost', this._host);
          return true;
        }
        window.localStorage.setItem('currentServerHost', '');
        return false;
      })
    )
  }

  getCurrentGalaxy(gameID: string): Observable<Galaxy> {
    return this.http.get<BEGalaxy>(`${this._host}/galaxy/${gameID}`).pipe(
      map(g => {
        const mappedGalaxy = mapGalaxy(g);
        this._galaxy.next(mappedGalaxy);
        return mappedGalaxy;
      }))
  }

  clearCurrentGame(): void {
    this._currentGameID = '';
    window.localStorage.removeItem('currentGameID');
  }

  setCurrentGame(gameID: string) {
    this._currentGameID = gameID;
    this._galaxyIDSub.next(gameID);
    window.localStorage.setItem('currentGameID', gameID);
  }

  getStarById(id: number): Observable<Star> {
    return this.currentGalaxyUpdates.pipe(
      take(1),
      map(galaxy => {
        const star = galaxy.stars.find(star => star.id === id);
        if (star === undefined) {
          throw new Error(`star with id '${id}' was not found`)
        }
        return star;
      })
    )
  }


  getPlanetById(planetId: number): Observable<Planet | undefined> {
    return this.currentGalaxyUpdates.pipe(
      map(galaxy => {
        for (let star of galaxy.stars) {
          const planet = star.planets.find(p => p.id === planetId);
          if (planet !== undefined) {
            return planet;
          }
        }
        return undefined;
      })
    )
  }

  colonize(ownerID: number, planetID: number): Observable<boolean> {
    const url = `${this._host}/action/${ownerID}/colonize/${planetID}`
    return this.http.post<BEGalaxy>(url, {game_id: this._currentGameID}).pipe(map(() => true))
  }
}
