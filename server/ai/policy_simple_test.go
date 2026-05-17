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

func TestSimplePolicyStarvationShiftsWorkersToFarmers(t *testing.T) {
	planet := &model.Planet{
		ID:         10,
		Owner:      1,
		Name:       "Earth",
		Population: 8000, // 8 pop units; food demand is 8
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
		TurnStatus:         TurnStatus{WaitingPlayerIDs: []uint{1}},
	}
	d := SimplePolicy{}.Decide(vg, 1)
	if len(d.Commands.ColonizeMany) != 1 || d.Commands.ColonizeMany[0].PlanetID != 42 {
		t.Fatalf("expected colonize command for planet 42, got %+v", d.Commands.ColonizeMany)
	}
}

func TestSimplePolicyMovesIdleFleetToNearestUnexplored(t *testing.T) {
	me := &model.Player{ID: 1}
	// star 1 explored (has Name), star 2 unexplored (no Name), at different distances.
	stars := []*model.Star{
		{ID: 1, Name: "Sol", X: 0, Y: 0},
		{ID: 2, X: 5, Y: 0},
		{ID: 3, X: 50, Y: 0},
	}
	fleet := &model.Fleet{ID: 7, Owner: 1, StarID: 1, X: 0, Y: 0}
	vg := snapshotWith([]*model.Player{me}, stars, []*model.Fleet{fleet}, TurnStatus{WaitingPlayerIDs: []uint{1}})
	d := SimplePolicy{}.Decide(vg, 1)
	if len(d.Commands.FleetMove) != 1 || d.Commands.FleetMove[0].DestinationStarID != 2 {
		t.Fatalf("expected fleet move to star 2, got %+v", d.Commands.FleetMove)
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
