import {Component, OnInit} from '@angular/core';
import {Planet, PlanetGravity, PlanetMineral, PlanetSize} from "../model/model";
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

  size!: string;
  gravity!: string;
  mineral_class!: string;

  constructor(
    private readonly gameService: GameService,
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
    this.route.params.pipe(
      map(params => [parseInt(params['planet_id']), parseInt(params['star_id'])]),
      switchMap(([planetId, starId]) => this.gameService.getPlanetById(planetId, starId))
    ).subscribe(planet => {
      this.planet = planet;
      this.size = PlanetViewComponent.getSizeString(planet.size);
      this.gravity = PlanetViewComponent.getGravityString(planet.gravity);
      this.mineral_class = PlanetViewComponent.getMineralClassString(planet.mineral_class);
    });
  }

  goBack() {
    this.location.back();
  }
}
