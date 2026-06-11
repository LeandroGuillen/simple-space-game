package main

import "core:encoding/json"
import "core:fmt"
import "core:strings"

// Client-side pending-command buffer. The web client accumulates many
// commands locally and submits them in a single POST /turn; this mirrors
// that. Issuing a command appends to `pending`; "End Turn" assembles the
// whole buffer into JSON, POSTs it, and on success clears it.
//
// `pending` is owned and mutated only by the main (UI) thread, so it needs
// no locking — unlike `state`, which the WebSocket reader writes.
//
// Singular optional commands use Maybe(T) rather than pointers because
// Odin's json.marshal cannot serialise non-nil pointer fields; the JSON is
// assembled by hand in build_turn_body so only present commands are emitted.

Pending :: struct {
	build:                  [dynamic]Build_Command,
	move_population:        [dynamic]Move_Population_Command,
	transfer_population:    [dynamic]Transfer_Population_Command,
	colonize_many:          [dynamic]Colonize_Command,
	artificial_planet_many: [dynamic]Artificial_Planet_Command,
	fleet_move:             [dynamic]Fleet_Move_Command,
	invade:                 [dynamic]Invade_Command,
	buy_production:         [dynamic]Buy_Production_Command,
	select_research:        Maybe(Select_Research_Command),
	mutate_race:            Maybe(Mutate_Race_Command),
}

pending: Pending

// How many commands are queued, across every kind. Drives the "N queued"
// hint next to the End Turn button.
pending_count :: proc() -> int {
	n := len(pending.build)
	n += len(pending.move_population)
	n += len(pending.transfer_population)
	n += len(pending.colonize_many)
	n += len(pending.artificial_planet_many)
	n += len(pending.fleet_move)
	n += len(pending.invade)
	n += len(pending.buy_production)
	if _, ok := pending.select_research.?; ok do n += 1
	if _, ok := pending.mutate_race.?; ok do n += 1
	return n
}

// Queue a whole-fleet move (ship_indexes left empty => whole fleet).
// Re-targeting the same fleet replaces its queued move rather than stacking
// a second one. Mutually exclusive with a queued colonize for the fleet
// (matches the web client).
enqueue_fleet_move :: proc(fleet_id, destination_star_id: int) {
	cancel_colonize_fleet(fleet_id)
	for i := len(pending.fleet_move) - 1; i >= 0; i -= 1 {
		if pending.fleet_move[i].fleet_id == fleet_id {
			ordered_remove(&pending.fleet_move, i)
		}
	}
	append(&pending.fleet_move, Fleet_Move_Command{
		fleet_id            = fleet_id,
		destination_star_id = destination_star_id,
	})
}

// Queue a partial-fleet move: exactly `ship_indexes` go to the destination.
// Mirrors the web client's merge semantics: the selected indexes are pulled
// out of any other queued move for this fleet, then merged into an existing
// same-destination entry (or appended as a new one). An empty selection
// falls back to a whole-fleet move.
enqueue_fleet_move_ships :: proc(fleet_id, destination_star_id: int, ship_indexes: []int) {
	if len(ship_indexes) == 0 {
		enqueue_fleet_move(fleet_id, destination_star_id)
		return
	}
	cancel_colonize_fleet(fleet_id)

	selected: map[int]bool
	defer delete(selected)
	for idx in ship_indexes do selected[idx] = true

	// Remove the selected indexes from this fleet's other queued moves;
	// drop entries left empty. A queued whole-fleet move (no indexes) is
	// dropped outright — it's superseded by the explicit selection.
	for i := len(pending.fleet_move) - 1; i >= 0; i -= 1 {
		c := &pending.fleet_move[i]
		if c.fleet_id != fleet_id do continue
		if len(c.ship_indexes) == 0 {
			ordered_remove(&pending.fleet_move, i)
			continue
		}
		kept := make([dynamic]int)
		for idx in c.ship_indexes {
			if !selected[idx] do append(&kept, idx)
		}
		if len(kept) == 0 {
			delete(kept)
			ordered_remove(&pending.fleet_move, i)
		} else {
			c.ship_indexes = kept[:]
		}
	}

	// Merge into an existing same-destination entry if present.
	for &c in pending.fleet_move {
		if c.fleet_id == fleet_id && c.destination_star_id == destination_star_id {
			merged := make([dynamic]int)
			for idx in c.ship_indexes do append(&merged, idx)
			for idx in ship_indexes do append(&merged, idx)
			c.ship_indexes = merged[:]
			return
		}
	}

	idxs := make([]int, len(ship_indexes))
	copy(idxs, ship_indexes)
	append(&pending.fleet_move, Fleet_Move_Command{
		fleet_id            = fleet_id,
		destination_star_id = destination_star_id,
		ship_indexes        = idxs,
	})
}

// Ship indexes of `fleet_id` already committed to a queued move.
fleet_pending_move_indexes :: proc(fleet_id: int, allocator := context.allocator) -> []int {
	out := make([dynamic]int, allocator)
	for c in pending.fleet_move {
		if c.fleet_id != fleet_id do continue
		for idx in c.ship_indexes do append(&out, idx)
	}
	return out[:]
}

fleet_move_queued :: proc(fleet_id: int) -> bool {
	for c in pending.fleet_move {
		if c.fleet_id == fleet_id do return true
	}
	return false
}

// --- Colonize / artificial planet ----------------------------------------

// Queue a colonize. fleet_id == 0 means "from a completed colony base"
// (no ship involved); otherwise the colony ship fleet to consume. Replaces
// any queued colonize by the same fleet or onto the same planet, and clears
// the fleet's queued moves (the ship stays to colonize).
enqueue_colonize :: proc(planet_id: int, fleet_id := 0) {
	if fleet_id != 0 {
		for i := len(pending.fleet_move) - 1; i >= 0; i -= 1 {
			if pending.fleet_move[i].fleet_id == fleet_id {
				ordered_remove(&pending.fleet_move, i)
			}
		}
	}
	for i := len(pending.colonize_many) - 1; i >= 0; i -= 1 {
		c := pending.colonize_many[i]
		if (fleet_id != 0 && c.fleet_id == fleet_id) || c.planet_id == planet_id {
			ordered_remove(&pending.colonize_many, i)
		}
	}
	append(&pending.colonize_many, Colonize_Command{planet_id = planet_id, fleet_id = fleet_id})
}

colonize_queued_for_fleet :: proc(fleet_id: int) -> bool {
	if fleet_id == 0 do return false
	for c in pending.colonize_many {
		if c.fleet_id == fleet_id do return true
	}
	return false
}

colonize_queued_for_planet :: proc(planet_id: int) -> bool {
	for c in pending.colonize_many {
		if c.planet_id == planet_id do return true
	}
	return false
}

cancel_colonize_fleet :: proc(fleet_id: int) {
	if fleet_id == 0 do return
	for i := len(pending.colonize_many) - 1; i >= 0; i -= 1 {
		if pending.colonize_many[i].fleet_id == fleet_id {
			ordered_remove(&pending.colonize_many, i)
		}
	}
}

cancel_colonize_planet :: proc(planet_id: int) {
	for i := len(pending.colonize_many) - 1; i >= 0; i -= 1 {
		if pending.colonize_many[i].planet_id == planet_id {
			ordered_remove(&pending.colonize_many, i)
		}
	}
}

// Queue an artificial planet build onto `target_planet_id`, consuming the
// completed Artificial Planet project on `source_planet_id`. Replaces any
// queued command with the same source or target.
enqueue_artificial_planet :: proc(source_planet_id, target_planet_id: int) {
	for i := len(pending.artificial_planet_many) - 1; i >= 0; i -= 1 {
		c := pending.artificial_planet_many[i]
		if c.source_planet_id == source_planet_id || c.target_planet_id == target_planet_id {
			ordered_remove(&pending.artificial_planet_many, i)
		}
	}
	append(&pending.artificial_planet_many, Artificial_Planet_Command{
		source_planet_id = source_planet_id,
		target_planet_id = target_planet_id,
	})
}

artificial_planet_queued_for_target :: proc(target_planet_id: int) -> bool {
	for c in pending.artificial_planet_many {
		if c.target_planet_id == target_planet_id do return true
	}
	return false
}

artificial_planet_queued_for_source :: proc(source_planet_id: int) -> bool {
	for c in pending.artificial_planet_many {
		if c.source_planet_id == source_planet_id do return true
	}
	return false
}

cancel_artificial_planet_target :: proc(target_planet_id: int) {
	for i := len(pending.artificial_planet_many) - 1; i >= 0; i -= 1 {
		if pending.artificial_planet_many[i].target_planet_id == target_planet_id {
			ordered_remove(&pending.artificial_planet_many, i)
		}
	}
}

// --- Invade ---------------------------------------------------------------

// Queue an invasion of `target_planet_id` by the fleet's transports
// (`ship_indexes`). Replaces any queued invasion by the same fleet or onto
// the same planet, and clears the fleet's queued moves.
enqueue_invade :: proc(fleet_id, target_planet_id: int, ship_indexes: []int) {
	for i := len(pending.fleet_move) - 1; i >= 0; i -= 1 {
		if pending.fleet_move[i].fleet_id == fleet_id {
			ordered_remove(&pending.fleet_move, i)
		}
	}
	for i := len(pending.invade) - 1; i >= 0; i -= 1 {
		c := pending.invade[i]
		if c.fleet_id == fleet_id || c.target_planet_id == target_planet_id {
			ordered_remove(&pending.invade, i)
		}
	}
	idxs := make([]int, len(ship_indexes))
	copy(idxs, ship_indexes)
	append(&pending.invade, Invade_Command{
		fleet_id         = fleet_id,
		target_planet_id = target_planet_id,
		ship_indexes     = idxs,
	})
}

invade_queued_for_fleet :: proc(fleet_id: int) -> bool {
	for c in pending.invade {
		if c.fleet_id == fleet_id do return true
	}
	return false
}

cancel_invade_fleet :: proc(fleet_id: int) {
	for i := len(pending.invade) - 1; i >= 0; i -= 1 {
		if pending.invade[i].fleet_id == fleet_id {
			ordered_remove(&pending.invade, i)
		}
	}
}

// --- Research / mutation ---------------------------------------------------

set_select_research :: proc(tech_name: string) {
	pending.select_research = Select_Research_Command{tech_name = strings.clone(tech_name)}
}

queued_research_name :: proc() -> (string, bool) {
	if sr, ok := pending.select_research.?; ok do return sr.tech_name, true
	return "", false
}

clear_select_research :: proc() {
	pending.select_research = nil
}

set_mutate_race :: proc(trait_ids: []string) {
	ids := make([]string, len(trait_ids))
	for t, i in trait_ids do ids[i] = strings.clone(t)
	pending.mutate_race = Mutate_Race_Command{trait_ids = ids}
}

mutate_race_queued :: proc() -> bool {
	_, ok := pending.mutate_race.?
	return ok
}

clear_mutate_race :: proc() {
	pending.mutate_race = nil
}

// --- Population (move_population) ---------------------------------------
//
// The wire command carries absolute farmer/worker/scientist counts. We seed
// the queued allocation from the planet's current split and reallocate within
// it; the sum stays <= the planet's assignable units.

// The allocation to display/edit for a planet: the queued one if present,
// otherwise the planet's current split.
planet_allocation :: proc(p: Api_Planet) -> (farmers, workers, scientists: int) {
	for c in pending.move_population {
		if c.planet_id == p.id do return c.farmers, c.workers, c.scientists
	}
	return p.food.farmers, p.industry.workers, p.science.scientists
}

set_planet_allocation :: proc(planet_id, farmers, workers, scientists: int) {
	for &c in pending.move_population {
		if c.planet_id == planet_id {
			c.farmers, c.workers, c.scientists = farmers, workers, scientists
			return
		}
	}
	append(&pending.move_population, Move_Population_Command{
		planet_id = planet_id, farmers = farmers, workers = workers, scientists = scientists,
	})
}

// --- Construction (build) ----------------------------------------------

// The queued build list for a planet (current project first), if any.
planet_build_queue :: proc(planet_id: int) -> []string {
	for c in pending.build {
		if c.planet_id == planet_id do return c.build_queue
	}
	return nil
}

// Append a building to a planet's queued build list. The first entry is sent
// as the immediate project. Names are cloned so they survive snapshot churn.
enqueue_build :: proc(planet_id: int, building: string) {
	name := strings.clone(building)
	for &c in pending.build {
		if c.planet_id == planet_id {
			q := make([]string, len(c.build_queue) + 1)
			copy(q, c.build_queue)
			q[len(c.build_queue)] = name
			c.build_queue = q
			c.building = q[0]
			return
		}
	}
	q := make([]string, 1)
	q[0] = name
	append(&pending.build, Build_Command{planet_id = planet_id, building = name, build_queue = q})
}

clear_build :: proc(planet_id: int) {
	for i in 0 ..< len(pending.build) {
		if pending.build[i].planet_id == planet_id {
			ordered_remove(&pending.build, i)
			return
		}
	}
}

// --- Buy production (rush) ---------------------------------------------

buy_production_queued :: proc(planet_id: int) -> bool {
	for c in pending.buy_production {
		if c.planet_id == planet_id do return true
	}
	return false
}

toggle_buy_production :: proc(planet_id: int) {
	for i in 0 ..< len(pending.buy_production) {
		if pending.buy_production[i].planet_id == planet_id {
			ordered_remove(&pending.buy_production, i)
			return
		}
	}
	append(&pending.buy_production, Buy_Production_Command{planet_id = planet_id})
}

// Drop every queued command. Called after a turn is submitted successfully.
clear_pending :: proc() {
	clear(&pending.build)
	clear(&pending.move_population)
	clear(&pending.transfer_population)
	clear(&pending.colonize_many)
	clear(&pending.artificial_planet_many)
	clear(&pending.fleet_move)
	clear(&pending.invade)
	clear(&pending.buy_production)
	pending.select_research = nil
	pending.mutate_race = nil
}

// Assemble the full POST /turn body: {"game_id":...,"commands":{...}}.
// Only command kinds with queued entries are emitted. The allocator owns the
// returned bytes.
build_turn_body :: proc(game_id: string, allocator := context.allocator) -> []byte {
	b := strings.builder_make(allocator)
	strings.write_string(&b, `{"game_id":`)
	write_json_string(&b, game_id)
	strings.write_string(&b, `,"commands":{`)

	first := true
	emit_array(&b, &first, "build", pending.build[:])
	emit_array(&b, &first, "movePopulation", pending.move_population[:])
	emit_array(&b, &first, "transfer_population", pending.transfer_population[:])
	emit_array(&b, &first, "colonize_many", pending.colonize_many[:])
	emit_array(&b, &first, "artificial_planet_many", pending.artificial_planet_many[:])
	emit_array(&b, &first, "fleet_move", pending.fleet_move[:])
	emit_array(&b, &first, "invade", pending.invade[:])
	emit_array(&b, &first, "buy_production", pending.buy_production[:])
	if sr, ok := pending.select_research.?; ok do emit_value(&b, &first, "selectResearch", sr)
	if mr, ok := pending.mutate_race.?; ok do emit_value(&b, &first, "mutate_race", mr)

	strings.write_string(&b, "}}")
	return transmute([]byte)strings.to_string(b)
}

@(private = "file")
emit_array :: proc(b: ^strings.Builder, first: ^bool, key: string, items: []$T) {
	if len(items) == 0 do return
	emit_raw(b, first, key, items)
}

@(private = "file")
emit_value :: proc(b: ^strings.Builder, first: ^bool, key: string, item: $T) {
	emit_raw(b, first, key, item)
}

@(private = "file")
emit_raw :: proc(b: ^strings.Builder, first: ^bool, key: string, value: $T) {
	if !first^ do strings.write_byte(b, ',')
	first^ = false
	write_json_string(b, key)
	strings.write_byte(b, ':')
	data, err := json.marshal(value, allocator = context.temp_allocator)
	if err != nil {
		fmt.eprintfln("commands: marshal %s failed: %v", key, err)
		strings.write_string(b, "null")
		return
	}
	strings.write_bytes(b, data)
}

// Minimal JSON string escaper — enough for game ids and our fixed keys.
@(private = "file")
write_json_string :: proc(b: ^strings.Builder, s: string) {
	strings.write_byte(b, '"')
	for r in s {
		switch r {
		case '"':  strings.write_string(b, `\"`)
		case '\\': strings.write_string(b, `\\`)
		case '\n': strings.write_string(b, `\n`)
		case '\r': strings.write_string(b, `\r`)
		case '\t': strings.write_string(b, `\t`)
		case:      strings.write_rune(b, r)
		}
	}
	strings.write_byte(b, '"')
}
