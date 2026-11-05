package game

import (
	"math"
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/require"
)

func TestMovePopulationRequiresAllUnits(t *testing.T) {
	p := &model.Planet{ID: 1, Owner: 1, Population: 8500, Environment: model.PlanetEnvironmentTerran, Food: model.PlanetFoodOutput{ProducedPerFarmer: 2}}
	tests := []struct {
		name                         string
		farmers, workers, scientists int64
		valid                        bool
	}{
		{"zero", 0, 0, 0, false},
		{"partial", 2, 2, 0, false},
		{"excess", 4, 4, 1, false},
		{"overflow", math.MaxInt64, math.MaxInt64, 10, false},
		{"exact", 4, 3, 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMovePopulationCommand(&model.MovePopulationCommand{PlanetID: 1, Farmers: tt.farmers, Workers: tt.workers, Scientists: tt.scientists}, p)
			if tt.valid {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, "bad pop movement")
			}
		})
	}
}
