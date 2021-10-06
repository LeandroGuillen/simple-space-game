import { Component, Input } from '@angular/core';

import { Planet } from '../../model/model';
import { StringService } from '../../services/string.service';

@Component({
  selector: 'crn-planet-mini-view',
  templateUrl: './planet-mini-view.component.html',
  styleUrls: [ './planet-mini-view.component.scss' ]
})
export class PlanetMiniViewComponent {
  shortDescription = '';
  private _planet!: Planet;

  constructor(private readonly strings: StringService) { }

  @Input() set planet(p: Planet) {
    this._planet = p;
    this.shortDescription = this.strings.getPlanetShortDescription(p);
  }

  get planet(): Planet {
    return this._planet;
  }

}
