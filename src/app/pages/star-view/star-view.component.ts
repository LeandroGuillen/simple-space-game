import { Location } from '@angular/common';
import { Component, OnInit } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { Store } from '@ngrx/store';
import { map, switchMap } from 'rxjs/operators';

import { GameService } from '../../game.service';
import { Star } from '../../model/model';
import { AppState } from '../../store/reducers';
import { selectStar } from '../../store/selectors';

@Component({
  selector: 'crn-star-view',
  templateUrl: './star-view.component.html',
  styleUrls: [ './star-view.component.scss' ]
})
export class StarViewComponent implements OnInit {

  star!: Star;

  constructor(
    private readonly store: Store<AppState>,
    private readonly gameService: GameService,
    private readonly route: ActivatedRoute,
    private readonly location: Location) {
  }

  ngOnInit() {
    this.route.params.pipe(
      map(params => parseInt(params[ 'star_id' ])),
      switchMap(id => this.store.select(selectStar(id)))
    ).subscribe(star => {
      if (!star) {
        return;
      }
      this.star = star;
    });
  }

  goBack() {
    this.location.back();
  }

  colonize(planetID: number) {
    this.gameService.colonize(1, planetID).subscribe(ok => {
      if (ok) {
        const planet = this.star.planets.find(p => p.id === planetID);
        if (!planet) {
          return;
        }
        planet.owner = 1;
      }
    });
  }

}
