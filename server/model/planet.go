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
	Population        int64 `json:"population"`
	LastTurnPopGrowth int64 `json:"last_turn_pop_growth"`

	Industry  PlanetIndustryOutput `json:"industry"`
	Food      PlanetFoodOutput     `json:"food"`
	Science   PlanetScienceOutput  `json:"science"`
	Income    PlanetIncomeOutput   `json:"income"`
	Pollution int64                `json:"pollution"`

	Marines int64 `json:"marines"`
	Armored int64 `json:"armor"`

	// building

	CurrentProject string   `json:"current_project"`
	Buildings      []string `json:"buildings"`
	// IndustryStack contains the leftover production of a planet.
	IndustryStack int64 `json:"industry_stack"`
	// BuildBucket represents the production that is left for the current project to complete.
	BuildBucket int64 `json:"build_bucket"`
	IsBuilding  bool  `json:"is_building"`
}

type PlanetFoodOutput struct {
	Farmers             int64 `json:"farmers"`
	ProducedPerFarmer   int64 `json:"produced_per_farmer"`
	ProducedPerBuilding int64 `json:"produced_per_building"`
}

type PlanetIndustryOutput struct {
	Workers             int64 `json:"workers"`
	ProducedPerWorker   int64 `json:"produced_per_worker"`
	ProducedPerBuilding int64 `json:"produced_per_building"`
}

type PlanetScienceOutput struct {
	Scientists           int64 `json:"scientists"`
	ProducedPerScientist int64 `json:"produced_per_scientist"`
	ProducedPerBuilding  int64 `json:"produced_per_building"`
}

type PlanetIncomeOutput struct {
	Taxes       int64 `json:"taxes"`
	Buildings   int64 `json:"buildings"`
	TradeGoods  int64 `json:"trade_goods"`
	Maintenance int64 `json:"maintenance"`
	Total       int64 `json:"total"`
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
