package main

import "core:fmt"
import "core:sync"

import rl "vendor:raylib"

// Galaxy view + planet panel + input. Pulls game state from `state` which
// is updated by the WebSocket reader thread under `state_mu`.

WINDOW_W :: 1280
WINDOW_H :: 800
PANEL_W  :: 320

// Galaxy state shared between WS reader thread and main thread.
// `current` is replaced atomically (under the mutex) each time a new
// snapshot arrives. We never mutate fields of an existing snapshot — the
// reader always installs a freshly-allocated Api_Game.
Galaxy_State :: struct {
	game:     Api_Game,
	loaded:   bool,
	arena:    [dynamic]byte, // reserved for future arena-backed allocator usage
}

state:    Galaxy_State
state_mu: sync.Mutex

ws_set_game :: proc(g: Api_Game) {
	sync.lock(&state_mu)
	state.game = g
	state.loaded = true
	sync.unlock(&state_mu)
}

ws_snapshot :: proc() -> (Api_Game, bool) {
	sync.lock(&state_mu)
	defer sync.unlock(&state_mu)
	return state.game, state.loaded
}

Selection :: struct {
	star_id:  int,
	has_star: bool,
	fleet_id: int,
	has_fleet:bool,
}

run_ui :: proc(client: ^Api_Client, game_id_arg: string) {
	// Pick a game: either the arg, or the first in the list.
	game_id := game_id_arg
	if game_id == "" {
		games, ok := api_get_games(client)
		if !ok || len(games) == 0 {
			fmt.eprintln("ui: no game to join (pass game-id as 2nd arg)")
			return
		}
		game_id = games[0].game_id
		fmt.printfln("ui: auto-joining game %s", game_id)
	}

	// Initial full snapshot via REST so we have something to render
	// before the first WS frame lands.
	if game, ok := api_get_game(client, game_id); ok {
		ws_set_game(game)
	} else {
		fmt.eprintfln("ui: failed to GET /games/%s", game_id)
		return
	}

	// Spawn WS reader (M4). It mutates `state` under state_mu.
	ws_start(client, game_id)

	rl.InitWindow(WINDOW_W, WINDOW_H, "SSG — Odin Client")
	defer rl.CloseWindow()
	rl.SetTargetFPS(60)

	camera := rl.Camera2D {
		zoom   = 1.0,
		offset = {WINDOW_W * 0.5, WINDOW_H * 0.5},
	}
	camera_initialized := false

	sel: Selection
	last_mouse: rl.Vector2

	for !rl.WindowShouldClose() {
		game, loaded := ws_snapshot()

		if loaded && !camera_initialized {
			// Center camera on the galaxy bounding box and pick a zoom
			// that fits all stars roughly within the visible area
			// (minus the side panel).
			cx, cy, span := galaxy_extent(game)
			camera.target = {f32(cx), f32(cy)}
			vis_w := f32(WINDOW_W - PANEL_W)
			vis_h := f32(WINDOW_H)
			camera.offset = {vis_w * 0.5, vis_h * 0.5}
			if span > 0 {
				fit := min(vis_w, vis_h) / (f32(span) * 1.2)
				camera.zoom = fit
			}
			camera_initialized = true
		}

		// Pan with right mouse drag, zoom with wheel
		if rl.IsMouseButtonPressed(.RIGHT) do last_mouse = rl.GetMousePosition()
		if rl.IsMouseButtonDown(.RIGHT) {
			m := rl.GetMousePosition()
			dx := (m.x - last_mouse.x) / camera.zoom
			dy := (m.y - last_mouse.y) / camera.zoom
			camera.target.x -= dx
			camera.target.y -= dy
			last_mouse = m
		}
		wheel := rl.GetMouseWheelMove()
		if wheel != 0 {
			// Zoom around the mouse cursor.
			world_before := rl.GetScreenToWorld2D(rl.GetMousePosition(), camera)
			camera.zoom *= 1 + wheel * 0.1
			camera.zoom = clamp(camera.zoom, 0.05, 50.0)
			world_after := rl.GetScreenToWorld2D(rl.GetMousePosition(), camera)
			camera.target.x += world_before.x - world_after.x
			camera.target.y += world_before.y - world_after.y
		}

		// Left click: select nearest star (M5) or, if a fleet is already
		// selected, target a star for movement (M6).
		if rl.IsMouseButtonPressed(.LEFT) {
			mp := rl.GetMousePosition()
			if mp.x < f32(WINDOW_W - PANEL_W) && loaded {
				world := rl.GetScreenToWorld2D(mp, camera)
				star_id, hit := pick_star(game, world, 18.0 / camera.zoom)
				if hit {
					if sel.has_fleet {
						// Send a fleet_move command and end the turn for this player.
						cmds := Turn_Commands_Body {
							fleet_move = []Fleet_Move_Command{
								{
									fleet_id            = sel.fleet_id,
									destination_star_id = star_id,
									ship_indexes        = []int{},
								},
							},
						}
						if resp, ok := api_end_turn(client, game_id, cmds); ok {
							fmt.printfln("turn: ok=%v advanced=%v", resp.ok, resp.advanced)
						}
						sel.has_fleet = false
					} else {
						sel.star_id = star_id
						sel.has_star = true
					}
				}
			}
		}

		// Number keys 1..9 select that fleet from the current player's
		// fleet list (super crude but enough for M6 demo).
		if loaded {
			for n in 1 ..= 9 {
				key := rl.KeyboardKey(int(rl.KeyboardKey.ZERO) + n)
				if rl.IsKeyPressed(key) {
					my_fleets := player_fleets(game, client.player_id, context.temp_allocator)
					if n - 1 < len(my_fleets) {
						sel.fleet_id = my_fleets[n - 1].id
						sel.has_fleet = true
						fmt.printfln("selected fleet %d", sel.fleet_id)
					}
				}
			}
		}

		// --- Draw ---
		rl.BeginDrawing()
		rl.ClearBackground({8, 10, 18, 255})

		rl.BeginMode2D(camera)
		if loaded {
			draw_galaxy(game)
			if sel.has_star do draw_star_highlight(game, sel.star_id)
		} else {
			rl.DrawText("loading...", -50, 0, 24, rl.RAYWHITE)
		}
		rl.EndMode2D()

		// Side panel (screen space)
		draw_panel(game, loaded, sel, client.player_id)

		rl.EndDrawing()

		free_all(context.temp_allocator)
	}
}

galaxy_extent :: proc(g: Api_Game) -> (cx, cy, span: f64) {
	if len(g.galaxy.stars) == 0 do return 0, 0, 1
	min_x, min_y, max_x, max_y := g.galaxy.stars[0].x, g.galaxy.stars[0].y, g.galaxy.stars[0].x, g.galaxy.stars[0].y
	for s in g.galaxy.stars {
		if s.x < min_x do min_x = s.x
		if s.y < min_y do min_y = s.y
		if s.x > max_x do max_x = s.x
		if s.y > max_y do max_y = s.y
	}
	cx = (min_x + max_x) / 2
	cy = (min_y + max_y) / 2
	span = max(max_x - min_x, max_y - min_y)
	if span < 1 do span = 1
	return
}

pick_star :: proc(g: Api_Game, world: rl.Vector2, threshold: f32) -> (int, bool) {
	best_d := threshold * threshold
	best_id := 0
	found := false
	for s in g.galaxy.stars {
		dx := f32(s.x) - world.x
		dy := f32(s.y) - world.y
		d := dx * dx + dy * dy
		if d < best_d {
			best_d = d
			best_id = s.id
			found = true
		}
	}
	return best_id, found
}

draw_galaxy :: proc(g: Api_Game) {
	// Stars first
	for s in g.galaxy.stars {
		c := star_color(s.color)
		rl.DrawCircleV({f32(s.x), f32(s.y)}, 6, c)
	}
	// Fleets
	for f in g.fleets {
		col := player_color(g, f.owner)
		rl.DrawRectangleV({f32(f.x) - 3, f32(f.y) - 3}, {6, 6}, col)
		if f.destination_star_id != 0 {
			for s in g.galaxy.stars {
				if s.id == f.destination_star_id {
					rl.DrawLineV(
						{f32(f.x), f32(f.y)},
						{f32(s.x), f32(s.y)},
						{200, 200, 200, 90},
					)
					break
				}
			}
		}
	}
}

draw_star_highlight :: proc(g: Api_Game, id: int) {
	for s in g.galaxy.stars {
		if s.id == id {
			rl.DrawCircleLinesV({f32(s.x), f32(s.y)}, 12, rl.YELLOW)
			return
		}
	}
}

star_color :: proc(code: int) -> rl.Color {
	// The server's `color` is an enum-ish int from the GalaxyMap palette.
	// Match the rough buckets used by the Solid client. Anything we don't
	// recognise falls back to a neutral grey.
	switch code {
	case 0:  return {220, 240, 255, 255} // blue-white
	case 1:  return {255, 255, 220, 255} // yellow
	case 2:  return {255, 200, 120, 255} // orange
	case 3:  return {255, 110, 100, 255} // red
	case 4:  return {200, 220, 255, 255} // blue
	case 5:  return {255, 240, 200, 255} // white
	case 6:  return {180, 130, 255, 255} // purple
	case:    return {200, 200, 200, 255}
	}
}

player_color :: proc(g: Api_Game, owner: int) -> rl.Color {
	for p in g.players {
		if p.id == owner do return parse_hex_color(p.color)
	}
	return {150, 150, 150, 255}
}

parse_hex_color :: proc(s: string) -> rl.Color {
	// Accepts "#RRGGBB" or "RRGGBB". Falls back to white on parse errors.
	x := s
	if len(x) > 0 && x[0] == '#' do x = x[1:]
	if len(x) != 6 do return rl.WHITE
	hex :: proc(c: byte) -> u8 {
		switch c {
		case '0' ..= '9': return u8(c - '0')
		case 'a' ..= 'f': return u8(c - 'a' + 10)
		case 'A' ..= 'F': return u8(c - 'A' + 10)
		}
		return 0
	}
	r := hex(x[0]) * 16 + hex(x[1])
	g := hex(x[2]) * 16 + hex(x[3])
	b := hex(x[4]) * 16 + hex(x[5])
	return {r, g, b, 255}
}

player_fleets :: proc(g: Api_Game, player_id: int, allocator := context.allocator) -> []Api_Fleet {
	out := make([dynamic]Api_Fleet, 0, len(g.fleets), allocator)
	for f in g.fleets {
		if f.owner == player_id do append(&out, f)
	}
	return out[:]
}

draw_panel :: proc(g: Api_Game, loaded: bool, sel: Selection, player_id: int) {
	x := i32(WINDOW_W - PANEL_W)
	rl.DrawRectangle(x, 0, PANEL_W, WINDOW_H, {18, 22, 34, 255})
	rl.DrawLine(x, 0, x, WINDOW_H, {60, 70, 90, 255})

	cx := x + 16
	cy: i32 = 16
	rl.DrawText("SSG — Odin Client", cx, cy, 20, rl.RAYWHITE)
	cy += 30

	if !loaded {
		rl.DrawText("waiting for snapshot...", cx, cy, 16, rl.GRAY)
		return
	}

	turn_line := fmt.ctprintf("turn: %d   players: %d", g.current_turn, len(g.players))
	rl.DrawText(turn_line, cx, cy, 16, rl.LIGHTGRAY)
	cy += 22

	stars_line := fmt.ctprintf("stars: %d   fleets: %d", len(g.galaxy.stars), len(g.fleets))
	rl.DrawText(stars_line, cx, cy, 16, rl.LIGHTGRAY)
	cy += 30

	rl.DrawText("controls:", cx, cy, 14, {180, 200, 220, 255})
	cy += 18
	rl.DrawText("- right-drag to pan", cx, cy, 14, rl.GRAY); cy += 16
	rl.DrawText("- wheel to zoom", cx, cy, 14, rl.GRAY); cy += 16
	rl.DrawText("- left-click to select star", cx, cy, 14, rl.GRAY); cy += 16
	rl.DrawText("- 1..9 to select your fleet", cx, cy, 14, rl.GRAY); cy += 16
	rl.DrawText("- with fleet selected,", cx, cy, 14, rl.GRAY); cy += 16
	rl.DrawText("  click a star to move + end turn", cx, cy, 14, rl.GRAY); cy += 24

	if sel.has_fleet {
		line := fmt.ctprintf("selected fleet: %d", sel.fleet_id)
		rl.DrawText(line, cx, cy, 16, rl.YELLOW)
		cy += 22
	}

	if sel.has_star {
		for s in g.galaxy.stars {
			if s.id != sel.star_id do continue
			heading := fmt.ctprintf("star: %s (#%d)", s.name, s.id)
			rl.DrawText(heading, cx, cy, 18, rl.RAYWHITE)
			cy += 26
			if len(s.planets) == 0 {
				rl.DrawText("(no planets)", cx, cy, 14, rl.GRAY)
				cy += 18
			}
			for p in s.planets {
				owner_name := player_name_for(g, p.owner)
				name := p.name
				if name == "" do name = "?"
				line := fmt.ctprintf("- %s  pop=%d  owner=%s", name, p.population, owner_name)
				rl.DrawText(line, cx, cy, 14, rl.LIGHTGRAY)
				cy += 18
			}
			break
		}
	}

	// Fleets list (current player)
	cy += 8
	rl.DrawText("your fleets:", cx, cy, 14, {180, 200, 220, 255})
	cy += 18
	idx := 1
	for f in g.fleets {
		if f.owner != player_id do continue
		dst := ""
		if f.destination_star_id != 0 do dst = fmt.tprintf("  -> %d (%d turns)", f.destination_star_id, f.turns_remaining)
		line := fmt.ctprintf("[%d] fleet %d  ships=%d%s", idx, f.id, len(f.ships), dst)
		rl.DrawText(line, cx, cy, 14, rl.LIGHTGRAY)
		cy += 16
		idx += 1
		if idx > 9 do break
	}
}

player_name_for :: proc(g: Api_Game, id: int) -> string {
	if id == 0 do return "—"
	for p in g.players {
		if p.id == id do return p.name
	}
	return "?"
}

