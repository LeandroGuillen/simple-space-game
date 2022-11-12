package game

import (
	. "ssg/game/model"
)

func MaxPop(p *Planet) int64 {
	// TODO should take into account other factors
	// Planetary Env.:  Gaia Terran Arid Swamp Other
	// Tiny              5     4     3    2     1     <-- The table shows how
	// Small            10     8     6    4     3         the base max. pop. of
	// Medium           15    12     9    6     4         a planet is determined
	// Large            20    16    12    8     5     <-- by size and climate.
	// Huge             25    20    15   10     6
	return int64(p.Size*4) * 1000
}
