package model

import (
	"math"
	"math/rand"
)

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

	CurrentProject Project   `json:"current_project"`
	BuildQueue     []Project `json:"build_queue"`
	Buildings      []Project `json:"buildings"`
	industryStack  int64
}

func (p *Planet) GetID() uint {
	return p.ID
}
func (p *Planet) GetName() string {
	return p.Name
}
func (p *Planet) GetOwner() uint {
	return p.Owner
}
func (p *Planet) Colonize(ownerID uint) {
	p.Owner = ownerID
	p.Population = 1000
	p.Workers = 1
}
func (p *Planet) CurrentPop() int64 {
	return int64(math.Floor(float64(p.Population) / 1000))
}
func (p *Planet) CurrentPopMillis() int64 {
	popIntegers := p.CurrentPop()
	return p.Population - popIntegers
}

// PopGrowth returns how much population grows every turn.
func (p *Planet) PopGrowth() int64 {
	// TODO should take into account tech, race traits, leaders, etc.
	return 450
}
func (p *Planet) MaxPop() int64 {
	// TODO should take into account other factors
	// Planetary Env.:  Gaia Terran Arid Swamp Other
	// Tiny              5     4     3    2     1     <-- The table shows how
	// Small            10     8     6    4     3         the base max. pop. of
	// Medium           15    12     9    6     4         a planet is determined
	// Large            20    16    12    8     5     <-- by size and climate.
	// Huge             25    20    15   10     6
	return int64(p.Size*4) * 1000
}
func (p *Planet) Morale() int64 {
	// TODO should take into account tech, race traits, leaders, etc.
	return 0
}
func (p *Planet) increasePopulation() {
	// put one more where most people are
	if p.NumFarmers() >= p.NumWorkers() && p.NumFarmers() >= p.NumScientists() {
		p.Farmers++
	} else if p.Workers >= p.Farmers && p.Workers >= p.Scientists {
		p.Workers++
	} else {
		p.Scientists++
	}
}
func (p *Planet) decreasePopulation() {
	if p.Population <= 0 {
		return
	}
	found := false
	for !found {
		popGroup := rand.Intn(3)
		switch popGroup {
		case 0:
			if p.Farmers > 0 {
				p.Farmers--
				found = true
			}
		case 1:
			if p.Workers > 0 {
				p.Workers--
				found = true
			}
		case 2:
			if p.Scientists > 0 {
				p.Scientists--
				found = true
			}
		}
	}
}
func (p *Planet) UpdatePopulation() {
	// should the colony population grow or shrink?
	//p.Population += p.PopGrowth()
	expectedPopulation := p.Population + p.PopGrowth()
	//expectedPopulation := int64(math.Floor(float64(p.CurrentPopMillis()) / 1000))
	if expectedPopulation < p.Population {
		// somebody died
		p.decreasePopulation()
	} else if expectedPopulation >= p.MaxPop() {
		// cannot fit more people on this planet
		expectedPopulation = p.MaxPop()
	} else if expectedPopulation/1000 > p.CurrentPop() {
		// somebody was born
		p.increasePopulation()
	}
	p.Population = expectedPopulation
}
func (p *Planet) NumFarmers() int64 {
	return p.Farmers
}
func (p *Planet) Food() int64 {
	// TODO should take into account planet environment and other factors
	return p.NumFarmers() * 3
}
func (p *Planet) NumWorkers() int64 {
	return p.Workers
}
func (p *Planet) Industry() int64 {
	// TODO add Buildings, morale, tech, racial traits, etc. to this industry effort
	return p.NumWorkers() * IndustryPerMineralClass(p.MineralClass)
}
func (p *Planet) UpdateProduction() {
	// workers labor
	industry := p.Industry()
	// apply labor to current project
	p.industryStack += industry

	if p.CurrentProject == nil {
		return
	}

	if p.industryStack >= p.CurrentProject.GetCost() {
		// finished project
		p.AddBuilding(p.CurrentProject)
		p.industryStack -= p.CurrentProject.GetCost()
		p.CurrentProject = nil
	}
}
func (p *Planet) Income() int64 {
	return p.CurrentPop()
}
func (p *Planet) Build(project Project) {
	p.CurrentProject = project
}
func (p *Planet) AddBuilding(project Project) {
	p.Buildings = append(p.Buildings, project)
}
func (p *Planet) GetCurrentProject() Project {
	return p.CurrentProject
}
func (p *Planet) GetBuildQueue() []Project {
	return p.BuildQueue
}
func (p *Planet) GetBuildings() []Project {
	return p.Buildings
}
func (p *Planet) NumScientists() int64 {
	return p.Scientists
}
func (p *Planet) Research() int64 {
	// TODO should take into account Buildings, gravity and other factors
	return p.NumScientists() * 3
}
func (p *Planet) NumMarines() int64 {
	return p.Marines
}
func (p *Planet) NumArmored() int64 {
	return p.Armored
}
func (p *Planet) Turn() {
	p.UpdatePopulation()
	p.UpdateProduction()
}
