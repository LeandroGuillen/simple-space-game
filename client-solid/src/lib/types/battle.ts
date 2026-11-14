// Wire types for server-authoritative tactical battles (see server
// model/tactical_battle.go). These are consumed raw, without a view-model
// mapping layer, because the Phaser scene works directly off server state.

export type ApiBeam = {
  name: string;
  minDamage: number;
  maxDamage: number;
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
