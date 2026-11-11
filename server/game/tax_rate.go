package game

import (
	"fmt"
	"math"
)

// validateTaxRate accepts continuous rates; UI step sizes are not a server rule.
func validateTaxRate(rate float64) error {
	if math.IsNaN(rate) || math.IsInf(rate, 0) || rate < 0 || rate > 0.5 {
		return fmt.Errorf("invalid tax rate %v (want 0..0.5)", rate)
	}
	return nil
}
