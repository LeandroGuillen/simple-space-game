import {Component, OnInit} from '@angular/core';
import {Planet} from "../model/model";
import {GameService} from "../game.service";
import {ActivatedRoute} from "@angular/router";
import {Location} from "@angular/common";
import {map, switchMap} from "rxjs/operators";

@Component({
  selector: 'crn-planet-view',
  templateUrl: './planet-view.component.html',
  styleUrls: ['./planet-view.component.scss']
})
export class PlanetViewComponent implements OnInit {

  planet!: Planet;

  constructor(
    private readonly gameService: GameService,
    private readonly route: ActivatedRoute,
    private readonly location: Location) {
  }

  ngOnInit() {

    this.route.params.pipe(
      map(params => [parseInt(params['planet_id']), parseInt(params['star_id'])]),
      switchMap(([planetId, starId]) => this.gameService.getPlanetById(planetId, starId))
    ).subscribe(star => {
      this.planet = star;
    });
  }

  goBack() {
    this.location.back();
  }
}
