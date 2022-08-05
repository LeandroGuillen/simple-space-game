package game

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	. "ssg/game/model"
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

func NewGame(playerName string, numStars uint) (*Game, error) {
	g := &Game{
		Galaxy: NewGalaxy(numStars),
		Players: []*Player{
			{ID: 1, Name: playerName},
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

	return g, nil
}

func (g *Game) Turn(playerID uint) error {
	player, err := g.getPlayer(playerID)
	if err != nil {
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

func (g *Game) getPlayer(ID uint) (*Player, error) {
	for _, player := range g.Players {
		if player.ID == ID {
			return player, nil
		}
	}
	return nil, errors.New(fmt.Sprintf("player with ID %d not found", ID))
}

func (g *Game) getPlanet(ID uint) (*Planet, error) {
	for _, star := range g.Galaxy.Stars {
		for _, planet := range star.Planets {
			if planet.ID == ID {
				return planet, nil
			}
		}
	}
	return nil, errors.New(fmt.Sprintf("planet with ID %d not found", ID))
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

	_, err := g.getPlanet(planetID)
	if err != nil {
		return err
	}

	// get building from its ID
	_ = g.Resources.Buildings[buildingID]

	// TODO add build functionality
	//Build(planet, building)
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
				return errors.New(fmt.Sprintf("cannot create a design with name '%s', name already exists", name))
			}
		}
		for _, design := range g.Resources.ShipDesigns {
			if design.Name == name {
				return errors.New(fmt.Sprintf("cannot create a design with name '%s', name already exists", name))
			}
		}
		player.ShipDesigns = append(player.ShipDesigns, basicDesign)
		return nil
	}
}
