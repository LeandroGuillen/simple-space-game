export type BEShipDesign = {
  name: string;
  weapons: BEWeapon[];
  special_systems: BESpecialSystem[];
  base_space: number;
  drive: BEDrive;
  computer: BEComputer;
  shield: BEShield;
  armor: BEArmor;
  hull_size: BEHullSize;
  fuel_cell: BEFuelCell;
};

export type BEWeapon = {};
export type BESpecialSystem = {};
export type BEDrive = {
  name: string;
  interstellar_speed: number;
  combat_speed: number;
};
export type BEComputer = {
  name: string;
  beam_attack_bonus: number;
};
export type BEShield = {
  name: string;
  recharge_rate: number;
  reduce_damage: number;
  absortion_factor: number;
};

export type BEArmor = {
  name: string;
  multiplier: number;
  ground_troop_strength: number;
};
export type BEHullSize = {
  name: string;
  space: number;
  marines: number;
  base_speed: number;
  base_hp: number;
  computer_hp: number;
  drive_hp: number;
  shield_hp: number;
};
export type BEFuelCell = {
  name: string;
  autonomy: number;
};
