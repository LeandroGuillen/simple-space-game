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
// a second one.
enqueue_fleet_move :: proc(fleet_id, destination_star_id: int) {
	for &c in pending.fleet_move {
		if c.fleet_id == fleet_id {
			c.destination_star_id = destination_star_id
			return
		}
	}
	append(&pending.fleet_move, Fleet_Move_Command{
		fleet_id            = fleet_id,
		destination_star_id = destination_star_id,
	})
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
