import {Galaxy, Planet, Star} from "./model";
import {BEGalaxy, BEPlanet, BEStar} from "./be-model";

export function mapGalaxy(g: BEGalaxy): Galaxy {
  return {
    stars: g.stars.map(mapStar)
  }
}

function mapStar(s: BEStar): Star {
  return {
    id: s.id,
    name: s.name,
    planets: s.planets.map((p, i) => mapPlanet(s.name, i, p)),
    color: s.color,
    x: s.x,
    y: s.y,
    numPlanets: s.planets.length,
  }
}

function mapPlanet(starName: string, idx: number, p: BEPlanet): Planet {
  return {
    id: p.id,
    parent_star: p.parent_star,
    name: `${starName} ${romanNumber(idx + 1)}`,
    mineral_class: p.mineral_class,
    size: p.size,
    gravity: p.gravity,
    owner: p.owner,
    money: p.money,
    food: p.food,
    industry: p.industry,
    research: p.research,
    morale: p.morale,
    current_pop: p.current_pop,
    pop_growth: p.pop_growth,
    max_pop: p.max_pop,
    farmers: p.farmers,
    workers: p.workers,
    scientists: p.scientists,
    marines: p.marines,
    armor: p.armor,
    current_project: p.current_project,
    build_list: p.build_list || [],
    buildings: p.buildings || [],
  }
}

function romanNumber(n: number): string {
  switch(n) {
    case 0: return '';
    case 1: return 'I';
    case 2: return 'II';
    case 3: return 'III';
    case 4: return 'IV';
    case 5: return 'V';
    case 6: return 'VI';
    case 7: return 'VII';
    case 8: return 'VIII';
    case 9: return 'IX';
    case 10: return 'X';
    default: return '?'
  }
}
