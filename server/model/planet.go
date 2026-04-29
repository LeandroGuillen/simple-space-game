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
	Population  int64 `json:"population"`
	Farmers     int64 `json:"farmers"`
	Workers    int64 `json:"workers"`
	Scientists int64 `json:"scientists"`

	Industry PlanetIndustryOutput `json:"industry,omitempty"`
	Food     PlanetFoodOutput     `json:"food,omitempty"`
	Science  PlanetScienceOutput  `json:"science,omitempty"`

	Marines int64 `json:"marines"`
	Armored int64 `json:"armor"`

	// building

	CurrentProject Building   `json:"current_project"`
	BuildQueue     []Building `json:"build_queue"`
	Buildings      []Building `json:"buildings"`
	// IndustryStack contains the leftover production of a planet.
	IndustryStack int64 `json:"industry_stack"`
	// BuildBucket represents the production that is left for the current project to complete.
	BuildBucket int64 `json:"build_bucket"`
	IsBuilding  bool  `json:"is_building"`
}

type PlanetOutputContributor struct {
	ID         string               `json:"id"`
	SourceType string               `json:"source_type"`
	SourceID   string               `json:"source_id"`
	Label      string               `json:"label"`
	Effects    []PlanetOutputEffect `json:"effects,omitempty"`
}

type PlanetOutputEffect struct {
	Kind  string  `json:"kind"`
	Value float64 `json:"value"`
}

type PlanetIndustryOutput struct {
	Workers             int64                     `json:"workers"`
	BasePerWorker       int64                     `json:"base_per_worker"`
	PerWorkerBonusTotal int64                     `json:"per_worker_bonus_total"`
	FlatBonusTotal      int64                     `json:"flat_bonus_total"`
	PercentBonusTotal   float64                   `json:"percent_bonus_total"`
	Contributors        []PlanetOutputContributor `json:"contributors"`
	Total               int64                     `json:"total"`
}

type PlanetFoodOutput struct {
	Farmers             int64                     `json:"farmers"`
	BasePerFarmer       int64                     `json:"base_per_farmer"`
	PerFarmerBonusTotal int64                     `json:"per_farmer_bonus_total"`
	FlatBonusTotal      int64                     `json:"flat_bonus_total"`
	PercentBonusTotal   float64                   `json:"percent_bonus_total"`
	Contributors        []PlanetOutputContributor `json:"contributors"`
	Total               int64                     `json:"total"`
}

type PlanetScienceOutput struct {
	Scientists             int64                     `json:"scientists"`
	BasePerScientist       int64                     `json:"base_per_scientist"`
	PerScientistBonusTotal int64                     `json:"per_scientist_bonus_total"`
	FlatBonusTotal         int64                     `json:"flat_bonus_total"`
	PercentBonusTotal      float64                   `json:"percent_bonus_total"`
	Contributors           []PlanetOutputContributor `json:"contributors"`
	Total                  int64                     `json:"total"`
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
