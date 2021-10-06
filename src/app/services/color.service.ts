import { Injectable } from '@angular/core';

import { Planet, PlanetEnvironment } from '../model/model';

@Injectable({ providedIn: "root" })
export class ColorService {

  getPlanetColors(p: Planet) {
    switch (p.environment) {
      case PlanetEnvironment.GASGIANT: return '#a35f23';
      case PlanetEnvironment.ASTEROIDBELT: return '#a37044';
      case PlanetEnvironment.TOXIC: return '#f28eeb';
      case PlanetEnvironment.RADIATED: return '#f77f0e';
      case PlanetEnvironment.BARREN: return '#b7b7b7';
      case PlanetEnvironment.DESERT: return '#e8c09b';
      case PlanetEnvironment.ARID: return '#c68e59';
      case PlanetEnvironment.TUNDRA: return '#1d7a52';
      case PlanetEnvironment.SWAMP: return '#154731';
      case PlanetEnvironment.OCEAN: return '#2994ce';
      case PlanetEnvironment.TERRAN: return '#186a96';
      case PlanetEnvironment.GAIA: return '#01960b';
      default:
        return 'Invalid!'
    }

  }


}
