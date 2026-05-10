package game

import (
	"fmt"
	. "ssg/model"
	"time"

	"github.com/google/uuid"
)

type Turnable interface {
	Turn()
}

type Game struct {
	Galaxy             Galaxy          `json:"galaxy"`
	Players            []*Player       `json:"players"`
	Fleets             []*Fleet        `json:"fleets"`
	ID                 string          `json:"id"`
	CurrentTurn        int64           `json:"current_turn"`
	CreatedAt          time.Time       `json:"created_at"`
	LastTurnAt         *time.Time      `json:"last_turn_at,omitempty"`
	Resources          *Resources      `json:"resources"`
	PendingColonyBases map[uint][]uint `json:"pending_colony_bases"`
}

func NewGame(resources *ResourceManager, playerName string, numStars uint, playerColor string) (*Game, error) {
	if playerColor == "" {
		playerColor = DefaultPlayerColor()
	}
	if !IsValidPlayerColor(playerColor) {
		return nil, fmt.Errorf("invalid player color %q", playerColor)
	}

	firstPlayerID := uint(1)
	player1 := NewPlayer(firstPlayerID, playerName, playerColor)
	created := time.Now().UTC()
	g := &Game{
		Galaxy:             NewGalaxy(numStars),
		Players:            []*Player{player1},
		Fleets:             []*Fleet{},
		ID:                 uuid.New().String(),
		CurrentTurn:        3500,
		CreatedAt:          created,
		Resources:          &resources.Resources,
		PendingColonyBases: make(map[uint][]uint),
	}
	g.hydratePlayerAvailability(player1)

	_ = g.CreateCapitalPlanet(firstPlayerID)
	_ = g.ExploreStar(firstPlayerID, 1)

	return g, nil
}

/*
Turn sequence

Each time "Turn" button was pushed next actions take place:

 1. Population is increased/decreased.
 2. FP/PP/RP/Money are generated.
 3. New buildings are constructed.
 4. New colonists arrive by freighters (and battles take place).
 5. Research projects are finished (and, possibly, spies finish their jobs).

https://strategywiki.org/wiki/Master_of_Orion_II:_Battle_at_Antares/Calculations
*/
func (g *Game) Turn(playerID uint, commands Commands) error {
	player, err := g.getPlayer(playerID)
	if err != nil {
		return err
	}

	if commands.SelectResearch != nil {
		if err := g.selectResearch(player, *commands.SelectResearch); err != nil {
			return err
		}
	}
	if err := g.issueFleetMoveCommands(playerID, commands.FleetMove); err != nil {
		return err
	}

	// colonies turn processing
	colonyList, err := g.GetColonyList(playerID)
	if err != nil {
		return err
	}
	totalIncome := int64(0)
	totalResearch := int64(0)
	for _, colony := range colonyList {
		if err := planetTurn(colony, commands, player, g); err != nil {
			return err
		}
		totalIncome += colony.Income.Total
		totalResearch += PlanetScienceProduced(colony)
	}

	// process colonize command
	if commands.Colonize != nil {
		if g.PendingColonyBases == nil {
			g.PendingColonyBases = make(map[uint][]uint)
		}
		pending := g.PendingColonyBases[playerID]
		if len(pending) == 0 {
			return fmt.Errorf("no pending colony base")
		}
		sourcePlanetID := pending[len(pending)-1]
		g.PendingColonyBases[playerID] = pending[:len(pending)-1]

		if err := g.ColonizePlanet(playerID, sourcePlanetID, commands.Colonize.PlanetID); err != nil {
			return err
		}
	}
	if len(g.PendingColonyBases[playerID]) > 0 && !g.hasColonizablePlanetsInSourceSystems(g.PendingColonyBases[playerID], playerID) {
		delete(g.PendingColonyBases, playerID)
	}

	// empire turn processing
	player.Treasury += totalIncome
	g.HydrateCommandPoints()
	player.Treasury -= player.CommandOverdrawBC
	if err := g.applyResearch(player, uint64(max(totalResearch, 0))); err != nil {
		return err
	}
	g.advanceFleetMovements()

	g.CurrentTurn += 1
	now := time.Now().UTC()
	g.LastTurnAt = &now
	return nil
}

func (g *Game) CreateCapitalPlanet(ownerID uint) error {
	player, err := g.getPlayer(ownerID)
	if err != nil {
		return err
	}

	// TODO find a better way to get the first planet
	planet, err := g.getPlanet(1)
	if err != nil {
		return err
	}

	planet.Owner = ownerID

	// default capital terran world
	planet.Environment = PlanetEnvironmentTerran
	planet.Size = PlanetSizeMedium
	planet.MineralClass = PlanetMineralAbundant
	planet.Gravity = PlanetGravityNormal

	// population setup
	planet.Population = 8000
	planet.Food.Farmers = 4
	planet.Industry.Workers = 3
	planet.Science.Scientists = 1

	buildOnPlanet(g, planet, "Capitol")
	buildOnPlanet(g, planet, "Star Base")
	buildOnPlanet(g, planet, "Marine Barracks")

	hydratePlanetPopulationCapacity(g, planet, player)
	hydratePlanetMorale(g, planet, player)
	hydratePlanetOutputs(g, planet)
	hydratePlanetIncome(g, planet, 0)

	g.HydrateCommandPoints()

	return nil
}

func (g *Game) ColonizePlanet(ownerID, sourcePlanetID, targetPlanetID uint) error {
	if _, err := g.getPlayer(ownerID); err != nil {
		return err
	}

	source, err := g.getPlanet(sourcePlanetID)
	if err != nil {
		return err
	}
	if source.Owner != ownerID {
		return fmt.Errorf("source planet is not owned by player")
	}

	target, err := g.getPlanet(targetPlanetID)
	if err != nil {
		return err
	}
	if target.ParentStar != source.ParentStar {
		return fmt.Errorf("colony base can only colonize planets in the same system")
	}
	if target.Owner != 0 {
		return fmt.Errorf("planet already colonized")
	}
	if target.Environment == PlanetEnvironmentGasGiant || target.Environment == PlanetEnvironmentAsteroidBelt {
		return fmt.Errorf("cannot colonize this planet type")
	}

	target.Owner = ownerID
	target.Population = 1000

	if target.Environment >= PlanetEnvironmentDesert {
		target.Food.Farmers = 1
	} else {
		target.Industry.Workers = 1
	}

	target.CurrentProject = "Trade Goods"
	target.IsBuilding = false
	target.BuildBucket = 0

	return nil
}

func (g *Game) hasColonizablePlanetsInSourceSystems(sourcePlanetIDs []uint, ownerID uint) bool {
	sourceStars := make(map[uint]bool)
	for _, sourcePlanetID := range sourcePlanetIDs {
		source, err := g.getPlanet(sourcePlanetID)
		if err != nil || source.Owner != ownerID {
			continue
		}
		sourceStars[source.ParentStar] = true
	}
	if len(sourceStars) == 0 {
		return false
	}
	for _, star := range g.Galaxy.Stars {
		if !sourceStars[star.ID] {
			continue
		}
		for _, planet := range star.Planets {
			if planet.Owner == 0 && planet.Environment != PlanetEnvironmentGasGiant && planet.Environment != PlanetEnvironmentAsteroidBelt {
				return true
			}
		}
	}
	return false
}

// GetColonyList returns a list of planets that have a colony owned by given player.
func (g *Game) GetColonyList(playerID uint) ([]*Planet, error) {
	player, err := g.getPlayer(playerID)
	if err != nil {
		return nil, err
	}

	colonyList := make([]*Planet, 0)
	for _, star := range g.Galaxy.Stars {
		for _, planet := range star.Planets {
			if planet.Owner == playerID {
				hydratePlanetMorale(g, planet, player)
				hydratePlanetOutputs(g, planet)
				hydratePlanetPopulationCapacity(g, planet, player)
				hydratePlanetIncome(g, planet, PlanetIndustryProduced(planet))
				colonyList = append(colonyList, planet)
			}
		}
	}
	return colonyList, nil
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
