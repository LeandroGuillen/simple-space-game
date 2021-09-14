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

type Planet struct {
	ID         uint `json:"id"`
	ParentStar uint `json:"parent_star"`

	// physical characteristics
	MineralClass PlanetMineral `json:"mineral_class"`
	Size         PlanetSize    `json:"size"`
	Gravity      PlanetGravity `json:"gravity"`

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

	CurrentProject string   `json:"current_project"`
	BuildList      []string `json:"build_list"`
	Buildings      []string `json:"buildings"`
}
