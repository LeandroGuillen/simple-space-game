import {Component, OnInit} from '@angular/core';
import {Location} from '@angular/common';
import {GameService} from "../game.service";
import {ActivatedRoute} from "@angular/router";
import {map, switchMap} from "rxjs/operators";
import {Star} from "../model/model";

@Component({
  selector: 'crn-star-view',
  templateUrl: './star-view.component.html',
  styleUrls: ['./star-view.component.scss']
})
export class StarViewComponent implements OnInit {

  star!: Star;

  constructor(
    private readonly gameService: GameService,
    private readonly route: ActivatedRoute,
    private readonly location: Location) {
  }

  ngOnInit() {
    this.route.params.pipe(
      map(params => parseInt(params['star_id'])),
      switchMap(id => this.gameService.getStarById(id))
    ).subscribe(star => {
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
