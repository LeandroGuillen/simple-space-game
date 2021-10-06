package model

type PlanetMineral uint64

const (
	PlanetMineralInvalid PlanetMineral = iota
	PlanetMineralPoor
	PlanetMineralAbundant
	PlanetMineralRich
	PlanetMineralMax
)

type PlanetSize uint64

const (
	PlanetSizeInvalid PlanetSize = iota
	PlanetSizeTiny
	PlanetSizeSmall
	PlanetSizeMedium
	PlanetSizeLarge
	PlanetSizeHuge
	PlanetSizeMax
)

type PlanetGravity uint64

const (
	PlanetGravityInvalid PlanetGravity = iota
	PlanetGravityLow
	PlanetGravityNormal
	PlanetGravityHigh
	PlanetGravityMax
)

type PlanetEnvironment uint64

const (
	PlanetEnvironmentInvalid PlanetEnvironment = iota
	PlanetEnvironmentGasGiant
	PlanetEnvironmentAsteroidBelt
	PlanetEnvironmentToxic
	PlanetEnvironmentRadiated
	PlanetEnvironmentBarren
	PlanetEnvironmentDesert
	PlanetEnvironmentArid
	PlanetEnvironmentTundra
	PlanetEnvironmentSwamp
	PlanetEnvironmentOcean
	PlanetEnvironmentTerran
	PlanetEnvironmentGaia
	//PlanetEnvironmentIceGiant
	//PlanetEnvironmentIce
	//PlanetEnvironmentInferno
	//PlanetEnvironmentVolcanic
	//PlanetEnvironmentTropical
	//PlanetEnvironmentGrassland
	//PlanetEnvironmentBrownDwarf
	// see https://astronomical.fandom.com/wiki/List_of_planet_types for more
	PlanetEnvironmentMax
	// terraforming transitions
	// ========================
	// - Planet Construction
	// GasGiant -> Barren
	// AsteroidBelt -> Barren
	// - Requires Weather Controller
	// Toxic -> Barren
	// - Requires Radiation Shield
	// Radiated -> Barren
	// - Normal Terraforming
	// Barren -> Desert OR Tundra
	// Desert -> Arid
	// Tundra -> Swamp
	// Arid OR Swamp -> Terran
	// Ocean -> Terran
	// Terran -> Gaia
	// - Extra
	// GasGiant -> Barren OR Volcanic
	// AsteroidBelt -> Barren OR Ice OR Volcanic
	// IceGiant -> Ice
	// Ice -> Ocean
	// Volcanic -> Inferno -> Barren
	// Arid -> Grassland -> Terran
	// Swamp -> Tropical -> Terran
)

type Planet struct {
	ID         uint   `json:"id"`
	ParentStar uint   `json:"parent_star"`
	Name       string `json:"name"`

	// physical characteristics
	MineralClass PlanetMineral     `json:"mineral_class"`
	Size         PlanetSize        `json:"size"`
	Gravity      PlanetGravity     `json:"gravity"`
	Environment  PlanetEnvironment `json:"environment"`

	Owner uint `json:"owner"`

	// colony output
	Money    int64 `json:"money"`
	Food     int64 `json:"food"`
	Industry int64 `json:"industry"`
	Research int64 `json:"research"`
	Morale   int64 `json:"morale"`

	// population
	CurrentPop int64 `json:"current_pop"`
	PopGrowth  int64 `json:"pop_growth"`
	MaxPop     int64 `json:"max_pop"`
	Farmers    int64 `json:"farmers"`
	Workers    int64 `json:"workers"`
	Scientists int64 `json:"scientists"`

	Marines int64 `json:"marines"`
	Armor   int64 `json:"armor"`

	CurrentProject uint     `json:"current_project"`
	BuildList      []string `json:"build_list"`
	Buildings      []string `json:"buildings"`
}

func CalculatePlanetMaxPop(size PlanetSize, env PlanetEnvironment) int64 {
	// Planetary Env.:  Gaia Terran Arid Swamp Other
	// Tiny              5     4     3    2     1     <-- The table shows how
	// Small            10     8     6    4     3         the base max. pop. of
	// Medium           15    12     9    6     4         a planet is determined
	// Large            20    16    12    8     5     <-- by size and climate.
	// Huge             25    20    15   10     6
	return int64(size * 4)
}
