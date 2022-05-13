export type ShipDesign = {
  name: string;

  weapons?: Weapon[];
  specialSystems?: SpecialSystem[];
  baseSpace?: number;

  drive: Drive;
  computer: Computer;
  shield: Shield;
  armor: Armor;
  hullSize: HullSize;
  fuelCell: FuelCell;

  isCommon: boolean;
};

export type ShipDesignRef = {
  [property in keyof ShipDesign as Exclude<
    property,
    'isCommon' | 'baseSpace' | 'specialSystems' | 'weapons'
  >]: string;
};

export type Weapon = {};
export type SpecialSystem = {};
export type Drive = {
  name: string;
  interstellarSpeed: number;
  combatSpeed: number;
};
export type Computer = {
  name: string;
  beamAttackBonus: number;
};
export type Shield = {
  name: string;
  rechargeRate: number;
  reduceDamage: number;
  absortionFactor: number;
};

export type Armor = {
  name: string;
  multiplier: number;
  groundTroopStrength: number;
};
export type HullSize = {
  name: string;
  sizeRating: number;
  space: number;
  marines: number;
  baseSpeed: number;
  baseHP: number;
  computerHP: number;
  driveHP: number;
  shieldHP: number;
};
export type FuelCell = {
  name: string;
  autonomy: number;
};
