package main

import "core:fmt"

import rl "vendor:raylib"

// Phase 2: fleet helpers + the Fleets tab (per-ship inspector, partial
// moves, colonize, invade, invasion reports).

find_fleet :: proc(g: Api_Game, fleet_id: int) -> (Api_Fleet, bool) {
	for f in g.fleets {
		if f.id == fleet_id do return f, true
	}
	return {}, false
}

find_star :: proc(g: Api_Game, star_id: int) -> (Api_Star, bool) {
	for s in g.galaxy.stars {
		if s.id == star_id do return s, true
	}
	return {}, false
}

fleet_in_orbit :: proc(f: Api_Fleet) -> bool {
	return f.turns_remaining == 0
}

// Same rule as the web client's isPlanetSuitableForColonization.
planet_suitable_for_colonization :: proc(p: Api_Planet) -> bool {
	return p.owner == 0 && p.environment != 1 && p.environment != 2
}

// Gas giants / asteroid belts: targets for artificial planet construction.
planet_artificial_target :: proc(p: Api_Planet) -> bool {
	return p.owner == 0 && (p.environment == 1 || p.environment == 2)
}

// Reads one player's id list out of the game's pending_* maps, whose JSON
// object keys are the player ids as strings.
pending_ids_for_player :: proc(m: map[string][]int, player_id: int) -> []int {
	key := fmt.tprintf("%d", player_id)
	if ids, ok := m[key]; ok do return ids
	return nil
}

fleet_has_ship_type :: proc(f: Api_Fleet, type: string) -> bool {
	for s in f.ships {
		if s.type == type do return true
	}
	return false
}

// --- Fleets tab state -------------------------------------------------------

fleets_selected_fleet: int      // fleet id shown in the inspector (0 = none)
fleet_ship_sel: [dynamic]bool   // selection per original ship index
fleets_colonize_idx: int        // cycle position among suitable target planets
fleets_invade_idx: int          // cycle position among enemy target planets

// Select a fleet in the inspector, defaulting the ship selection to "all".
fleet_select :: proc(g: Api_Game, fleet_id: int) {
	fleets_selected_fleet = fleet_id
	fleets_colonize_idx = 0
	fleets_invade_idx = 0
	clear(&fleet_ship_sel)
	if f, ok := find_fleet(g, fleet_id); ok {
		for _ in f.ships do append(&fleet_ship_sel, true)
	}
}

// The ship indexes a queued move should carry for this fleet: nil (whole
// fleet) unless the inspector holds a proper subset selection for it.
selected_ship_indexes_for_move :: proc(g: Api_Game, fleet_id: int, allocator := context.allocator) -> []int {
	if fleet_id != fleets_selected_fleet do return nil
	f, ok := find_fleet(g, fleet_id)
	if !ok || len(fleet_ship_sel) != len(f.ships) do return nil
	out := make([dynamic]int, allocator)
	for selected, i in fleet_ship_sel {
		if selected do append(&out, i)
	}
	if len(out) == len(f.ships) || len(out) == 0 {
		delete(out)
		return nil // all (or none) selected => whole-fleet move
	}
	return out[:]
}

ship_type_label :: proc(type, name: string) -> string {
	switch type {
	case "colony_ship": return "Colony Ship"
	case "transport":   return "Transport"
	case:
		if name != "" do return name
		return type
	}
}

draw_fleets_tab :: proc(g: Api_Game, cx, cy_in: i32, sel: ^Selection, player_id: int) -> i32 {
	cy := cy_in

	my_fleets := player_fleets(g, player_id, context.temp_allocator)
	rl.DrawText(fmt.ctprintf("Your Fleets (%d)", len(my_fleets)), cx, cy, pxi(16), rl.RAYWHITE); cy += pxi(22)

	if len(my_fleets) == 0 {
		rl.DrawText("(no fleets)", cx, cy, pxi(14), rl.GRAY)
		cy += pxi(18)
		return draw_invasion_reports(g, cx, cy, player_id)
	}

	// --- Fleet list ---
	for f in my_fleets {
		row := rl.Rectangle{f32(cx), f32(cy), f32(pxi(268)), f32(pxi(18))}
		hover := ui_hover(row)
		active := f.id == fleets_selected_fleet
		if active do rl.DrawRectangleRec(row, {44, 60, 90, 255})
		else if hover do rl.DrawRectangleRec(row, {34, 42, 60, 255})

		status := fleet_status_label(g, f)
		if fleet_move_queued(f.id) do status = fmt.tprintf("%s  [move queued]", status)
		if colonize_queued_for_fleet(f.id) do status = fmt.tprintf("%s  [colonizing]", status)
		if invade_queued_for_fleet(f.id) do status = fmt.tprintf("%s  [invading]", status)
		rl.DrawText(fmt.ctprintf("#%d  %d ships  %s", f.id, len(f.ships), status), cx, cy, pxi(12), active ? rl.YELLOW : rl.LIGHTGRAY)

		if hover && rl.IsMouseButtonPressed(.LEFT) {
			fleet_select(g, f.id)
		}
		cy += pxi(18)
	}
	cy += pxi(8)

	// --- Inspector for the selected fleet ---
	if _, fok := find_fleet(g, fleets_selected_fleet); fok {
		cy = draw_fleet_inspector(g, cx, cy, sel, player_id, fleets_selected_fleet)
	} else {
		rl.DrawText("click a fleet to inspect it", cx, cy, pxi(13), rl.GRAY)
		cy += pxi(18)
	}

	return draw_invasion_reports(g, cx, cy, player_id)
}

fleet_status_label :: proc(g: Api_Game, f: Api_Fleet) -> string {
	if fleet_in_orbit(f) {
		s, sok := find_star(g, f.star_id)
		return fmt.tprintf("orbiting %s", s.name if sok else "?")
	}
	s, sok := find_star(g, f.destination_star_id)
	return fmt.tprintf("-> %s (%dt)", s.name if sok else "?", f.turns_remaining)
}

// Ship-level view of one fleet: selection checkboxes, move targeting, and
// the colonize / invade actions. Shared between the Fleets tab and the
// contextual Galaxy sidebar (fleet selected on the map).
draw_fleet_inspector :: proc(g: Api_Game, cx, cy_in: i32, sel: ^Selection, player_id: int, fleet_id: int) -> i32 {
	cy := cy_in

	fleet, fok := find_fleet(g, fleet_id)
	if !fok {
		rl.DrawText("(fleet no longer exists)", cx, cy, pxi(13), rl.GRAY)
		sel.has_fleet = false
		return cy + pxi(18)
	}
	if fleets_selected_fleet != fleet.id || len(fleet_ship_sel) != len(fleet.ships) {
		// Different fleet than last time, or the snapshot changed under us.
		fleet_select(g, fleet.id)
	}

	rl.DrawText(fmt.ctprintf("Fleet #%d  %s", fleet.id, fleet_status_label(g, fleet)), cx, cy, pxi(15), {180, 200, 220, 255}); cy += pxi(20)

	// Ship list with per-ship selection toggles.
	pending_idx := fleet_pending_move_indexes(fleet.id, context.temp_allocator)
	n_selected := 0
	for ship, i in fleet.ships {
		row := rl.Rectangle{f32(cx), f32(cy), f32(pxi(268)), f32(pxi(16))}
		hover := ui_hover(row)
		if hover do rl.DrawRectangleRec(row, {34, 42, 60, 255})

		mark: string = fleet_ship_sel[i] ? "[x]" : "[ ]"
		if fleet_ship_sel[i] do n_selected += 1
		tag := ""
		for pi in pending_idx {
			if pi == i {
				tag = "  (move queued)"
				break
			}
		}
		rl.DrawText(fmt.ctprintf("%s %d. %s%s", mark, i + 1, ship_type_label(ship.type, ship.name), tag), cx, cy, pxi(12), rl.LIGHTGRAY)
		if hover && rl.IsMouseButtonPressed(.LEFT) do fleet_ship_sel[i] = !fleet_ship_sel[i]
		cy += pxi(16)
	}
	cy += pxi(4)

	if gui_button({f32(cx), f32(cy), f32(pxi(90)), f32(pxi(20))}, n_selected == len(fleet.ships) ? "None" : "All") {
		all := n_selected != len(fleet.ships)
		for i in 0 ..< len(fleet_ship_sel) do fleet_ship_sel[i] = all
	}
	targeting := sel.move_targeting && sel.has_fleet && sel.fleet_id == fleet.id
	move_label: cstring = targeting ? "click a star..." : "Move"
	if gui_button({f32(cx + pxi(96)), f32(cy), f32(pxi(120)), f32(pxi(20))}, move_label, n_selected > 0) {
		sel.fleet_id = fleet.id
		sel.has_fleet = true
		sel.move_targeting = !targeting // clicking again disarms
		active_tab = .Galaxy
		open_window = .None // the map needs to be clickable to pick a star
	}
	cy += pxi(26)

	// --- Colonize (orbiting colony ship + suitable planet at this star) ---
	if fleet_in_orbit(fleet) && fleet_has_ship_type(fleet, "colony_ship") {
		star, sok := find_star(g, fleet.star_id)
		targets := make([dynamic]Api_Planet, context.temp_allocator)
		if sok {
			for p in star.planets {
				if planet_suitable_for_colonization(p) do append(&targets, p)
			}
		}
		if colonize_queued_for_fleet(fleet.id) {
			if gui_button({f32(cx), f32(cy), f32(pxi(216)), f32(pxi(22))}, "Cancel Colonization") {
				cancel_colonize_fleet(fleet.id)
			}
			cy += pxi(26)
		} else if len(targets) > 0 {
			if fleets_colonize_idx >= len(targets) do fleets_colonize_idx = 0
			target := targets[fleets_colonize_idx]
			if len(targets) > 1 {
				if gui_button({f32(cx), f32(cy), f32(pxi(24)), f32(pxi(20))}, "<") {
					fleets_colonize_idx = (fleets_colonize_idx - 1 + len(targets)) % len(targets)
				}
				if gui_button({f32(cx + pxi(240)), f32(cy), f32(pxi(24)), f32(pxi(20))}, ">") {
					fleets_colonize_idx = (fleets_colonize_idx + 1) % len(targets)
				}
			}
			rl.DrawText(fmt.ctprintf("%s (%s)", target.name, env_name(target.environment)), cx + pxi(30), cy + pxi(3), pxi(12), rl.LIGHTGRAY)
			cy += pxi(22)
			if gui_button({f32(cx), f32(cy), f32(pxi(216)), f32(pxi(22))}, "Colonize Planet") {
				enqueue_colonize(target.id, fleet.id)
			}
			cy += pxi(26)
		}
	}

	// --- Invade (orbiting transport + enemy planet at this star) ---
	if fleet_in_orbit(fleet) && fleet_has_ship_type(fleet, "transport") {
		star, sok := find_star(g, fleet.star_id)
		targets := make([dynamic]Api_Planet, context.temp_allocator)
		if sok {
			for p in star.planets {
				if p.owner != 0 && p.owner != player_id do append(&targets, p)
			}
		}
		if invade_queued_for_fleet(fleet.id) {
			if gui_button({f32(cx), f32(cy), f32(pxi(216)), f32(pxi(22))}, "Cancel Invasion") {
				cancel_invade_fleet(fleet.id)
			}
			cy += pxi(26)
		} else if len(targets) > 0 {
			if fleets_invade_idx >= len(targets) do fleets_invade_idx = 0
			target := targets[fleets_invade_idx]
			if len(targets) > 1 {
				if gui_button({f32(cx), f32(cy), f32(pxi(24)), f32(pxi(20))}, "<") {
					fleets_invade_idx = (fleets_invade_idx - 1 + len(targets)) % len(targets)
				}
				if gui_button({f32(cx + pxi(240)), f32(cy), f32(pxi(24)), f32(pxi(20))}, ">") {
					fleets_invade_idx = (fleets_invade_idx + 1) % len(targets)
				}
			}
			rl.DrawText(fmt.ctprintf("%s (%s, %d marines)", target.name, player_name_for(g, target.owner), target.marines), cx + pxi(30), cy + pxi(3), pxi(12), {220, 150, 140, 255})
			cy += pxi(22)
			if gui_button({f32(cx), f32(cy), f32(pxi(216)), f32(pxi(22))}, "Invade Planet") {
				transports := make([dynamic]int, context.temp_allocator)
				for s, i in fleet.ships {
					if s.type == "transport" do append(&transports, i)
				}
				enqueue_invade(fleet.id, target.id, transports[:])
			}
			cy += pxi(26)
		}
	}

	return cy
}

// Latest invasion reports involving the player (Fleets tab only).
draw_invasion_reports :: proc(g: Api_Game, cx, cy_in: i32, player_id: int) -> i32 {
	cy := cy_in
	shown := 0
	for ri := len(g.invasion_reports) - 1; ri >= 0 && shown < 4; ri -= 1 {
		r := g.invasion_reports[ri]
		if r.attacker_owner_id != player_id && r.defender_owner_id != player_id do continue
		if shown == 0 {
			cy += pxi(6)
			rl.DrawText("Invasion Reports", cx, cy, pxi(15), {180, 200, 220, 255})
			cy += pxi(20)
		}
		outcome := invasion_outcome_label(r)
		col: rl.Color
		if r.cancelled {
			// Neutral/yellowish — no fight happened; avoid win/loss colours.
			col = {210, 190, 120, 255}
		} else {
			col = {210, 130, 120, 255}
			if (r.attacker_owner_id == player_id) == r.success do col = {140, 200, 150, 255}
		}
		atk0 := ground_force_units(r.initial_attacker)
		def0 := ground_force_units(r.initial_defender)
		surv := ground_force_units(r.surviving_units)
		rl.DrawText(fmt.ctprintf("T%d %s: %s", r.turn, r.planet_name, outcome), cx, cy, pxi(12), col)
		cy += pxi(15)
		rl.DrawText(fmt.ctprintf("  %s attacked %s  (%d vs %d, %d left)", player_name_for(g, r.attacker_owner_id), player_name_for(g, r.defender_owner_id), atk0, def0, surv), cx, cy, pxi(11), rl.GRAY)
		cy += pxi(16)
		shown += 1
	}

	return cy
}

ground_force_units :: proc(f: Api_Ground_Force) -> int {
	return len(f.marines) + len(f.armored)
}
