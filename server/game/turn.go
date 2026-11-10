package game

import (
	"fmt"
	"math"
	"slices"
	. "ssg/model"
	"strings"
)

const (
	tradeGoodsProject = "Trade Goods"
	housingProject    = "Housing"
)

func planetTurn(p *Planet, commands Commands, player *Player, game *Game, foodReceivedValues ...int64) error {
	foodReceived := int64(0)
	if len(foodReceivedValues) > 0 {
		foodReceived = foodReceivedValues[0]
	}

	hydratePlanetMorale(game, p, player)
	hydratePlanetGovernment(game, p, player)
	hydratePlanetOutputs(game, p)

	if err := applyMovePopulationCommands(p, commands.MovePopulation); err != nil {
		return err
	}

	maxPop := MaxPop(game, p, player)
	p.MaxPopulation = maxPop
	if p.Population > maxPop {
		p.Population = maxPop
		capAssignedPopulation(p)
	}

	baseIndustryProduced := GamePlanetIndustryProduced(game, p)

	// Calculate food demand first to enable Food Replicators conversion
	foodDemand := int64(math.Ceil(PopulationFoodDemand(player, p.Population)))
	// Calculate food with Food Replicators converting production to food if needed
	replicatorResult := CalculateFoodWithReplicators(game, p, foodDemand, baseIndustryProduced)
	totalFood := replicatorResult.FoodProduced + foodReceived

	// Industry is reduced by what Food Replicators consumed (2 production = 1 food)
	industryProduced := baseIndustryProduced - replicatorResult.IndustryConsumed

	planetBuildCommand := findPlanetBuildCommand(p, commands)
	buildingName := ""
	if planetBuildCommand != nil {
		buildingName = planetBuildCommand.Building
	}
	if buildingName == housingProject {
		if p.Population >= maxPop {
			p.CurrentProject = tradeGoodsProject
		} else {
			p.CurrentProject = housingProject
		}
		p.IsBuilding = false
		p.BuildBucket = 0
	}
	if planetBuildCommand == nil && p.CurrentProject == housingProject && p.Population >= maxPop {
		p.CurrentProject = tradeGoodsProject
	}

	// population growth calculations
	oldPopulation := p.Population
	housingBonus := int64(0)
	housingActive := isHousingProject(p, commands) && p.Population < maxPop
	if housingActive {
		housingBonus = housingGrowthBonus(industryProduced, p.Population)
	}

	growth := PopulationGrowthFromFoodWithHousing(
		player,
		p.Population,
		totalFood,
		maxPop,
		player.Race.Stats.PopulationGrowthPercent+MedicineGrowthPercent(game, player),
		PopulationGrowthBonus(game, p),
		housingBonus,
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
	if totalFood < foodDemand {
		game.recordTurnEvent(TurnEvent{Owner: player.ID, Kind: "starvation", Text: fmt.Sprintf("Food shortage on %s: %d population lost", p.Name, max(0, oldPopulation-p.Population)), PlanetID: p.ID, StarID: p.ParentStar})
	}
	if housingActive && p.Population >= maxPop {
		p.CurrentProject = tradeGoodsProject
	}

	// should we increase/decrease pop units?
	popIncrease := int64(math.Floor(float64(p.Population/1000)) - math.Floor(float64(oldPopulation/1000)))
	if popIncrease > 0 {
		assignNewPopulation(game, player, p, popIncrease)
	}
	capAssignedPopulation(p)

	if planetBuildCommand != nil {
		if planetBuildCommand.BuildQueue != nil {
			p.BuildQueue = sanitizeBuildQueue(p, buildingName, planetBuildCommand.BuildQueue, player.Available.Buildings, game)
			if buildingName != "" && len(p.BuildQueue) > 0 && p.BuildQueue[0] == buildingName {
				p.BuildQueue = p.BuildQueue[1:]
			}
			if buildingName == "" && len(p.BuildQueue) > 0 {
				buildingName = popNextQueuedBuild(p)
			}
		}
	}
	if buildingName == "" && !(p.IsBuilding && p.CurrentProject != "") && !isSpecialNonBuildingProject(p.CurrentProject) {
		if len(p.BuildQueue) > 0 {
			buildingName = popNextQueuedBuild(p)
			if !Build(p, buildingName, player.Available.Buildings, game) {
				game.recordBlockedProduction(p, buildingName)
				p.CurrentProject = tradeGoodsProject
				p.IsBuilding = false
				p.BuildBucket = 0
			}
		} else {
			if p.Autobuild {
				buildingName = nextAutobuildProject(p, player.Available.Buildings, game)
			}
			if buildingName == "" || !Build(p, buildingName, player.Available.Buildings, game) {
				p.CurrentProject = tradeGoodsProject
				p.IsBuilding = false
				p.BuildBucket = 0
			}
		}
	} else if buildingName == tradeGoodsProject {
		p.CurrentProject = tradeGoodsProject
		p.IsBuilding = false
		p.BuildBucket = 0
	} else if buildingName == housingProject {
		if p.Population >= maxPop {
			p.CurrentProject = tradeGoodsProject
		} else {
			p.CurrentProject = housingProject
		}
		p.IsBuilding = false
		p.BuildBucket = 0
	} else if buildingName != "" {
		// A stale order can become invalid after submission. Keep the current
		// project and its invested production when the switch cannot be made.
		Build(p, buildingName, player.Available.Buildings, game)
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
		availableProduction := p.IndustryStack + industryProduced
		p.IndustryStack = 0
		for p.IsBuilding {
			nextBucket := p.BuildBucket - availableProduction
			if nextBucket > 0 {
				// project is not complete
				p.BuildBucket = nextBucket
				break
			}

			completedProject := p.CurrentProject
			// Resolve once before spending production. A missing/unbuildable ship
			// must never be interpreted as a newly invented building.
			design, isDesign := game.findBuildableShipDesign(player, completedProject)
			_, buildingErr := getBuildingFromName(game, completedProject)
			if !isDesign && buildingErr != nil {
				p.IndustryStack += availableProduction
				game.recordBlockedProduction(p, completedProject)
				break
			}
			kind := "building"
			if isDesign || completedProject == "Colony Ship" || completedProject == "Transport" {
				kind = "ship"
			}
			game.recordTurnEvent(TurnEvent{Owner: player.ID, Kind: kind, Text: fmt.Sprintf("%s completed at %s", completedProject, p.Name), PlanetID: p.ID, StarID: p.ParentStar})
			availableProduction = -nextBucket
			p.IsBuilding = false
			p.BuildBucket = 0
			p.CurrentProject = ""

			if completedProject == "Colony Base" {
				if game.PendingColonyBases == nil {
					game.PendingColonyBases = make(map[uint][]uint)
				}
				game.PendingColonyBases[player.ID] = append(game.PendingColonyBases[player.ID], p.ID)
			} else if completedProject == artificialPlanetProject {
				if game.hasArtificialPlanetTargetsInStar(p.ParentStar) {
					if game.PendingArtificialPlanets == nil {
						game.PendingArtificialPlanets = make(map[uint][]uint)
					}
					game.PendingArtificialPlanets[player.ID] = append(game.PendingArtificialPlanets[player.ID], p.ID)
				}
			} else if completedProject == terraformingProject {
				terraformPlanet(p)
			} else if completedProject == gaiaTransformationProject {
				gaiaTransformPlanet(p)
			} else if completedProject == "Colony Ship" {
				game.addShipToOrbitFleet(player.ID, p.ParentStar, FleetShip{
					Type: "colony_ship",
					Name: "Colony Ship",
					Icon: "🚀",
				})
			} else if completedProject == "Transport" {
				game.addShipToOrbitFleet(player.ID, p.ParentStar, FleetShip{
					Type: "transport",
					Name: "Transport",
					Icon: "🚢",
				})
			} else if completedProject == "Freighters" {
				player.Freighters += 5
			} else if isDesign {
				game.addShipToOrbitFleet(player.ID, p.ParentStar, fleetShipFromDesign(*design))
			} else {
				buildOnPlanet(game, p, completedProject)
			}

			if len(p.BuildQueue) == 0 {
				p.IndustryStack += availableProduction
				break
			}
			nextProject := popNextQueuedBuild(p)
			if !Build(p, nextProject, player.Available.Buildings, game) {
				game.recordBlockedProduction(p, nextProject)
				p.CurrentProject = tradeGoodsProject
				p.IsBuilding = false
				p.BuildBucket = 0
				p.IndustryStack += availableProduction
				break
			}
		}
	} else if p.CurrentProject != tradeGoodsProject && p.CurrentProject != housingProject {
		// if nothing to work on, save production
		p.IndustryStack += industryProduced
	}

	// Produce ground units from barracks buildings
	produceGroundUnits(p, game, player)

	return nil
}

func findPlanetBuildCommand(p *Planet, commands Commands) *BuildCommand {
	for _, command := range commands.Build {
		if command.PlanetID == p.ID {
			return command
		}
	}
	return nil
}

func isHousingProject(p *Planet, commands Commands) bool {
	command := findPlanetBuildCommand(p, commands)
	if command != nil {
		return command.Building == housingProject
	}
	return p.CurrentProject == housingProject
}

func housingGrowthBonus(industryProduced, population int64) int64 {
	if industryProduced <= 0 || population <= 0 {
		return 0
	}
	populationUnits := max(int64(1), population/1000)
	return int64(math.Floor(float64(industryProduced*40) / float64(populationUnits)))
}

func isSpecialNonBuildingProject(project string) bool {
	return project == tradeGoodsProject || project == housingProject
}

// farmingPossible reports whether a farmer can produce any food on the planet.
// Hostile worlds (Toxic, Barren, Radiated) yield zero food per farmer until a
// breakthrough such as Biomorphic Fungi makes farming viable, so assigning
// colonists to farming there is pointless.
func farmingPossible(p *Planet) bool {
	return p.Food.ProducedPerFarmer > 0
}

func assignNewPopulation(game *Game, player *Player, p *Planet, popIncrease int64) {
	foodDemand := int64(math.Ceil(PopulationFoodDemand(player, p.Population)))
	for range popIncrease {
		if farmingPossible(p) && GamePlanetFoodProduced(game, p) < foodDemand {
			p.Food.Farmers++
		} else if p.Industry.Workers >= p.Science.Scientists {
			p.Industry.Workers++
		} else {
			p.Science.Scientists++
		}
	}
}

func applyMovePopulationCommands(p *Planet, commands []*MovePopulationCommand) error {
	for _, movePopulation := range commands {
		if movePopulation.PlanetID != p.ID {
			continue
		}
		// Defensive validation at apply time
		if err := validateMovePopulationCommand(movePopulation, p); err != nil {
			return err
		}
		p.Food.Farmers = movePopulation.Farmers
		p.Industry.Workers = movePopulation.Workers
		p.Science.Scientists = movePopulation.Scientists
		break
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
		artificialPlanetProject,
		terraformingProject,
		gaiaTransformationProject,
		"Outpost Ship",
		"Transport",
		"Freighters",
		"Spy",
		"Marine Barracks",
		"Armor Barracks",
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
		case p.Food.Farmers > 0:
			p.Food.Farmers--
		case p.Industry.Workers > 0:
			p.Industry.Workers--
		case p.Science.Scientists > 0:
			p.Science.Scientists--
		default:
			return
		}
	}
}

func sanitizeBuildQueue(p *Planet, selectedProject string, buildQueue []string, availableBuildings []string, game *Game) []string {
	queue := make([]string, 0, min(len(buildQueue), 10))
	seen := make(map[string]bool)
	for _, buildingName := range buildQueue {
		if len(queue) >= 10 {
			break
		}
		if buildingName == "" || isSpecialNonBuildingProject(buildingName) {
			continue
		}
		if !isRepeatableProject(buildingName) && !isShipDesignProductionProject(p, buildingName, game) && seen[buildingName] {
			continue
		}
		if p.IsBuilding && p.CurrentProject == buildingName && selectedProject != buildingName {
			queue = append(queue, buildingName)
			seen[buildingName] = true
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
	if buildingName == "" {
		return false
	}
	if buildingName == artificialPlanetProject && p.IsBuilding && p.CurrentProject == buildingName {
		return true
	}
	if buildingName == artificialPlanetProject && !canBuildArtificialPlanetProject(p, availableBuildings, game) {
		return false
	}
	if isTerraformingProject(buildingName) && p.IsBuilding && p.CurrentProject == buildingName {
		return true
	}
	if isTerraformingProject(buildingName) && !canTerraformPlanet(p) {
		return false
	}
	if isGaiaTransformationProject(buildingName) && p.IsBuilding && p.CurrentProject == buildingName {
		return true
	}
	if isGaiaTransformationProject(buildingName) && !canGaiaTransformPlanet(p) {
		return false
	}
	if buildShipDesignProject(p, buildingName, game) {
		return true
	}

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
	if !buildingApplicableToPlanet(p, buildingName, game) {
		return false
	}

	projectCost := effectiveBuildingCost(p, building, game)
	investedProduction := consumeStackedBuildProduction(p, projectCost, game)
	p.CurrentProject = buildingName
	p.IsBuilding = true
	p.BuildBucket = max(0, projectCost-investedProduction)

	return true
}

func currentBuildProgress(p *Planet, game *Game) int64 {
	if !p.IsBuilding || p.CurrentProject == "" {
		return 0
	}
	if cost, ok := effectiveShipDesignCost(p, p.CurrentProject, game); ok {
		return max(0, cost-p.BuildBucket)
	}
	building, err := getBuildingFromName(game, p.CurrentProject)
	if err != nil {
		return 0
	}
	return max(0, effectiveBuildingCost(p, building, game)-p.BuildBucket)
}

func consumeStackedBuildProduction(p *Planet, projectCost int64, game *Game) int64 {
	stackedProduction := p.IndustryStack + currentBuildProgress(p, game)
	investedProduction := min(projectCost, stackedProduction)
	p.IndustryStack = max(0, stackedProduction-projectCost)
	return investedProduction
}

func canBuild(p *Planet, buildingName string, availableBuildings []string, game *Game) bool {
	if canBuildShipDesignProject(p, buildingName, game) {
		return true
	}
	if buildingName == artificialPlanetProject {
		return canBuildArtificialPlanetProject(p, availableBuildings, game)
	}
	if isTerraformingProject(buildingName) {
		return canTerraformPlanet(p)
	}
	if isGaiaTransformationProject(buildingName) {
		return canGaiaTransformPlanet(p)
	}
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
	if !buildingApplicableToPlanet(p, buildingName, game) {
		return false
	}
	return true
}

func effectiveBuildingCost(p *Planet, building Building, game *Game) int64 {
	if isTerraformingProject(building.Name) {
		return effectiveTerraformingCost(p)
	}
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

func canBuildShipDesignProject(p *Planet, projectName string, game *Game) bool {
	if p == nil || game == nil || p.IsBuilding && p.CurrentProject == projectName {
		return false
	}
	player := gamePlayerForPlanet(game, p)
	_, ok := game.findBuildableShipDesign(player, projectName)
	return ok
}

func buildShipDesignProject(p *Planet, projectName string, game *Game) bool {
	cost, ok := effectiveShipDesignCost(p, projectName, game)
	if !ok {
		return false
	}
	investedProduction := consumeStackedBuildProduction(p, cost, game)
	p.CurrentProject = projectName
	p.IsBuilding = true
	p.BuildBucket = max(0, cost-investedProduction)
	return true
}

func effectiveShipDesignCost(p *Planet, projectName string, game *Game) (int64, bool) {
	player := gamePlayerForPlanet(game, p)
	design, ok := game.findBuildableShipDesign(player, projectName)
	if !ok {
		return 0, false
	}
	cost := shipDesignProductionCost(*design)
	multiplier := governmentShipCostMultiplier(game, player)
	if multiplier < 0 {
		multiplier = 0
	}
	if multiplier != 0 && multiplier != 1 {
		cost = int64(math.Ceil(float64(cost)*multiplier - 0.0000001))
	}
	return cost, true
}

func isShipDesignProductionProject(p *Planet, projectName string, game *Game) bool {
	_, ok := effectiveShipDesignCost(p, projectName, game)
	return ok
}

func isShipProductionProject(buildingName string) bool {
	return strings.HasSuffix(buildingName, " Ship")
}

// buildingApplicableToPlanet reports whether a building has any meaningful
// effect on the given planet, given the planet owner's race. Some buildings
// only matter under specific planetary conditions and are hidden otherwise.
func buildingApplicableToPlanet(p *Planet, buildingName string, game *Game) bool {
	if p == nil {
		return true
	}
	switch buildingName {
	case gravityGeneratorBuildingName:
		return gravityGeneratorApplicable(p, gamePlayerForPlanet(game, p))
	case terraformingProject:
		return canTerraformPlanet(p)
	case gaiaTransformationProject:
		return canGaiaTransformPlanet(p)
	}
	if !farmerBonusBuildingAllowedOnPlanet(p, buildingName, game) {
		return false
	}
	return true
}

// farmerBonusBuildingAllowedOnPlanet reports whether a building whose only
// benefit is boosting food per farmer may be built on the planet. Such buildings
// are useless on hostile worlds (Toxic, Barren, Radiated) until Biomorphic Fungi
// lets farmers produce food there, so they are hidden until that tech is researched.
// Buildings that grant flat food (or any non-farmer modifier) are unaffected.
func farmerBonusBuildingAllowedOnPlanet(p *Planet, buildingName string, game *Game) bool {
	if !isHostileEnvironment(p.Environment) {
		return true
	}
	building, err := getBuildingFromName(game, buildingName)
	if err != nil || !buildingIsFarmerBonusOnly(building) {
		return true
	}
	if player := gamePlayerForPlanet(game, p); player != nil && isTechResearched(player, "Biomorphic Fungi") {
		return true
	}
	return false
}

// buildingIsFarmerBonusOnly reports whether a building's sole effect is increasing
// food per farmer (e.g. Soil Enrichment, Weather Controller, Android Farmers).
func buildingIsFarmerBonusOnly(building Building) bool {
	hasFarmerBonus := false
	for _, modifier := range building.Grants.Modifiers {
		if modifier.Name != "food_per_farmer" {
			return false
		}
		hasFarmerBonus = true
	}
	return hasFarmerBonus
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
	return buildingName == "Colony Base" || buildingName == "Colony Ship" || buildingName == "Transport" || buildingName == "Freighters" || buildingName == artificialPlanetProject || isTerraformingProject(buildingName)
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
	effectiveCost, remaining, cost, err := quoteBuyProduction(p, game)
	if err != nil {
		return err
	}
	if effectiveCost <= 0 {
		p.BuildBucket = 0
		return nil
	}
	if remaining <= 0 {
		p.IndustryStack = max(0, p.IndustryStack-p.BuildBucket)
		p.BuildBucket = 0
		return nil
	}
	if player.Treasury < float64(cost) {
		return fmt.Errorf("cannot buy production: need %d BC, have %.1f BC", cost, player.Treasury)
	}
	player.Treasury -= float64(cost)
	p.IndustryStack = 0
	p.BuildBucket = 0
	return nil
}

// produceGroundUnits adds marines and armored units based on building modifiers
func produceGroundUnits(p *Planet, game *Game, player *Player) {
	if game == nil || game.Resources == nil || game.Resources.BuildingCollection == nil {
		return
	}

	marineProduction := int64(0)
	armoredProduction := int64(0)

	// Check buildings for ground unit production modifiers
	for _, buildingName := range p.Buildings {
		if building, err := getBuildingFromName(game, buildingName); err == nil {
			marineProduction += groundUnitModifierValue(building.Grants.Modifiers, "marine_production")
			armoredProduction += groundUnitModifierValue(building.Grants.Modifiers, "armored_production")
		}
	}

	// Check technologies for ground unit production bonuses
	if game.Resources.TechTree != nil && player != nil {
		for _, techName := range player.ResearchedTechs {
			if tech, ok := game.findTechDefinition(techName); ok {
				marineProduction += groundUnitModifierValue(tech.Grants.Modifiers, "marine_production")
				armoredProduction += groundUnitModifierValue(tech.Grants.Modifiers, "armored_production")
			}
		}
	}

	// Add produced units to planet
	p.Marines += marineProduction
	p.Armored += armoredProduction
}

// groundUnitModifierValue extracts ground unit production values from modifier lists
func groundUnitModifierValue(modifiers []Modifier, modifierName string) int64 {
	total := int64(0)
	for _, modifier := range modifiers {
		if modifier.Name == modifierName {
			total += int64(modifier.Value)
		}
	}
	return total
}

func (g *Game) recordBlockedProduction(p *Planet, project string) {
	g.recordTurnEvent(TurnEvent{Owner: p.Owner, Kind: "production_blocked", Text: fmt.Sprintf("%s could no longer be built at %s; production kept", project, p.Name), PlanetID: p.ID, StarID: p.ParentStar})
}
