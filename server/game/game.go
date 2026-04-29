package game

import (
	"fmt"
	. "ssg/model"

	"github.com/google/uuid"
)

type Turnable interface {
	Turn()
}

type Game struct {
	Galaxy      Galaxy    `json:"galaxy"`
	Players     []*Player `json:"players"`
	ID          string    `json:"id"`
	CurrentTurn int64     `json:"current_turn"`
	Resources   Resources `json:"resources"`

	resourceManager *ResourceManager
}

func NewGame(playerName string, numStars uint, playerColor string) (*Game, error) {
	if playerColor == "" {
		playerColor = DefaultPlayerColor()
	}
	if !IsValidPlayerColor(playerColor) {
		return nil, fmt.Errorf("invalid player color %q", playerColor)
	}

	g := &Game{
		Galaxy: NewGalaxy(numStars),
		Players: []*Player{
			{ID: 1, Name: playerName, Color: playerColor},
		},
		ID:          uuid.New().String(),
		CurrentTurn: 3500,
		Resources:   Resources{},
	}
	if rm, err := NewResourceManager(g); err != nil {
		return nil, err
	} else {
		g.resourceManager = rm
	}

	_ = g.CreateCapitalPlanet(1, 1)
	_ = g.ExploreStar(1, 1)

	return g, nil
}

func (g *Game) Turn(playerID uint, commands Commands) error {
	player, err := g.getPlayer(playerID)
	if err != nil {
		return err
	}

	if err := g.ApplyMovePopulationCommands(playerID, commands.MovePopulation); err != nil {
		return err
	}

	player.Treasury += 2

	colonyList, err := g.GetColonyList(playerID)
	if err != nil {
		return err
	}
	for _, colony := range colonyList {
		planetTurn(colony)
	}

	g.CurrentTurn += 1
	return nil
}

func (g *Game) CreateCapitalPlanet(ownerID, planetID uint) error {
	if _, err := g.getPlayer(ownerID); err != nil {
		return err
	}

	planet, err := g.getPlanet(planetID)
	if err != nil {
		return err
	}

	// default capital terran world
	planet.Environment = PlanetEnvironmentTerran
	planet.Size = PlanetSizeMedium
	planet.MineralClass = PlanetMineralAbundant
	planet.Gravity = PlanetGravityNormal

	planet.Owner = ownerID
	planet.Population = 8000
	planet.Workers = 3
	planet.Farmers = 2
	planet.Scientists = 3

	// TODO add building functionality
	//AddBuilding(planet, g.Resources.Buildings[0])

	return nil
}

func (g *Game) ColonizePlanet(ownerID, planetID uint) error {
	if _, err := g.getPlayer(ownerID); err != nil {
		return err
	}

	_, err := g.getPlanet(planetID)
	if err != nil {
		return err
	}

	// TODO add colonize functionality
	//Colonize(planet, ownerID)

	return nil
}

// GetColonyList returns a list of planets that have a colony owned by given player.
func (g *Game) GetColonyList(playerID uint) ([]*Planet, error) {
	if _, err := g.getPlayer(playerID); err != nil {
		return nil, err
	}

	colonyList := make([]*Planet, 0)
	for _, star := range g.Galaxy.Stars {
		for _, planet := range star.Planets {
			if planet.Owner == playerID {
				colonyList = append(colonyList, planet)
			}
		}
	}
	return colonyList, nil
}

func (g *Game) BuildOnPlanet(planetID, ownerID, buildingID uint) error {
	if _, err := g.getPlayer(ownerID); err != nil {
		return err
	}

	planet, err := g.getPlanet(planetID)
	if err != nil {
		return err
	}

	building := g.Resources.Buildings[buildingID]

	planet.CurrentProject = building
	planet.IsBuilding = true
	planet.BuildBucket = building.Cost

	return nil
}

func (g *Game) GetShipDesignsList(playerID uint) ([]*ShipDesign, error) {
	if player, err := g.getPlayer(playerID); err != nil {
		return nil, err
	} else {
		var shipDesigns []*ShipDesign
		for _, d := range g.Resources.ShipDesigns {
			shipDesigns = append(shipDesigns, &d)
		}
		for _, d := range player.ShipDesigns {
			shipDesigns = append(shipDesigns, &d)
		}
		return shipDesigns, nil
	}
}

func (g *Game) CreateShipDesign(
	playerID uint,
	name,
	driveName,
	computerName,
	shieldName,
	armorName,
	hullSizeName,
	fuelCellName string) error {

	if err := g.resourceManager.canICreateThisDesign(driveName, computerName, shieldName, armorName, hullSizeName, fuelCellName); err != nil {
		return err
	}

	parts := g.Resources.ShipPartsCollection

	// I can ignore errors here because I just checked that all part names are valid
	armor, _ := ArmorFromName(armorName, parts.Armors)
	drive, _ := DriveFromName(driveName, parts.Drives)
	hullSize, _ := HullSizeFromName(hullSizeName, parts.HullSizes)
	computer, _ := ComputerFromName(computerName, parts.Computers)
	fuelCell, _ := FuelCellFromName(fuelCellName, parts.FuelCells)
	shield, _ := ShieldFromName(shieldName, parts.Shields)

	basicDesign := NewShipDesign(
		name,
		[]Beam{},
		[]SpecialSystem{},
		hullSize,
		drive,
		armor,
		computer,
		fuelCell,
		shield,
	)

	if player, err := g.getPlayer(playerID); err != nil {
		return err
	} else {
		for _, design := range player.ShipDesigns {
			if design.Name == name {
				return fmt.Errorf("cannot create a design with name '%s', name already exists", name)
			}
		}
		for _, design := range g.Resources.ShipDesigns {
			if design.Name == name {
				return fmt.Errorf("cannot create a design with name %q, name already exists", name)
			}
		}
		player.ShipDesigns = append(player.ShipDesigns, basicDesign)
		return nil
	}
}

func (g *Game) UpdateShipDesign(
	playerID uint,
	name,
	driveName,
	computerName,
	shieldName,
	armorName,
	hullSizeName,
	fuelCellName string) error {

	if err := g.resourceManager.canICreateThisDesign(driveName, computerName, shieldName, armorName, hullSizeName, fuelCellName); err != nil {
		return err
	}

	parts := g.Resources.ShipPartsCollection

	// I can ignore errors here because I just checked that all part names are valid
	armor, _ := ArmorFromName(armorName, parts.Armors)
	drive, _ := DriveFromName(driveName, parts.Drives)
	hullSize, _ := HullSizeFromName(hullSizeName, parts.HullSizes)
	computer, _ := ComputerFromName(computerName, parts.Computers)
	fuelCell, _ := FuelCellFromName(fuelCellName, parts.FuelCells)
	shield, _ := ShieldFromName(shieldName, parts.Shields)

	// get player reference
	player, err := g.getPlayer(playerID)
	if err != nil {
		return err
	}

	// find ship design to update
	for i, d := range player.ShipDesigns {
		if d.Name == name {
			design := &player.ShipDesigns[i]
			design.Armor = armor
			design.Drive = drive
			design.HullSize = hullSize
			design.Computer = computer
			design.FuelCell = fuelCell
			design.Shield = shield
			return nil
		}
	}

	return fmt.Errorf("cannot update, did not find a design with name %q", name)
}

func (g *Game) ApplyMovePopulationCommands(playerID uint, commands []MovePopulationCommand) error {
	for _, cmd := range commands {
		planet, err := g.getPlanet(cmd.PlanetID)
		if err != nil {
			return err
		}
		if planet.Owner != playerID {
			return fmt.Errorf("planet %d not owned by player %d", cmd.PlanetID, playerID)
		}

		fromCount := g.getPopulationCount(planet, cmd.From)
		if fromCount < cmd.Count {
			return fmt.Errorf("not enough %s on planet %d", cmd.From, cmd.PlanetID)
		}

		totalPop := planet.Farmers + planet.Workers + planet.Scientists + planet.Marines
		newTotalPop := totalPop - cmd.Count + cmd.Count
		maxPop := MaxPop(planet)
		if newTotalPop > maxPop {
			return fmt.Errorf("population %d exceeds max %d on planet %d", newTotalPop, maxPop, cmd.PlanetID)
		}

		planet.Population = newTotalPop

		switch cmd.From {
		case PopCategoryFarmers:
			planet.Farmers -= cmd.Count
		case PopCategoryWorkers:
			planet.Workers -= cmd.Count
		case PopCategoryScientists:
			planet.Scientists -= cmd.Count
		case PopCategoryMarines:
			planet.Marines -= cmd.Count
		}

		switch cmd.To {
		case PopCategoryFarmers:
			planet.Farmers += cmd.Count
		case PopCategoryWorkers:
			planet.Workers += cmd.Count
		case PopCategoryScientists:
			planet.Scientists += cmd.Count
		case PopCategoryMarines:
			planet.Marines += cmd.Count
		}
	}
	return nil
}

func (g *Game) getPopulationCount(p *Planet, cat PopCategory) int64 {
	switch cat {
	case PopCategoryFarmers:
		return p.Farmers
	case PopCategoryWorkers:
		return p.Workers
	case PopCategoryScientists:
		return p.Scientists
	case PopCategoryMarines:
		return p.Marines
	}
	return 0
}
