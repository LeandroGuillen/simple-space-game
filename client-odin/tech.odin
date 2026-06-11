package main

import "core:fmt"
import "core:math"
import "core:strings"

import rl "vendor:raylib"

// Phase 3: research logic + the Technology tab. Mirrors the research-choice
// derivation in client-solid/src/components/AppShell.tsx: per subject, the
// first incomplete level provides the choices; Creative/force_all levels are
// taken whole; finished subjects offer "Hyper Advanced" repeats.

Research_Choice :: struct {
	subject:  string,
	level:    string,   // level display name
	cost:     int,      // RP to complete the level
	command:  string,   // tech_name to submit in selectResearch
	label:    string,   // row label
	techs:    []string, // techs granted (>1 for force_all/Creative levels)
	is_hyper: bool,
}

player_by_id :: proc(g: Api_Game, player_id: int) -> (Api_Player, bool) {
	for p in g.players {
		if p.id == player_id do return p, true
	}
	return {}, false
}

player_has_trait :: proc(p: Api_Player, trait_id: string) -> bool {
	for t in p.race.trait_ids {
		if t == trait_id do return true
	}
	return false
}

tech_known :: proc(p: Api_Player, name: string) -> bool {
	for t in p.researched_techs {
		if strings.equal_fold(t, name) do return true
	}
	return false
}

// True when completing the level grants every tech at that breakthrough
// (Creative race or force_all level; Uncreative overrides both).
level_unlocks_all :: proc(p: Api_Player, level: Api_Tech_Level) -> bool {
	if player_has_trait(p, "uncreative") do return false
	return level.force_all || player_has_trait(p, "creative")
}

level_complete :: proc(p: Api_Player, level: Api_Tech_Level) -> bool {
	if len(level.available_techs) == 0 do return false
	if !level_unlocks_all(p, level) {
		for t in level.available_techs {
			if tech_known(p, t) do return true
		}
		return false
	}
	for t in level.available_techs {
		if !tech_known(p, t) do return false
	}
	return true
}

find_tech :: proc(g: Api_Game, name: string) -> (Api_Tech, bool) {
	for t in g.resources.tech_collection.techs {
		if strings.equal_fold(t.name, name) do return t, true
	}
	return {}, false
}

// Government techs are only researchable as upgrades of the player's active
// government (mirrors isResearchChoiceAllowedForGovernment, minus the
// Unification morale filter — the server validates the rest).
research_allowed_for_government :: proc(g: Api_Game, p: Api_Player, tech_name: string) -> bool {
	tech, ok := find_tech(g, tech_name)
	if !ok do return true
	if tech.type != "government" do return true
	active := active_government(g, p)
	for up in tech.upgrades {
		if up == active do return true
	}
	return false
}

// The player's current government: the race's, upgraded along any researched
// government techs (each names its predecessors in `upgrades`).
active_government :: proc(g: Api_Game, p: Api_Player) -> string {
	active := p.race.government
	if active == "" {
		for t in g.resources.tech_collection.techs {
			if t.type == "government" && t.default {
				active = t.name
				break
			}
		}
	}
	for steps := 0; steps < 8; steps += 1 { // bounded walk; chains are short
		advanced := false
		for t in g.resources.tech_collection.techs {
			if t.type != "government" || !tech_known(p, t.name) do continue
			for up in t.upgrades {
				if up == active && t.name != active {
					active = t.name
					advanced = true
					break
				}
			}
			if advanced do break
		}
		if !advanced do break
	}
	return active
}

// All research projects the player may start right now.
research_choices :: proc(g: Api_Game, p: Api_Player, allocator := context.allocator) -> []Research_Choice {
	out := make([dynamic]Research_Choice, allocator)
	for subject in g.resources.tech_tree.subjects {
		level_found := false
		for level in subject.levels {
			if level_complete(p, level) do continue
			level_found = true
			if level_unlocks_all(p, level) {
				// One pick researches the whole breakthrough; list every
				// tech it grants (the web client does the same).
				techs := make([dynamic]string, allocator)
				for t in level.available_techs {
					if research_allowed_for_government(g, p, t) do append(&techs, t)
				}
				append(&out, Research_Choice{
					subject = subject.name, level = level.name, cost = level.cost,
					command = level.name, label = level.name, techs = techs[:],
				})
			} else {
				for tech in level.available_techs {
					if tech_known(p, tech) do continue
					if !research_allowed_for_government(g, p, tech) do continue
					append(&out, Research_Choice{
						subject = subject.name, level = level.name, cost = level.cost,
						command = tech, label = tech,
					})
				}
			}
			break
		}
		if !level_found && len(subject.levels) > 0 {
			// Subject finished: repeatable Hyper Advanced levels, doubling
			// in cost each time.
			done := p.hyper_advanced[subject.name]
			next := done + 1
			cost := subject.levels[len(subject.levels) - 1].cost * int(math.pow(2, f64(next)))
			append(&out, Research_Choice{
				subject  = subject.name,
				level    = fmt.aprintf("Hyper Advanced %d", next, allocator = allocator),
				cost     = cost,
				command  = fmt.aprintf("Hyper Advanced %s", subject.name, allocator = allocator),
				label    = fmt.aprintf("Hyper Advanced %d", next, allocator = allocator),
				is_hyper = true,
			})
		}
	}
	return out[:]
}

// Empire research output per turn (scientists + buildings on owned planets).
research_per_turn :: proc(g: Api_Game, player_id: int) -> int {
	total := 0
	for s in g.galaxy.stars {
		for p in s.planets {
			if p.owner != player_id do continue
			total += p.science.scientists * p.science.produced_per_scientist + p.science.produced_per_building
		}
	}
	return total
}

// The level a named breakthrough belongs to (by level name or member tech).
find_level_for :: proc(g: Api_Game, name: string) -> (Api_Tech_Level, bool) {
	if name == "" do return {}, false
	for subject in g.resources.tech_tree.subjects {
		for level in subject.levels {
			if strings.equal_fold(level.name, name) do return level, true
			for t in level.available_techs {
				if strings.equal_fold(t, name) do return level, true
			}
		}
	}
	return {}, false
}

// --- Top-bar Research button + popup picker -------------------------------

research_popup_open: bool
research_scroll: i32
research_content_h: i32
research_btn_rect: rl.Rectangle

// The active project (queued name wins over the server's current one),
// empire RP/turn, and the level cost/progress for the ETA.
research_status :: proc(g: Api_Game, player_id: int) -> (rpt: int, name: string, progress, cost: int, has: bool) {
	rpt = research_per_turn(g, player_id)
	p, ok := player_by_id(g, player_id)
	if !ok do return
	queued, has_q := queued_research_name()
	cur := p.current_research
	name = has_q ? queued : (cur.tech_name != "" ? cur.tech_name : (cur.level_name != "" ? cur.level_name : cur.subject))
	if name == "" do return
	has = true
	progress = cur.progress
	if level, lok := find_level_for(g, name); lok do cost = level.cost
	return
}

// Two-line button in the top bar: "Research" / "<N>RP ~<M> turns". Checks
// hover manually (not ui_hover) so it stays clickable while its own popup is
// open — clicking it again closes the popup.
draw_research_button :: proc(g: Api_Game, player_id: int) -> bool {
	th := topbar_h()
	r := rl.Rectangle{px(180), px(5), px(190), f32(th) - px(10)}
	research_btn_rect = r

	hover := rl.CheckCollisionPointRec(rl.GetMousePosition(), r)
	bg := research_popup_open ? rl.Color{52, 96, 150, 255} : (hover ? rl.Color{40, 50, 70, 255} : rl.Color{24, 30, 44, 255})
	rl.DrawRectangleRec(r, bg)
	rl.DrawRectangleLinesEx(r, max(1, px(1)), {70, 84, 110, 255})

	rpt, _, progress, cost, has := research_status(g, player_id)
	rl.DrawText("Research", i32(r.x) + pxi(10), i32(r.y) + pxi(4), pxi(13), rl.RAYWHITE)
	line2: cstring
	col2 := rl.Color{150, 165, 190, 255}
	if has && rpt > 0 && cost > 0 {
		turns := max(1, (max(0, cost - progress) + rpt - 1) / rpt)
		line2 = fmt.ctprintf("%dRP ~%d turns", rpt, turns)
	} else if has {
		line2 = fmt.ctprintf("%dRP", rpt)
	} else {
		line2 = fmt.ctprintf("%dRP - none!", rpt)
		col2 = {210, 170, 120, 255}
	}
	rl.DrawText(line2, i32(r.x) + pxi(10), i32(r.y) + pxi(21), pxi(12), col2)
	return hover && rl.IsMouseButtonPressed(.LEFT)
}

// Modal research picker: all startable projects grouped by subject; clicking
// one queues selectResearch and closes the popup. Click outside (or Close,
// Esc, T, or the Research button) dismisses it.
draw_research_popup :: proc(g: Api_Game, player_id: int) {
	r := modal_rect(680)
	sw := rl.GetScreenWidth()
	sh := rl.GetScreenHeight()
	rl.DrawRectangle(0, 0, sw, sh, {0, 0, 0, 130})
	rl.DrawRectangleRec(r, {18, 22, 34, 255})
	rl.DrawRectangleLinesEx(r, max(1, px(1)), {70, 84, 110, 255})

	cx := i32(r.x) + pxi(18)
	cy := i32(r.y) + pxi(14)
	rl.DrawText("Research", cx, cy, pxi(18), rl.RAYWHITE)
	if gui_button({r.x + r.width - px(86), r.y + px(12), px(70), px(26)}, "Close") {
		research_popup_open = false
	}
	cy += pxi(28)

	player, ok := player_by_id(g, player_id)
	if !ok do return

	rpt, name, progress, cost, has := research_status(g, player_id)
	queued, has_q := queued_research_name()
	if has {
		col := has_q ? rl.Color{200, 180, 120, 255} : rl.Color{140, 200, 150, 255}
		suffix: string = has_q ? "  [queued]" : ""
		if cost > 0 && rpt > 0 {
			turns := max(1, (max(0, cost - progress) + rpt - 1) / rpt)
			rl.DrawText(fmt.ctprintf("Current: %s  (%d/%d RP, ~%d turns)%s", name, progress, cost, turns, suffix), cx, cy, pxi(13), col)
		} else {
			rl.DrawText(fmt.ctprintf("Current: %s%s", name, suffix), cx, cy, pxi(13), col)
		}
	} else {
		rl.DrawText(fmt.ctprintf("No research selected  (+%d RP/turn)", rpt), cx, cy, pxi(13), {210, 130, 120, 255})
	}
	cy += pxi(24)

	// --- Scrollable choice list ---
	content_top := cy
	content_bottom := i32(r.y + r.height) - pxi(14)
	region_h := max(content_bottom - content_top, pxi(40))
	max_scroll := max(0, research_content_h - region_h)
	research_scroll = clamp(research_scroll, 0, max_scroll)

	rl.BeginScissorMode(i32(r.x), content_top, i32(r.width), region_h)
	ui_clip_on = true
	ui_clip = {r.x, f32(content_top), r.width, f32(region_h)}
	start_cy := content_top - research_scroll
	yy := start_cy
	choices := research_choices(g, player, context.temp_allocator)
	last_subject := ""
	for c in choices {
		if c.subject != last_subject {
			last_subject = c.subject
			rl.DrawText(fmt.ctprintf("%s - %s (%d RP)", c.subject, c.level, c.cost), cx, yy, pxi(13), {130, 150, 180, 255})
			yy += pxi(18)
		}
		// Force-all levels grant several techs with one pick: show them
		// under the level row, inside the same click target.
		row_h := pxi(17)
		if len(c.techs) > 1 do row_h += pxi(15)
		row := rl.Rectangle{f32(cx), f32(yy), r.width - px(36), f32(row_h)}
		hover := ui_hover(row)
		is_queued := has_q && strings.equal_fold(queued, c.command)
		if is_queued do rl.DrawRectangleRec(row, {44, 60, 90, 255})
		else if hover do rl.DrawRectangleRec(row, {34, 42, 60, 255})
		rl.DrawText(fmt.ctprintf("  %s", c.label), cx, yy + pxi(2), pxi(12), is_queued ? rl.YELLOW : rl.LIGHTGRAY)
		if len(c.techs) > 1 {
			granted := strings.join(c.techs, ", ", context.temp_allocator)
			rl.DrawText(fmt.ctprintf("    grants: %s", granted), cx, yy + pxi(18), pxi(11), {130, 145, 170, 255})
		}
		if hover && rl.IsMouseButtonPressed(.LEFT) {
			set_select_research(c.command)
			research_popup_open = false
		}
		yy += row_h
	}
	if len(choices) == 0 {
		rl.DrawText("(nothing left to research)", cx, yy, pxi(13), rl.GRAY)
		yy += pxi(18)
	}
	ui_clip_on = false
	rl.EndScissorMode()
	research_content_h = yy - start_cy

	if research_content_h > region_h {
		track_x := i32(r.x + r.width) - pxi(6)
		bar_h := max(region_h * region_h / research_content_h, pxi(24))
		bar_y := content_top + (region_h - bar_h) * research_scroll / max(1, max_scroll)
		rl.DrawRectangle(track_x, content_top, pxi(3), region_h, {30, 36, 50, 255})
		rl.DrawRectangle(track_x, bar_y, pxi(3), bar_h, {90, 104, 130, 255})
	}

	// Click outside dismisses (the Research button handles its own toggle).
	if rl.IsMouseButtonPressed(.LEFT) {
		mp := rl.GetMousePosition()
		if !rl.CheckCollisionPointRec(mp, r) && !rl.CheckCollisionPointRec(mp, research_btn_rect) {
			research_popup_open = false
		}
	}
}
