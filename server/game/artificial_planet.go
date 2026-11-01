package game

import (
	"fmt"
	. "ssg/model"
)

const artificialPlanetProject = "Artificial Planet Construction"

func isArtificialPlanetTarget(p *Planet) bool {
	return p != nil &&
		p.Owner == 0 &&
		(p.Environment == PlanetEnvironmentGasGiant || p.Environment == PlanetEnvironmentAsteroidBelt)
}

func (g *Game) hasArtificialPlanetTargetsInStar(starID uint) bool {
	star, err := g.getStar(starID)
	if err != nil {
		return false
	}
	for _, planet := range star.Planets {
		if isArtificialPlanetTarget(planet) {
			return true
		}
	}
	return false
}

func (g *Game) hasArtificialPlanetTargetsInSourceSystems(sourcePlanetIDs []uint, ownerID uint) bool {
	for _, sourcePlanetID := range sourcePlanetIDs {
		source, err := g.getPlanet(sourcePlanetID)
		if err != nil || source.Owner != ownerID {
			continue
		}
		if g.hasArtificialPlanetTargetsInStar(source.ParentStar) {
			return true
		}
	}
	return false
}

func canBuildArtificialPlanetProject(p *Planet, availableBuildings []string, game *Game) bool {
	if p == nil || game == nil || p.Owner == 0 {
		return false
	}
	if p.IsBuilding && p.CurrentProject == artificialPlanetProject {
		return false
	}
	known := false
	for _, availableBuilding := range availableBuildings {
		if availableBuilding == artificialPlanetProject {
			known = true
			break
		}
	}
	return known && game.hasArtificialPlanetTargetsInStar(p.ParentStar)
}

// artificialPlanetChoice validates against the supplied pending sources without mutating the game.
func (g *Game) artificialPlanetChoice(playerID uint, command *ArtificialPlanetCommand, pending []uint) (int, *Planet, error) {
	if len(pending) == 0 {
		return -1, nil, fmt.Errorf("no pending artificial planet construction")
	}
	target, err := g.getPlanet(command.TargetPlanetID)
	if err != nil {
		return -1, nil, err
	}

	sourceIndex := -1
	if command.SourcePlanetID != 0 {
		for i, sourcePlanetID := range pending {
			if sourcePlanetID == command.SourcePlanetID {
				sourceIndex = i
				break
			}
		}
		if sourceIndex < 0 {
			return -1, nil, fmt.Errorf("source planet has no pending artificial planet construction")
		}
	} else {
		for i := len(pending) - 1; i >= 0; i-- {
			source, err := g.getPlanet(pending[i])
			if err == nil && source.Owner == playerID && source.ParentStar == target.ParentStar {
				sourceIndex = i
				break
			}
		}
		if sourceIndex < 0 {
			return -1, nil, fmt.Errorf("no pending artificial planet construction in target system")
		}
	}

	source, err := g.getPlanet(pending[sourceIndex])
	if err != nil {
		return -1, nil, err
	}
	if source.Owner != playerID {
		return -1, nil, fmt.Errorf("source planet is not owned by player")
	}
	if target.ParentStar != source.ParentStar {
		return -1, nil, fmt.Errorf("artificial planet target must be in the same system")
	}
	if !isArtificialPlanetTarget(target) {
		return -1, nil, fmt.Errorf("invalid artificial planet target")
	}

	return sourceIndex, target, nil
}

func (g *Game) processArtificialPlanetCommand(playerID uint, command *ArtificialPlanetCommand) error {
	if command == nil {
		return nil
	}
	pending := g.PendingArtificialPlanets[playerID]
	sourceIndex, target, err := g.artificialPlanetChoice(playerID, command, pending)
	if err != nil {
		return err
	}
	g.PendingArtificialPlanets[playerID] = append(pending[:sourceIndex], pending[sourceIndex+1:]...)
	transformArtificialPlanetTarget(target)
	return nil
}

func transformArtificialPlanetTarget(target *Planet) {
	switch target.Environment {
	case PlanetEnvironmentGasGiant:
		target.Size = PlanetSizeHuge
	case PlanetEnvironmentAsteroidBelt:
		target.Size = PlanetSizeLarge
	}
	target.Environment = PlanetEnvironmentBarren
	target.MineralClass = PlanetMineralAbundant
	target.Gravity = PlanetGravityNormal
	target.Owner = 0
	target.Population = 0
	target.PopulationRace = ""
	target.MaxPopulation = 0
	target.LastTurnPopGrowth = 0
	target.Food = PlanetFoodOutput{}
	target.Industry = PlanetIndustryOutput{}
	target.Science = PlanetScienceOutput{}
	target.Income = PlanetIncomeOutput{}
	target.Morale = PlanetMoraleOutput{}
	target.Government = PlanetGovernmentOutput{}
	target.Pollution = 0
	target.HomeworldScienceBonus = 0
	target.Marines = 0
	target.Armored = 0
	target.CurrentProject = ""
	target.Buildings = nil
	target.BuildQueue = nil
	target.IndustryStack = 0
	target.BuildBucket = 0
	target.IsBuilding = false
	target.Autobuild = false
}

func (g *Game) prunePendingArtificialPlanets(playerID uint) {
	if g.PendingArtificialPlanets == nil {
		return
	}
	pending := g.PendingArtificialPlanets[playerID]
	if len(pending) == 0 {
		return
	}
	next := make([]uint, 0, len(pending))
	for _, sourcePlanetID := range pending {
		source, err := g.getPlanet(sourcePlanetID)
		if err != nil || source.Owner != playerID {
			continue
		}
		if g.hasArtificialPlanetTargetsInStar(source.ParentStar) {
			next = append(next, sourcePlanetID)
		}
	}
	if len(next) == 0 {
		delete(g.PendingArtificialPlanets, playerID)
		return
	}
	g.PendingArtificialPlanets[playerID] = next
}
