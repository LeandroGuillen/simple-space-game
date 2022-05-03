import { BEArmor, BEComputer, BEDrive, BEFuelCell, BEHullSize, BEShield, BEShipDesign } from './be-combat';
import { BEGalaxy, BEGame, BEPlanet, BEPlayer, BEStar } from './be-model';
import { Armor, Computer, Drive, FuelCell, HullSize, Shield, ShipDesign } from './combat';
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
    resources: {
      shipParts: {
        hullSizes: g.resources.ship_parts.hull_sizes.map(mapHullSize),
        drives: g.resources.ship_parts.drives.map(mapDrive),
        armors: g.resources.ship_parts.armors.map(mapArmor),
        computers: g.resources.ship_parts.computers.map(mapComputer),
        fuelCells: g.resources.ship_parts.fuel_cells.map(mapFuelCell),
        shields: g.resources.ship_parts.shields.map(mapShield),
      },
    },
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
    shipDesigns: p.ship_designs?.map(mapShipDesign),
  };
}

export function mapShipDesign(s: BEShipDesign): ShipDesign {
  return {
    name: s.name,
    weapons: s.weapons,
    specialSystems: s.special_systems,
    baseSpace: s.base_space,
    drive: mapDrive(s.drive),
    computer: mapComputer(s.computer),
    shield: mapShield(s.shield),
    armor: mapArmor(s.armor),
    hullSize: mapHullSize(s.hull_size),
    fuelCell: mapFuelCell(s.fuel_cell),
  };
}

function mapDrive(d: BEDrive): Drive {
  return {
    name: d.name,
    combatSpeed: d.combat_speed,
    interstellarSpeed: d.interstellar_speed,
  };
}

function mapComputer(c: BEComputer): Computer {
  return {
    name: c.name,
    beamAttackBonus: c.beam_attack_bonus,
  };
}

function mapShield(s: BEShield): Shield {
  return {
    name: s.name,
    rechargeRate: s.recharge_rate,
    reduceDamage: s.reduce_damage,
    absortionFactor: s.absortion_factor,
  };
}

function mapArmor(a: BEArmor): Armor {
  return {
    name: a.name,
    multiplier: a.multiplier,
    groundTroopStrength: a.ground_troop_strength,
  };
}

function mapHullSize(h: BEHullSize): HullSize {
  return {
    name: h.name,
    space: h.space,
    marines: h.marines,
    baseSpeed: h.base_speed,
    baseHP: h.base_hp,
    computerHP: h.computer_hp,
    driveHP: h.drive_hp,
    shieldHP: h.shield_hp,
  };
}

function mapFuelCell(f: BEFuelCell): FuelCell {
  return {
    name: f.name,
    autonomy: f.autonomy,
  };
}
