package model

type Planet struct {
	ID         uint   `json:"id"`
	Name       string `json:"name"`
	Owner      uint   `json:"owner"`
	ParentStar uint   `json:"parent_star"`

	// physical characteristics
	MineralClass PlanetMineral     `json:"mineral_class"`
	Size         PlanetSize        `json:"size"`
	Gravity      PlanetGravity     `json:"gravity"`
	Environment  PlanetEnvironment `json:"environment"`

	// population
	Population int64 `json:"population"`
	Farmers    int64 `json:"farmers"`
	Workers    int64 `json:"workers"`
	Scientists int64 `json:"scientists"`

	Marines int64 `json:"marines"`
	Armored int64 `json:"armor"`

	CurrentProject Building   `json:"current_project"`
	BuildQueue     []Building `json:"build_queue"`
	Buildings      []Building `json:"buildings"`
	industryStack  int64
}

//go:generate stringer -type=PlanetMineral
type PlanetMineral uint64

const (
	PlanetMineralInvalid PlanetMineral = iota
	PlanetMineralUltraPoor
	PlanetMineralPoor
	PlanetMineralAbundant
	PlanetMineralRich
	PlanetMineralUltraRich
	PlanetMineralMax
)

func IndustryPerMineralClass(p PlanetMineral) int64 {
	switch p {
	case PlanetMineralUltraPoor:
		return 1
	case PlanetMineralPoor:
		return 2
	case PlanetMineralAbundant:
		return 3
	case PlanetMineralRich:
		return 5
	case PlanetMineralUltraRich:
		return 8
	default:
		return 0
	}
}

//go:generate stringer -type=PlanetSize
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

//go:generate stringer -type=PlanetGravity
type PlanetGravity uint64

const (
	PlanetGravityInvalid PlanetGravity = iota
	PlanetGravityLow
	PlanetGravityNormal
	PlanetGravityHigh
	PlanetGravityMax
)

//go:generate stringer -type=PlanetEnvironment
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
	// see https://astronomical.fandom.com/wiki/List_of_planet_types for more
	PlanetEnvironmentMax
	//PlanetEnvironmentIceGiant
	//PlanetEnvironmentIce
	//PlanetEnvironmentInferno
	//PlanetEnvironmentVolcanic
	//PlanetEnvironmentTropical
	//PlanetEnvironmentGrassland
	//PlanetEnvironmentBrownDwarf
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
