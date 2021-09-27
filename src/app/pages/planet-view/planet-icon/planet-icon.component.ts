import { Component, Input, OnInit } from '@angular/core';

import { Planet } from '../../../model/model';

@Component({
  selector: 'crn-planet-icon',
  templateUrl: './planet-icon.component.svg',
  styleUrls: [ './planet-icon.component.scss' ]
})
export class PlanetIconComponent implements OnInit {
  fillColor = 'rgb(255, 0, 0)';
  radius: number = 5;

  @Input() planet!: Planet;

  constructor() {
  }

  changeColor() {
    const r = Math.floor(Math.random() * 256);
    const g = Math.floor(Math.random() * 256);
    const b = Math.floor(Math.random() * 256);
    this.fillColor = `rgb(${r}, ${g}, ${b})`;
  }


  ngOnInit(): void {
    this.radius = (this.planet.size + 1) * 10
  }

}
