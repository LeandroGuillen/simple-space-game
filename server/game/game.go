package game

import (
	"errors"
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

type moveBatch struct {
	destinationStarID uint
	indexes           []uint
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
	// PendingInvasions holds Invade orders accepted this turn until
	// resolveGroundInvasions runs. Turn-ephemeral: cleared on hydrate and
	// at the start of AdvanceTurn; a failed AdvanceTurn restores the
	// pre-advance queue via snapshot. Omitted/empty in older saves.
	PendingInvasions []PendingInvasion `json:"pending_invasions,omitempty"`
	// PostponedInvasions holds Invade orders deferred because a tactical
	// battle was still pending at the target star. Unlike PendingInvasions,
	// these persist across saves while a matching pending battle remains
	// (see HydrateLoadedGame / ValidateLoadedGame). Cleared when the battle
	// finishes (resolve or cancel). Omitted/empty in older saves.
	PostponedInvasions []PendingInvasion `json:"postponed_invasions,omitempty"`
	TurnSubmissions    map[uint]Commands `json:"-"`
	TurnStatus         TurnStatus        `json:"turn_status"`
	// Debug is set when a debug-only endpoint mutates the game (e.g. research-all).
	// Omitted/false in older saves.
	Debug bool          `json:"debug,omitempty"`
	mu    *sync.RWMutex `json:"-"`
}

// PendingInvasion is a validated Invade order waiting for ground resolution.
type PendingInvasion struct {
	OwnerID        uint    `json:"owner_id"`
	FleetID        uint    `json:"fleet_id"`
	TargetPlanetID uint    `json:"target_planet_id"`
	ShipIndexes    *[]uint `json:"ship_indexes,omitempty"`
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
		mu:                       &sync.RWMutex{},
	}
	for _, player := range g.Players {
		g.hydratePlayerAvailability(player)
	}
	g.resetTurnStatus()

	startingStars := SelectHomeStarIDs(g.Galaxy.Stars, numPlayers)
	for index, player := range g.Players {
		starID := startingStars[index]
		if err := g.CreateCapitalPlanetAtStar(player.ID, starID); err != nil {
			return nil, err
		}
		// Construction is single-threaded; use the unlocked helper so we do not
		// take g.mu before the game is published to concurrent readers.
		if err := g.exploreStarUnlocked(player.ID, starID); err != nil {
			return nil, err
		}
	}

	return g, nil
}

// StartingStarIDs is the legacy count-only stride over star IDs. This is the
// default home pick; SelectHomeStarIDs returns these IDs whenever they already
// keep homes more than one parsec apart (or the map has no room to do better).
func StartingStarIDs(numPlayers, numStars uint) []uint {
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

const homeMinSeparationPixels = float64(parsecInPixels) // more than 1 parsec → dist > 60

type homeCand struct {
	id   uint
	x, y float64
}

// SelectHomeStarIDs returns the legacy StartingStarIDs stride by default so
// game creation matches pre-farthest-point home IDs whenever that pick is fine.
// It falls back to greedy farthest-point sampling ONLY when the stride would
// place two homes within one parsec (≤60px) AND the map has room to avoid it
// (a farthest-point pick among the stars can keep every pair > 60px).
func SelectHomeStarIDs(stars []*Star, numPlayers uint) []uint {
	if numPlayers == 0 {
		return []uint{}
	}
	cands := make([]homeCand, 0, len(stars))
	byID := make(map[uint]homeCand, len(stars))
	for _, s := range stars {
		if s == nil {
			continue
		}
		c := homeCand{id: s.ID, x: float64(s.X), y: float64(s.Y)}
		cands = append(cands, c)
		byID[s.ID] = c
	}
	if len(cands) == 0 {
		return []uint{}
	}

	stride := StartingStarIDs(numPlayers, uint(len(cands)))
	if !homesWithinOneParsec(stride, byID) {
		return stride
	}
	fallback := farthestPointHomeIDs(cands, int(numPlayers))
	if uint(len(fallback)) < numPlayers || homesWithinOneParsec(fallback, byID) {
		// Map has no room to do better; keep the legacy pick.
		return stride
	}
	return fallback
}

func homesWithinOneParsec(ids []uint, byID map[uint]homeCand) bool {
	positions := make([]homeCand, 0, len(ids))
	for _, id := range ids {
		c, ok := byID[id]
		if !ok {
			continue
		}
		positions = append(positions, c)
	}
	for i := 0; i < len(positions); i++ {
		for j := i + 1; j < len(positions); j++ {
			dx := positions[i].x - positions[j].x
			dy := positions[i].y - positions[j].y
			if math.Sqrt(dx*dx+dy*dy) <= homeMinSeparationPixels+1e-9 {
				return true
			}
		}
	}
	return false
}

func farthestPointHomeIDs(cands []homeCand, n int) []uint {
	if n <= 0 || len(cands) == 0 {
		return nil
	}
	if n > len(cands) {
		n = len(cands)
	}
	// Pin first home to the lowest star ID (legacy stride always starts at 1).
	first := 0
	for i := 1; i < len(cands); i++ {
		if cands[i].id < cands[first].id {
			first = i
		}
	}
	chosenIdx := make([]int, 0, n)
	chosenIdx = append(chosenIdx, first)
	used := make([]bool, len(cands))
	used[first] = true
	for len(chosenIdx) < n {
		pick := -1
		pickMin := -1.0
		for i, c := range cands {
			if used[i] {
				continue
			}
			minD := math.MaxFloat64
			for _, ci := range chosenIdx {
				dx := c.x - cands[ci].x
				dy := c.y - cands[ci].y
				d := math.Sqrt(dx*dx + dy*dy)
				if d < minD {
					minD = d
				}
			}
			if minD > pickMin {
				pickMin = minD
				pick = i
			}
		}
		if pick < 0 {
			break
		}
		used[pick] = true
		chosenIdx = append(chosenIdx, pick)
	}
	ids := make([]uint, len(chosenIdx))
	for i, ci := range chosenIdx {
		ids[i] = cands[ci].id
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

	// Research is applied first in applyPlayerTurn; validate against current player state.
	if commands.SelectResearch != nil {
		if _, err := g.validateSelectResearchOrder(player, *commands.SelectResearch); err != nil {
			return err
		}
	}

	// Mutation runs after research select; select does not change mutation eligibility.
	if commands.MutateRace != nil {
		if _, err := g.validateMutateRaceOrder(player, *commands.MutateRace); err != nil {
			return err
		}
	}

	// Validate fleet move commands
	if err := g.validateFleetMoveCommands(playerID, commands.FleetMove); err != nil {
		return err
	}

	// Validate invade commands (pass fleet moves to check for conflicts)
	if err := g.validateInvadeCommands(playerID, commands.Invade, commands.FleetMove); err != nil {
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

	// Validate colonize commands (fleet moves run before colonize in apply)
	colonizeCommands := commands.ColonizeMany
	if commands.Colonize != nil {
		colonizeCommands = append(colonizeCommands, commands.Colonize)
	}
	for _, command := range colonizeCommands {
		if err := g.validateColonizeCommand(playerID, command, commands.FleetMove); err != nil {
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

	// Validate planet commands (buy production, move population).
	// Write lock already held by SubmitTurn — use unlocked helper.
	colonyList, err := g.colonyListUnlocked(playerID)
	if err != nil {
		return err
	}
	remainingTreasury := player.Treasury
	totalBuyCost := int64(0)
	for _, colony := range colonyList {
		// Turn processing visits each colony once and applies at most one buy
		// command for it, so duplicates must not consume the budget twice.
		for _, command := range commands.BuyProduction {
			if command == nil || command.PlanetID != colony.ID {
				continue
			}

			projected := *colony
			if buildCommand := findPlanetBuildCommand(colony, commands); buildCommand != nil {
				if buildCommand.Building == housingProject {
					projected.CurrentProject = housingProject
					projected.IsBuilding = false
					projected.BuildBucket = 0
				} else if !Build(&projected, buildCommand.Building, player.Available.Buildings, g) {
					projected.CurrentProject = tradeGoodsProject
					projected.IsBuilding = false
					projected.BuildBucket = 0
				}
			}
			_, _, cost, err := quoteBuyProduction(&projected, g)
			if err != nil {
				return err
			}
			if remainingTreasury < float64(cost) {
				return fmt.Errorf("cannot buy production: total purchases need %d BC, have %.1f BC", totalBuyCost+cost, player.Treasury)
			}
			remainingTreasury -= float64(cost)
			totalBuyCost += cost
			break
		}

		// Validate move population commands
		if err := validateMovePopulationCommands(colony, commands.MovePopulation); err != nil {
			return err
		}
	}

	return nil
}

// TurnSubmissionError identifies the player whose stored orders failed to apply.
// SubmitTurn drops that submission after AdvanceTurn restores the world.
type TurnSubmissionError struct {
	PlayerID   uint
	PlayerName string
	Err        error
}

func (e *TurnSubmissionError) Error() string {
	return fmt.Sprintf("player %s (ID %d) must resubmit their turn: %v", e.PlayerName, e.PlayerID, e.Err)
}

func (e *TurnSubmissionError) Unwrap() error { return e.Err }

func (g *Game) SubmitTurn(playerID uint, commands Commands) (TurnStatus, bool, error) {
	g.ensureMu()
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
		// AdvanceTurn restored the world and all stored submissions. Drop the
		// orders that failed, preserving the last submitter's valid orders.
		// Unattributed failures retain the existing caller-retry behavior.
		failedPlayerID := playerID
		var submissionErr *TurnSubmissionError
		if errors.As(err, &submissionErr) {
			failedPlayerID = submissionErr.PlayerID
		}
		delete(g.TurnSubmissions, failedPlayerID)
		g.TurnStatus = g.calculateTurnStatus()
		return g.TurnStatus, false, err
	}
	return g.TurnStatus, true, nil
}

// advanceTurnTestHook, when non-nil, is called at named stages inside
// AdvanceTurn so tests can inject failures. Production keeps it nil (zero cost).
var advanceTurnTestHook func(stage string) error

func (g *Game) AdvanceTurn(submissions map[uint]Commands) error {
	// Snapshot-then-restore: on any error the live *Game is rewritten from the
	// pre-advance deep copy so callers (and external pointers to this Game)
	// see the prior world. Resources and mu are shared, not cloned.
	snap := snapshotGameState(g)
	if err := g.advanceTurnApplying(submissions); err != nil {
		restoreGameState(g, snap)
		return err
	}
	return nil
}

func (g *Game) advanceTurnApplying(submissions map[uint]Commands) error {
	// A pending battle should have been fought via the battle API before any
	// turn submission, but never let a stale one wedge the game.
	g.autoResolveLeftoverBattles() // also releases PostponedInvasions via finishTacticalBattle
	if advanceTurnTestHook != nil {
		if err := advanceTurnTestHook("after_auto_resolve"); err != nil {
			return err
		}
	}
	// Drop any leftover PendingInvasions from a prior incomplete turn (or a
	// bad save). Orders for THIS turn are recorded below during applyPlayerTurn,
	// then consumed by resolveGroundInvasions. PostponedInvasions are NOT
	// cleared here — they persist with pending battles. On failure the outer
	// AdvanceTurn restore puts both invasion queues (and battles) back.
	g.PendingInvasions = nil
	existingPopulationTransfers := g.populationTransferIDs()
	playerIDs := g.playerIDs()
	for _, playerID := range playerIDs {
		commands, ok := submissions[playerID]
		if !ok {
			return fmt.Errorf("missing turn submission for player %d", playerID)
		}
		if err := g.applyPlayerTurn(playerID, commands); err != nil {
			player, _ := g.getPlayer(playerID) // playerIDs contains existing players
			return &TurnSubmissionError{PlayerID: playerID, PlayerName: player.Name, Err: err}
		}
		if advanceTurnTestHook != nil {
			if err := advanceTurnTestHook(fmt.Sprintf("after_apply:%d", playerID)); err != nil {
				return err
			}
		}
	}

	if advanceTurnTestHook != nil {
		if err := advanceTurnTestHook("before_fleet_movements"); err != nil {
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

	// colonies turn processing (write lock already held by SubmitTurn)
	colonyList, err := g.colonyListUnlocked(playerID)
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

// validateInvadeOrder performs read-only validation of an invade command.
// Returns the validated fleet and target planet, or an error.
func (g *Game) validateInvadeOrder(playerID uint, command *InvadeCommand) (*Fleet, *Planet, error) {
	if command == nil {
		return nil, nil, nil
	}

	// Validate fleet ownership
	fleet, err := g.getFleet(command.FleetID)
	if err != nil {
		return nil, nil, err
	}
	if fleet.Owner != playerID {
		return nil, nil, fmt.Errorf("fleet %d does not belong to player %d", fleet.ID, playerID)
	}

	// Validate fleet is not in transit
	if fleet.TurnsRemaining > 0 || fleet.DestinationStarID != 0 {
		return nil, nil, fmt.Errorf("fleet %d is in transit and cannot invade", fleet.ID)
	}

	// Validate target planet exists
	planet, err := g.getPlanet(command.TargetPlanetID)
	if err != nil {
		return nil, nil, err
	}

	// Validate planet is at the same star as fleet
	if planet.ParentStar != fleet.StarID {
		return nil, nil, fmt.Errorf("planet %d is not at the same star as fleet %d", planet.ID, fleet.ID)
	}

	// Validate planet is owned by another player (not empty, not same owner)
	if planet.Owner == 0 {
		return nil, nil, fmt.Errorf("cannot invade uncolonized planet %d", planet.ID)
	}
	if planet.Owner == playerID {
		return nil, nil, fmt.Errorf("cannot invade your own planet %d", planet.ID)
	}

	// ShipIndexes semantics:
	//   nil  → all transport ships in the fleet participate
	//   non-nil (including empty) → only the named indexes; empty is rejected
	if command.ShipIndexes != nil {
		if len(*command.ShipIndexes) == 0 {
			return nil, nil, fmt.Errorf("invade ship_indexes must name at least one transport")
		}
		seen := make(map[uint]bool, len(*command.ShipIndexes))
		for _, index := range *command.ShipIndexes {
			if seen[index] {
				return nil, nil, fmt.Errorf("duplicate ship index %d in invade order for fleet %d", index, fleet.ID)
			}
			seen[index] = true
			if int(index) >= len(fleet.Ships) {
				return nil, nil, fmt.Errorf("ship index %d is out of range for fleet %d", index, fleet.ID)
			}
			ship := fleet.Ships[index]
			if ship.Type != "transport" {
				return nil, nil, fmt.Errorf("ship at index %d is not a transport ship", index)
			}
		}
	} else {
		hasTransport := false
		for _, ship := range fleet.Ships {
			if ship.Type == "transport" {
				hasTransport = true
				break
			}
		}
		if !hasTransport {
			return nil, nil, fmt.Errorf("fleet %d has no transport ships for invasion", fleet.ID)
		}
	}

	return fleet, planet, nil
}

// validateSingleFleetMoveOrder performs read-only validation of a single fleet move command.
// Returns the validated fleet, player, ship indexes, and error.
func (g *Game) validateSingleFleetMoveOrder(playerID uint, command *FleetMoveCommand) (*Fleet, *Player, []uint, error) {
	fleet, err := g.getFleet(command.FleetID)
	if err != nil {
		return nil, nil, nil, err
	}
	player, err := g.getPlayer(playerID)
	if err != nil {
		return nil, nil, nil, err
	}

	// Validate ship indexes
	shipIdxs, err := normalizeFleetMoveShipIndexes(command, len(fleet.Ships))
	if err != nil {
		return nil, nil, nil, err
	}

	// Destination cannot be 0
	if command.DestinationStarID == 0 {
		return nil, nil, nil, fmt.Errorf("invalid fleet destination")
	}

	// Destination must exist
	if _, err := g.getStar(command.DestinationStarID); err != nil {
		return nil, nil, nil, err
	}

	// Recall to origin is always allowed
	if command.DestinationStarID == fleet.StarID {
		return fleet, player, shipIdxs, nil
	}

	// Check if fleet is already moving
	inTransit := fleet.TurnsRemaining > 0 && fleet.DestinationStarID != 0

	// Cannot reroute without Hyperspace Communications
	if inTransit && !playerHasTech(player, "Hyperspace Communications") {
		return nil, nil, nil, fmt.Errorf("fleet %d is already moving", fleet.ID)
	}

	// Validate fuel range (skip for Star Gate travel)
	if !g.isStarGateTravel(playerID, fleet.StarID, command.DestinationStarID) {
		if err := g.validateFleetMoveFuelRange(player, shipsByIndexes(fleet.Ships, shipIdxs), command.DestinationStarID); err != nil {
			return nil, nil, nil, err
		}
	}

	return fleet, player, shipIdxs, nil
}

// validateFleetMoveGroupOrders performs read-only validation of grouped fleet move commands.
// Returns the validated fleet, player, and batches of moves, or an error.
func (g *Game) validateFleetMoveGroupOrders(playerID uint, commands []*FleetMoveCommand) (*Fleet, *Player, []moveBatch, error) {
	if len(commands) == 0 {
		return nil, nil, nil, nil
	}

	fleet, err := g.getFleet(commands[0].FleetID)
	if err != nil {
		return nil, nil, nil, err
	}
	player, err := g.getPlayer(playerID)
	if err != nil {
		return nil, nil, nil, err
	}

	inTransit := fleet.TurnsRemaining > 0 && fleet.DestinationStarID != 0

	// If in transit, validate each move individually
	if inTransit {
		for _, command := range commands {
			if _, _, _, err := g.validateSingleFleetMoveOrder(playerID, command); err != nil {
				return nil, nil, nil, err
			}
		}
		return fleet, player, nil, nil
	}

	// Not in transit: validate grouped moves
	batches := make([]moveBatch, 0, len(commands))
	selected := make(map[uint]struct{})

	for _, command := range commands {
		// Destination cannot be 0
		if command.DestinationStarID == 0 {
			return nil, nil, nil, fmt.Errorf("invalid fleet destination")
		}

		// Destination must exist
		if _, err := g.getStar(command.DestinationStarID); err != nil {
			return nil, nil, nil, err
		}

		// Skip recall to origin
		if command.DestinationStarID == fleet.StarID {
			continue
		}

		// Validate ship indexes and check for duplicates
		indexes, err := normalizeFleetMoveShipIndexes(command, len(fleet.Ships))
		if err != nil {
			return nil, nil, nil, err
		}
		for _, index := range indexes {
			if _, exists := selected[index]; exists {
				return nil, nil, nil, fmt.Errorf("ship index %d assigned to multiple fleet moves", index)
			}
			selected[index] = struct{}{}
		}

		// Validate fuel range for this destination (skip for Star Gate travel)
		if !g.isStarGateTravel(playerID, fleet.StarID, command.DestinationStarID) {
			if err := g.validateFleetMoveFuelRange(player, shipsByIndexes(fleet.Ships, indexes), command.DestinationStarID); err != nil {
				return nil, nil, nil, err
			}
		}

		batches = append(batches, moveBatch{destinationStarID: command.DestinationStarID, indexes: indexes})
	}

	return fleet, player, batches, nil
}

func (g *Game) validateSingleFleetMove(playerID uint, command *FleetMoveCommand) error {
	_, _, _, err := g.validateSingleFleetMoveOrder(playerID, command)
	return err
}

func (g *Game) validateFleetMoveGroup(playerID uint, commands []*FleetMoveCommand) error {
	_, _, _, err := g.validateFleetMoveGroupOrders(playerID, commands)
	return err
}

func (g *Game) validateFleetMoveCommands(playerID uint, commands []*FleetMoveCommand) error {
	// Group commands by fleet ID (same structure as issueFleetMoveCommands)
	grouped := make(map[uint][]*FleetMoveCommand)
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
		grouped[command.FleetID] = append(grouped[command.FleetID], command)
	}

	// Validate each fleet's commands
	for _, group := range grouped {
		if len(group) == 1 {
			if err := g.validateSingleFleetMove(playerID, group[0]); err != nil {
				return err
			}
		} else {
			if err := g.validateFleetMoveGroup(playerID, group); err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *Game) validateInvadeCommands(playerID uint, commands []*InvadeCommand, fleetMoves []*FleetMoveCommand) error {
	// Build a set of fleet IDs that are being moved this turn
	movingFleetIDs := make(map[uint]bool)
	for _, moveCmd := range fleetMoves {
		if moveCmd != nil && moveCmd.DestinationStarID != 0 {
			movingFleetIDs[moveCmd.FleetID] = true
		}
	}

	// At most one Invade order per fleet per submit (avoids stale indexes after
	// removeInvasionTransports shifts the fleet). Also reject fleets that already
	// have a recorded pending invasion from earlier this AdvanceTurn.
	seenFleets := make(map[uint]bool)
	for _, pending := range g.PendingInvasions {
		if pending.OwnerID == playerID {
			seenFleets[pending.FleetID] = true
		}
	}

	for _, command := range commands {
		if command == nil {
			continue
		}

		// Check if this fleet is also being moved this turn
		if movingFleetIDs[command.FleetID] {
			return fmt.Errorf("fleet %d cannot invade and move in the same turn", command.FleetID)
		}

		if seenFleets[command.FleetID] {
			return fmt.Errorf("fleet %d already has an invade order this turn", command.FleetID)
		}
		seenFleets[command.FleetID] = true

		// Use shared validator
		_, _, err := g.validateInvadeOrder(playerID, command)
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *Game) validatePopulationTransferCommands(playerID uint, commands []*TransferPopulationCommand) error {
	player, err := g.getPlayer(playerID)
	if err != nil {
		return err
	}
	// Simulated commitments from earlier commands in this same submit (no mutation).
	inboundExtra := map[uint]uint{}
	outboundExtra := map[uint]uint{}
	freightersReserved := uint(0)

	for _, command := range commands {
		if command == nil {
			continue
		}
		if command.PopulationUnits == 0 {
			return fmt.Errorf("population transfer must move at least 1 population unit")
		}
		if command.SourcePlanetID == command.DestinationPlanetID {
			return fmt.Errorf("population transfer source and destination must differ")
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

		sourceUnits := uint(source.Population/1000) - outboundExtra[source.ID]
		if sourceUnits <= command.PopulationUnits {
			return fmt.Errorf("population transfer source must retain at least 1 population unit")
		}

		hydratePlanetPopulationCapacity(g, dest, player)
		destinationUnits := uint(dest.Population / 1000)
		destinationCapacity := uint(dest.MaxPopulation / 1000)
		pendingInbound := g.pendingInboundPopulationUnits(playerID, dest.ID) + inboundExtra[dest.ID]
		if destinationUnits+pendingInbound+command.PopulationUnits > destinationCapacity {
			return fmt.Errorf("population transfer exceeds destination capacity")
		}

		requiredFreighters := command.PopulationUnits * 5
		available := g.availableFreighters(playerID)
		if available < freightersReserved+requiredFreighters {
			return fmt.Errorf("population transfer needs %d freighters", requiredFreighters)
		}

		inboundExtra[dest.ID] += command.PopulationUnits
		outboundExtra[source.ID] += command.PopulationUnits
		freightersReserved += requiredFreighters
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

// validateColonizePlanetTarget checks that a planet can be colonized (unowned, colonizable type).
func validateColonizePlanetTarget(target *Planet) error {
	if target.Owner != 0 {
		return fmt.Errorf("planet already colonized")
	}
	if target.Environment == PlanetEnvironmentGasGiant || target.Environment == PlanetEnvironmentAsteroidBelt {
		return fmt.Errorf("cannot colonize this planet type")
	}
	return nil
}

// validateColonizeWithShipOrder performs all read-only checks for colonizing with a fleet.
// Both SubmitTurn and ColonizePlanetWithShip call this so the rules cannot drift.
// If fleetID is stale/missing, falls back to any orbiting colony ship in the target system
// (matching apply behavior).
func (g *Game) validateColonizeWithShipOrder(ownerID, fleetID, targetPlanetID uint) (*Fleet, *Planet, error) {
	target, err := g.getPlanet(targetPlanetID)
	if err != nil {
		return nil, nil, err
	}
	fleet, err := g.getFleet(fleetID)
	if err != nil {
		fleet = g.findOrbitColonyShipFleet(ownerID, target.ParentStar)
		if fleet == nil {
			return nil, nil, fmt.Errorf("no orbiting colony ship in target system")
		}
	}
	if fleet.Owner != ownerID {
		return nil, nil, fmt.Errorf("fleet %d does not belong to player %d", fleet.ID, ownerID)
	}
	if fleet.DestinationStarID != 0 || fleet.TurnsRemaining != 0 {
		return nil, nil, fmt.Errorf("colony ship must be in orbit")
	}
	if !fleetHasColonyShip(fleet) {
		return nil, nil, fmt.Errorf("fleet has no colony ship")
	}
	if target.ParentStar != fleet.StarID {
		return nil, nil, fmt.Errorf("colony ship can only colonize planets in the same system")
	}
	if err := validateColonizePlanetTarget(target); err != nil {
		return nil, nil, err
	}
	return fleet, target, nil
}

func (g *Game) validateColonizeCommand(playerID uint, command *ColonizeCommand, fleetMoves []*FleetMoveCommand) error {
	if command == nil {
		return nil
	}
	if command.FleetID != 0 {
		fleet, err := g.getFleet(command.FleetID)
		if err == nil {
			// Same-turn move to another star leaves the fleet in transit before colonize runs
			// (fleet moves are applied before colonize). Reject so submit mirrors apply.
			// A recall (destination == current star) is allowed: apply lands the fleet first.
			for _, move := range fleetMoves {
				if move == nil || move.FleetID != command.FleetID {
					continue
				}
				if move.DestinationStarID != 0 && move.DestinationStarID != fleet.StarID {
					return fmt.Errorf("fleet %d cannot colonize and move in the same turn", command.FleetID)
				}
			}
			// Same-turn recall of an in-transit fleet lands it before colonize; treat as in orbit.
			recalling := false
			if fleet.DestinationStarID != 0 || fleet.TurnsRemaining != 0 {
				for _, move := range fleetMoves {
					if move != nil && move.FleetID == command.FleetID && move.DestinationStarID == fleet.StarID {
						recalling = true
						break
					}
				}
			}
			if recalling {
				target, err := g.getPlanet(command.PlanetID)
				if err != nil {
					return err
				}
				if target.ParentStar != fleet.StarID {
					return fmt.Errorf("colony ship can only colonize planets in the same system")
				}
				if !fleetHasColonyShip(fleet) {
					return fmt.Errorf("fleet has no colony ship")
				}
				return validateColonizePlanetTarget(target)
			}
		}
		_, _, err = g.validateColonizeWithShipOrder(playerID, command.FleetID, command.PlanetID)
		return err
	}
	// Colony-base path: pending base must exist; target checked at apply against the source system.
	if g.PendingColonyBases == nil {
		return fmt.Errorf("no pending colony base")
	}
	pending := g.PendingColonyBases[playerID]
	if len(pending) == 0 {
		return fmt.Errorf("no pending colony base")
	}
	target, err := g.getPlanet(command.PlanetID)
	if err != nil {
		return err
	}
	source, err := g.getPlanet(pending[len(pending)-1])
	if err != nil {
		return err
	}
	if target.ParentStar != source.ParentStar {
		return fmt.Errorf("colony base can only colonize planets in the same system")
	}
	return validateColonizePlanetTarget(target)
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

func quoteBuyProduction(p *Planet, game *Game) (effectiveCost, remaining, cost int64, err error) {
	if isSpecialNonBuildingProject(p.CurrentProject) {
		return 0, 0, 0, fmt.Errorf("cannot buy %s", p.CurrentProject)
	}
	if !p.IsBuilding || p.CurrentProject == "" {
		return 0, 0, 0, fmt.Errorf("cannot buy production: planet %d is not building", p.ID)
	}
	effectiveCost, ok := effectiveShipDesignCost(p, p.CurrentProject, game)
	if !ok {
		building, err := getBuildingFromName(game, p.CurrentProject)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("cannot buy production: unknown project %q on planet %d", p.CurrentProject, p.ID)
		}
		effectiveCost = effectiveBuildingCost(p, building, game)
	}
	if effectiveCost <= 0 {
		return effectiveCost, 0, 0, nil
	}
	remaining = max(0, p.BuildBucket-p.IndustryStack)
	return effectiveCost, remaining, BuyProductionCost(effectiveCost, remaining), nil
}

func (g *Game) validateBuyProduction(p *Planet, player *Player) error {
	_, _, cost, err := quoteBuyProduction(p, g)
	if err != nil {
		return err
	}
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
	// Write lock already held by SubmitTurn.
	return g.colonizePlanetUnlocked(playerID, sourcePlanetID, command.PlanetID)
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
	g.ensureMu()
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.colonizePlanetUnlocked(ownerID, sourcePlanetID, targetPlanetID)
}

// colonizePlanetUnlocked applies a colony-base colonization. Caller must hold
// the write lock (or be single-threaded construction/tests via ColonizePlanet).
func (g *Game) colonizePlanetUnlocked(ownerID, sourcePlanetID, targetPlanetID uint) error {
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
	if err := validateColonizePlanetTarget(target); err != nil {
		return err
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

	fleet, target, err := g.validateColonizeWithShipOrder(ownerID, fleetID, targetPlanetID)
	if err != nil {
		return err
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
	if err := validateColonizePlanetTarget(target); err != nil {
		return err
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

// GetColonyList returns deep-copied, hydrated planets owned by playerID.
// Takes the read lock so concurrent turn processing cannot race the listing.
func (g *Game) GetColonyList(playerID uint) ([]*Planet, error) {
	g.ensureMu()
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.colonyListCopiesUnlocked(playerID)
}

// colonyListUnlocked returns live planet pointers and hydrates them in place.
// Caller must hold the write lock (turn processing) or operate on a private copy.
func (g *Game) colonyListUnlocked(playerID uint) ([]*Planet, error) {
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

// colonyListCopiesUnlocked returns hydrated deep copies. Caller holds RLock
// on the live game, or g is a private copy with no shared mutex.
func (g *Game) colonyListCopiesUnlocked(playerID uint) ([]*Planet, error) {
	player, err := g.getPlayer(playerID)
	if err != nil {
		return nil, err
	}

	colonyList := make([]*Planet, 0)
	for _, star := range g.Galaxy.Stars {
		for _, planet := range star.Planets {
			if planet == nil || planet.Owner != playerID {
				continue
			}
			planetCopy := copyPlanet(planet)
			hydratePlanetGovernment(g, planetCopy, player)
			hydratePlanetMorale(g, planetCopy, player)
			hydratePlanetOutputs(g, planetCopy)
			hydratePlanetPopulationCapacity(g, planetCopy, player)
			hydratePlanetIncome(g, planetCopy, GamePlanetIndustryProduced(g, planetCopy))
			colonyList = append(colonyList, planetCopy)
		}
	}
	return colonyList, nil
}

func (g *Game) SetPlanetAutobuild(playerID, planetID uint, enabled bool) error {
	g.ensureMu()
	g.mu.Lock()
	defer g.mu.Unlock()
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

// ShipPartsCollection returns the immutable ship-parts catalog under RLock so
// handlers that hold a live Game still go through a locked accessor.
func (g *Game) ShipPartsCollection() *ShipPartsCollection {
	g.ensureMu()
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.Resources == nil {
		return nil
	}
	return g.Resources.ShipPartsCollection
}

func (g *Game) GetShipDesignsList(playerID uint) ([]*ShipDesign, error) {
	g.ensureMu()
	g.mu.RLock()
	defer g.mu.RUnlock()
	player, err := g.getPlayer(playerID)
	if err != nil {
		return nil, err
	}
	var shipDesigns []*ShipDesign
	for _, d := range g.Resources.ShipDesigns {
		cp := copyShipDesign(d)
		shipDesigns = append(shipDesigns, &cp)
	}
	for _, d := range player.ShipDesigns {
		cp := copyShipDesign(d)
		shipDesigns = append(shipDesigns, &cp)
	}
	return shipDesigns, nil
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

	g.ensureMu()
	g.mu.Lock()
	defer g.mu.Unlock()

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

	g.ensureMu()
	g.mu.Lock()
	defer g.mu.Unlock()

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
