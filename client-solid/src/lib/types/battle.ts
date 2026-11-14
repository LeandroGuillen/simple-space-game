// Wire types for server-authoritative tactical battles (see server
// model/tactical_battle.go). These are consumed raw, without a view-model
// mapping layer, because the Phaser scene works directly off server state.

export type ApiBeam = {
  name: string;
  minDamage: number;
  maxDamage: number;
};

/** One missile rack's per-battle magazine. */
export type ApiMissileRack = {
  name: string;
  damage: number;
  ammo_left: number;
  ammo_max: number;
};

export type ApiTacticalShipStatus = {
  current_armor_hp: number;
  current_structure_hp: number;
  max_armor_hp: number;
  max_structure_hp: number;
  current_computer_hp: number;
  current_drive_hp: number;
  current_shield_hp: number;
};

export type ApiTacticalShip = {
  id: number;
  owner_id: number;
  fleet_id: number;
  ship_index: number;
  name: string;
  design_name?: string;
  hull_size?: string;
  size_rating: number;
  x: number;
  y: number;
  combat_speed: number;
  initiative: number;
  beams?: ApiBeam[];
  beam_attack_bonus: number;
  beam_defense: number;
  // Absent on battles saved before shields reduced damage: no shield.
  shield_name?: string;
  shield_reduce_damage?: number;
  // Absent on beam ships and battles saved before missiles: no rack, ME 0.
  missiles?: ApiMissileRack[];
  missile_evasion?: number;
  status: ApiTacticalShipStatus;
  moves_left: number;
  has_fired: boolean;
  destroyed: boolean;
};

export type ApiBattleEvent = {
  seq: number;
  type:
    | 'round_start'
    | 'activation'
    | 'move'
    | 'fire'
    | 'destroyed'
    | 'finished';
  round?: number;
  ship_id?: number;
  target_id?: number;
  from_x?: number;
  from_y?: number;
  x?: number;
  y?: number;
  damage?: number;
  hits?: number;
  misses?: number;
  // "missile" for missile launches; absent for beam fire.
  weapon_kind?: 'missile';
  weapon_name?: string;
  winner_owner_id?: number;
  draw?: boolean;
};

export type ApiTacticalBattle = {
  id: number;
  game_turn: number;
  star_id: number;
  status: 'active' | 'finished';
  grid_width: number;
  grid_height: number;
  round: number;
  ships: ApiTacticalShip[];
  turn_order: number[];
  turn_index: number;
  human_owner_ids: number[];
  winner_owner_id?: number;
  draw?: boolean;
  events: ApiBattleEvent[];
};

export type BattleActionRequest = {
  type: 'move' | 'fire' | 'done' | 'wait' | 'auto';
  ship_id?: number;
  x?: number;
  y?: number;
  target_id?: number;
};

export function currentBattleShip(
  battle: ApiTacticalBattle,
): ApiTacticalShip | null {
  if (battle.status !== 'active') {
    return null;
  }
  const shipId = battle.turn_order[battle.turn_index];
  return battle.ships.find((ship) => ship.id === shipId && !ship.destroyed) ?? null;
}

/**
 * Battle HUD shield readout: the shield class and how much damage it blocks
 * from every beam hit. Ships without shield data (older battles) show None.
 */
export function formatShieldStrength(
  ship: Pick<ApiTacticalShip, 'shield_name' | 'shield_reduce_damage'>,
): string {
  const reduce = Math.max(0, ship.shield_reduce_damage ?? 0);
  const name = ship.shield_name?.trim() || 'None';
  if (reduce === 0) {
    return name === 'None' ? 'None' : `${name} (-0/hit)`;
  }
  return `${name} (-${reduce}/hit)`;
}

/** HUD weapons row for a missile rack, e.g. "Nuclear Missile · 8 dmg · 3/5 left". */
export function formatMissileRack(rack: ApiMissileRack): string {
  return `${rack.name} · ${rack.damage} dmg · ${Math.max(0, rack.ammo_left)}/${rack.ammo_max} left`;
}

/** True when the ship's only weapons are missile racks and all are empty. */
export function isOutOfMissiles(
  ship: Pick<ApiTacticalShip, 'beams' | 'missiles'>,
): boolean {
  const racks = ship.missiles ?? [];
  if ((ship.beams?.length ?? 0) > 0 || racks.length === 0) {
    return false;
  }
  return racks.every((rack) => rack.ammo_left <= 0);
}

/** Whether the ship can still fire this activation. */
export function canShipFire(
  ship: Pick<ApiTacticalShip, 'beams' | 'missiles' | 'has_fired'>,
): boolean {
  if (ship.has_fired) return false;
  if ((ship.beams?.length ?? 0) > 0) return true;
  return (ship.missiles ?? []).some((rack) => rack.ammo_left > 0);
}

/** Why the Fire command is unavailable, or "" when the ship can fire. */
export function fireDisabledReason(
  ship: Pick<ApiTacticalShip, 'beams' | 'missiles' | 'has_fired'> | null,
): string {
  if (!ship) return 'No ship is acting';
  if (isOutOfMissiles(ship)) return 'Out of missiles';
  if ((ship.beams?.length ?? 0) === 0 && (ship.missiles?.length ?? 0) === 0) {
    return 'No weapons';
  }
  if (ship.has_fired) return 'Already fired this round';
  return '';
}

/** Missile launches animate as projectiles; everything else as beams. */
export function isMissileFire(event: Pick<ApiBattleEvent, 'type' | 'weapon_kind'>): boolean {
  return event.type === 'fire' && event.weapon_kind === 'missile';
}

/** Nearest living enemy by grid (Chebyshev) distance, lowest id on ties. */
export function nearestEnemyShip(
  battle: ApiTacticalBattle,
  ship: ApiTacticalShip,
): ApiTacticalShip | null {
  let best: ApiTacticalShip | null = null;
  let bestDist = Number.POSITIVE_INFINITY;
  for (const other of battle.ships) {
    if (other.destroyed || other.owner_id === ship.owner_id) continue;
    const dist = Math.max(Math.abs(other.x - ship.x), Math.abs(other.y - ship.y));
    if (dist < bestDist || (dist === bestDist && best && other.id < best.id)) {
      best = other;
      bestDist = dist;
    }
  }
  return best;
}
