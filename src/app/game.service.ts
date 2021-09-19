import {Injectable} from "@angular/core";
import {BehaviorSubject, merge, Observable} from "rxjs";
import {HttpClient} from "@angular/common/http";
import {filter, map, take, takeLast} from "rxjs/operators";
import {Galaxy, Planet, Star} from "./model/model";
import {mapGalaxy} from "./model/mapper";
import {BEGalaxy} from "./model/be-model";
import {createUrlResolverWithoutPackagePrefix} from "@angular/compiler";

@Injectable({providedIn: "root"})
export class GameService {
  private _galaxy = new BehaviorSubject<Galaxy>({} as Galaxy);
  private currentGameID: string;

  constructor(private readonly http: HttpClient) {
    this._host = window.localStorage.getItem('currentServerHost') || '';
    this.currentGameID = window.localStorage.getItem('currentGameID') || '';
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
    if (this.currentGameID !== '') {
      const currentGalaxy = this.getCurrentGalaxy(this.currentGameID)
      // return merge(galaxyWatcher, currentGalaxy)//.pipe(takeLast(1));
      return currentGalaxy;
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
    this.currentGameID = '';
    window.localStorage.removeItem('currentGameID');
  }

  setCurrentGame(gameID: string) {
    this.currentGameID = gameID;
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

  getPlanetById(planetId: number, starId: number): Observable<Planet> {
    return this.getStarById(starId).pipe(
      take(1),
      map(star => {
        const planet = star.planets.find(star => star.id === planetId);
        if (planet === undefined) {
          throw new Error(`planet with id '${planetId}' was not found`)
        }
        return planet;
      })
    )
  }

  colonize(ownerID: number, planetID: number): Observable<boolean> {
    const url = `${this._host}/action/${ownerID}/colonize/${planetID}`
    return this.http.post<BEGalaxy>(url, {game_id: this.currentGameID}).pipe(map(g => true))
  }
}
