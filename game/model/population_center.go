package model

type Colonizable interface {
	GetID() uint
	GetName() string
	GetOwner() uint
	Colonize(ownerID uint)
}

type BuildCenter interface {
	Build(uint)
	GetCurrentProject() uint
	GetBuildQueue() []string
	GetBuildings() []string
}

type PopulationCenter interface {
	CurrentPop() int64
	CurrentPopMillis() int64
	PopGrowth() int64
	MaxPop() int64
	Morale() int64
	UpdatePopulation()
}

type AgriculturalCenter interface {
	NumFarmers() int64
	Food() int64
}

type IndustrialCenter interface {
	NumWorkers() int64
	Industry() int64
}

type ResearchCenter interface {
	NumScientists() int64
	Research() int64
}

type MilitaryCenter interface {
	NumMarines() int64
	NumArmored() int64
}
