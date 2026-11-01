package ai

import (
	"testing"

	"ssg/model"
)

func snapshotWith(players []*model.Player, stars []*model.Star, fleets []*model.Fleet, ts TurnStatus) *VisibleGame {
	return &VisibleGame{
		CurrentTurn: 3500,
		Galaxy:      Galaxy{Stars: stars},
		Players:     players,
		Fleets:      fleets,
		TurnStatus:  ts,
	}
}

func TestSimplePolicyNoOpWhenNotWaiting(t *testing.T) {
	// The Agent skips Decide when not in waiting set, but if Decide is called
	// directly with no work to do, it should produce empty Commands.
	me := &model.Player{ID: 1}
	vg := snapshotWith([]*model.Player{me}, nil, nil, TurnStatus{ReadyPlayerIDs: []uint{1}})
	d := SimplePolicy{}.Decide(vg, 1)
	if len(d.Commands.Build) != 0 || len(d.Commands.MovePopulation) != 0 || len(d.Commands.FleetMove) != 0 {
		t.Fatalf("expected empty commands, got %+v", d.Commands)
	}
}

func TestSimplePolicyLithovoreDoesNotShiftWorkersToFarmers(t *testing.T) {
	planet := &model.Planet{
		ID:         10,
		Owner:      1,
		Population: 8000,
		Food:       model.PlanetFoodOutput{Farmers: 2, ProducedPerFarmer: 2, ProducedPerBuilding: 0},
		Industry:   model.PlanetIndustryOutput{Workers: 4},
		Science:    model.PlanetScienceOutput{Scientists: 2},
	}
	star := &model.Star{ID: 1, Name: "Sol", Planets: []*model.Planet{planet}}
	me := &model.Player{ID: 1, Race: model.Race{TraitIDs: []string{"lithovore"}}}
	vg := snapshotWith([]*model.Player{me}, []*model.Star{star}, nil, TurnStatus{WaitingPlayerIDs: []uint{1}})
	d := SimplePolicy{}.Decide(vg, 1)
	for _, cmd := range d.Commands.MovePopulation {
		if cmd.PlanetID == 10 && cmd.Farmers > planet.Food.Farmers {
			t.Fatalf("lithovore should not add farmers for food, got %+v", cmd)
		}
	}
}

func TestSimplePolicyCyberneticUsesHalvedFoodDemand(t *testing.T) {
	planet := &model.Planet{
		ID:         10,
		Owner:      1,
		Population: 8000,
		Food:       model.PlanetFoodOutput{Farmers: 2, ProducedPerFarmer: 2, ProducedPerBuilding: 0},
		Industry:   model.PlanetIndustryOutput{Workers: 4},
		Science:    model.PlanetScienceOutput{Scientists: 2},
	}
	star := &model.Star{ID: 1, Name: "Sol", Planets: []*model.Planet{planet}}
	me := &model.Player{ID: 1, Race: model.Race{TraitIDs: []string{"cybernetic"}}}
	vg := snapshotWith([]*model.Player{me}, []*model.Star{star}, nil, TurnStatus{WaitingPlayerIDs: []uint{1}})
	d := SimplePolicy{}.Decide(vg, 1)
	if len(d.Commands.MovePopulation) != 0 {
		t.Fatalf("cybernetic with 4 food vs demand 4 should not shift workers, got %+v", d.Commands.MovePopulation)
	}
}

func TestSimplePolicyStarvationShiftsWorkersToFarmers(t *testing.T) {
	planet := &model.Planet{
		ID:         10,
		Owner:      1,
		Name:       "Earth",
		Population: 8000,                                                                             // 8 pop units; food demand is 8
		Food:       model.PlanetFoodOutput{Farmers: 2, ProducedPerFarmer: 2, ProducedPerBuilding: 0}, // 4 food vs 8 needed
		Industry:   model.PlanetIndustryOutput{Workers: 4},
		Science:    model.PlanetScienceOutput{Scientists: 2},
	}
	star := &model.Star{ID: 1, Name: "Sol", Planets: []*model.Planet{planet}}
	me := &model.Player{ID: 1}
	vg := snapshotWith([]*model.Player{me}, []*model.Star{star}, nil, TurnStatus{WaitingPlayerIDs: []uint{1}})
	d := SimplePolicy{}.Decide(vg, 1)
	if len(d.Commands.MovePopulation) != 1 {
		t.Fatalf("expected 1 move population command, got %d", len(d.Commands.MovePopulation))
	}
	cmd := d.Commands.MovePopulation[0]
	if cmd.PlanetID != 10 {
		t.Fatalf("wrong planet id: %d", cmd.PlanetID)
	}
	if cmd.Farmers < 4 {
		t.Fatalf("expected at least 4 farmers to meet pop_units=8 with 2 food/farmer, got %d", cmd.Farmers)
	}
	if cmd.Farmers+cmd.Workers != planet.Food.Farmers+planet.Industry.Workers {
		t.Fatalf("farmer/worker totals not preserved: %d + %d != %d + %d", cmd.Farmers, cmd.Workers, planet.Food.Farmers, planet.Industry.Workers)
	}
}

func TestSimplePolicyColonizesPendingBase(t *testing.T) {
	me := &model.Player{ID: 1}
	vg := &VisibleGame{
		Players:            []*model.Player{me},
		PendingColonyBases: map[uint][]uint{1: {42}},
		Galaxy:             Galaxy{Stars: []*model.Star{{ID: 1, Name: "Sol", Planets: []*model.Planet{{ID: 42, Owner: 1}, {ID: 43, Environment: model.PlanetEnvironmentTerran}}}}},
		TurnStatus:         TurnStatus{WaitingPlayerIDs: []uint{1}},
	}
	d := SimplePolicy{}.Decide(vg, 1)
	if len(d.Commands.ColonizeMany) != 1 || d.Commands.ColonizeMany[0].PlanetID != 43 {
		t.Fatalf("expected colonize command for planet 43, got %+v", d.Commands.ColonizeMany)
	}
}

func TestSimplePolicyMovesIdleFleetToNearestUnexplored(t *testing.T) {
	me := &model.Player{ID: 1}
	// star 1 explored (has Name), star 2 unexplored (no Name), at different distances.
	// Owned colony at Sol provides the fuel-range supply origin.
	home := &model.Planet{ID: 1, Owner: 1}
	stars := []*model.Star{
		{ID: 1, Name: "Sol", X: 0, Y: 0, Planets: []*model.Planet{home}},
		{ID: 2, X: 5, Y: 0},
		{ID: 3, X: 50, Y: 0},
	}
	fleet := &model.Fleet{
		ID: 7, Owner: 1, StarID: 1, X: 0, Y: 0,
		Ships: []model.FleetShip{{Type: "ship_design", Name: "Scout", FuelRangeParsecs: 4}},
	}
	vg := snapshotWith([]*model.Player{me}, stars, []*model.Fleet{fleet}, TurnStatus{WaitingPlayerIDs: []uint{1}})
	d := SimplePolicy{}.Decide(vg, 1)
	if len(d.Commands.FleetMove) != 1 || d.Commands.FleetMove[0].DestinationStarID != 2 {
		t.Fatalf("expected fleet move to star 2, got %+v", d.Commands.FleetMove)
	}
}

func TestSimplePolicySkipsUnexploredOutsideFuelRange(t *testing.T) {
	// Nearest unexplored star (by fleet position) is outside fuel range from
	// the owned supply colony; a farther unexplored star is reachable.
	// On da20acf the AI picks the nearest (out of range) star 2.
	me := &model.Player{ID: 1}
	home := &model.Planet{ID: 1, Owner: 1}
	stars := []*model.Star{
		{ID: 1, Name: "Sol", X: 0, Y: 0, Planets: []*model.Planet{home}},
		// 520px ≈ 8.7pc from Sol — outside range 4; nearest to the fleet.
		{ID: 2, X: 520, Y: 0},
		// 120px = 2pc from Sol — inside range 4; farther from the fleet.
		{ID: 3, X: 120, Y: 0},
	}
	fleet := &model.Fleet{
		ID: 7, Owner: 1, StarID: 1, X: 500, Y: 0,
		Ships: []model.FleetShip{{Type: "ship_design", Name: "Scout", FuelRangeParsecs: 4}},
	}
	vg := snapshotWith([]*model.Player{me}, stars, []*model.Fleet{fleet}, TurnStatus{WaitingPlayerIDs: []uint{1}})
	d := SimplePolicy{}.Decide(vg, 1)
	if len(d.Commands.FleetMove) != 1 {
		t.Fatalf("expected one fleet move, got %+v", d.Commands.FleetMove)
	}
	if d.Commands.FleetMove[0].DestinationStarID != 3 {
		t.Fatalf("expected reachable star 3, got destination %d (notes=%v)",
			d.Commands.FleetMove[0].DestinationStarID, d.Notes)
	}
}

func TestSimplePolicyStaysIdleWhenNoReachableUnexplored(t *testing.T) {
	me := &model.Player{ID: 1}
	home := &model.Planet{ID: 1, Owner: 1}
	stars := []*model.Star{
		{ID: 1, Name: "Sol", X: 0, Y: 0, Planets: []*model.Planet{home}},
		{ID: 2, X: 600, Y: 0}, // 10pc — outside range 4
	}
	fleet := &model.Fleet{
		ID: 7, Owner: 1, StarID: 1, X: 0, Y: 0,
		Ships: []model.FleetShip{{Type: "ship_design", Name: "Scout", FuelRangeParsecs: 4}},
	}
	vg := snapshotWith([]*model.Player{me}, stars, []*model.Fleet{fleet}, TurnStatus{WaitingPlayerIDs: []uint{1}})
	d := SimplePolicy{}.Decide(vg, 1)
	if len(d.Commands.FleetMove) != 0 {
		t.Fatalf("expected no fleet moves when none reachable, got %+v", d.Commands.FleetMove)
	}
}

func TestSimplePolicyPicksFirstAvailableResearch(t *testing.T) {
	// Level 1 has Tech A researched; per server rules a non-ForceAll level is
	// complete as soon as one sibling is researched, so we must skip the
	// remaining Tech B and move on to Level 2's Tech C.
	me := &model.Player{
		ID:              1,
		ResearchedTechs: []string{"Tech A"},
	}
	vg := &VisibleGame{
		Players:    []*model.Player{me},
		TurnStatus: TurnStatus{WaitingPlayerIDs: []uint{1}},
		Resources: &VisibleResources{
			TechTree: &model.TechTree{
				Subjects: []model.TechSubject{
					{
						Name: "Construction",
						Levels: []model.TechLevel{
							{Name: "Level 1", Cost: 50, AvailableTechs: []string{"Tech A", "Tech B"}},
							{Name: "Level 2", Cost: 150, AvailableTechs: []string{"Tech C"}},
						},
					},
				},
			},
		},
	}
	d := SimplePolicy{}.Decide(vg, 1)
	if d.Commands.SelectResearch == nil {
		t.Fatalf("expected SelectResearch command, got nil")
	}
	if d.Commands.SelectResearch.TechName != "Tech C" {
		t.Fatalf("expected to skip the completed level and pick Tech C, got %q", d.Commands.SelectResearch.TechName)
	}
}

func TestSimplePolicyForceAllLevelRequiresAllTechs(t *testing.T) {
	// A ForceAll level with one tech researched is still incomplete; the AI
	// should select the level by name and the server will research the rest.
	me := &model.Player{
		ID:              1,
		ResearchedTechs: []string{"Tech A"},
	}
	vg := &VisibleGame{
		Players:    []*model.Player{me},
		TurnStatus: TurnStatus{WaitingPlayerIDs: []uint{1}},
		Resources: &VisibleResources{
			TechTree: &model.TechTree{
				Subjects: []model.TechSubject{{
					Name: "Construction",
					Levels: []model.TechLevel{
						{Name: "Heavy Industry", Cost: 200, ForceAll: true, AvailableTechs: []string{"Tech A", "Tech B"}},
					},
				}},
			},
		},
	}
	d := SimplePolicy{}.Decide(vg, 1)
	if d.Commands.SelectResearch == nil || d.Commands.SelectResearch.TechName != "Heavy Industry" {
		t.Fatalf("expected SelectResearch=Heavy Industry, got %+v", d.Commands.SelectResearch)
	}
}

func TestSimplePolicyKeepsExistingResearch(t *testing.T) {
	me := &model.Player{
		ID:              1,
		CurrentResearch: &model.ResearchProject{TechName: "Tech B"},
	}
	vg := &VisibleGame{
		Players:    []*model.Player{me},
		TurnStatus: TurnStatus{WaitingPlayerIDs: []uint{1}},
		Resources: &VisibleResources{
			TechTree: &model.TechTree{Subjects: []model.TechSubject{{Name: "Construction", Levels: []model.TechLevel{{AvailableTechs: []string{"Tech A"}}}}}},
		},
	}
	d := SimplePolicy{}.Decide(vg, 1)
	if d.Commands.SelectResearch != nil {
		t.Fatalf("expected no SelectResearch when one is in progress, got %+v", d.Commands.SelectResearch)
	}
}

func TestSimplePolicyOmniscientSkipsScoutExploration(t *testing.T) {
	me := &model.Player{ID: 1, Race: model.Race{TraitIDs: []string{"omniscient"}}}
	fleet := &model.Fleet{ID: 7, Owner: 1, X: 0, Y: 0}
	vg := snapshotWith([]*model.Player{me}, []*model.Star{{ID: 2}}, []*model.Fleet{fleet}, TurnStatus{WaitingPlayerIDs: []uint{1}})

	d := SimplePolicy{}.Decide(vg, 1)
	if len(d.Commands.FleetMove) != 0 {
		t.Fatalf("expected no scout moves for omniscient player, got %+v", d.Commands.FleetMove)
	}
}

func TestSimplePolicyQueuesAvailableBuilding(t *testing.T) {
	me := &model.Player{
		ID:        1,
		Available: model.AvailableTech{Buildings: []string{"Automated Factory"}},
	}
	planet := &model.Planet{
		ID:         11,
		Owner:      1,
		Population: 4000, // 4 pop units, 8 food from farmers — well fed
		Food:       model.PlanetFoodOutput{Farmers: 4, ProducedPerFarmer: 2},
		Industry:   model.PlanetIndustryOutput{Workers: 0},
	}
	star := &model.Star{ID: 1, Name: "Sol", Planets: []*model.Planet{planet}}
	vg := snapshotWith([]*model.Player{me}, []*model.Star{star}, nil, TurnStatus{WaitingPlayerIDs: []uint{1}})
	d := SimplePolicy{}.Decide(vg, 1)
	if len(d.Commands.Build) != 1 || d.Commands.Build[0].Building != "Automated Factory" {
		t.Fatalf("expected build of Automated Factory, got %+v", d.Commands.Build)
	}
}

func TestSimplePolicyIssuesInvadeForTransportsAtEnemyPlanet(t *testing.T) {
	me := &model.Player{ID: 1, Race: model.Race{TraitIDs: []string{"omniscient"}}}
	enemyPlanet := &model.Planet{ID: 20, Owner: 2, ParentStar: 1, Name: "Target"}
	star := &model.Star{ID: 1, Name: "Sol", Planets: []*model.Planet{enemyPlanet}}
	fleet := &model.Fleet{
		ID:     7,
		Owner:  1,
		StarID: 1,
		X:      0,
		Y:      0,
		Ships: []model.FleetShip{
			{Type: "transport", Name: "T1"},
			{Type: "ship_design", Name: "Escort"},
		},
	}
	vg := snapshotWith([]*model.Player{me}, []*model.Star{star}, []*model.Fleet{fleet}, TurnStatus{WaitingPlayerIDs: []uint{1}})
	d := SimplePolicy{}.Decide(vg, 1)
	if len(d.Commands.Invade) != 1 {
		t.Fatalf("expected one invade command, got %+v", d.Commands.Invade)
	}
	cmd := d.Commands.Invade[0]
	if cmd.FleetID != 7 || cmd.TargetPlanetID != 20 {
		t.Fatalf("unexpected invade command %+v", cmd)
	}
	if cmd.ShipIndexes == nil || len(*cmd.ShipIndexes) != 1 || (*cmd.ShipIndexes)[0] != 0 {
		t.Fatalf("expected ship index [0] for the transport, got %+v", cmd.ShipIndexes)
	}
	if len(d.Commands.FleetMove) != 0 {
		t.Fatalf("invading fleet should not also scout-move, got %+v", d.Commands.FleetMove)
	}
}

func TestSimplePolicyUsesFuelCellCatalogOverStamp(t *testing.T) {
	// Stamp says range 9 but FuelCell Standard (autonomy 2) must win — same as
	// validateFleetMoveFuelRange / fleetShipFuelRange.
	me := &model.Player{ID: 1}
	home := &model.Planet{ID: 1, Owner: 1}
	stars := []*model.Star{
		{ID: 1, Name: "Sol", X: 0, Y: 0, Planets: []*model.Planet{home}},
		{ID: 2, X: 300, Y: 0}, // 5pc — inside stamp 9, outside Standard 2
		{ID: 3, X: 60, Y: 0},  // 1pc — inside Standard 2
	}
	fleet := &model.Fleet{
		ID: 7, Owner: 1, StarID: 1, X: 0, Y: 0,
		Ships: []model.FleetShip{{
			Type: "ship_design", Name: "Scout",
			FuelCell: "Standard", FuelRangeParsecs: 9,
		}},
	}
	vg := snapshotWith([]*model.Player{me}, stars, []*model.Fleet{fleet}, TurnStatus{WaitingPlayerIDs: []uint{1}})
	vg.Resources = &VisibleResources{ShipParts: &VisibleShipParts{
		FuelCells: []model.FuelCell{{Name: "Standard", Autonomy: 2}},
	}}
	d := SimplePolicy{}.Decide(vg, 1)
	if len(d.Commands.FleetMove) != 1 || d.Commands.FleetMove[0].DestinationStarID != 3 {
		t.Fatalf("expected star 3 (Standard range), got %+v notes=%v", d.Commands.FleetMove, d.Notes)
	}
}

func TestSimplePolicyScoutsDesignOnlyShipViaPlayerDesign(t *testing.T) {
	// On 2e1d8aa ShipsListedFuelRange fails (no stamp) and the fleet stays idle.
	me := &model.Player{
		ID: 1,
		ShipDesigns: []model.ShipDesign{{
			Name:     "Laser Gunship",
			FuelCell: model.FuelCell{Name: "Standard", Autonomy: 2},
		}},
	}
	home := &model.Planet{ID: 1, Owner: 1}
	stars := []*model.Star{
		{ID: 1, Name: "Sol", X: 0, Y: 0, Planets: []*model.Planet{home}},
		{ID: 2, X: 60, Y: 0}, // 1pc — inside Standard 2
	}
	fleet := &model.Fleet{
		ID: 7, Owner: 1, StarID: 1, X: 0, Y: 0,
		Ships: []model.FleetShip{{
			Type: "ship_design", Name: "Dark Blade", DesignName: "Laser Gunship",
		}},
	}
	vg := snapshotWith([]*model.Player{me}, stars, []*model.Fleet{fleet}, TurnStatus{WaitingPlayerIDs: []uint{1}})
	d := SimplePolicy{}.Decide(vg, 1)
	if len(d.Commands.FleetMove) != 1 || d.Commands.FleetMove[0].DestinationStarID != 2 {
		t.Fatalf("expected design-only scout move to star 2, got %+v notes=%v", d.Commands.FleetMove, d.Notes)
	}
}

func TestSimplePolicyScoutsTransportViaBestAvailableFuelCell(t *testing.T) {
	me := &model.Player{
		ID:        1,
		Available: model.AvailableTech{FuelCells: []string{"None", "Standard"}},
	}
	home := &model.Planet{ID: 1, Owner: 1}
	stars := []*model.Star{
		{ID: 1, Name: "Sol", X: 0, Y: 0, Planets: []*model.Planet{home}},
		{ID: 2, X: 60, Y: 0},
	}
	fleet := &model.Fleet{
		ID: 7, Owner: 1, StarID: 1, X: 0, Y: 0,
		Ships: []model.FleetShip{{Type: "transport", Name: "Transport"}},
	}
	vg := snapshotWith([]*model.Player{me}, stars, []*model.Fleet{fleet}, TurnStatus{WaitingPlayerIDs: []uint{1}})
	vg.Resources = &VisibleResources{ShipParts: &VisibleShipParts{
		FuelCells: []model.FuelCell{
			{Name: "None", Autonomy: 0},
			{Name: "Standard", Autonomy: 2},
		},
	}}
	d := SimplePolicy{}.Decide(vg, 1)
	if len(d.Commands.FleetMove) != 1 || d.Commands.FleetMove[0].DestinationStarID != 2 {
		t.Fatalf("expected transport scout move to star 2, got %+v notes=%v", d.Commands.FleetMove, d.Notes)
	}
}

func TestSimplePolicyScoutsColonyShipViaBestAvailableFuelCell(t *testing.T) {
	me := &model.Player{
		ID:        1,
		Available: model.AvailableTech{FuelCells: []string{"Standard"}},
	}
	home := &model.Planet{ID: 1, Owner: 1}
	stars := []*model.Star{
		{ID: 1, Name: "Sol", X: 0, Y: 0, Planets: []*model.Planet{home}},
		{ID: 2, X: 60, Y: 0},
	}
	fleet := &model.Fleet{
		ID: 7, Owner: 1, StarID: 1, X: 0, Y: 0,
		Ships: []model.FleetShip{{Type: "colony_ship", Name: "Colony Ship"}},
	}
	vg := snapshotWith([]*model.Player{me}, stars, []*model.Fleet{fleet}, TurnStatus{WaitingPlayerIDs: []uint{1}})
	vg.Resources = &VisibleResources{ShipParts: &VisibleShipParts{
		FuelCells: []model.FuelCell{{Name: "Standard", Autonomy: 2}},
	}}
	d := SimplePolicy{}.Decide(vg, 1)
	if len(d.Commands.FleetMove) != 1 || d.Commands.FleetMove[0].DestinationStarID != 2 {
		t.Fatalf("expected colony-ship scout move to star 2, got %+v notes=%v", d.Commands.FleetMove, d.Notes)
	}
}

func TestSimplePolicyMergedScoutUsesShortestRange(t *testing.T) {
	// Stamped scout (range 4) merged with unstamped Transport (best available = 2)
	// must use the shortest range (2), still scout, and skip out-of-range targets.
	me := &model.Player{
		ID:        1,
		Available: model.AvailableTech{FuelCells: []string{"Standard"}},
	}
	home := &model.Planet{ID: 1, Owner: 1}
	stars := []*model.Star{
		{ID: 1, Name: "Sol", X: 0, Y: 0, Planets: []*model.Planet{home}},
		{ID: 2, X: 300, Y: 0}, // 5pc — outside Standard 2 / inside stamp 4
		{ID: 3, X: 60, Y: 0},  // 1pc — inside both
	}
	fleet := &model.Fleet{
		ID: 7, Owner: 1, StarID: 1, X: 0, Y: 0,
		Ships: []model.FleetShip{
			{Type: "ship_design", Name: "Scout", FuelRangeParsecs: 4},
			{Type: "transport", Name: "Transport"},
		},
	}
	vg := snapshotWith([]*model.Player{me}, stars, []*model.Fleet{fleet}, TurnStatus{WaitingPlayerIDs: []uint{1}})
	vg.Resources = &VisibleResources{ShipParts: &VisibleShipParts{
		FuelCells: []model.FuelCell{{Name: "Standard", Autonomy: 2}},
	}}
	d := SimplePolicy{}.Decide(vg, 1)
	if len(d.Commands.FleetMove) != 1 || d.Commands.FleetMove[0].DestinationStarID != 3 {
		t.Fatalf("expected merged fleet to scout star 3 (shortest range), got %+v notes=%v",
			d.Commands.FleetMove, d.Notes)
	}
}

func TestScoutsReserveDistinctReachableDestinations(t *testing.T) {
	me := &model.Player{ID: 1}
	stars := []*model.Star{
		{ID: 1, Name: "Sol", Planets: []*model.Planet{{ID: 1, Owner: 1}}},
		{ID: 2, X: 30}, {ID: 3, X: 90}, {ID: 4, X: 150}, {ID: 5, X: 900},
	}
	fleets := []*model.Fleet{
		{ID: 10, Owner: 1, StarID: 1, Ships: []model.FleetShip{{Name: "A", FuelRangeParsecs: 4}}},
		{ID: 11, Owner: 1, StarID: 1, Ships: []model.FleetShip{{Name: "B", FuelRangeParsecs: 4}}},
		{ID: 12, Owner: 1, StarID: 1, Ships: []model.FleetShip{{Name: "C", FuelRangeParsecs: 4}}},
		{ID: 13, Owner: 1, DestinationStarID: 2, TurnsRemaining: 1, Ships: []model.FleetShip{{Name: "En route", FuelRangeParsecs: 4}}},
	}
	vg := snapshotWith([]*model.Player{me}, stars, fleets, TurnStatus{WaitingPlayerIDs: []uint{1}})
	d := SimplePolicy{}.Decide(vg, 1)
	if len(d.Commands.FleetMove) != 2 {
		t.Fatalf("expected two distinct reachable targets, got %+v", d.Commands.FleetMove)
	}
	if d.Commands.FleetMove[0].DestinationStarID != 3 || d.Commands.FleetMove[1].DestinationStarID != 4 {
		t.Fatalf("expected stars 3 and 4, avoiding en-route star 2 and unreachable star 5: %+v", d.Commands.FleetMove)
	}
}

func TestScoutKeepsFleetPreparingColonyShipInOrbit(t *testing.T) {
	me := &model.Player{ID: 1}
	stars := []*model.Star{{ID: 1, Name: "Sol", Planets: []*model.Planet{{ID: 1, Owner: 1}}}, {ID: 2, X: 30}}
	fleet := &model.Fleet{ID: 7, Owner: 1, StarID: 1, Ships: []model.FleetShip{{Type: "colony_ship", FuelRangeParsecs: 4}}}
	vg := snapshotWith([]*model.Player{me}, stars, []*model.Fleet{fleet}, TurnStatus{WaitingPlayerIDs: []uint{1}})
	vg.PendingColonyShips = map[uint][]uint{1: {7}}
	d := SimplePolicy{}.Decide(vg, 1)
	if d.Commands.ColonyShip == nil || d.Commands.ColonyShip.FleetID != 7 {
		t.Fatal("expected colony ship preparation")
	}
	if len(d.Commands.FleetMove) != 0 {
		t.Fatalf("fleet with a colony order should not move: %+v", d.Commands.FleetMove)
	}
}

func TestScoutHoldsColonyShipAtSuitableLocalPlanet(t *testing.T) {
	for _, tc := range []struct {
		name     string
		env      model.PlanetEnvironment
		owner    uint
		wantMove bool
	}{
		{"habitable", model.PlanetEnvironmentTerran, 0, false},
		{"gas giant", model.PlanetEnvironmentGasGiant, 0, true},
		{"asteroid belt", model.PlanetEnvironmentAsteroidBelt, 0, true},
		{"occupied", model.PlanetEnvironmentTerran, 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			me := &model.Player{ID: 1}
			stars := []*model.Star{{ID: 1, Name: "Sol", Planets: []*model.Planet{{ID: 1, Owner: 1}, {ID: 2, Owner: tc.owner, Environment: tc.env}}}, {ID: 3, X: 30}}
			fleet := &model.Fleet{ID: 7, Owner: 1, StarID: 1, Ships: []model.FleetShip{{Type: "colony_ship", FuelRangeParsecs: 4}}}
			vg := snapshotWith([]*model.Player{me}, stars, []*model.Fleet{fleet}, TurnStatus{WaitingPlayerIDs: []uint{1}})
			d := SimplePolicy{}.Decide(vg, 1)
			if got := len(d.Commands.FleetMove) > 0; got != tc.wantMove {
				t.Fatalf("got move=%v, want %v", got, tc.wantMove)
			}
		})
	}
}
