import {Injectable} from "@angular/core";
import {BehaviorSubject, Observable} from "rxjs";
import {HttpClient} from "@angular/common/http";
import {filter, map, shareReplay, take} from "rxjs/operators";
import {Galaxy, Planet, Star} from "./model/model";
import {mapGalaxy} from "./model/mapper";
import {BEGalaxy} from "./model/be-model";

@Injectable({providedIn: "root"})
export class GameService {
  private static readonly HOST = 'http://localhost:8080'

  private _galaxy = new BehaviorSubject<Galaxy>({} as Galaxy);

  constructor(private readonly http: HttpClient) {
  }

  get currentGalaxyUpdates(): Observable<Galaxy> {
    return this._galaxy.asObservable().pipe(filter(g => !!g.stars));
  }

  createGalaxy(numStars: number): Observable<Galaxy | undefined> {
    return this.http.post<BEGalaxy>(`${GameService.HOST}/galaxy`, {numStars})
      .pipe(
        map(g => {
          const mappedGalaxy = mapGalaxy(g);
          this._galaxy.next(mappedGalaxy);
          return mappedGalaxy;
        }),
        shareReplay(1),
      )
  }

  getRandomGalaxy(): Observable<Galaxy> {
    return this.http.post<Galaxy>(`${GameService.HOST}/randomGalaxy`, {})
      .pipe(shareReplay(1))
  }

  getStarByName(name: string): Observable<Star> {
    return this.currentGalaxyUpdates.pipe(
      take(1),
      map(galaxy => {
        const star = galaxy.stars.find(star => star.name === name);
        if (star === undefined) {
          throw new Error(`star with name '${name}' was not found`)
        }
        return star;
      })
    )
  }

  getPlanetByName(name: string): Observable<Planet> {
    const starName = name.split(' ')[0];

    return this.getStarByName(starName).pipe(
      take(1),
      map(star => {
        const planet = star.planets.find(star => star.name === name);
        if (planet === undefined) {
          throw new Error(`planet with name '${name}' was not found`)
        }
        return planet;
      })
    )
  }
}
