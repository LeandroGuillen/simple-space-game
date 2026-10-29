package ai

import (
	"context"
	"errors"
	"strings"
	"testing"

	"ssg/model"
)

type stubSubmitter struct {
	calls    int
	rejectIf func(model.Commands) error
	accepts  []*TurnResponse
}

func (s *stubSubmitter) SubmitTurn(_ context.Context, commands model.Commands) (*TurnResponse, error) {
	s.calls++
	if s.rejectIf != nil {
		if err := s.rejectIf(commands); err != nil {
			return nil, err
		}
	}
	if len(s.accepts) > 0 {
		resp := s.accepts[0]
		s.accepts = s.accepts[1:]
		return resp, nil
	}
	return &TurnResponse{OK: true, Advanced: true}, nil
}

func TestAgentRetriesAfterFuelRangeRejection(t *testing.T) {
	// Reject the first submit when a fleet move is present; accept once moves are stripped.
	stub := &stubSubmitter{
		rejectIf: func(cmds model.Commands) error {
			if len(cmds.FleetMove) > 0 {
				return errors.New("destination is outside fleet fuel range (4 parsecs)")
			}
			return nil
		},
	}
	a := newAgent("test", &Client{PlayerID: 1, GameID: "g", BaseURL: "http://example"}, SimplePolicy{}, NewManager(context.Background(), 0), 0)
	vg := &VisibleGame{
		CurrentTurn: 3500.1,
		TurnStatus:  TurnStatus{WaitingPlayerIDs: []uint{1}},
		Players:     []*model.Player{{ID: 1}},
	}
	decision := Decision{
		Commands: model.Commands{
			FleetMove: []*model.FleetMoveCommand{{FleetID: 7, DestinationStarID: 99}},
			Build:     []*model.BuildCommand{{PlanetID: 1, Building: "Automated Factory"}},
		},
		Notes: []string{"scout"},
	}

	a.submitDecisionWithRetries(context.Background(), vg, stub, decision)

	if stub.calls != 2 {
		t.Fatalf("expected 2 submit attempts (reject then retry), got %d", stub.calls)
	}
	entries := a.Log.Snapshot()
	var sawRetry, sawReject bool
	for _, e := range entries {
		if strings.Contains(e.Message, "retrying turn without rejected fleet move") {
			sawRetry = true
		}
		if e.Error != "" && strings.Contains(e.Error, "fuel range") {
			sawReject = true
		}
	}
	if !sawReject {
		t.Fatalf("expected a logged fuel-range rejection, entries=%v", entries)
	}
	if !sawRetry {
		t.Fatalf("expected a retry info log, entries=%v", entries)
	}
	if a.Summary().Status == StatusError {
		t.Fatalf("agent should not stay in error after successful retry, status=%s err=%s",
			a.Summary().Status, a.Summary().LastError)
	}
}

func TestStripRejectedFleetMovesOnlyOnFuelErrors(t *testing.T) {
	cmds := model.Commands{
		FleetMove: []*model.FleetMoveCommand{{FleetID: 1, DestinationStarID: 2}},
		Build:     []*model.BuildCommand{{PlanetID: 1, Building: "X"}},
	}
	next, ok := stripRejectedFleetMoves(cmds, errors.New("destination is outside fleet fuel range"))
	if !ok || len(next.FleetMove) != 0 || len(next.Build) != 1 {
		t.Fatalf("expected fleet moves stripped, got ok=%v cmds=%+v", ok, next)
	}
	_, ok = stripRejectedFleetMoves(cmds, errors.New("treasury too low"))
	if ok {
		t.Fatalf("non-fleet errors must not strip moves")
	}
	_, ok = stripRejectedFleetMoves(model.Commands{}, errors.New("outside fleet fuel range"))
	if ok {
		t.Fatalf("nothing to strip when no fleet moves")
	}
}
