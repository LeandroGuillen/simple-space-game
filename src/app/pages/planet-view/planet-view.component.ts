import { Location } from '@angular/common';
import { Component, OnDestroy, OnInit } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { Store } from '@ngrx/store';
import { Subscription } from 'rxjs';
import { map, switchMap } from 'rxjs/operators';

import { Planet, PlanetGravity, PlanetMineral, PlanetSize } from '../../model/model';
import { AppState } from '../../store/reducers';
import { getCurrentRouteState, selectPlanet } from '../../store/selectors';

@Component({
  selector: 'crn-planet-view',
  templateUrl: './planet-view.component.html',
  styleUrls: [ './planet-view.component.scss' ]
})
export class PlanetViewComponent implements OnInit, OnDestroy {
  planet!: Planet;
  size!: string;
  gravity!: string;
  mineral_class!: string;

  sub!: Subscription;

  constructor(
    private readonly store: Store<AppState>,
    private readonly route: ActivatedRoute,
    private readonly location: Location) {
  }

  private static getSizeString(ps: PlanetSize): string {
    switch (ps) {
      case 1:
        return 'Tiny';
      case 2:
        return 'Small';
      case 3:
        return 'Medium';
      case 4:
        return 'Large';
      case 5:
        return 'Huge';
      default:
        return 'Invalid!'
    }
  }

  private static getGravityString(ps: PlanetGravity): string {
    switch (ps) {
      case 1:
        return 'Low';
      case 2:
        return 'Normal';
      case 3:
        return 'High';
      default:
        return 'Invalid!'
    }
  }

  private static getMineralClassString(ps: PlanetMineral): string {
    switch (ps) {
      case 1:
        return 'Poor';
      case 2:
        return 'Abundant';
      case 3:
        return 'Rich';
      default:
        return 'Invalid!'
    }
  }

  ngOnInit() {
    this.sub = this.store.select(getCurrentRouteState).pipe(
      map(route => parseInt((route as any).params.planet_id)),
      switchMap(starID => this.store.select(selectPlanet(starID)))
    ).subscribe(planet => {
      if (!planet) {
        return;
      }
      this.planet = planet;
      this.size = PlanetViewComponent.getSizeString(planet.size)
      this.gravity = PlanetViewComponent.getGravityString(planet.gravity);
      this.mineral_class = PlanetViewComponent.getMineralClassString(planet.mineral_class);
    })
  }

  ngOnDestroy() {
    this.sub.unsubscribe();
  }

  goBack() {
    this.location.back();
  }
}
