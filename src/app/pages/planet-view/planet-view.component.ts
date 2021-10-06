import { Location } from '@angular/common';
import { Component, OnDestroy, OnInit } from '@angular/core';
import { Store } from '@ngrx/store';
import { Subscription } from 'rxjs';
import { map, switchMap } from 'rxjs/operators';

import { Planet } from '../../model/model';
import { StringService } from '../../services/string.service';
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
    private readonly strings: StringService,
    private readonly location: Location) {
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
      this.size = this.strings.getSize(planet.size)
      this.gravity = this.strings.getGravity(planet.gravity);
      this.mineral_class = this.strings.getMineralClass(planet.mineral_class);
    })
  }

  ngOnDestroy() {
    this.sub.unsubscribe();
  }

  goBack() {
    this.location.back();
  }
}
