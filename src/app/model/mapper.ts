import { BEGalaxy, BEPlanet, BEStar } from './be-model';
import { Galaxy, Planet, Star } from './model';

export function mapGalaxy(g: BEGalaxy): Galaxy {
  return {
    stars: g.stars.map(mapStar)
  }
}

function mapStar(s: BEStar): Star {
  return {
    id: s.id,
    name: s.name,
    planets: s.planets.map((p, i) => mapPlanet(p)),
    color: s.color,
    x: s.x,
    y: s.y,
    numPlanets: s.planets.length,
  }
}

export function mapPlanet(p: BEPlanet): Planet {
  return {
    id: p.id,
    parent_star: p.parent_star,
    name: p.name,
    mineral_class: p.mineral_class,
    size: p.size,
    gravity: p.gravity,
    environment: p.environment,
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
