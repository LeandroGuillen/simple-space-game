package game

import (
	"fmt"
	"math"
	"slices"
	. "ssg/model"
	"strings"
)

func planetTurn(p *Planet, commands Commands, player *Player, game *Game) error {

	hydratePlanetMorale(game, p, player)
	hydratePlanetGovernment(game, p, player)
	hydratePlanetOutputs(game, p)

	// population movement
	for _, movePopulation := range commands.MovePopulation {
		if movePopulation.PlanetID != p.ID {
			continue
		}
		newTotal := movePopulation.Farmers + movePopulation.Workers + movePopulation.Scientists
		if newTotal > p.Population/1000 {
			return fmt.Errorf("bad pop movement: too many units")
		}
		p.Food.Farmers = movePopulation.Farmers
		p.Industry.Workers = movePopulation.Workers
		p.Science.Scientists = movePopulation.Scientists
		break
	}

	maxPop := MaxPop(game, p, player)
	p.MaxPopulation = maxPop
	if p.Population > maxPop {
		p.Population = maxPop
		capAssignedPopulation(p)
	}

	// population growth calculations
	oldPopulation := p.Population
	growth := PopulationGrowthFromFood(
		p.Population,
		GamePlanetFoodProduced(game, p),
		maxPop,
		player.Race.Stats.PopulationGrowthPercent,
		PopulationGrowthBonus(game, p),
	)
	p.Population += growth
	p.LastTurnPopGrowth = growth

	// minimum pop units is 1
	if p.Population < 1000 {
		p.Population = 1000
	}

	// capping population to planet max
	if p.Population > maxPop {
		p.Population = maxPop
	}

	// should we increase/decrease pop units?
	popIncrease := int64(math.Floor(float64(p.Population/1000)) - math.Floor(float64(oldPopulation/1000)))
	if popIncrease != 0 {
		// whoever has more units, add one more. This reinforces colonies that are
		// focused on one type of production
		if p.Industry.Workers >= p.Food.Farmers && p.Industry.Workers >= p.Science.Scientists {
			p.Industry.Workers += popIncrease
		} else if p.Food.Farmers >= p.Industry.Workers && p.Food.Farmers >= p.Science.Scientists {
			p.Food.Farmers += popIncrease
		} else {
			p.Science.Scientists += popIncrease
		}
	}
	capAssignedPopulation(p)

	// industry output
	industryProduced := GamePlanetIndustryProduced(game, p)

	var planetBuildCommand *BuildCommand
	for _, command := range commands.Build {
		if command.PlanetID == p.ID {
			planetBuildCommand = command
		}
	}
	buildingName := ""
	if planetBuildCommand != nil {
		buildingName = planetBuildCommand.Building
		p.BuildQueue = sanitizeBuildQueue(p, planetBuildCommand.BuildQueue, player.Available.Buildings, game)
		if buildingName != "" && len(p.BuildQueue) > 0 && p.BuildQueue[0] == buildingName {
			p.BuildQueue = p.BuildQueue[1:]
		}
		if buildingName == "" && len(p.BuildQueue) > 0 {
			buildingName = popNextQueuedBuild(p)
		}
	}
	if buildingName == "" && !(p.IsBuilding && p.CurrentProject != "") {
		if len(p.BuildQueue) > 0 {
			buildingName = popNextQueuedBuild(p)
			if !Build(p, buildingName, player.Available.Buildings, game) {
				p.CurrentProject = "Trade Goods"
				p.IsBuilding = false
				p.BuildBucket = 0
			}
		} else {
			if p.Autobuild {
				buildingName = nextAutobuildProject(p, player.Available.Buildings, game)
			}
			if buildingName == "" || !Build(p, buildingName, player.Available.Buildings, game) {
				p.CurrentProject = "Trade Goods"
				p.IsBuilding = false
				p.BuildBucket = 0
			}
		}
	} else if buildingName == "Trade Goods" {
		p.CurrentProject = "Trade Goods"
		p.IsBuilding = false
		p.BuildBucket = 0
		p.BuildQueue = nil
	} else if buildingName != "" {
		if !Build(p, buildingName, player.Available.Buildings, game) {
			p.CurrentProject = "Trade Goods"
			p.IsBuilding = false
			p.BuildBucket = 0
			p.BuildQueue = nil
		}
	}

	for _, command := range commands.BuyProduction {
		if command == nil || command.PlanetID != p.ID {
			continue
		}
		if err := applyBuyProduction(p, player, game); err != nil {
			return err
		}
		break
	}

	hydratePlanetIncome(game, p, industryProduced)

	// Advance build project
	if p.IsBuilding {
		nextBucket := p.BuildBucket - industryProduced
		if nextBucket <= 0 {
			// project is complete
			completedProject := p.CurrentProject
			p.IsBuilding = false
			p.IndustryStack += -nextBucket
			p.BuildBucket = 0
			p.CurrentProject = ""

			if completedProject == "Colony Base" {
				if game.PendingColonyBases == nil {
					game.PendingColonyBases = make(map[uint][]uint)
				}
				game.PendingColonyBases[player.ID] = append(game.PendingColonyBases[player.ID], p.ID)
			} else if completedProject == "Colony Ship" {
				game.addShipToOrbitFleet(player.ID, p.ParentStar, FleetShip{
					Type: "colony_ship",
					Name: "Colony Ship",
					Icon: "🚀",
				})
			} else {
				buildOnPlanet(game, p, completedProject)
			}

			if len(p.BuildQueue) > 0 {
				nextProject := popNextQueuedBuild(p)
				if !Build(p, nextProject, player.Available.Buildings, game) {
					p.CurrentProject = "Trade Goods"
					p.IsBuilding = false
					p.BuildBucket = 0
				}
			}
		} else {
			// project is not complete
			p.BuildBucket = nextBucket
		}
	} else {
		// if nothing to work on, save production
		p.IndustryStack += industryProduced
	}

	return nil
}

func nextAutobuildProject(p *Planet, availableBuildings []string, game *Game) string {
	for _, buildingName := range availableBuildings {
		if !isAutobuildProject(buildingName) {
			continue
		}
		if canBuild(p, buildingName, availableBuildings, game) {
			return buildingName
		}
	}
	return ""
}

func isAutobuildProject(buildingName string) bool {
	switch buildingName {
	case "Colony Base",
		"Colony Ship",
		"Outpost Ship",
		"Transport",
		"Freighters",
		"Spy",
		"Android Farmers",
		"Android Workers",
		"Android Scientists":
		return false
	}
	return !strings.HasSuffix(buildingName, " Ship")
}

func capAssignedPopulation(p *Planet) {
	maxAssignedPopulation := p.Population / 1000
	for p.Food.Farmers+p.Industry.Workers+p.Science.Scientists > maxAssignedPopulation {
		switch {
		case p.Industry.Workers >= p.Food.Farmers && p.Industry.Workers >= p.Science.Scientists && p.Industry.Workers > 0:
			p.Industry.Workers--
		case p.Food.Farmers >= p.Industry.Workers && p.Food.Farmers >= p.Science.Scientists && p.Food.Farmers > 0:
			p.Food.Farmers--
		case p.Science.Scientists > 0:
			p.Science.Scientists--
		default:
			return
		}
	}
}

func sanitizeBuildQueue(p *Planet, buildQueue []string, availableBuildings []string, game *Game) []string {
	queue := make([]string, 0, min(len(buildQueue), 10))
	seen := make(map[string]bool)
	for _, buildingName := range buildQueue {
		if len(queue) >= 10 {
			break
		}
		if buildingName == "" || buildingName == "Trade Goods" {
			continue
		}
		if !isRepeatableProject(buildingName) && seen[buildingName] {
			continue
		}
		if !canBuild(p, buildingName, availableBuildings, game) {
			continue
		}
		queue = append(queue, buildingName)
		seen[buildingName] = true
	}
	return queue
}

func popNextQueuedBuild(p *Planet) string {
	if len(p.BuildQueue) == 0 {
		return ""
	}
	nextProject := p.BuildQueue[0]
	p.BuildQueue = p.BuildQueue[1:]
	return nextProject
}

func Build(p *Planet, buildingName string, availableBuildings []string, game *Game) bool {
	// is the building already built on this planet
	found := false
	for _, buildingInPlanet := range p.Buildings {
		if isRepeatableProject(buildingName) {
			continue
		}
		if buildingName == buildingInPlanet {
			found = true
			break
		}
	}

	// does the player have the tech to build this?
	known := false
	for _, availableBuilding := range availableBuildings {
		if buildingName == availableBuilding {
			known = true
			break
		}
	}

	if found || !known {
		return false
	}

	building, err := getBuildingFromName(game, buildingName)
	if err != nil {
		return false
	}

	if p.IsBuilding && p.CurrentProject == buildingName {
		return true
	}
	if building.Unique && isEmpireUniqueBuildingLocked(game, p.Owner, buildingName, p.ID) {
		return false
	}
	if hasSuperiorBuilding(p, buildingName, game) {
		return false
	}

	investedProduction := currentBuildProgress(p, game)
	p.CurrentProject = buildingName
	p.IsBuilding = true
	p.BuildBucket = max(0, effectiveBuildingCost(p, building, game)-investedProduction)

	return true
}

func currentBuildProgress(p *Planet, game *Game) int64 {
	if !p.IsBuilding || p.CurrentProject == "" {
		return 0
	}
	building, err := getBuildingFromName(game, p.CurrentProject)
	if err != nil {
		return 0
	}
	return max(0, effectiveBuildingCost(p, building, game)-p.BuildBucket)
}

func canBuild(p *Planet, buildingName string, availableBuildings []string, game *Game) bool {
	if p.IsBuilding && p.CurrentProject == buildingName {
		return false
	}
	for _, buildingInPlanet := range p.Buildings {
		if isRepeatableProject(buildingName) {
			continue
		}
		if buildingName == buildingInPlanet {
			return false
		}
	}
	foundAvailable := false
	for _, availableBuilding := range availableBuildings {
		if buildingName == availableBuilding {
			foundAvailable = true
			break
		}
	}
	if !foundAvailable {
		return false
	}
	building, err := getBuildingFromName(game, buildingName)
	if err != nil {
		return false
	}
	if building.Unique && isEmpireUniqueBuildingLocked(game, p.Owner, buildingName, p.ID) {
		return false
	}
	if hasSuperiorBuilding(p, buildingName, game) {
		return false
	}
	return true
}

func effectiveBuildingCost(p *Planet, building Building, game *Game) int64 {
	upgradeCredit := int64(0)
	for _, upgradedBuildingName := range building.Upgrades {
		if !slices.Contains(p.Buildings, upgradedBuildingName) {
			continue
		}
		upgradedBuilding, err := getBuildingFromName(game, upgradedBuildingName)
		if err != nil {
			continue
		}
		upgradeCredit = max(upgradeCredit, upgradedBuilding.Cost)
	}
	cost := max(0, building.Cost-upgradeCredit)
	if isShipProductionProject(building.Name) {
		player := gamePlayerForPlanet(game, p)
		multiplier := governmentShipCostMultiplier(game, player)
		if multiplier < 0 {
			multiplier = 0
		}
		if multiplier != 0 && multiplier != 1 {
			cost = int64(math.Ceil(float64(cost)*multiplier - 0.0000001))
		}
	}
	return cost
}

func isShipProductionProject(buildingName string) bool {
	return strings.HasSuffix(buildingName, " Ship")
}

func hasSuperiorBuilding(p *Planet, buildingName string, game *Game) bool {
	for _, builtBuildingName := range p.Buildings {
		builtBuilding, err := getBuildingFromName(game, builtBuildingName)
		if err != nil {
			continue
		}
		if slices.Contains(builtBuilding.Upgrades, buildingName) {
			return true
		}
	}
	return false
}

func isEmpireUniqueBuildingLocked(game *Game, ownerID uint, buildingName string, ignorePlanetID uint) bool {
	for _, star := range game.Galaxy.Stars {
		for _, planet := range star.Planets {
			if planet.Owner != ownerID || planet.ID == ignorePlanetID {
				continue
			}
			if slices.Contains(planet.Buildings, buildingName) ||
				(planet.IsBuilding && planet.CurrentProject == buildingName) ||
				slices.Contains(planet.BuildQueue, buildingName) {
				return true
			}
		}
	}
	return false
}

func isRepeatableProject(buildingName string) bool {
	return buildingName == "Colony Base" || buildingName == "Colony Ship"
}

// BuyProductionCost returns the treasury cost (in BC) to immediately finish a build project
// that has `remaining` PP left out of `effectiveCost` total. The rate is 4 BC/PP while the
// project is below 50% complete and 2 BC/PP once it reaches 50% or more.
func BuyProductionCost(effectiveCost, remaining int64) int64 {
	if remaining <= 0 || effectiveCost <= 0 {
		return 0
	}
	invested := effectiveCost - remaining
	if invested < 0 {
		invested = 0
	}
	rate := int64(4)
	if invested*2 >= effectiveCost {
		rate = 2
	}
	return remaining * rate
}

func applyBuyProduction(p *Planet, player *Player, game *Game) error {
	if p.CurrentProject == "Trade Goods" {
		return fmt.Errorf("cannot buy Trade Goods")
	}
	if !p.IsBuilding || p.CurrentProject == "" {
		return fmt.Errorf("cannot buy production: planet %d is not building", p.ID)
	}
	building, err := getBuildingFromName(game, p.CurrentProject)
	if err != nil {
		return fmt.Errorf("cannot buy production: unknown project %q on planet %d", p.CurrentProject, p.ID)
	}
	effectiveCost := effectiveBuildingCost(p, building, game)
	if effectiveCost <= 0 {
		p.BuildBucket = 0
		return nil
	}
	remaining := p.BuildBucket
	if remaining <= 0 {
		return nil
	}
	cost := BuyProductionCost(effectiveCost, remaining)
	if player.Treasury < cost {
		return fmt.Errorf("cannot buy production: need %d BC, have %d BC", cost, player.Treasury)
	}
	player.Treasury -= cost
	p.BuildBucket = 0
	return nil
}
