import { Injectable } from '@angular/core';

import {
  Planet,
  PlanetEnvironment,
  PlanetGravity,
  PlanetMineral,
  PlanetSize,
} from '../model/model';

@Injectable({ providedIn: 'root' })
export class StringService {
  getPlanetShortDescription(p: Planet): string {
    return this.getDescription(p, false);
  }

  getPlanetFullDescription(p: Planet): string {
    return this.getDescription(p, true);
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
        return 'Invalid!';
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
        return 'Invalid!';
    }
  }

  getMineralClass(ps: PlanetMineral): string {
    switch (ps) {
      case 1:
        return 'Ultra Poor';
      case 2:
        return 'Poor';
      case 3:
        return 'Abundant';
      case 4:
        return 'Rich';
      case 5:
        return 'Ultra Rich';
      default:
        return 'Invalid!';
    }
  }

  getEnvironment(env: PlanetEnvironment): string {
    switch (env) {
      case 1:
        return 'GasGiant';
      case 2:
        return 'AsteroidBelt';
      case 3:
        return 'Toxic';
      case 4:
        return 'Radiated';
      case 5:
        return 'Barren';
      case 6:
        return 'Desert';
      case 7:
        return 'Arid';
      case 8:
        return 'Tundra';
      case 9:
        return 'Swamp';
      case 10:
        return 'Ocean';
      case 11:
        return 'Terran';
      case 12:
        return 'Gaia';
      default:
        return 'Invalid!';
    }
  }

  private getDescription(p: Planet, fullDescription = false): string {
    // ex. Poor Small planet, Normal gravity
    const mineralClass = this.getMineralClass(p.mineral_class);
    const size = this.getSize(p.size);
    const gravity = this.getGravity(p.gravity);
    const env = this.getEnvironment(p.environment);

    let desc = '';
    if (fullDescription || p.mineral_class !== PlanetMineral.ABUNDANT) {
      desc += mineralClass;
    }
    if (fullDescription || p.size !== PlanetSize.MEDIUM) {
      desc += ' ' + size;
    }
    desc += ' ' + env + ' planet';
    if (fullDescription || p.gravity !== PlanetGravity.NORMAL) {
      desc += `, ${gravity}-G`;
    }
    return desc;
  }
}
