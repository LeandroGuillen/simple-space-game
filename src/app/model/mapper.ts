import { BEShipDesign } from './be-combat';
import { BEGalaxy, BEGame, BEPlanet, BEPlayer, BEStar } from './be-model';
import { ShipDesign } from './combat';
import { Galaxy, Game, Planet, Player, Star } from './model';

export function mapGalaxy(g: BEGalaxy): Galaxy {
  return {
    stars: g.stars.map(mapStar),
  };
}

function mapStar(s: BEStar): Star {
  return {
    id: s.id,
    name: s.name,
    planets: s.planets.map(mapPlanet),
    color: s.color,
    x: s.x,
    y: s.y,
    numPlanets: s.planets.length,
  };
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
  };
}

export function mapGame(g: BEGame): Game {
  return {
    id: g.id,
    currentTurn: g.current_turn,
    galaxy: mapGalaxy(g.galaxy),
    players: g.players.map(mapPlayer),
  };
}

export function mapPlayer(p: BEPlayer): Player {
  return {
    id: p.id,
    name: p.name,
    race: p.race,
    treasury: p.treasury,
    taxRate: p.taxRate,
    researchPoints: p.research_points,
    fleets: p.fleets,
    freighters: p.freighters,
    freightersUsed: p.freighters_used,
    shipDesigns: p.ship_designs.map(mapShipDesign),
  };
}

export function mapShipDesign(s: BEShipDesign): ShipDesign {
  return {
    name: s.name,
    weapons: s.weapons,
    specialSystems: s.special_systems,
    baseSpace: s.base_space,
    drive: {
      name: s.drive.name,
      combatSpeed: s.drive.combat_speed,
      interstellarSpeed: s.drive.interstellar_speed,
    },
    computer: s.computer,
    shield: {
      name: s.shield.name,
      rechargeRate: s.shield.recharge_rate,
      reduceDamage: s.shield.reduce_damage,
      absortionFactor: s.shield.absortion_factor,
    },
    armor: {
      name: s.armor.name,
      multiplier: s.armor.multiplier,
      groundTroopStrength: s.armor.ground_troop_strength,
    },
    hullSize: {
      name: s.hull_size.name,
      space: s.hull_size.space,
      marines: s.hull_size.marines,
      baseSpeed: s.hull_size.base_speed,
      baseHP: s.hull_size.base_hp,
      computerHP: s.hull_size.computer_hp,
      driveHP: s.hull_size.drive_hp,
      shieldHP: s.hull_size.shield_hp,
    },
    fuelCell: s.fuel_cell,
  };
}
