import { Component, Input } from '@angular/core';

import { Planet } from '../../model/model';
import { ColorService } from '../../services/color.service';

@Component({
  selector: 'ssg-planet-icon',
  templateUrl: './planet-icon.component.svg',
  styleUrls: [ './planet-icon.component.scss' ]
})
export class PlanetIconComponent {
  fillColor = 'rgb(255, 0, 0)';
  radius: number = 5;
  animationValues = '';

  constructor(private readonly colors: ColorService) { }

  private _planet!: Planet;

  @Input() set planet(p: Planet) {
    this._planet = p;
    const r = (this.planet.size + 2) * 8;
    this.radius = r
    this.animationValues = `${r * 0.4};${r * 0.6};${r * 0.4}`
    this.fillColor = this.colors.getPlanetColors(p);
  }

  get planet(): Planet {
    return this._planet;
  }
}
