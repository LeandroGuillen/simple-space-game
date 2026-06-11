package main

import "core:fmt"
import "core:sync"

import rl "vendor:raylib"

// Galaxy view + planet panel + input. Pulls game state from `state` which
// is updated by the WebSocket reader thread under `state_mu`.

WINDOW_W :: 1600
WINDOW_H :: 1000

// UI scale: physical pixels per design unit. Recomputed each frame from the
// framebuffer height so the interface scales with the window and stays crisp
// under HiDPI. This compositor reports the DPI scale as 1.0, so raylib renders
// at full physical resolution (crisp) but does not auto-scale our sizes — we
// do it here, sizing everything relative to a 1000px-tall design baseline.
ui_scale: f32 = 1

px  :: proc(v: f32) -> f32 { return v * ui_scale }
pxi :: proc(v: i32) -> i32 { return i32(f32(v) * ui_scale) }

// Side-panel width in physical pixels (scaled).
panel_w :: proc() -> i32 { return pxi(300) }

// Top-bar height in physical pixels (scaled).
topbar_h :: proc() -> i32 { return pxi(40) }

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

// Session generation. Bumped on every join/leave. A WS reader thread carries
// the generation it was started with and its updates are ignored once the
// generation moves on — so the (detached, un-stoppable) reader for a game we
// left can't clobber the state of the game we're now in.
ws_gen: int

ws_set_game :: proc(g: Api_Game, gen: int) {
	sync.lock(&state_mu)
	if gen == ws_gen {
		state.game = g
		state.loaded = true
	}
	sync.unlock(&state_mu)
}

ws_snapshot :: proc() -> (Api_Game, bool) {
	sync.lock(&state_mu)
	defer sync.unlock(&state_mu)
	return state.game, state.loaded
}

Selection :: struct {
	star_id:    int,
	has_star:   bool,
	fleet_id:   int,
	has_fleet:  bool,
	planet_id:  int,
	has_planet: bool,
}

// Actions the panel reports back to run_ui for handling outside the draw pass
// (turn submission, leaving the game, the immediate autobuild PUT).
Panel_Action :: struct {
	end_turn:         bool,
	back:             bool,
	autobuild_planet: int, // 0 = none
	autobuild_value:  bool,
}

Screen :: enum {
	Setup,
	In_Game,
}

// Side-panel tabs. Only Galaxy is implemented in Phase 0; the rest are
// placeholders that name the phase that fills them in.
Tab :: enum {
	Galaxy,
	Planet,
	Colonies,
	Fleets,
	Technology,
	Mutation,
}
active_tab: Tab

// Feedback from the most recent End Turn submission, shown in the panel.
Turn_Feedback :: struct {
	shown:    bool,
	ok:       bool,
	advanced: bool,
}
turn_feedback: Turn_Feedback

// Building picker state: tracks which building is selected per planet.
build_picker_planet: int // planet currently showing the picker (-1 = none)
build_picker_index: int  // index into buildable_buildings for that planet

// Join a game: pull an initial REST snapshot (so there's something to draw
// before the first WS frame) and start the live-update reader thread tagged
// with a fresh session generation.
ui_enter_game :: proc(c: ^Api_Client, game_id: string) -> bool {
	game, ok := api_get_game(c, game_id)
	if !ok {
		fmt.eprintfln("ui: failed to GET /games/%s", game_id)
		return false
	}
	cache_building_catalog(game)
	sync.lock(&state_mu)
	ws_gen += 1
	gen := ws_gen
	state.game = game
	state.loaded = true
	sync.unlock(&state_mu)
	ws_start(c, game_id, gen)
	return true
}

// Leave the current game and discard its state. The old WS reader keeps
// running (we can't stop a blocking socket read) but the generation bump
// makes its future updates no-ops.
ui_leave_game :: proc() {
	sync.lock(&state_mu)
	ws_gen += 1
	state = {}
	sync.unlock(&state_mu)
	clear_pending()
}

run_ui :: proc(client: ^Api_Client, game_id_arg: string) {
	// RESIZABLE: let the window be resized; the layout reads the live size.
	// MSAA: smoother circles/lines.
	//
	// NB: WINDOW_HIGHDPI is deliberately NOT set. On this niri/Wayland setup
	// raylib draws in physical-pixel space but reports mouse input in logical
	// space and GetWindowScaleDPI() as 1.0, giving no way to reconcile the two
	// — that made the right half of the UI unclickable. Without HIGHDPI,
	// drawing, layout and mouse all share one coordinate space (clicks work);
	// the compositor upscales the buffer, which is acceptably soft.
	rl.SetConfigFlags({.WINDOW_RESIZABLE, .MSAA_4X_HINT})
	rl.InitWindow(WINDOW_W, WINDOW_H, "SSG - Odin Client")
	defer rl.CloseWindow()
	rl.SetWindowMinSize(900, 600)
	rl.SetTargetFPS(60)

	screen := Screen.Setup
	setup: Setup_State
	setup_init(&setup, client)
	game_id := ""

	camera := rl.Camera2D {
		zoom   = 1.0,
		offset = {WINDOW_W * 0.5, WINDOW_H * 0.5},
	}
	camera_initialized := false

	sel: Selection
	last_mouse: rl.Vector2

	// CLI fast-path: a game id joins immediately as the default player.
	if game_id_arg != "" && ui_enter_game(client, game_id_arg) {
		game_id = game_id_arg
		screen = .In_Game
	}

	for !rl.WindowShouldClose() {
		// Scale the UI to the current framebuffer height (design baseline 1000).
		ui_scale = max(1, f32(rl.GetRenderHeight()) / 1000.0)

		// --- Setup screen ---
		if screen == .Setup {
			rl.BeginDrawing()
			rl.ClearBackground({8, 10, 18, 255})
			gid, pid, joined := setup_screen(&setup, client)
			rl.EndDrawing()
			if joined {
				client.player_id = pid
				if ui_enter_game(client, gid) {
					game_id = gid
					screen = .In_Game
					camera_initialized = false
					sel = {}
					turn_feedback = {}
				}
			}
			free_all(context.temp_allocator)
			continue
		}

		// --- In-game ---
		game, loaded := ws_snapshot()

		// Live window size — drives the layout so resizing works.
		vw := f32(rl.GetScreenWidth())
		vh := f32(rl.GetScreenHeight())
		map_w := vw - f32(panel_w())
		top_h := f32(topbar_h())
		// Keep the galaxy centred in the visible (non-panel, non-topbar) area.
		camera.offset = {map_w * 0.5, top_h + (vh - top_h) * 0.5}

		if loaded && !camera_initialized {
			// Center camera on the galaxy bounding box and pick a zoom that
			// fits all stars roughly within the visible (non-panel, non-topbar) area.
			cx, cy, span := galaxy_extent(game)
			camera.target = {f32(cx), f32(cy)}
			if span > 0 {
				camera.zoom = min(map_w, vh - top_h) / (f32(span) * 1.2)
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
			if mp.x < map_w && mp.y > f32(topbar_h()) && loaded {
				world := rl.GetScreenToWorld2D(mp, camera)
				star_id, hit := pick_star(game, world, 12.0)
				if hit {
					if sel.has_fleet {
						// Queue the move; it's submitted later via End Turn.
						enqueue_fleet_move(sel.fleet_id, star_id)
						turn_feedback.shown = false
						sel.has_fleet = false
					} else {
						sel.star_id = star_id
						sel.has_star = true
						// Auto-select first owned planet at this star.
						for s in game.galaxy.stars {
							if s.id == star_id {
								for p in s.planets {
									if p.owner == client.player_id {
										sel.planet_id = p.id
										sel.has_planet = true
										active_tab = .Planet
										break
									}
								}
								break
							}
						}
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

		// Top bar (empire stats)
		if loaded do draw_top_bar(game, client.player_id)

		rl.BeginMode2D(camera)
		if loaded {
			draw_galaxy(game)
			if sel.has_star do draw_star_highlight(game, sel.star_id)
		} else {
			rl.DrawText("loading...", -50, 0, 24, rl.RAYWHITE)
		}
		rl.EndMode2D()

		// Side panel (screen space). Reports End Turn / Menu clicks.
		act := draw_panel(game, loaded, &sel, client.player_id)
		end_turn := act.end_turn
		back := act.back
		if rl.IsKeyPressed(.E) do end_turn = true

		rl.EndDrawing()

		// Menu: leave the game and return to the setup screen.
		if back {
			ui_leave_game()
			screen = .Setup
			sel = {}
			turn_feedback = {}
			setup_refresh(&setup, client)
			free_all(context.temp_allocator)
			continue
		}

		// Submit the queued commands (outside Begin/EndDrawing — it blocks
		// briefly on the network round-trip).
		if end_turn && loaded {
			if resp, ok := api_end_turn(client, game_id); ok {
				clear_pending()
				sel.has_fleet = false
				turn_feedback = {shown = true, ok = resp.ok, advanced = resp.advanced}
				fmt.printfln("turn: ok=%v advanced=%v", resp.ok, resp.advanced)
			} else {
				turn_feedback = {shown = true, ok = false, advanced = false}
			}
		}

		// Autobuild toggle: fire the PUT immediately (not batched).
		if act.autobuild_planet != 0 && loaded {
			api_set_autobuild(client, game_id, act.autobuild_planet, act.autobuild_value)
		}

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
		// Star name, colored by the owner of its first planet (if any).
		name_col := rl.Color{120, 130, 150, 255}
		for p in s.planets {
			if p.owner != 0 {
				name_col = player_color(g, p.owner)
				break
			}
		}
		rl.DrawText(fmt.ctprintf("%s", s.name), i32(f32(s.x) - f32(rl.MeasureText(fmt.ctprintf("%s", s.name), 12)) / 2), i32(f32(s.y) + 8), 12, name_col)
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

// Draws the side panel and reports actions (End Turn / Menu / autobuild) back
// to run_ui. Takes the selection by pointer so tabs can change it (e.g. pick a
// planet). All sizes go through px()/pxi() so the panel scales.
draw_panel :: proc(g: Api_Game, loaded: bool, sel: ^Selection, player_id: int) -> (act: Panel_Action) {
	sw := rl.GetScreenWidth()
	sh := rl.GetScreenHeight()
	pw := panel_w()
	x := sw - pw
	rl.DrawRectangle(x, 0, pw, sh, {18, 22, 34, 255})
	rl.DrawLineEx({f32(x), 0}, {f32(x), f32(sh)}, px(1), {60, 70, 90, 255})

	cx := x + pxi(16)
	cy := pxi(16)
	rl.DrawText("SSG", cx, cy, pxi(20), rl.RAYWHITE)
	// Menu button (leave game -> setup), always available, top-right of panel.
	menu_w := pxi(78)
	act.back = gui_button({f32(sw - menu_w - pxi(14)), f32(pxi(13)), f32(menu_w), px(28)}, "Menu")
	cy += pxi(30)

	if !loaded {
		rl.DrawText("waiting for snapshot...", cx, cy, pxi(16), rl.GRAY)
		return
	}

	turn_line := fmt.ctprintf("turn %d   %d stars   %d fleets", g.current_turn, len(g.galaxy.stars), len(g.fleets))
	rl.DrawText(turn_line, cx, cy, pxi(15), rl.LIGHTGRAY)
	cy += pxi(26)

	// --- Tab bar (2 rows of 3) ---
	cy = draw_tab_bar(cx, cy)
	cy += pxi(10)

	// --- Active tab content ---
	switch active_tab {
	case .Galaxy:
		draw_galaxy_tab(g, cx, cy, sel^, player_id)
	case .Planet:
		draw_planet_tab(g, cx, cy, sel, player_id, &act)
	case .Colonies:
		draw_colonies_tab(g, cx, cy, sel, player_id)
	case .Fleets:
		draw_placeholder(cx, cy, "Fleets", "Phase 2: fleet inspector")
	case .Technology:
		draw_placeholder(cx, cy, "Technology", "Phase 3: research tree")
	case .Mutation:
		draw_placeholder(cx, cy, "Mutation", "Phase 3: race mutation")
	}

	// --- End Turn (always visible, anchored to the bottom) ---
	by: f32 = f32(sh) - px(86)
	count := pending_count()
	hint := fmt.ctprintf("%d command(s) queued", count)
	rl.DrawText(hint, cx, i32(by) - pxi(24), pxi(15), {200, 180, 120, 255})
	if turn_feedback.shown {
		fb := fmt.ctprintf("last turn: ok=%v advanced=%v", turn_feedback.ok, turn_feedback.advanced)
		col := turn_feedback.ok ? rl.Color{140, 200, 150, 255} : rl.Color{210, 130, 120, 255}
		rl.DrawText(fb, cx, i32(by) - pxi(44), pxi(14), col)
	}
	label: cstring = count > 0 ? "End Turn  >" : "End Turn (no commands)  >"
	act.end_turn = gui_button({f32(cx), by, f32(pw - pxi(32)), px(48)}, label)
	return
}

draw_tab_bar :: proc(cx, cy: i32) -> (next_cy: i32) {
	tabs := [?]struct {
		t:     Tab,
		label: cstring,
	}{
		{.Galaxy, "Galaxy"}, {.Planet, "Planet"}, {.Colonies, "Colony"},
		{.Fleets, "Fleets"}, {.Technology, "Tech"}, {.Mutation, "Mutate"},
	}
	gap := px(4)
	tw := (f32(panel_w() - pxi(32)) - gap * 2) / 3
	th := px(30)
	for tb, i in tabs {
		col := i % 3
		row := i / 3
		rec := rl.Rectangle{f32(cx) + f32(col) * (tw + gap), f32(cy) + f32(row) * (th + gap), tw, th}
		active := active_tab == tb.t
		mp := rl.GetMousePosition()
		hover := rl.CheckCollisionPointRec(mp, rec)
		bg := active ? rl.Color{52, 96, 150, 255} : (hover ? rl.Color{40, 50, 70, 255} : rl.Color{28, 34, 48, 255})
		rl.DrawRectangleRec(rec, bg)
		rl.DrawRectangleLinesEx(rec, max(1, px(1)), {70, 84, 110, 255})
		fs := pxi(14)
		tx := i32(rec.x) + (i32(rec.width) - rl.MeasureText(tb.label, fs)) / 2
		rl.DrawText(tb.label, tx, i32(rec.y + (th - f32(fs)) / 2), fs, active ? rl.RAYWHITE : rl.Color{170, 185, 205, 255})
		if hover && rl.IsMouseButtonPressed(.LEFT) do active_tab = tb.t
	}
	return cy + i32(th * 2 + gap)
}

draw_placeholder :: proc(cx, cy: i32, title, note: cstring) {
	rl.DrawText(title, cx, cy, pxi(20), rl.RAYWHITE)
	rl.DrawText(note, cx, cy + pxi(28), pxi(15), {150, 165, 190, 255})
	rl.DrawText("(not implemented yet)", cx, cy + pxi(50), pxi(14), rl.GRAY)
}

draw_galaxy_tab :: proc(g: Api_Game, cx, cy_in: i32, sel: Selection, player_id: int) {
	cy := cy_in
	rl.DrawText("right-drag pan, wheel zoom, click star", cx, cy, pxi(14), rl.GRAY); cy += pxi(18)
	rl.DrawText("1..9 select fleet, click star to queue", cx, cy, pxi(14), rl.GRAY); cy += pxi(18)
	rl.DrawText("a move, E or button to end turn", cx, cy, pxi(14), rl.GRAY); cy += pxi(26)

	if sel.has_fleet {
		rl.DrawText(fmt.ctprintf("selected fleet: %d", sel.fleet_id), cx, cy, pxi(16), rl.YELLOW)
		cy += pxi(24)
	}

	if sel.has_star {
		for s in g.galaxy.stars {
			if s.id != sel.star_id do continue
			rl.DrawText(fmt.ctprintf("star: %s (#%d)", s.name, s.id), cx, cy, pxi(18), rl.RAYWHITE)
			cy += pxi(28)
			if len(s.planets) == 0 {
				rl.DrawText("(no planets)", cx, cy, pxi(15), rl.GRAY)
				cy += pxi(20)
			}
			for p in s.planets {
				name := p.name
				if name == "" do name = "?"
				line := fmt.ctprintf("- %s  pop=%d  owner=%s", name, p.population, player_name_for(g, p.owner))
				rl.DrawText(line, cx, cy, pxi(15), rl.LIGHTGRAY)
				cy += pxi(20)
			}
			break
		}
	}

	cy += pxi(8)
	rl.DrawText("your fleets:", cx, cy, pxi(15), {180, 200, 220, 255})
	cy += pxi(20)
	idx := 1
	for f in g.fleets {
		if f.owner != player_id do continue
		dst := ""
		if f.destination_star_id != 0 do dst = fmt.tprintf("  -> %d (%d turns)", f.destination_star_id, f.turns_remaining)
		line := fmt.ctprintf("[%d] fleet %d  ships=%d%s", idx, f.id, len(f.ships), dst)
		rl.DrawText(line, cx, cy, pxi(15), rl.LIGHTGRAY)
		cy += pxi(18)
		idx += 1
		if idx > 9 do break
	}
}

draw_planet_tab :: proc(g: Api_Game, cx, cy_in: i32, sel: ^Selection, player_id: int, act: ^Panel_Action) {
	cy := cy_in

	if !sel.has_planet {
		rl.DrawText("Planet", cx, cy, pxi(20), rl.RAYWHITE); cy += pxi(28)
		rl.DrawText("click a star to select a planet", cx, cy, pxi(14), rl.GRAY)
		return
	}

	planet, ok := find_planet(g, sel.planet_id)
	if !ok {
		rl.DrawText("Planet not found", cx, cy, pxi(16), rl.RED)
		sel.has_planet = false
		return
	}

	star, _ := star_of_planet(g, planet.id)

	// --- Planet header ---
	rl.DrawText(fmt.ctprintf("%s", planet.name), cx, cy, pxi(18), rl.RAYWHITE); cy += pxi(24)
	rl.DrawText(fmt.ctprintf("%s  %s  %s  %s", env_name(planet.environment), size_name(planet.size), gravity_name(planet.gravity), mineral_name(planet.mineral_class)), cx, cy, pxi(13), {150, 165, 190, 255}); cy += pxi(18)
	rl.DrawText(fmt.ctprintf("System: %s", star.name), cx, cy, pxi(13), {150, 165, 190, 255}); cy += pxi(22)

	// --- Population metrics ---
	units := planet_units(planet)
	max_units := planet.max_population / 1000
	rl.DrawText(fmt.ctprintf("Population: %d / %d units", units, max_units), cx, cy, pxi(14), rl.LIGHTGRAY); cy += pxi(18)
	food_net := planet.food.farmers * planet.food.produced_per_farmer + planet.food.produced_per_building - units
	food_col := food_net >= 0 ? rl.Color{140, 200, 150, 255} : rl.Color{210, 130, 120, 255}
	rl.DrawText(fmt.ctprintf("Food: %d net", food_net), cx, cy, pxi(13), food_col); cy += pxi(16)
	ind_net := planet.industry.workers * planet.industry.produced_per_worker + planet.industry.produced_per_building
	rl.DrawText(fmt.ctprintf("Industry: %d/turn", ind_net), cx, cy, pxi(13), rl.LIGHTGRAY); cy += pxi(16)
	rl.DrawText(fmt.ctprintf("Income: %d BC", planet.income.total), cx, cy, pxi(13), rl.LIGHTGRAY); cy += pxi(20)

	// --- Population allocation ---
	farmers, workers, scientists := planet_allocation(planet)
	alloc: [3]int = {farmers, workers, scientists}

	rl.DrawText("Population Allocation", cx, cy, pxi(15), {180, 200, 220, 255}); cy += pxi(20)

	// Farmers
	rl.DrawText(fmt.ctprintf("Food:    %d", alloc[0]), cx, cy, pxi(14), rl.LIGHTGRAY)
	if gui_button({f32(cx + pxi(200)), f32(cy), f32(pxi(28)), f32(pxi(18))}, "-") do pop_dec(&alloc, 0)
	if gui_button({f32(cx + pxi(232)), f32(cy), f32(pxi(28)), f32(pxi(18))}, "+") do pop_inc(&alloc, units, 0)
	cy += pxi(20)

	// Workers
	rl.DrawText(fmt.ctprintf("Industry:%d", alloc[1]), cx, cy, pxi(14), rl.LIGHTGRAY)
	if gui_button({f32(cx + pxi(200)), f32(cy), f32(pxi(28)), f32(pxi(18))}, "-") do pop_dec(&alloc, 1)
	if gui_button({f32(cx + pxi(232)), f32(cy), f32(pxi(28)), f32(pxi(18))}, "+") do pop_inc(&alloc, units, 1)
	cy += pxi(20)

	// Scientists
	rl.DrawText(fmt.ctprintf("Science: %d", alloc[2]), cx, cy, pxi(14), rl.LIGHTGRAY)
	if gui_button({f32(cx + pxi(200)), f32(cy), f32(pxi(28)), f32(pxi(18))}, "-") do pop_dec(&alloc, 2)
	if gui_button({f32(cx + pxi(232)), f32(cy), f32(pxi(28)), f32(pxi(18))}, "+") do pop_inc(&alloc, units, 2)
	cy += pxi(22)

	if alloc[0] != farmers || alloc[1] != workers || alloc[2] != scientists {
		set_planet_allocation(planet.id, alloc[0], alloc[1], alloc[2])
	}

	// --- Build queue ---
	rl.DrawText("Construction", cx, cy, pxi(15), {180, 200, 220, 255}); cy += pxi(20)

	current_project := planet.current_project
	if current_project == "" do current_project = "None"
	rl.DrawText(fmt.ctprintf("Project: %s", current_project), cx, cy, pxi(13), rl.LIGHTGRAY); cy += pxi(16)

	if planet.is_building && planet.build_bucket > 0 {
		if info, ok := building_info(current_project); ok {
			progress := f32(planet.build_bucket) / f32(info.cost)
			bar_w := f32(pxi(220))
			rl.DrawRectangle(cx, cy, i32(bar_w), pxi(12), {40, 50, 70, 255})
			rl.DrawRectangle(cx, cy, i32(bar_w * progress), pxi(12), {80, 160, 120, 255})
			rl.DrawText(fmt.ctprintf("%d/%d PP", planet.build_bucket, info.cost), cx + pxi(224), cy, pxi(11), rl.GRAY)
			cy += pxi(16)
		}
	}

	// Queued build list
	q := planet_build_queue(planet.id)
	if len(q) > 0 {
		for name, i in q {
			rl.DrawText(fmt.ctprintf("  [%d] %s", i + 1, name), cx, cy, pxi(12), rl.LIGHTGRAY)
			cy += pxi(15)
		}
		cy += pxi(4)
	}

	// Building picker: shows available buildings, click to add.
	buildings := buildable_buildings(g, planet, player_id, context.temp_allocator)
	if len(buildings) > 0 {
		// Reset picker index if planet changed or out of range
		if build_picker_planet != planet.id || build_picker_index >= len(buildings) {
			build_picker_planet = planet.id
			build_picker_index = 0
		}
		selected := buildings[build_picker_index]

		// Cycle through buildings with < > buttons
		if gui_button({f32(cx), f32(cy), f32(pxi(24)), f32(pxi(20))}, "<") {
			build_picker_index = (build_picker_index - 1 + len(buildings)) % len(buildings)
		}
		rl.DrawText(fmt.ctprintf("%s", selected), cx + pxi(28), cy, pxi(13), rl.RAYWHITE)
		if gui_button({f32(cx + pxi(180)), f32(cy), f32(pxi(24)), f32(pxi(20))}, ">") {
			build_picker_index = (build_picker_index + 1) % len(buildings)
		}
		cy += pxi(22)

		// Add selected building to queue
		if gui_button({f32(cx), f32(cy), f32(pxi(100)), f32(pxi(22))}, "Add to Queue") {
			enqueue_build(planet.id, selected)
		}
		if gui_button({f32(cx + pxi(104)), f32(cy), f32(pxi(70)), f32(pxi(22))}, "Clear") {
			clear_build(planet.id)
		}
		cy += pxi(26)
	} else {
		rl.DrawText("(no buildings available)", cx, cy, pxi(12), rl.GRAY)
		cy += pxi(18)
	}

	// Buy production
	buying := buy_production_queued(planet.id)
	buy_label: cstring = buying ? "Buying (queued)" : "Buy Production"
	if gui_button({f32(cx), f32(cy), f32(pxi(180)), f32(pxi(22))}, buy_label) {
		toggle_buy_production(planet.id)
	}
	cy += pxi(24)

	// Autobuild toggle
	ab_label: cstring = planet.autobuild ? "Autobuild: ON" : "Autobuild: OFF"
	if gui_button({f32(cx), f32(cy), f32(pxi(180)), f32(pxi(22))}, ab_label) {
		act.autobuild_planet = planet.id
		act.autobuild_value = !planet.autobuild
	}
	cy += pxi(26)

	// --- Transfer population ---
	if units > 1 {
		rl.DrawText("Transfer Population", cx, cy, pxi(15), {180, 200, 220, 255}); cy += pxi(20)

		// Find destination planets
		dest_planets := owned_planets(g, player_id, context.temp_allocator)
		// Simple: use first other planet as destination (TODO: picker)
		dest_name: cstring = "(no other planet)"
		for dp in dest_planets {
			if dp.id != planet.id {
				s, sok := star_of_planet(g, dp.id)
				sn := s.name if sok else "?"
				dest_name = fmt.ctprintf("%s - %s", sn, dp.name)
				break
			}
		}

		rl.DrawText(fmt.ctprintf("Dest: %s", dest_name), cx, cy, pxi(13), rl.LIGHTGRAY); cy += pxi(16)
		rl.DrawText(fmt.ctprintf("Max transfer: %d units", units - 1), cx, cy, pxi(13), rl.GRAY); cy += pxi(20)
	}
}

draw_colonies_tab :: proc(g: Api_Game, cx, cy_in: i32, sel: ^Selection, player_id: int) {
	cy := cy_in

	planets := owned_planets(g, player_id, context.temp_allocator)
	rl.DrawText(fmt.ctprintf("Your Colonies (%d)", len(planets)), cx, cy, pxi(16), rl.RAYWHITE); cy += pxi(22)

	if len(planets) == 0 {
		rl.DrawText("(no colonies yet)", cx, cy, pxi(14), rl.GRAY)
		return
	}

	// Column headers
	rl.DrawText("Name", cx, cy, pxi(12), {130, 150, 180, 255})
	rl.DrawText("Pop", cx + pxi(100), cy, pxi(12), {130, 150, 180, 255})
	rl.DrawText("F/W/S", cx + pxi(140), cy, pxi(12), {130, 150, 180, 255})
	rl.DrawText("Project", cx + pxi(190), cy, pxi(12), {130, 150, 180, 255})
	cy += pxi(16)

	for p in planets {
		row_h := pxi(18)
		row_rec := rl.Rectangle{f32(cx), f32(cy), f32(pxi(280)), f32(row_h)}
		hover := rl.CheckCollisionPointRec(rl.GetMousePosition(), row_rec)

		if hover {
			rl.DrawRectangleRec(row_rec, {34, 42, 60, 255})
		}

		// Name (clickable)
		name := p.name
		if name == "" do name = "?"
		rl.DrawText(fmt.ctprintf("%s", name), cx, cy, pxi(12), hover ? rl.YELLOW : rl.LIGHTGRAY)

		// Pop
		units := planet_units(p)
		rl.DrawText(fmt.ctprintf("%d", units), cx + pxi(100), cy, pxi(12), rl.LIGHTGRAY)

		// Farmers/Workers/Scientists
		rl.DrawText(fmt.ctprintf("%d/%d/%d", p.food.farmers, p.industry.workers, p.science.scientists), cx + pxi(140), cy, pxi(11), rl.LIGHTGRAY)

		// Project
		proj := p.current_project
		if proj == "" do proj = "-"
		rl.DrawText(fmt.ctprintf("%s", proj), cx + pxi(190), cy, pxi(11), rl.LIGHTGRAY)

		// Click to jump to planet
		if hover && rl.IsMouseButtonPressed(.LEFT) {
			sel.planet_id = p.id
			sel.has_planet = true
			// Also select the parent star
			if s, sok := star_of_planet(g, p.id); sok {
				sel.star_id = s.id
				sel.has_star = true
			}
			active_tab = .Planet
		}

		cy += row_h
	}
}

player_name_for :: proc(g: Api_Game, id: int) -> string {
	if id == 0 do return "-"
	for p in g.players {
		if p.id == id do return p.name
	}
	return "?"
}

// Draw the top bar with empire-wide stats (treasury, income, food, research,
// stardate). Matches the Solid client's game-topbar layout.
draw_top_bar :: proc(g: Api_Game, player_id: int) {
	sw := rl.GetScreenWidth()
	th := topbar_h()

	// Bar background
	rl.DrawRectangle(0, 0, sw, th, {14, 18, 28, 255})
	rl.DrawLineEx({0, f32(th)}, {f32(sw), f32(th)}, px(1), {50, 60, 80, 255})

	// Find current player
	player: Api_Player
	for p in g.players {
		if p.id == player_id {
			player = p
			break
		}
	}

	// Compute empire totals from owned planets
	treasury := player.treasury
	total_income := 0
	total_food := 0
	planet_count := 0
	for s in g.galaxy.stars {
		for p in s.planets {
			if p.owner != player_id do continue
			planet_count += 1
			total_income += p.income.total
			units := planet_units(p)
			total_food += p.food.farmers * p.food.produced_per_farmer + p.food.produced_per_building - units
		}
	}

	cx := pxi(16)
	fs := pxi(14)
	fs_big := pxi(16)

	// Stardate (left)
	stardate := f64(g.current_turn) / 10.0
	rl.DrawText(fmt.ctprintf("Star date: %.1f", stardate), cx, i32(f32(th) - f32(fs_big) - px(2)), fs_big, rl.LIGHTGRAY)

	// Stats (right-aligned, flowing left)
	sx := f32(sw) - px(16)

	// End turn hint
	rl.DrawText("[E] End Turn", i32(sx - px(110)), i32(f32(th) - f32(fs) - px(4)), fs, {120, 140, 160, 255})

	// Stardate label on right side
	rl.DrawText(fmt.ctprintf("Turn %d", g.current_turn), i32(sx - px(200)), i32(f32(th) - f32(fs) - px(4)), fs, {120, 140, 160, 255})

	// Income
	income_col := total_income >= 0 ? rl.Color{140, 200, 150, 255} : rl.Color{210, 130, 120, 255}
	rl.DrawText(fmt.ctprintf("Income: %+d BC", total_income), i32(sx - px(380)), i32(f32(th) - f32(fs) - px(4)), fs, income_col)

	// Food
	food_col := total_food >= 0 ? rl.Color{140, 200, 150, 255} : rl.Color{210, 130, 120, 255}
	rl.DrawText(fmt.ctprintf("Food: %+d", total_food), i32(sx - px(500)), i32(f32(th) - f32(fs) - px(4)), fs, food_col)

	// Treasury (gold)
	rl.DrawText(fmt.ctprintf("%.0f BC", treasury), i32(sx - px(620)), i32(f32(th) - f32(fs) - px(4)), fs, {220, 200, 120, 255})

	// Colonies count
	rl.DrawText(fmt.ctprintf("Colonies: %d", planet_count), i32(sx - px(740)), i32(f32(th) - f32(fs) - px(4)), fs, {120, 140, 160, 255})
}

