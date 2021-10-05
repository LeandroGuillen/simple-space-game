import { Component, Input } from '@angular/core';

import { Planet } from '../../model/model';

@Component({
  selector: 'crn-planet-icon',
  templateUrl: './planet-icon.component.svg',
  styleUrls: [ './planet-icon.component.scss' ]
})
export class PlanetIconComponent {
  fillColor = 'rgb(255, 0, 0)';
  radius: number = 5;

  private _planet!: Planet;

  @Input() set planet(value: Planet) {
    this._planet = value;
    this.radius = (this.planet.size + 1) * 10
  }

  get planet(): Planet {
    return this._planet;
  }

  changeColor() {
    const r = Math.floor(Math.random() * 256);
    const g = Math.floor(Math.random() * 256);
    const b = Math.floor(Math.random() * 256);
    this.fillColor = `rgb(${r}, ${g}, ${b})`;
  }
}
