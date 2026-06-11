package main

import "core:fmt"

import rl "vendor:raylib"

// Phase 3: Evolutionary Mutation. A one-shot re-pick of race traits with a
// 14-point budget (10 base + 4 from the tech). The trait table mirrors
// client-solid/src/lib/race-design.ts (enabled traits only); the server
// re-validates the submitted set.

MUTATION_PICK_BUDGET :: 14
NEGATIVE_PICK_LIMIT :: 10

Trait_Def :: struct {
	id:    string,
	name:  string,
	cost:  int,
	group: string, // exclusive group ("" = stackable)
}

trait_defs := [?]Trait_Def{
	{"growth_minus_50", "-50% Growth", -4, "growth"},
	{"growth_plus_50", "+50% Growth", 3, "growth"},
	{"growth_plus_100", "+100% Growth", 6, "growth"},
	{"food_minus_1", "-1/2 Food", -3, "food"},
	{"food_plus_1", "+1 Food", 4, "food"},
	{"food_plus_2", "+2 Food", 7, "food"},
	{"production_minus_1", "-1 Production", -3, "production"},
	{"production_plus_1", "+1 Production", 3, "production"},
	{"production_plus_2", "+2 Production", 6, "production"},
	{"research_minus_1", "-1 Research", -3, "research"},
	{"research_plus_1", "+1 Research", 3, "research"},
	{"research_plus_2", "+2 Research", 6, "research"},
	{"money_minus_half", "-0.5 BC", -4, "money"},
	{"money_plus_half", "+0.5 BC", 5, "money"},
	{"money_plus_1", "+1 BC", 8, "money"},
	{"ship_defense", "+20% Ship Defense", 3, ""},
	{"ship_attack", "+20% Ship Attack", 3, ""},
	{"ground_combat", "Ground Combat", 5, ""},
	{"spying", "Spying", 3, ""},
	{"low_g_world", "Low-G World", -5, "gravity"},
	{"high_g_world", "High-G World", 6, "gravity"},
	{"aquatic", "Aquatic", 5, ""},
	{"subterranean", "Subterranean", 6, "capacity"},
	{"large_homeworld", "Large Home World", 1, "homeworld_size"},
	{"rich_homeworld", "Rich Home World", 2, "homeworld_minerals"},
	{"poor_homeworld", "Poor Home World", -1, "homeworld_minerals"},
	{"artifacts_homeworld", "Artifacts World", 3, "homeworld_artifacts"},
	{"cybernetic", "Cybernetic", 4, ""},
	{"lithovore", "Lithovore", 10, ""},
	{"uncreative", "Uncreative", -4, "creativity"},
	{"creative", "Creative", 8, "creativity"},
	{"tolerant", "Tolerant", 10, ""},
	{"fantastic_traders", "Fantastic Traders", 4, ""},
	{"lucky", "Lucky", 3, ""},
	{"omniscient", "Omniscient", 3, ""},
	{"stealthy_ships", "Stealthy Ships", 4, ""},
	{"trans_dimensional", "Trans-Dimensional", 5, ""},
	{"warlord", "Warlord", 4, ""},
}

// Government pick costs (fallbackGovernments in race-design.ts); they count
// against the same budget as traits.
government_pick_cost :: proc(government: string) -> int {
	switch government {
	case "Feudal":      return -4
	case "Dictatorship": return 0
	case "Democracy":   return 7
	case "Unification": return 6
	}
	return 0
}

// Whether the player can still perform their one-shot mutation (drives the
// bottom-bar Mutate button's visibility).
mutation_available :: proc(p: Api_Player) -> bool {
	return p.evolutionary_mutation_points > 0 && !p.has_used_evolutionary_mutation
}

// --- Mutation editor state ---------------------------------------------

mutation_editing: bool
mutation_traits: [dynamic]string

mutation_begin_edit :: proc(p: Api_Player) {
	mutation_editing = true
	clear(&mutation_traits)
	for t in p.race.trait_ids do append(&mutation_traits, t)
}

mutation_has :: proc(id: string) -> bool {
	for t in mutation_traits {
		if t == id do return true
	}
	return false
}

mutation_toggle :: proc(def: Trait_Def) {
	for t, i in mutation_traits {
		if t == def.id {
			ordered_remove(&mutation_traits, i)
			return
		}
	}
	if def.group != "" {
		// Exclusive group: drop any sibling first.
		for other in trait_defs {
			if other.group == def.group && other.id != def.id {
				for t, i in mutation_traits {
					if t == other.id {
						ordered_remove(&mutation_traits, i)
						break
					}
				}
			}
		}
	}
	append(&mutation_traits, def.id)
}

mutation_pick_cost :: proc(government: string) -> (used, negative: int) {
	used = government_pick_cost(government)
	negative = max(0, -used)
	for t in mutation_traits {
		for def in trait_defs {
			if def.id == t {
				used += def.cost
				if def.cost < 0 do negative += -def.cost
				break
			}
		}
	}
	return
}

draw_mutation_tab :: proc(g: Api_Game, cx, cy_in: i32, player_id: int) -> i32 {
	cy := cy_in

	player, ok := player_by_id(g, player_id)
	if !ok {
		rl.DrawText("(player not found)", cx, cy, pxi(14), rl.RED)
		return cy + pxi(18)
	}

	if mutate_race_queued() {
		rl.DrawText("Mutation queued for next turn", cx, cy, pxi(13), {140, 200, 150, 255}); cy += pxi(18)
		if gui_button({f32(cx), f32(cy), f32(pxi(160)), f32(pxi(22))}, "Cancel Mutation") {
			clear_mutate_race()
		}
		return cy + pxi(26)
	}

	if player.has_used_evolutionary_mutation {
		rl.DrawText("Already used (one-time only).", cx, cy, pxi(13), rl.GRAY)
		return cy + pxi(18)
	}
	if player.evolutionary_mutation_points <= 0 {
		rl.DrawText("Locked. Research Evolutionary", cx, cy, pxi(13), rl.GRAY); cy += pxi(16)
		rl.DrawText("Mutation to unlock.", cx, cy, pxi(13), rl.GRAY)
		return cy + pxi(18)
	}

	if !mutation_editing {
		rl.DrawText(fmt.ctprintf("Race: %s (%s)", player.race.name, player.race.government), cx, cy, pxi(13), rl.LIGHTGRAY); cy += pxi(18)
		rl.DrawText("One-time re-pick of race traits", cx, cy, pxi(12), rl.GRAY); cy += pxi(15)
		rl.DrawText(fmt.ctprintf("with a %d-point budget.", MUTATION_PICK_BUDGET), cx, cy, pxi(12), rl.GRAY); cy += pxi(20)
		if gui_button({f32(cx), f32(cy), f32(pxi(160)), f32(pxi(24))}, "Edit Traits") {
			mutation_begin_edit(player)
		}
		return cy + pxi(28)
	}

	// --- Editor ---
	used, negative := mutation_pick_cost(player.race.government)
	remaining := MUTATION_PICK_BUDGET - used
	valid := remaining >= 0 && negative <= NEGATIVE_PICK_LIMIT

	rem_col := remaining < 0 ? rl.Color{210, 130, 120, 255} : rl.Color{140, 200, 150, 255}
	rl.DrawText(fmt.ctprintf("Budget %d  used %d  left %d", MUTATION_PICK_BUDGET, used, remaining), cx, cy, pxi(13), rem_col); cy += pxi(16)
	neg_col := negative > NEGATIVE_PICK_LIMIT ? rl.Color{210, 130, 120, 255} : rl.Color{150, 165, 190, 255}
	rl.DrawText(fmt.ctprintf("Negative picks %d/%d", negative, NEGATIVE_PICK_LIMIT), cx, cy, pxi(12), neg_col); cy += pxi(18)

	if gui_button({f32(cx), f32(cy), f32(pxi(104)), f32(pxi(22))}, "Apply", valid) {
		set_mutate_race(mutation_traits[:])
		mutation_editing = false
	}
	if gui_button({f32(cx + pxi(112)), f32(cy), f32(pxi(104)), f32(pxi(22))}, "Cancel") {
		mutation_editing = false
	}
	cy += pxi(28)

	for def in trait_defs {
		row := rl.Rectangle{f32(cx), f32(cy), f32(pxi(268)), f32(pxi(16))}
		hover := ui_hover(row)
		selected := mutation_has(def.id)
		if selected do rl.DrawRectangleRec(row, {44, 60, 90, 255})
		else if hover do rl.DrawRectangleRec(row, {34, 42, 60, 255})

		mark: string = selected ? "[x]" : "[ ]"
		rl.DrawText(fmt.ctprintf("%s %s", mark, def.name), cx, cy, pxi(12), selected ? rl.YELLOW : rl.LIGHTGRAY)
		cost_col := def.cost < 0 ? rl.Color{140, 200, 150, 255} : rl.Color{210, 170, 120, 255}
		rl.DrawText(fmt.ctprintf("%+d", def.cost), cx + pxi(240), cy, pxi(12), cost_col)
		if hover && rl.IsMouseButtonPressed(.LEFT) do mutation_toggle(def)
		cy += pxi(16)
	}
	return cy
}
