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

type BEWeapon = {};
type BESpecialSystem = {};
type BEDrive = {
  name: string;
  interstellar_speed: number;
  combat_speed: number;
};
type BEComputer = {};
type BEShield = {
  name: string;
  recharge_rate: number;
  reduce_damage: number;
  absortion_factor: number;
};

type BEArmor = {
  name: string;
  multiplier: number;
  ground_troop_strength: number;
};
type BEHullSize = {
  name: string;
  space: number;
  marines: number;
  base_speed: number;
  base_hp: number;
  computer_hp: number;
  drive_hp: number;
  shield_hp: number;
};
type BEFuelCell = {
  name: string;
  autonomy: number;
};
