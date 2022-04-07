import { Location } from '@angular/common';
import { Component, OnDestroy, OnInit } from '@angular/core';
import { Store } from '@ngrx/store';
import { Subscription } from 'rxjs';
import { map, switchMap } from 'rxjs/operators';

import { Planet, Star } from '../../model/model';
import { colonizeRequest } from '../../store/actions';
import { AppState } from '../../store/reducers';
import { selectStar } from '../../store/selectors/galaxy.selectors';
import { getCurrentRouteState } from '../../store/selectors/router.selectors';

@Component({
  selector: 'ssg-star-view',
  templateUrl: './star-view.component.html',
  styleUrls: [ './star-view.component.scss' ]
})
export class StarViewComponent implements OnInit, OnDestroy {
  star!: Star;
  planets: Planet[] = [];
  selectedPlanetID = -1;
  selectedPlanet!: Planet;

  sub!: Subscription;


  constructor(
    private readonly store: Store<AppState>,
    private readonly location: Location) {
  }

  ngOnInit() {
    this.sub = this.store.select(getCurrentRouteState).pipe(
      map(route => parseInt((route as any).params.star_id)),
      switchMap(starID => this.store.select(selectStar(starID)))
    ).subscribe(star => {
      if (star === undefined) {
        return;
      }
      this.star = star;
      this.planets = star.planets;
    })
  }

  ngOnDestroy() {
    this.sub.unsubscribe();
  }

  goBack() {
    this.location.back();
  }

  colonize(planetID: number) {
    this.store.dispatch(colonizeRequest({ planetID, playerID: 1 }));
  }

  clicked(p: Planet) {
    this.selectedPlanetID = p.id;
    this.selectedPlanet = p;
  }

}
