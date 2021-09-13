import {Injectable} from "@angular/core";
import {BehaviorSubject, Observable} from "rxjs";
import {HttpClient} from "@angular/common/http";
import {filter, map, shareReplay, take, tap} from "rxjs/operators";
import {Galaxy, Planet, Star} from "./model/model";
import {mapGalaxy} from "./model/mapper";
import {BEGalaxy} from "./model/be-model";

@Injectable({providedIn: "root"})
export class GameService {
  private _galaxy = new BehaviorSubject<Galaxy>({} as Galaxy);
  private _host: string;

  constructor(private readonly http: HttpClient) {
    this._host = window.localStorage.getItem('currentServerHost') || '';
  }


  set host(host: string) {
    if (!host.startsWith('http')) {
      this._host = 'http://'+host;
    }
    this._host = host;
  }

  get currentGalaxyUpdates(): Observable<Galaxy> {
    return this._galaxy.asObservable().pipe(filter(g => !!g.stars));
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

  createGalaxy(numStars: number): Observable<Galaxy | undefined> {
    return this.http.post<BEGalaxy>(`${this._host}/galaxy`, {numStars}).pipe(
      map(g => {
        const mappedGalaxy = mapGalaxy(g);
        this._galaxy.next(mappedGalaxy);
        return mappedGalaxy;
      }),
      shareReplay(1),
    )
  }

  createRandomGalaxy(): Observable<Galaxy> {
    return this.http.post<Galaxy>(`${this._host}/randomGalaxy`, {})
      .pipe(shareReplay(1))
  }

  getCurrentGalaxy(): Observable<Galaxy> {
    return this.http.get<BEGalaxy>(`${this._host}/galaxy`).pipe(
      map(g => {
        const mappedGalaxy = mapGalaxy(g);
        this._galaxy.next(mappedGalaxy);
        return mappedGalaxy;
      })
    )
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
    return this.http.get<BEGalaxy>(url).pipe(map(g => true))
  }
}
