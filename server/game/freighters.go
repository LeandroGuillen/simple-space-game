package game

import (
	"fmt"
	"log"
	"math"
	"sort"

	. "ssg/model"
)

type FoodTransferResult struct {
	PlanetReceived  map[uint]int64
	FreightersUsed  uint
	OperatingCostBC float64
}

type foodBalance struct {
	planet   *Planet
	amount   int64
	planetID uint
}

func (g *Game) calculateAutomaticFoodTransfers(player *Player, colonies []*Planet, commands Commands) FoodTransferResult {
	result := FoodTransferResult{PlanetReceived: map[uint]int64{}}
	if player == nil || player.Freighters == 0 {
		return result
	}

	availableFreighters := player.Freighters
	if player.FreightersUsed >= availableFreighters {
		return result
	}
	availableFreighters -= player.FreightersUsed

	surpluses := []foodBalance{}
	deficits := []foodBalance{}
	for _, colony := range colonies {
		hydratePlanetMorale(g, colony, player)
		hydratePlanetGovernment(g, colony, player)
		hydratePlanetOutputs(g, colony)
		if err := applyMovePopulationCommands(colony, commands.MovePopulation); err != nil {
			continue
		}
		produced := GamePlanetFoodProduced(g, colony)
		consumed := int64(math.Ceil(PopulationFoodDemand(player, colony.Population)))
		if produced > consumed {
			surpluses = append(surpluses, foodBalance{planet: colony, amount: produced - consumed, planetID: colony.ID})
		} else if consumed > produced {
			deficits = append(deficits, foodBalance{planet: colony, amount: consumed - produced, planetID: colony.ID})
		}
	}

	sort.Slice(surpluses, func(i, j int) bool { return surpluses[i].planetID < surpluses[j].planetID })
	sort.Slice(deficits, func(i, j int) bool { return deficits[i].planetID < deficits[j].planetID })

	remainingCapacity := int64(availableFreighters)
	for deficitIndex := range deficits {
		if remainingCapacity <= 0 {
			break
		}
		needed := deficits[deficitIndex].amount
		for surplusIndex := range surpluses {
			if needed <= 0 || remainingCapacity <= 0 {
				break
			}
			if surpluses[surplusIndex].amount <= 0 {
				continue
			}
			moved := min(needed, surpluses[surplusIndex].amount, remainingCapacity)
			surpluses[surplusIndex].amount -= moved
			needed -= moved
			remainingCapacity -= moved
			result.PlanetReceived[deficits[deficitIndex].planet.ID] += moved
			result.FreightersUsed += uint(moved)
		}
	}
	result.OperatingCostBC = float64(result.FreightersUsed) * 0.5
	return result
}

func (g *Game) issuePopulationTransferCommands(playerID uint, commands []*TransferPopulationCommand) error {
	for _, command := range commands {
		if command == nil {
			continue
		}
		if err := g.issuePopulationTransferCommand(playerID, command); err != nil {
			return err
		}
	}
	g.recalculateFreightersUsed(playerID)
	return nil
}

func (g *Game) issuePopulationTransferCommand(playerID uint, command *TransferPopulationCommand) error {
	if command.PopulationUnits == 0 {
		return fmt.Errorf("population transfer must move at least 1 population unit")
	}
	if command.SourcePlanetID == command.DestinationPlanetID {
		return fmt.Errorf("population transfer source and destination must differ")
	}
	player, err := g.getPlayer(playerID)
	if err != nil {
		return err
	}
	source, err := g.getPlanet(command.SourcePlanetID)
	if err != nil {
		return err
	}
	destination, err := g.getPlanet(command.DestinationPlanetID)
	if err != nil {
		return err
	}
	if source.Owner != playerID || destination.Owner != playerID {
		return fmt.Errorf("population transfer planets must belong to player %d", playerID)
	}
	sourceUnits := uint(source.Population / 1000)
	if sourceUnits <= command.PopulationUnits {
		return fmt.Errorf("population transfer source must retain at least 1 population unit")
	}

	hydratePlanetPopulationCapacity(g, destination, player)
	destinationUnits := uint(destination.Population / 1000)
	destinationCapacity := uint(destination.MaxPopulation / 1000)
	pendingInbound := g.pendingInboundPopulationUnits(playerID, destination.ID)
	if destinationUnits+pendingInbound+command.PopulationUnits > destinationCapacity {
		return fmt.Errorf("population transfer exceeds destination capacity")
	}

	requiredFreighters := command.PopulationUnits * 5
	if g.availableFreighters(playerID) < requiredFreighters {
		return fmt.Errorf("population transfer needs %d freighters", requiredFreighters)
	}

	sourceStar, err := g.getStar(source.ParentStar)
	if err != nil {
		return err
	}
	destinationStar, err := g.getStar(destination.ParentStar)
	if err != nil {
		return err
	}
	turns := calculateFleetTravelTurns(sourceStar.X, sourceStar.Y, destinationStar.X, destinationStar.Y)

	source.Population -= int64(command.PopulationUnits) * 1000
	capAssignedPopulation(source)
	g.PopulationTransfers = append(g.PopulationTransfers, PopulationTransfer{
		ID:                  g.nextPopulationTransferID(),
		Owner:               playerID,
		SourcePlanetID:      source.ID,
		DestinationPlanetID: destination.ID,
		PopulationUnits:     command.PopulationUnits,
		FreightersReserved:  requiredFreighters,
		OriginStarID:        source.ParentStar,
		DestinationStarID:   destination.ParentStar,
		TotalTurns:          turns,
		TurnsRemaining:      turns,
	})
	g.recalculateFreightersUsed(playerID)
	return nil
}

func (g *Game) availableFreighters(playerID uint) uint {
	player, err := g.getPlayer(playerID)
	if err != nil {
		return 0
	}
	g.recalculateFreightersUsed(playerID)
	if player.FreightersUsed >= player.Freighters {
		return 0
	}
	return player.Freighters - player.FreightersUsed
}

func (g *Game) recalculateFreightersUsed(playerID uint) {
	player, err := g.getPlayer(playerID)
	if err != nil {
		return
	}
	used := uint(0)
	for _, transfer := range g.PopulationTransfers {
		if transfer.Owner == playerID {
			used += transfer.FreightersReserved
		}
	}
	player.FreightersUsed = used
}

func (g *Game) populationTransferIDs() map[uint]bool {
	ids := make(map[uint]bool, len(g.PopulationTransfers))
	for _, transfer := range g.PopulationTransfers {
		ids[transfer.ID] = true
	}
	return ids
}

func (g *Game) nextPopulationTransferID() uint {
	maxID := uint(0)
	for _, transfer := range g.PopulationTransfers {
		if transfer.ID > maxID {
			maxID = transfer.ID
		}
	}
	return maxID + 1
}

func (g *Game) advancePopulationTransfers(existing map[uint]bool) {
	next := make([]PopulationTransfer, 0, len(g.PopulationTransfers))
	arrivedOwners := map[uint]bool{}
	for _, transfer := range g.PopulationTransfers {
		if !existing[transfer.ID] {
			next = append(next, transfer)
			continue
		}
		if transfer.TurnsRemaining > 0 {
			transfer.TurnsRemaining--
		}
		if transfer.TurnsRemaining > 0 {
			next = append(next, transfer)
			continue
		}
		g.completePopulationTransfer(transfer)
		arrivedOwners[transfer.Owner] = true
	}
	g.PopulationTransfers = next
	for ownerID := range arrivedOwners {
		g.recalculateFreightersUsed(ownerID)
	}
}

func (g *Game) completePopulationTransfer(transfer PopulationTransfer) {
	player, err := g.getPlayer(transfer.Owner)
	if err != nil {
		return
	}
	destination, err := g.getPlanet(transfer.DestinationPlanetID)
	if err != nil || destination.Owner != transfer.Owner {
		return
	}
	hydratePlanetPopulationCapacity(g, destination, player)
	beforeUnits := destination.Population / 1000
	arrivingPopulation := int64(transfer.PopulationUnits) * 1000
	room := destination.MaxPopulation - destination.Population
	if room < 0 {
		room = 0
	}
	landed := arrivingPopulation
	if landed > room {
		landed = room
	}
	overflow := arrivingPopulation - landed
	destination.Population += landed
	afterUnits := destination.Population / 1000
	if afterUnits > beforeUnits {
		assignNewPopulation(g, player, destination, afterUnits-beforeUnits)
	}
	capAssignedPopulation(destination)

	if overflow <= 0 {
		return
	}
	source, err := g.getPlanet(transfer.SourcePlanetID)
	if err != nil || source.Owner != transfer.Owner {
		log.Printf(
			"population transfer %d: %d population lost (no owned source planet %d to refund)",
			transfer.ID, overflow, transfer.SourcePlanetID,
		)
		return
	}
	beforeSourceUnits := source.Population / 1000
	source.Population += overflow
	afterSourceUnits := source.Population / 1000
	if afterSourceUnits > beforeSourceUnits {
		assignNewPopulation(g, player, source, afterSourceUnits-beforeSourceUnits)
	}
	capAssignedPopulation(source)
}

// pendingInboundPopulationUnits sums PopulationUnits already in flight to destination
// for the given owner. Earlier same-submit orders are included once issued/appended.
func (g *Game) pendingInboundPopulationUnits(ownerID, destinationPlanetID uint) uint {
	var sum uint
	for _, transfer := range g.PopulationTransfers {
		if transfer.Owner == ownerID && transfer.DestinationPlanetID == destinationPlanetID {
			sum += transfer.PopulationUnits
		}
	}
	return sum
}
