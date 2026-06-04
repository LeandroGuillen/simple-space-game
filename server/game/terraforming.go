package game

import (
	"math/rand"
	. "ssg/model"
)

const terraformingProject = "Terraforming"
const terraformingCostStep = int64(250)

var terraformTransitions = map[PlanetEnvironment][]PlanetEnvironment{
	PlanetEnvironmentToxic:    {PlanetEnvironmentBarren},
	PlanetEnvironmentRadiated: {PlanetEnvironmentBarren},
	PlanetEnvironmentBarren:   {PlanetEnvironmentDesert, PlanetEnvironmentTundra},
	PlanetEnvironmentDesert:   {PlanetEnvironmentArid},
	PlanetEnvironmentTundra:   {PlanetEnvironmentSwamp},
	PlanetEnvironmentArid:     {PlanetEnvironmentTerran},
	PlanetEnvironmentSwamp:    {PlanetEnvironmentTerran},
	PlanetEnvironmentOcean:    {PlanetEnvironmentTerran},
}

var terraformPrerequisites = map[PlanetEnvironment]string{
	PlanetEnvironmentToxic:    "Weather Controller",
	PlanetEnvironmentRadiated: "Planetary Radiation Shield",
}

func isTerraformingProject(name string) bool {
	return name == terraformingProject
}

func canTerraformPlanet(p *Planet, game *Game) bool {
	if p == nil || p.Owner == 0 {
		return false
	}
	targets, ok := terraformTransitions[p.Environment]
	if !ok || len(targets) == 0 {
		return false
	}
	if prereq, hasPrereq := terraformPrerequisites[p.Environment]; hasPrereq {
		if !hasBuilding(p, prereq) {
			return false
		}
	}
	return true
}

func terraformPlanet(p *Planet) {
	targets, ok := terraformTransitions[p.Environment]
	if !ok || len(targets) == 0 {
		return
	}
	p.Environment = targets[rand.Intn(len(targets))]
	p.TerraformingCount++
}

func effectiveTerraformingCost(p *Planet) int64 {
	return terraformingCostStep + p.TerraformingCount*terraformingCostStep
}
