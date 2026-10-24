package game

import (
	"fmt"
	"math"
	"slices"
	"sort"
	. "ssg/model"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Turnable interface {
	Turn()
}

type TurnStatus struct {
	Mode              string `json:"mode"`
	ReadyPlayerIDs    []uint `json:"ready_player_ids"`
	WaitingPlayerIDs  []uint `json:"waiting_player_ids"`
	RequiredPlayerIDs []uint `json:"required_player_ids"`
}

type Game struct {
	Galaxy                   Galaxy               `json:"galaxy"`
	Players                  []*Player            `json:"players"`
	Fleets                   []*Fleet             `json:"fleets"`
	PopulationTransfers      []PopulationTransfer `json:"population_transfers"`
	ID                       string               `json:"id"`
	CurrentTurn              float64              `json:"current_turn"`
	CreatedAt                time.Time            `json:"created_at"`
	LastTurnAt               *time.Time           `json:"last_turn_at,omitempty"`
	Resources                *Resources           `json:"resources"`
	PendingColonyBases       map[uint][]uint      `json:"pending_colony_bases"`
	PendingColonyShips       map[uint][]uint      `json:"pending_colony_ships"`
	PendingArtificialPlanets map[uint][]uint      `json:"pending_artificial_planets"`
	BattleReports            []*BattleReport      `json:"battle_reports,omitempty"`
	PendingBattles           []*TacticalBattle    `json:"pending_battles,omitempty"`
	InvasionReports          []*InvasionReport    `json:"invasion_reports,omitempty"`
	TurnSubmissions          map[uint]Commands    `json:"-"`
	TurnStatus               TurnStatus           `json:"turn_status"`
	mu                       *sync.Mutex          `json:"-"`
}

type NewGameOptions struct {
	PlayerName  string
	PlayerColor string
	NumStars    uint
	GalaxySize  string
	Race        Race
	NumPlayers  uint
}

func NewGame(resources *ResourceManager, options NewGameOptions) (*Game, error) {
	playerName := options.PlayerName
	playerColor := options.PlayerColor
	numStars := options.NumStars
	numPlayers := options.NumPlayers
	if numPlayers == 0 {
		numPlayers = 1
	}
	if numPlayers > MaxPlayers {
		return nil, fmt.Errorf("players must be between 1 and %d", MaxPlayers)
	}
	if numPlayers > numStars {
		return nil, fmt.Errorf("players (%d) cannot exceed stars (%d)", numPlayers, numStars)
	}
	if playerColor == "" {
		playerColor = DefaultPlayerColor()
	}
	if !IsValidPlayerColor(playerColor) {
		return nil, fmt.Errorf("invalid player color %q", playerColor)
	}
	race := options.Race
	if race.Name == "" {
		race = DefaultRace("Human")
	}
	if race.Government == "" {
		race.Government = defaultGovernmentName(&resources.Resources)
	}

	firstPlayerID := uint(1)
	players := make([]*Player, 0, numPlayers)
	player1 := NewPlayer(firstPlayerID, playerName, playerColor, race)
	player1.IsHuman = true
	players = append(players, player1)

	// Build the pool of colors still available for non-interactive slots by
	// removing player 1's choice from the full list, then shuffling so each
	// new game looks visually distinct.
	remaining := make([]string, 0, len(PlayerColors))
	for _, c := range PlayerColors {
		if c != playerColor {
			remaining = append(remaining, c)
		}
	}
	randomShuffle(len(remaining), func(i, j int) { remaining[i], remaining[j] = remaining[j], remaining[i] })
	if uint(len(remaining)) < numPlayers-1 {
		return nil, fmt.Errorf("not enough player colors for %d players", numPlayers)
	}
	for slot := uint(2); slot <= numPlayers; slot++ {
		color := remaining[slot-2]
		players = append(players, NewPlayer(slot, fmt.Sprintf("Player %d", slot), color, DefaultRace("Human")))
	}

	created := time.Now().UTC()
	g := &Game{
		Galaxy:                   NewGalaxyForSize(numStars, options.GalaxySize),
		Players:                  players,
		Fleets:                   []*Fleet{},
		PopulationTransfers:      []PopulationTransfer{},
		ID:                       uuid.New().String(),
		CurrentTurn:              3500.0,
		CreatedAt:                created,
		Resources:                &resources.Resources,
		PendingColonyBases:       make(map[uint][]uint),
		PendingColonyShips:       make(map[uint][]uint),
		PendingArtificialPlanets: make(map[uint][]uint),
		TurnSubmissions:          make(map[uint]Commands),
		mu:                       &sync.Mutex{},
	}
	for _, player := range g.Players {
		g.hydratePlayerAvailability(player)
	}
	g.resetTurnStatus()

	startingStars := startingStarIDs(numPlayers, numStars)
	for index, player := range g.Players {
		starID := startingStars[index]
		if err := g.CreateCapitalPlanetAtStar(player.ID, starID); err != nil {
			return nil, err
		}
		if err := g.ExploreStar(player.ID, starID); err != nil {
			return nil, err
		}
	}

	return g, nil
}

func startingStarIDs(numPlayers, numStars uint) []uint {
	if numPlayers == 0 {
		return []uint{}
	}
	ids := make([]uint, 0, numPlayers)
	used := make(map[uint]bool, numPlayers)
	for slot := uint(1); slot <= numPlayers; slot++ {
		starID := uint(1)
		if slot > 1 {
			starID = uint(math.Round(float64(slot-1)*float64(numStars)/float64(numPlayers))) + 1
			if starID > numStars {
				starID = numStars
			}
		}
		for used[starID] {
			starID++
			if starID > numStars {
				starID = 1
			}
		}
		used[starID] = true
		ids = append(ids, starID)
	}
	return ids
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
	_, _, err := g.SubmitTurn(playerID, commands)
	return err
}

// validatePlayerCommands checks that all commands are valid without mutating game state.
// This must be called before any state mutations to ensure rejected turns don't change the game.
func (g *Game) validatePlayerCommands(playerID uint, commands Commands) error {
	player, err := g.getPlayer(playerID)
	if err != nil {
		return err
	}

	// Validate fleet move commands
	if err := g.validateFleetMoveCommands(playerID, commands.FleetMove); err != nil {
		return err
	}

	// Validate invade commands
	if err := g.validateInvadeCommands(playerID, commands.Invade); err != nil {
		return err
	}

	// Validate population transfer commands
	if err := g.validatePopulationTransferCommands(playerID, commands.TransferPopulation); err != nil {
		return err
	}

	// Validate colony ship preparation
	if commands.ColonyShip != nil {
		if err := g.validatePrepareColonyShip(playerID, commands.ColonyShip.FleetID); err != nil {
			return err
		}
	}

	// Validate colonize commands
	colonizeCommands := commands.ColonizeMany
	if commands.Colonize != nil {
		colonizeCommands = append(colonizeCommands, commands.Colonize)
	}
	for _, command := range colonizeCommands {
		if err := g.validateColonizeCommand(playerID, command); err != nil {
			return err
		}
	}

	// Validate artificial planet commands
	artificialPlanetCommands := commands.ArtificialPlanetMany
	if commands.ArtificialPlanet != nil {
		artificialPlanetCommands = append(artificialPlanetCommands, commands.ArtificialPlanet)
	}
	for _, command := range artificialPlanetCommands {
		if err := g.validateArtificialPlanetCommand(playerID, command); err != nil {
			return err
		}
	}

	// Validate planet commands (buy production, move population)
	colonyList, err := g.GetColonyList(playerID)
	if err != nil {
		return err
	}
	for _, colony := range colonyList {
		// Validate buy production commands
		for _, command := range commands.BuyProduction {
			if command == nil || command.PlanetID != colony.ID {
				continue
			}
			if err := g.validateBuyProduction(colony, player); err != nil {
				return err
			}
			break
		}

		// Validate move population commands
		if err := validateMovePopulationCommands(colony, commands.MovePopulation); err != nil {
			return err
		}
	}

	return nil
}

func (g *Game) SubmitTurn(playerID uint, commands Commands) (TurnStatus, bool, error) {
	if g.mu == nil {
		g.mu = &sync.Mutex{}
	}
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, err := g.getPlayer(playerID); err != nil {
		return g.currentTurnStatus(), false, err
	}
	for _, battle := range g.PendingBattles {
		if battle != nil && battle.Status == TacticalBattleActive && battle.IsHumanOwner(playerID) {
			return g.currentTurnStatus(), false, fmt.Errorf("resolve pending battle %d at star %d before ending the turn", battle.ID, battle.StarID)
		}
	}

	// Validate commands BEFORE accepting the submission
	if err := g.validatePlayerCommands(playerID, commands); err != nil {
		return g.currentTurnStatus(), false, err
	}

	if g.TurnSubmissions == nil {
		g.TurnSubmissions = make(map[uint]Commands)
	}
	g.TurnSubmissions[playerID] = commands
	status := g.calculateTurnStatus()
	g.TurnStatus = status
	if len(status.WaitingPlayerIDs) > 0 {
		return status, false, nil
	}

	submissions := make(map[uint]Commands, len(g.TurnSubmissions))
	for id, submitted := range g.TurnSubmissions {
		submissions[id] = submitted
	}
	if err := g.AdvanceTurn(submissions); err != nil {
		return g.currentTurnStatus(), false, err
	}
	return g.TurnStatus, true, nil
}

func (g *Game) AdvanceTurn(submissions map[uint]Commands) error {
	// A pending battle should have been fought via the battle API before any
	// turn submission, but never let a stale one wedge the game.
	g.autoResolveLeftoverBattles()
	existingPopulationTransfers := g.populationTransferIDs()
	playerIDs := g.playerIDs()
	for _, playerID := range playerIDs {
		commands, ok := submissions[playerID]
		if !ok {
			return fmt.Errorf("missing turn submission for player %d", playerID)
		}
		if err := g.applyPlayerTurn(playerID, commands); err != nil {
			return err
		}
	}

	g.advanceFleetMovements()
	g.resolveStarBattles()
	g.resolveGroundInvasions()
	g.advancePopulationTransfers(existingPopulationTransfers)
	g.HydrateCommandPoints()
	g.CurrentTurn = advanceStardate(g.CurrentTurn)
	now := time.Now().UTC()
	g.LastTurnAt = &now
	g.TurnSubmissions = make(map[uint]Commands)
	g.resetTurnStatus()
	return nil
}

func (g *Game) applyPlayerTurn(playerID uint, commands Commands) error {
	player, err := g.getPlayer(playerID)
	if err != nil {
		return err
	}

	if commands.SelectResearch != nil {
		if err := g.selectResearch(player, *commands.SelectResearch); err != nil {
			return err
		}
	}

	if commands.MutateRace != nil {
		if err := g.ApplyRaceMutation(playerID, *commands.MutateRace); err != nil {
			return err
		}
	}
	if err := g.issueFleetMoveCommands(playerID, commands.FleetMove); err != nil {
		return err
	}
	if err := g.issueInvadeCommands(playerID, commands.Invade); err != nil {
		return err
	}
	if err := g.issuePopulationTransferCommands(playerID, commands.TransferPopulation); err != nil {
		return err
	}
	if commands.ColonyShip != nil {
		if err := g.prepareColonyShip(playerID, commands.ColonyShip.FleetID); err != nil {
			return err
		}
	}

	// colonies turn processing
	colonyList, err := g.GetColonyList(playerID)
	if err != nil {
		return err
	}
	foodTransfers := g.calculateAutomaticFoodTransfers(player, colonyList, commands)
	totalIncome := int64(0)
	totalResearch := int64(0)
	for _, colony := range colonyList {
		if err := planetTurn(colony, commands, player, g, foodTransfers.PlanetReceived[colony.ID]); err != nil {
			return err
		}
		totalIncome += colony.Income.Total
		totalResearch += GamePlanetScienceProduced(g, colony)
	}

	colonizeCommands := commands.ColonizeMany
	if commands.Colonize != nil {
		colonizeCommands = append(colonizeCommands, commands.Colonize)
	}
	for _, command := range colonizeCommands {
		if err := g.processColonizeCommand(playerID, command); err != nil {
			return err
		}
	}
	artificialPlanetCommands := commands.ArtificialPlanetMany
	if commands.ArtificialPlanet != nil {
		artificialPlanetCommands = append(artificialPlanetCommands, commands.ArtificialPlanet)
	}
	for _, command := range artificialPlanetCommands {
		if err := g.processArtificialPlanetCommand(playerID, command); err != nil {
			return err
		}
	}
	if len(g.PendingColonyBases[playerID]) > 0 && !g.hasColonizablePlanetsInSourceSystems(g.PendingColonyBases[playerID], playerID) {
		delete(g.PendingColonyBases, playerID)
	}
	if len(g.PendingColonyShips[playerID]) > 0 && !g.hasColonizablePlanetsForPendingShips(g.PendingColonyShips[playerID], playerID) {
		delete(g.PendingColonyShips, playerID)
	}
	g.prunePendingArtificialPlanets(playerID)

	// empire turn processing
	player.Treasury += float64(totalIncome)
	player.Treasury -= foodTransfers.OperatingCostBC
	g.HydrateCommandPoints()
	player.Treasury -= float64(player.CommandOverdrawBC)
	totalResearch += int64(luckyTurnResearchBonus(player))
	if err := g.applyResearch(player, uint64(max(totalResearch, 0))); err != nil {
		return err
	}
	return nil
}

func (g *Game) playerIDs() []uint {
	ids := make([]uint, 0, len(g.Players))
	for _, player := range g.Players {
		ids = append(ids, player.ID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func (g *Game) calculateTurnStatus() TurnStatus {
	required := g.playerIDs()
	ready := make([]uint, 0, len(g.TurnSubmissions))
	waiting := make([]uint, 0, len(required))
	for _, playerID := range required {
		if _, ok := g.TurnSubmissions[playerID]; ok {
			ready = append(ready, playerID)
		} else {
			waiting = append(waiting, playerID)
		}
	}
	return TurnStatus{
		Mode:              "simultaneous",
		ReadyPlayerIDs:    ready,
		WaitingPlayerIDs:  waiting,
		RequiredPlayerIDs: required,
	}
}

func (g *Game) currentTurnStatus() TurnStatus {
	if g.TurnStatus.Mode == "" {
		return g.calculateTurnStatus()
	}
	return g.TurnStatus
}

func (g *Game) resetTurnStatus() {
	required := g.playerIDs()
	g.TurnStatus = TurnStatus{
		Mode:              "simultaneous",
		ReadyPlayerIDs:    []uint{},
		WaitingPlayerIDs:  required,
		RequiredPlayerIDs: required,
	}
}

func advanceStardate(stardate float64) float64 {
	return math.Round((stardate+0.1)*10) / 10
}

// Validation helper functions

func (g *Game) validateFleetMoveCommands(playerID uint, commands []*FleetMoveCommand) error {
	for _, command := range commands {
		if command == nil {
			continue
		}
		fleet, err := g.getFleet(command.FleetID)
		if err != nil {
			return err
		}
		if fleet.Owner != playerID {
			return fmt.Errorf("fleet %d does not belong to player %d", fleet.ID, playerID)
		}
		// Additional range/validity checks would go here
	}
	return nil
}

func (g *Game) validateInvadeCommands(playerID uint, commands []*InvadeCommand) error {
	for _, command := range commands {
		if command == nil {
			continue
		}
		fleet, err := g.getFleet(command.FleetID)
		if err != nil {
			return err
		}
		if fleet.Owner != playerID {
			return fmt.Errorf("fleet %d does not belong to player %d", fleet.ID, playerID)
		}
	}
	return nil
}

func (g *Game) validatePopulationTransferCommands(playerID uint, commands []*TransferPopulationCommand) error {
	for _, command := range commands {
		if command == nil {
			continue
		}
		source, err := g.getPlanet(command.SourcePlanetID)
		if err != nil {
			return err
		}
		if source.Owner != playerID {
			return fmt.Errorf("source planet %d does not belong to player %d", command.SourcePlanetID, playerID)
		}
		dest, err := g.getPlanet(command.DestinationPlanetID)
		if err != nil {
			return err
		}
		if dest.Owner != playerID {
			return fmt.Errorf("destination planet %d does not belong to player %d", command.DestinationPlanetID, playerID)
		}
	}
	return nil
}

func (g *Game) validatePrepareColonyShip(playerID uint, fleetID uint) error {
	fleet, err := g.getFleet(fleetID)
	if err != nil {
		return err
	}
	if fleet.Owner != playerID {
		return fmt.Errorf("fleet %d does not belong to player %d", fleetID, playerID)
	}
	hasColonyShip := false
	for _, ship := range fleet.Ships {
		if ship.Type == "colony_ship" {
			hasColonyShip = true
			break
		}
	}
	if !hasColonyShip {
		return fmt.Errorf("fleet %d has no colony ship", fleetID)
	}
	return nil
}

func (g *Game) validateColonizeCommand(playerID uint, command *ColonizeCommand) error {
	if command == nil {
		return nil
	}
	if command.FleetID != 0 {
		// Validate colony ship exists and belongs to player
		// Note: if fleet ID is stale, the actual colonize will fall back to finding
		// any colony ship in the target system, so we allow that case here too
		fleet, err := g.getFleet(command.FleetID)
		if err != nil {
			// Fleet not found - check if there's a valid target planet and colony ship in its system
			target, err := g.getPlanet(command.PlanetID)
			if err != nil {
				return err
			}
			fleet = g.findOrbitColonyShipFleet(playerID, target.ParentStar)
			if fleet == nil {
				return fmt.Errorf("no orbiting colony ship in target system")
			}
			// Found a valid fallback fleet, validation passes
			return nil
		}
		if fleet.Owner != playerID {
			return fmt.Errorf("fleet %d does not belong to player %d", fleet.ID, playerID)
		}
		hasColonyShip := false
		for _, ship := range fleet.Ships {
			if ship.Type == "colony_ship" {
				hasColonyShip = true
				break
			}
		}
		if !hasColonyShip {
			return fmt.Errorf("fleet %d has no colony ship", fleet.ID)
		}
		return nil
	}
	// Validate pending colony base exists
	if g.PendingColonyBases == nil {
		return fmt.Errorf("no pending colony base")
	}
	pending := g.PendingColonyBases[playerID]
	if len(pending) == 0 {
		return fmt.Errorf("no pending colony base")
	}
	return nil
}

func (g *Game) validateArtificialPlanetCommand(playerID uint, command *ArtificialPlanetCommand) error {
	if command == nil {
		return nil
	}
	if g.PendingArtificialPlanets == nil {
		return fmt.Errorf("no pending artificial planet construction")
	}
	pending := g.PendingArtificialPlanets[playerID]
	if len(pending) == 0 {
		return fmt.Errorf("no pending artificial planet construction")
	}
	return nil
}

func (g *Game) validateBuyProduction(p *Planet, player *Player) error {
	if isSpecialNonBuildingProject(p.CurrentProject) {
		return fmt.Errorf("cannot buy %s", p.CurrentProject)
	}
	if !p.IsBuilding || p.CurrentProject == "" {
		return fmt.Errorf("cannot buy production: planet %d is not building", p.ID)
	}
	effectiveCost, ok := effectiveShipDesignCost(p, p.CurrentProject, g)
	if !ok {
		building, err := getBuildingFromName(g, p.CurrentProject)
		if err != nil {
			return fmt.Errorf("cannot buy production: unknown project %q on planet %d", p.CurrentProject, p.ID)
		}
		effectiveCost = effectiveBuildingCost(p, building, g)
	}
	if effectiveCost <= 0 {
		return nil
	}
	remaining := max(0, p.BuildBucket-p.IndustryStack)
	if remaining <= 0 {
		return nil
	}
	cost := BuyProductionCost(effectiveCost, remaining)
	if player.Treasury < float64(cost) {
		return fmt.Errorf("cannot buy production: need %d BC, have %.1f BC", cost, player.Treasury)
	}
	return nil
}

func validateMovePopulationCommands(p *Planet, commands []*MovePopulationCommand) error {
	for _, movePopulation := range commands {
		if movePopulation.PlanetID != p.ID {
			continue
		}
		if err := validateMovePopulationCommand(movePopulation, p); err != nil {
			return err
		}
		break
	}
	return nil
}

// validateMovePopulationCommand checks that a single movePopulation command is valid.
// This is called at submit time and defensively at apply time.
func validateMovePopulationCommand(cmd *MovePopulationCommand, p *Planet) error {
	if cmd.Farmers < 0 {
		return fmt.Errorf("bad pop movement: farmers cannot be negative")
	}
	if cmd.Workers < 0 {
		return fmt.Errorf("bad pop movement: workers cannot be negative")
	}
	if cmd.Scientists < 0 {
		return fmt.Errorf("bad pop movement: scientists cannot be negative")
	}
	newTotal := cmd.Farmers + cmd.Workers + cmd.Scientists
	if newTotal > p.Population/1000 {
		return fmt.Errorf("bad pop movement: too many units")
	}
	if cmd.Farmers > 0 && !farmingPossible(p) {
		return fmt.Errorf("bad pop movement: farming is not possible on this planet")
	}
	if p.Owner == 0 {
		return fmt.Errorf("bad pop movement: planet %d is not colonized", p.ID)
	}
	return nil
}

func (g *Game) processColonizeCommand(playerID uint, command *ColonizeCommand) error {
	if command == nil {
		return nil
	}
	if command.FleetID != 0 {
		return g.ColonizePlanetWithShip(playerID, command.FleetID, command.PlanetID)
	}
	if g.PendingColonyBases == nil {
		g.PendingColonyBases = make(map[uint][]uint)
	}
	pending := g.PendingColonyBases[playerID]
	if len(pending) == 0 {
		return fmt.Errorf("no pending colony base")
	}
	sourcePlanetID := pending[len(pending)-1]
	g.PendingColonyBases[playerID] = pending[:len(pending)-1]
	return g.ColonizePlanet(playerID, sourcePlanetID, command.PlanetID)
}

func (g *Game) CreateCapitalPlanet(ownerID uint) error {
	return g.CreateCapitalPlanetAtStar(ownerID, 1)
}

func (g *Game) CreateCapitalPlanetAtStar(ownerID, starID uint) error {
	player, err := g.getPlayer(ownerID)
	if err != nil {
		return err
	}

	star, err := g.getStar(starID)
	if err != nil {
		return err
	}
	if len(star.Planets) == 0 {
		return fmt.Errorf("star %d has no planets", starID)
	}
	planet := star.Planets[0]

	planet.Owner = ownerID

	// default capital terran world
	planet.Environment = PlanetEnvironmentTerran
	planet.Size = PlanetSizeMedium
	planet.MineralClass = PlanetMineralAbundant
	planet.Gravity = PlanetGravityNormal
	applyRaceHomeworldTraits(player, planet)

	// population setup
	planet.Population = 8000
	planet.Food.Farmers = 4
	planet.Industry.Workers = 3
	planet.Science.Scientists = 1

	buildOnPlanet(g, planet, "Capitol")
	buildOnPlanet(g, planet, "Star Base")
	buildOnPlanet(g, planet, "Marine Barracks")

	hydratePlanetGovernment(g, planet, player)
	hydratePlanetPopulationCapacity(g, planet, player)
	hydratePlanetMorale(g, planet, player)
	hydratePlanetOutputs(g, planet)
	hydratePlanetIncome(g, planet, 0)

	g.HydrateCommandPoints()

	return nil
}

func applyRaceHomeworldTraits(player *Player, planet *Planet) {
	if playerHasRaceTrait(player, "large_homeworld") {
		planet.Size = PlanetSizeLarge
	}
	if playerHasRaceTrait(player, "rich_homeworld") {
		planet.MineralClass = PlanetMineralRich
	}
	if playerHasRaceTrait(player, "poor_homeworld") {
		planet.MineralClass = PlanetMineralPoor
	}
	if playerHasRaceTrait(player, "artifacts_homeworld") {
		planet.HomeworldScienceBonus = 2
	}
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

	target.CurrentProject = tradeGoodsProject
	target.IsBuilding = false
	target.BuildBucket = 0

	return nil
}

func (g *Game) prepareColonyShip(ownerID, fleetID uint) error {
	fleet, err := g.getFleet(fleetID)
	if err != nil {
		return err
	}
	if fleet.Owner != ownerID {
		return fmt.Errorf("fleet %d does not belong to player %d", fleet.ID, ownerID)
	}
	if fleet.DestinationStarID != 0 || fleet.TurnsRemaining != 0 {
		return fmt.Errorf("colony ship must be in orbit")
	}
	if !fleetHasColonyShip(fleet) {
		return fmt.Errorf("fleet has no colony ship")
	}
	if !g.hasColonizablePlanetsInStar(fleet.StarID) {
		return nil
	}
	if g.PendingColonyShips == nil {
		g.PendingColonyShips = make(map[uint][]uint)
	}
	if slices.Contains(g.PendingColonyShips[ownerID], fleetID) {
		return nil
	}
	g.PendingColonyShips[ownerID] = append(g.PendingColonyShips[ownerID], fleetID)
	return nil
}

func (g *Game) ColonizePlanetWithShip(ownerID, fleetID, targetPlanetID uint) error {
	if g.PendingColonyShips == nil {
		g.PendingColonyShips = make(map[uint][]uint)
	}
	target, err := g.getPlanet(targetPlanetID)
	if err != nil {
		return err
	}
	pending := g.PendingColonyShips[ownerID]
	found := false
	nextPending := make([]uint, 0, len(pending))
	for _, pendingFleetID := range pending {
		if pendingFleetID == fleetID && !found {
			found = true
			continue
		}
		nextPending = append(nextPending, pendingFleetID)
	}

	fleet, err := g.getFleet(fleetID)
	if err != nil {
		fleet = g.findOrbitColonyShipFleet(ownerID, target.ParentStar)
		if fleet == nil {
			return fmt.Errorf("no orbiting colony ship in target system")
		}
	}
	if fleet.Owner != ownerID {
		return fmt.Errorf("fleet %d does not belong to player %d", fleet.ID, ownerID)
	}
	if fleet.DestinationStarID != 0 || fleet.TurnsRemaining != 0 {
		return fmt.Errorf("colony ship must be in orbit")
	}
	if !fleetHasColonyShip(fleet) {
		return fmt.Errorf("fleet has no colony ship")
	}

	if target.ParentStar != fleet.StarID {
		return fmt.Errorf("colony ship can only colonize planets in the same system")
	}
	if err := g.colonizePlanet(ownerID, target); err != nil {
		return err
	}
	removeFirstColonyShip(fleet)
	if len(fleet.Ships) == 0 {
		g.removeFleet(fleet)
	}
	if found {
		g.PendingColonyShips[ownerID] = nextPending
	}
	return nil
}

func (g *Game) findOrbitColonyShipFleet(ownerID, starID uint) *Fleet {
	for _, fleet := range g.Fleets {
		if fleet.Owner == ownerID &&
			fleet.StarID == starID &&
			fleet.DestinationStarID == 0 &&
			fleet.TurnsRemaining == 0 &&
			fleetHasColonyShip(fleet) {
			return fleet
		}
	}
	return nil
}

func (g *Game) colonizePlanet(ownerID uint, target *Planet) error {
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

	target.CurrentProject = tradeGoodsProject
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

func (g *Game) hasColonizablePlanetsForPendingShips(fleetIDs []uint, ownerID uint) bool {
	for _, fleetID := range fleetIDs {
		fleet, err := g.getFleet(fleetID)
		if err != nil || fleet.Owner != ownerID || !fleetHasColonyShip(fleet) {
			continue
		}
		if g.hasColonizablePlanetsInStar(fleet.StarID) {
			return true
		}
	}
	return false
}

func (g *Game) hasColonizablePlanetsInStar(starID uint) bool {
	star, err := g.getStar(starID)
	if err != nil {
		return false
	}
	for _, planet := range star.Planets {
		if planet.Owner == 0 && planet.Environment != PlanetEnvironmentGasGiant && planet.Environment != PlanetEnvironmentAsteroidBelt {
			return true
		}
	}
	return false
}

func fleetHasColonyShip(fleet *Fleet) bool {
	for _, ship := range fleet.Ships {
		if ship.Type == "colony_ship" {
			return true
		}
	}
	return false
}

func removeFirstColonyShip(fleet *Fleet) {
	for i, ship := range fleet.Ships {
		if ship.Type == "colony_ship" {
			fleet.Ships = append(fleet.Ships[:i], fleet.Ships[i+1:]...)
			return
		}
	}
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
				hydratePlanetGovernment(g, planet, player)
				hydratePlanetMorale(g, planet, player)
				hydratePlanetOutputs(g, planet)
				hydratePlanetPopulationCapacity(g, planet, player)
				hydratePlanetIncome(g, planet, GamePlanetIndustryProduced(g, planet))
				colonyList = append(colonyList, planet)
			}
		}
	}
	return colonyList, nil
}

func (g *Game) SetPlanetAutobuild(playerID, planetID uint, enabled bool) error {
	planet, err := g.getPlanet(planetID)
	if err != nil {
		return err
	}
	if planet.Owner != playerID {
		return fmt.Errorf("planet %d is not owned by player %d", planetID, playerID)
	}
	planet.Autobuild = enabled
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

	parts := g.Resources.ShipPartsCollection
	player, err := g.getPlayer(playerID)
	if err != nil {
		return err
	}

	armor, err := ArmorFromName(armorName, parts.Armors)
	if err != nil {
		return err
	}
	drive, err := DriveFromName(driveName, parts.Drives)
	if err != nil {
		return err
	}
	hullSize, err := HullSizeFromName(hullSizeName, parts.HullSizes)
	if err != nil {
		return err
	}
	computer, err := ComputerFromName(computerName, parts.Computers)
	if err != nil {
		return err
	}
	fuelCell, err := FuelCellFromName(fuelCellName, parts.FuelCells)
	if err != nil {
		return err
	}
	shield, err := ShieldFromName(shieldName, parts.Shields)
	if err != nil {
		return err
	}

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

	if err := validatePlayerCanUseShipDesignParts(player, basicDesign); err != nil {
		return err
	}
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

	armor, err := ArmorFromName(armorName, parts.Armors)
	if err != nil {
		return err
	}
	drive, err := DriveFromName(driveName, parts.Drives)
	if err != nil {
		return err
	}
	hullSize, err := HullSizeFromName(hullSizeName, parts.HullSizes)
	if err != nil {
		return err
	}
	computer, err := ComputerFromName(computerName, parts.Computers)
	if err != nil {
		return err
	}
	fuelCell, err := FuelCellFromName(fuelCellName, parts.FuelCells)
	if err != nil {
		return err
	}
	shield, err := ShieldFromName(shieldName, parts.Shields)
	if err != nil {
		return err
	}

	// get player reference
	player, err := g.getPlayer(playerID)
	if err != nil {
		return err
	}

	updatedDesign := NewShipDesign(
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
	if err := validatePlayerCanUseShipDesignParts(player, updatedDesign); err != nil {
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
