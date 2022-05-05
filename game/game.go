package game

import (
	"errors"
	"fmt"
	"ssg/data"
	"ssg/game/builder"
	"ssg/game/combat"
	"ssg/game/combat/computers"
	"ssg/game/combat/weapons"
	"ssg/game/model"

	"github.com/google/uuid"
)

type Turnable interface {
	Turn()
}

type Game struct {
	Galaxy      model.Galaxy    `json:"galaxy"`
	Players     []*model.Player `json:"players"`
	ID          string          `json:"id"`
	CurrentTurn float64         `json:"current_turn"`
	Resources   Resource        `json:"resources"`

	resourceManager ResourceManager
}

type Resource struct {
	ShipPartsCollection `json:"ship_parts"`
}

func NewGame(playerName string, numStars uint) *Game {
	g := &Game{
		Galaxy: builder.NewGalaxy(numStars),
		Players: []*model.Player{
			{ID: 1, Name: playerName},
		},
		ID:          uuid.New().String(),
		CurrentTurn: 3500,
		Resources:   Resource{},
	}
	g.Resources.ShipPartsCollection, _ = g.resourceManager.GetShipPartsList()

	_ = g.CreateCapitalPlanet(1, 1)

	return g
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
		colony.Turn()
	}

	g.CurrentTurn += 0.1
	return nil
}

func (g *Game) getPlayer(ID uint) (*model.Player, error) {
	for _, player := range g.Players {
		if player.ID == ID {
			return player, nil
		}
	}
	return nil, errors.New(fmt.Sprintf("player with ID %d not found", ID))
}

func (g *Game) getPlanet(ID uint) (*model.Planet, error) {
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
	planet.Environment = model.PlanetEnvironmentTerran
	planet.Size = model.PlanetSizeMedium
	planet.MineralClass = model.PlanetMineralAbundant
	planet.Gravity = model.PlanetGravityNormal

	planet.Owner = ownerID
	planet.Population = 8000
	planet.Workers = 3
	planet.Farmers = 2
	planet.Scientists = 3

	planet.AddBuilding(data.Buildings[0])

	return nil
}

func (g *Game) ColonizePlanet(ownerID, planetID uint) error {
	if _, err := g.getPlayer(ownerID); err != nil {
		return err
	}

	planet, err := g.getPlanet(planetID)
	if err != nil {
		return err
	}

	planet.Colonize(ownerID)

	return nil
}

// GetColonyList returns a list of planets that have a colony owned by given player.
func (g *Game) GetColonyList(playerID uint) ([]*model.Planet, error) {
	if _, err := g.getPlayer(playerID); err != nil {
		return nil, err
	}

	colonyList := make([]*model.Planet, 0)
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

	p, err := g.getPlanet(planetID)
	if err != nil {
		return err
	}

	// get building from its ID
	b := data.Buildings[buildingID]
	p.Build(b)
	return nil
}

func (g *Game) GetShipDesignsList(playerID uint) ([]*combat.ShipDesign, error) {
	if player, err := g.getPlayer(playerID); err != nil {
		return nil, err
	} else {
		shipDesigns := make([]*combat.ShipDesign, len(player.ShipDesigns))
		for i := range player.ShipDesigns {
			shipDesigns[i] = &player.ShipDesigns[i]
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

	parts, _ := g.resourceManager.GetShipPartsList()

	// I can ignore errors here because I just checked that all part names are valid
	armor, _ := ArmorFromName(armorName, parts.Armors)
	drive, _ := DriveFromName(driveName, parts.Drives)
	hullSize, _ := HullSizeFromName(hullSizeName, parts.HullSizes)
	computer, _ := ComputerFromName(computerName, parts.Computers)
	fuelCell, _ := FuelCellFromName(fuelCellName, parts.FuelCells)
	shield, _ := ShieldFromName(shieldName, parts.Shields)

	basicDesign := combat.NewShipDesign(
		name,
		[]weapons.Weapon{},
		[]combat.SpecialSystem{},
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
		player.ShipDesigns = append(player.ShipDesigns, basicDesign)
		return nil
	}
}

type ShipPartsCollection struct {
	HullSizes []combat.HullSize    `json:"hull_sizes"`
	Drives    []combat.Drive       `json:"drives"`
	Armors    []combat.Armor       `json:"armors"`
	Computers []computers.Computer `json:"computers"`
	FuelCells []combat.FuelCell    `json:"fuel_cells"`
	Shields   []combat.Shield      `json:"shields"`
}
