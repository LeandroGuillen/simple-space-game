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
	MaxPopulation     int64 `json:"max_population"`
	LastTurnPopGrowth int64 `json:"last_turn_pop_growth"`
	// PopulationRace is set on conquest and survives subsequent captures.
	PopulationRace string `json:"population_race,omitempty"`

	Industry   PlanetIndustryOutput   `json:"industry"`
	Food       PlanetFoodOutput       `json:"food"`
	Science    PlanetScienceOutput    `json:"science"`
	Income     PlanetIncomeOutput     `json:"income"`
	Morale     PlanetMoraleOutput     `json:"morale"`
	Government PlanetGovernmentOutput `json:"government,omitempty"`
	Pollution  int64                  `json:"pollution"`
	// GravityMultiplier is the per-unit labor output multiplier resulting from
	// planetary gravity vs. the owning race's adaptation trait.
	GravityMultiplier float64 `json:"gravity_multiplier,omitempty"`
	// HomeworldScienceBonus is currently used for the custom race Artifacts World trait.
	HomeworldScienceBonus int64 `json:"homeworld_science_bonus,omitempty"`

	Marines int64 `json:"marines"`
	Armored int64 `json:"armor"`

	// building

	CurrentProject string   `json:"current_project"`
	Buildings      []string `json:"buildings"`
	BuildQueue     []string `json:"build_queue"`
	// IndustryStack contains the leftover production of a planet.
	IndustryStack int64 `json:"industry_stack"`
	// BuildBucket represents the production that is left for the current project to complete.
	BuildBucket int64 `json:"build_bucket"`
	IsBuilding  bool  `json:"is_building"`
	Autobuild   bool  `json:"autobuild"`

	TerraformingCount int64 `json:"terraforming_count,omitempty"`
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
	ProducedPerColonist int64 `json:"produced_per_colonist"`
}

type PlanetScienceOutput struct {
	Scientists           int64 `json:"scientists"`
	ProducedPerScientist int64 `json:"produced_per_scientist"`
	ProducedPerBuilding  int64 `json:"produced_per_building"`
}

type PlanetIncomeOutput struct {
	Taxes       int64   `json:"taxes"`
	Buildings   int64   `json:"buildings"`
	TradeGoods  int64   `json:"trade_goods"`
	Maintenance int64   `json:"maintenance"`
	Total       int64   `json:"total"`
	Bonus       float64 `json:"bonus"`
}

type PlanetMoraleOutput struct {
	Bonus      float64              `json:"bonus"`
	Multiplier float64              `json:"multiplier"`
	Sources    []PlanetMoraleSource `json:"sources,omitempty"`
}

type PlanetMoraleSource struct {
	Name  string  `json:"name"`
	Type  string  `json:"type"`
	Bonus float64 `json:"bonus"`
}

type PlanetGovernmentOutput struct {
	Name                    string  `json:"name,omitempty"`
	FoodLaborMultiplier     float64 `json:"food_labor_multiplier,omitempty"`
	IndustryLaborMultiplier float64 `json:"industry_labor_multiplier,omitempty"`
	ScienceLaborMultiplier  float64 `json:"science_labor_multiplier,omitempty"`
	TaxMultiplier           float64 `json:"tax_multiplier,omitempty"`
	MoraleNeutral           bool    `json:"morale_neutral,omitempty"`
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
