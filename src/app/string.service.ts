import { Injectable } from '@angular/core';

import { Planet, PlanetGravity, PlanetMineral, PlanetSize } from './model/model';

@Injectable({ providedIn: "root" })
export class StringService {

  getPlanetShortDescription(p: Planet): string {
    // ex. Poor Small planet, Normal gravity
    const mineralClass = this.getMineralClass(p.mineral_class);
    const size = this.getSize(p.size);
    const gravity = this.getGravity(p.gravity);
    return `${mineralClass} ${size} planet, ${gravity}-G`
  }

  getSize(ps: PlanetSize): string {
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

  getGravity(ps: PlanetGravity): string {
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

  getMineralClass(ps: PlanetMineral): string {
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

}
