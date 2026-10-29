package main

// Pure outcome label for an invasion report. Cancelled (orbit lost) is
// checked before captured/won/repelled so a no-fight cancel is not shown
// as a ground-combat "repelled".
invasion_outcome_label :: proc(r: Api_Invasion_Report) -> string {
	if r.cancelled do return "cancelled (orbit lost)"
	if r.planet_captured do return "captured"
	if r.success do return "won"
	return "repelled"
}
