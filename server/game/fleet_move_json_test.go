package game

import (
	"encoding/json"
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/require"
)

func TestFleetMoveCommandJSONIncludesShipIndexes(t *testing.T) {
	payload := `{"commands":{"build":[],"fleet_move":[{"fleet_id":1,"destination_star_id":2,"ship_indexes":[1]}]}}`
	var req struct {
		Commands model.Commands `json:"commands"`
	}
	require.NoError(t, json.Unmarshal([]byte(payload), &req))
	require.Len(t, req.Commands.FleetMove, 1)
	require.NotNil(t, req.Commands.FleetMove[0].ShipIndexes)
	require.Equal(t, []uint{1}, *req.Commands.FleetMove[0].ShipIndexes)
}

func TestFleetMoveCommandJSONOmitsShipIndexes(t *testing.T) {
	payload := `{"commands":{"fleet_move":[{"fleet_id":1,"destination_star_id":2}]}}`
	var req struct {
		Commands model.Commands `json:"commands"`
	}
	require.NoError(t, json.Unmarshal([]byte(payload), &req))
	require.Len(t, req.Commands.FleetMove, 1)
	require.Nil(t, req.Commands.FleetMove[0].ShipIndexes)
}
