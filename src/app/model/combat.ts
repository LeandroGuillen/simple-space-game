export type ShipDesign = {
  name: string;
  weapons: Weapon[];
  specialSystems: SpecialSystem[];
  baseSpace: number;
  drive: Drive;
  computer: Computer;
  shield: Shield;
  armor: Armor;
  hullSize: HullSize;
  fuelCell: FuelCell;
};

type Weapon = {};
type SpecialSystem = {};
type Drive = {
  name: string;
  interstellarSpeed: number;
  combatSpeed: number;
};
type Computer = {};
type Shield = {
  name: string;
  rechargeRate: number;
  reduceDamage: number;
  absortionFactor: number;
};

type Armor = {
  name: string;
  multiplier: number;
  groundTroopStrength: number;
};
type HullSize = {
  name: string;
  space: number;
  marines: number;
  baseSpeed: number;
  baseHP: number;
  computerHP: number;
  driveHP: number;
  shieldHP: number;
};
type FuelCell = {
  name: string;
  autonomy: number;
};
