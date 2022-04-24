package combat

//https://gamefaqs.gamespot.com/pc/197873-master-of-orion-ii-battle-at-antares/faqs/16743

type HullSize struct {
	Name      string
	Space     int64
	Marines   int64
	BaseSpeed int64 // squares/turn

	// hit points
	BaseHP     int64
	ComputerHP int64
	DriveHP    int64
	ShieldHP   int64
}

type Drive struct {
	Name              string
	InterstellarSpeed int64 // parsecs/turn
	CombatSpeed       int64 // squares/turn
}

type Armor struct {
	Name                string
	Multiplier          int64
	GroundTroopStrength int64
}

type FuelCell struct {
	Name     string
	Autonomy int64 // in parsecs
}

type Shield struct {
	Name            string
	RechargeRate    float64
	ReduceDamage    int64
	AbsortionFactor int64 // Absorbes AbsortionFactor * BaseHP
}

type ShipSystem struct {
	Name             string
	BeamAttackBonus  int64
	BeamDefenseBonus int64
}
