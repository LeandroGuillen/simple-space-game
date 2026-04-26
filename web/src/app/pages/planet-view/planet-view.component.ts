import { CdkDragDrop, transferArrayItem } from '@angular/cdk/drag-drop';
import { Location } from '@angular/common';
import { Component, OnDestroy, OnInit } from '@angular/core';
import { Store } from '@ngrx/store';
import { Subscription } from 'rxjs';
import { map, switchMap } from 'rxjs/operators';

import { Planet } from '../../model/model';
import { StringService } from '../../services/string.service';
import { AppState } from '../../store/reducers';
import { selectPlanet } from '../../store/selectors/galaxy.selectors';
import { getCurrentRouteState } from '../../store/selectors/router.selectors';

@Component({
  selector: 'ssg-planet-view',
  standalone: false,
  templateUrl: './planet-view.component.html',
  styleUrls: ['./planet-view.component.scss'],
})
export class PlanetViewComponent implements OnInit, OnDestroy {
  planet!: Planet;
  size!: string;
  gravity!: string;
  mineral_class!: string;
  description = '';

  sub!: Subscription;

  farmers: string[] = [];
  workers: string[] = [];
  scientists: string[] = [];

  constructor(
    private readonly store: Store<AppState>,
    private readonly strings: StringService,
    private readonly location: Location
  ) {}

  ngOnInit() {
    this.sub = this.store
      .select(getCurrentRouteState)
      .pipe(
        map(route => parseInt((route as any).params.planet_id)),
        switchMap(starID => this.store.select(selectPlanet(starID)))
      )
      .subscribe(planet => {
        if (!planet) {
          return;
        }
        this.planet = planet;
        this.size = this.strings.getSize(planet.size);
        this.gravity = this.strings.getGravity(planet.gravity);
        this.mineral_class = this.strings.getMineralClass(planet.mineral_class);
        this.description = this.strings.getPlanetFullDescription(planet);

        for (let i = 0; i < this.planet.farmers; i++) {
          this.farmers.push('F');
        }
        for (let i = 0; i < this.planet.workers; i++) {
          this.workers.push('W');
        }
        for (let i = 0; i < this.planet.scientists; i++) {
          this.scientists.push('S');
        }
      });
  }

  ngOnDestroy() {
    this.sub.unsubscribe();
  }

  goBack() {
    this.location.back();
  }

  drop(event: CdkDragDrop<string[]>) {
    if (event.previousContainer !== event.container) {
      transferArrayItem(
        event.previousContainer.data,
        event.container.data,
        event.previousIndex,
        event.currentIndex
      );
    }
  }
}
