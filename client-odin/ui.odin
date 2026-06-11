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

// Top-bar height in physical pixels (scaled). Tall enough for the two-line
// Research button.
topbar_h :: proc() -> i32 { return pxi(48) }

// Bottom navigation bar (Galaxy / Colonies / Fleets / Mutate) height.
bottombar_h :: proc() -> i32 { return pxi(44) }

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

// Whether a WS reader's session generation is still the live one. Readers
// for left/replaced games see false and exit their reconnect loop.
ws_gen_is_current :: proc(gen: int) -> bool {
	sync.lock(&state_mu)
	defer sync.unlock(&state_mu)
	return gen == ws_gen
}

Selection :: struct {
	star_id:    int,
	has_star:   bool,
	fleet_id:   int,
	has_fleet:  bool,
	planet_id:  int,
	has_planet: bool,
	// Armed by the Move button (or 1..9 quick-select): the next star click
	// queues a move for fleet_id instead of selecting the star.
	move_targeting: bool,
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

// Side-panel modes. The panel exists for galaxy-view selections only:
// Galaxy shows the selected star's planets or the selected fleet's ships;
// Planet is a drill-in from a star's planet list. Everything else lives in
// modal windows (bottom bar) or the research popup (top bar).
Tab :: enum {
	Galaxy,
	Planet,
}
active_tab: Tab

// Modal windows opened from the bottom bar. At most one is open, mutually
// exclusive with the research popup.
Window_Kind :: enum {
	None,
	Colonies,
	Fleets,
	Mutation,
}
open_window: Window_Kind
window_scroll:    [Window_Kind]i32
window_content_h: [Window_Kind]i32

// Centered modal geometry over the map area, `design_w` design units wide.
modal_rect :: proc(design_w: f32) -> rl.Rectangle {
	sw := f32(rl.GetScreenWidth())
	sh := f32(rl.GetScreenHeight())
	map_w := sw - f32(panel_w())
	usable_h := sh - f32(topbar_h()) - f32(bottombar_h())
	w := min(px(design_w), map_w - px(40))
	h := min(px(620), usable_h - px(30))
	return {(map_w - w) / 2, f32(topbar_h()) + (usable_h - h) / 2, w, h}
}

// The rect of whichever modal is currently open (zero rect when none).
current_modal_rect :: proc() -> rl.Rectangle {
	if research_popup_open do return modal_rect(680)
	#partial switch open_window {
	case .Colonies, .Fleets, .Mutation:
		return modal_rect(420)
	}
	return {}
}

// Toggle a bottom-bar window; windows and the research popup are exclusive.
toggle_window :: proc(k: Window_Kind) {
	open_window = open_window == k ? .None : k
	if open_window != .None do research_popup_open = false
}

// Per-tab scroll offset (physical px) and last measured content height,
// used to clamp scrolling. The mouse wheel scrolls whichever tab is active
// while the cursor is over the panel (it zooms the map otherwise).
panel_scroll:    [Tab]i32
panel_content_h: [Tab]i32

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
	// Esc closes modals (research popup / windows), not the app.
	rl.SetExitKey(.KEY_NULL)

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
		bot_h := f32(bottombar_h())
		map_h := vh - top_h - bot_h
		// Keep the galaxy centred in the visible (non-panel, non-bars) area.
		camera.offset = {map_w * 0.5, top_h + map_h * 0.5}

		if loaded && !camera_initialized {
			// Center camera on the galaxy bounding box and pick a zoom that
			// fits all stars roughly within the visible map area.
			cx, cy, span := galaxy_extent(game)
			camera.target = {f32(cx), f32(cy)}
			if span > 0 {
				camera.zoom = min(map_w, map_h) / (f32(span) * 1.2)
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
			if research_popup_open {
				// The popup is modal: the wheel scrolls its choice list.
				research_scroll -= i32(wheel * f32(pxi(40)))
			} else if open_window != .None {
				window_scroll[open_window] -= i32(wheel * f32(pxi(40)))
			} else if rl.GetMousePosition().x >= map_w {
				// Over the side panel: scroll the active tab's content.
				panel_scroll[active_tab] -= i32(wheel * f32(pxi(40)))
			} else {
				// Zoom around the mouse cursor.
				world_before := rl.GetScreenToWorld2D(rl.GetMousePosition(), camera)
				camera.zoom *= 1 + wheel * 0.1
				camera.zoom = clamp(camera.zoom, 0.05, 50.0)
				world_after := rl.GetScreenToWorld2D(rl.GetMousePosition(), camera)
				camera.target.x += world_before.x - world_after.x
				camera.target.y += world_before.y - world_after.y
			}
		}

		// Left click on the map: queue a move when move-targeting is armed,
		// otherwise select the nearest star or own fleet (whichever is
		// closer). The selection drives the contextual side panel.
		if rl.IsMouseButtonPressed(.LEFT) && !research_popup_open && open_window == .None {
			mp := rl.GetMousePosition()
			if mp.x < map_w && mp.y > top_h && mp.y < vh - bot_h && loaded {
				world := rl.GetScreenToWorld2D(mp, camera)
				star_id, star_d, star_hit := pick_star(game, world, 12.0)
				fleet_id, fleet_d, fleet_hit := pick_fleet(game, world, 10.0, client.player_id)
				if sel.move_targeting && sel.has_fleet && star_hit {
					// Queue the move (partial if the inspector holds a
					// subset selection); submitted later via End Turn.
					idxs := selected_ship_indexes_for_move(game, sel.fleet_id, context.temp_allocator)
					enqueue_fleet_move_ships(sel.fleet_id, star_id, idxs)
					turn_feedback.shown = false
					sel.move_targeting = false
				} else if fleet_hit && (!star_hit || fleet_d < star_d) {
					sel.fleet_id = fleet_id
					sel.has_fleet = true
					sel.move_targeting = false
					fleet_select(game, fleet_id)
					active_tab = .Galaxy
				} else if star_hit {
					sel.star_id = star_id
					sel.has_star = true
					sel.has_fleet = false
					sel.move_targeting = false
					active_tab = .Galaxy
				}
			}
		}

		// Keyboard navigation: windows on the bottom bar, T toggles the
		// research popup, Esc closes whatever modal is open.
		if rl.IsKeyPressed(.G) {
			active_tab = .Galaxy
			open_window = .None
		}
		if rl.IsKeyPressed(.P) && sel.has_planet do active_tab = .Planet
		if rl.IsKeyPressed(.C) do toggle_window(.Colonies)
		if rl.IsKeyPressed(.F) do toggle_window(.Fleets)
		if rl.IsKeyPressed(.T) {
			research_popup_open = !research_popup_open
			if research_popup_open do open_window = .None
		}
		if rl.IsKeyPressed(.ESCAPE) {
			research_popup_open = false
			open_window = .None
		}
		if loaded {
			player, pok := player_by_id(game, client.player_id)
			mutate_visible := pok && (mutation_available(player) || mutate_race_queued() || mutation_editing)
			if rl.IsKeyPressed(.M) && mutate_visible do toggle_window(.Mutation)
			// The Mutate window disappears once the mutation is spent.
			if open_window == .Mutation && !mutate_visible do open_window = .None
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
						sel.move_targeting = true // next star click queues the move
						fleet_select(game, sel.fleet_id)
						active_tab = .Galaxy
						fmt.printfln("selected fleet %d", sel.fleet_id)
					}
				}
			}
		}

		// --- Draw ---
		rl.BeginDrawing()
		rl.ClearBackground({8, 10, 18, 255})

		// While a modal (research popup or a window) is open, everything
		// outside it goes inert.
		ui_modal_on = (research_popup_open || open_window != .None) && loaded
		ui_modal_rect = current_modal_rect()

		// Top bar (empire stats + Research button).
		research_clicked := false
		if loaded do research_clicked = draw_top_bar(game, client.player_id)

		rl.BeginMode2D(camera)
		if loaded {
			draw_galaxy(game)
			if sel.has_star do draw_star_highlight(game, sel.star_id)
		} else {
			rl.DrawText("loading...", -50, 0, 24, rl.RAYWHITE)
		}
		rl.EndMode2D()

		// Bottom navigation bar. Exempt from the modal block so its buttons
		// can switch/close windows with a single click.
		if loaded {
			saved_modal := ui_modal_on
			ui_modal_on = false
			draw_bottom_bar(game, client.player_id)
			ui_modal_on = saved_modal
		}

		// Side panel (screen space). Reports End Turn / Menu clicks.
		act := draw_panel(game, loaded, &sel, client.player_id)
		end_turn := act.end_turn
		back := act.back
		if rl.IsKeyPressed(.E) do end_turn = true

		// Modals, drawn over everything else.
		if loaded {
			if research_popup_open do draw_research_popup(game, client.player_id)
			else if open_window != .None do draw_window(game, &sel, client.player_id)
		}

		rl.EndDrawing()
		ui_modal_on = false

		if research_clicked {
			research_popup_open = !research_popup_open
			if research_popup_open do open_window = .None
		}

		// Menu: leave the game and return to the setup screen.
		if back {
			ui_leave_game()
			screen = .Setup
			sel = {}
			turn_feedback = {}
			research_popup_open = false
			open_window = .None
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

// Nearest star within `threshold` world units. Returns the squared distance
// so callers can compare against other pick results.
pick_star :: proc(g: Api_Game, world: rl.Vector2, threshold: f32) -> (id: int, dist2: f32, found: bool) {
	dist2 = threshold * threshold
	for s in g.galaxy.stars {
		dx := f32(s.x) - world.x
		dy := f32(s.y) - world.y
		d := dx * dx + dy * dy
		if d < dist2 {
			dist2 = d
			id = s.id
			found = true
		}
	}
	return
}

// Nearest fleet owned by `owner` within `threshold` world units.
pick_fleet :: proc(g: Api_Game, world: rl.Vector2, threshold: f32, owner: int) -> (id: int, dist2: f32, found: bool) {
	dist2 = threshold * threshold
	for f in g.fleets {
		if f.owner != owner do continue
		dx := f32(f.x) - world.x
		dy := f32(f.y) - world.y
		d := dx * dx + dy * dy
		if d < dist2 {
			dist2 = d
			id = f.id
			found = true
		}
	}
	return
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
	// Mirrors the web client's STAR_COLORS palette (lib/utils.ts); anything
	// out of range falls back to entry 0, same as the web client.
	switch code {
	case 0:  return {241, 245, 249, 255} // #f1f5f9 white
	case 1:  return {125, 211, 252, 255} // #7dd3fc blue
	case 2:  return {253, 230, 138, 255} // #fde68a yellow
	case 3:  return {251, 146, 60, 255}  // #fb923c orange
	case 4:  return {248, 113, 113, 255} // #f87171 red
	case:    return {241, 245, 249, 255}
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

	turn_line := fmt.ctprintf("turn %.1f   %d stars   %d fleets", g.current_turn, len(g.galaxy.stars), len(g.fleets))
	rl.DrawText(turn_line, cx, cy, pxi(15), rl.LIGHTGRAY)
	cy += pxi(26)

	cy += pxi(4)

	// --- Active mode content (scrollable, scissored region) ---
	by: f32 = f32(sh) - px(86)          // End Turn button top
	content_top := cy
	content_bottom := i32(by) - pxi(50) // room for the feedback lines
	region_h := max(content_bottom - content_top, pxi(40))

	max_scroll := max(0, panel_content_h[active_tab] - region_h)
	panel_scroll[active_tab] = clamp(panel_scroll[active_tab], 0, max_scroll)
	scroll := panel_scroll[active_tab]

	rl.BeginScissorMode(x, content_top, pw, region_h)
	ui_clip_on = true
	ui_clip = {f32(x), f32(content_top), f32(pw), f32(region_h)}
	start_cy := content_top - scroll
	end_cy := start_cy
	switch active_tab {
	case .Galaxy:
		end_cy = draw_galaxy_tab(g, cx, start_cy, sel, player_id)
	case .Planet:
		end_cy = draw_planet_tab(g, cx, start_cy, sel, player_id, &act)
	}
	ui_clip_on = false
	rl.EndScissorMode()
	panel_content_h[active_tab] = end_cy - start_cy

	// Thin scrollbar on the panel's right edge when content overflows.
	if panel_content_h[active_tab] > region_h {
		track_x := sw - pxi(5)
		bar_h := max(region_h * region_h / panel_content_h[active_tab], pxi(24))
		bar_y := content_top + (region_h - bar_h) * scroll / max(1, max_scroll)
		rl.DrawRectangle(track_x, content_top, pxi(3), region_h, {30, 36, 50, 255})
		rl.DrawRectangle(track_x, bar_y, pxi(3), bar_h, {90, 104, 130, 255})
	}

	// --- End Turn (always visible, anchored to the bottom) ---
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

// Bottom navigation bar across the map area: Galaxy plus the Colonies /
// Fleets windows, and Mutate while an evolutionary mutation is available
// (or queued / being edited). Galaxy closes any window and backs out of the
// Planet drill-in.
draw_bottom_bar :: proc(g: Api_Game, player_id: int) {
	sw := rl.GetScreenWidth()
	sh := rl.GetScreenHeight()
	bh := bottombar_h()
	w := sw - panel_w()
	y := sh - bh
	rl.DrawRectangle(0, y, w, bh, {14, 18, 28, 255})
	rl.DrawLineEx({0, f32(y)}, {f32(w), f32(y)}, px(1), {50, 60, 80, 255})

	Nav_Entry :: struct {
		win:   Window_Kind,
		label: cstring,
	}
	entries := make([dynamic]Nav_Entry, 0, 4, context.temp_allocator)
	append(&entries, Nav_Entry{.None, "Galaxy"})
	append(&entries, Nav_Entry{.Colonies, "Colonies"})
	append(&entries, Nav_Entry{.Fleets, "Fleets"})
	if p, ok := player_by_id(g, player_id); ok && (mutation_available(p) || mutate_race_queued() || mutation_editing) {
		append(&entries, Nav_Entry{.Mutation, "Mutate"})
	}

	bw := pxi(110)
	gap := pxi(8)
	btn_h := pxi(30)
	x := pxi(12)
	by := y + (bh - btn_h) / 2
	for e in entries {
		rec := rl.Rectangle{f32(x), f32(by), f32(bw), f32(btn_h)}
		active := open_window == e.win
		hover := ui_hover(rec)
		bg := active ? rl.Color{52, 96, 150, 255} : (hover ? rl.Color{40, 50, 70, 255} : rl.Color{28, 34, 48, 255})
		rl.DrawRectangleRec(rec, bg)
		rl.DrawRectangleLinesEx(rec, max(1, px(1)), {70, 84, 110, 255})
		fs := pxi(14)
		tx := i32(rec.x) + (i32(rec.width) - rl.MeasureText(e.label, fs)) / 2
		rl.DrawText(e.label, tx, by + (btn_h - fs) / 2, fs, active ? rl.RAYWHITE : rl.Color{170, 185, 205, 255})
		if hover && rl.IsMouseButtonPressed(.LEFT) {
			if e.win == .None {
				open_window = .None
				active_tab = .Galaxy
			} else {
				toggle_window(e.win)
			}
		}
		x += bw + gap
	}
}

window_title :: proc(k: Window_Kind) -> cstring {
	#partial switch k {
	case .Colonies: return "Colonies"
	case .Fleets:   return "Fleets"
	case .Mutation: return "Evolutionary Mutation"
	}
	return ""
}

// Modal window for the bottom-bar screens. Hosts the same content procs the
// side panel used to, inside a scrollable scissored region.
draw_window :: proc(g: Api_Game, sel: ^Selection, player_id: int) {
	kind := open_window // local copy: content clicks may close the window
	r := current_modal_rect()
	sw := rl.GetScreenWidth()
	sh := rl.GetScreenHeight()
	rl.DrawRectangle(0, 0, sw, sh, {0, 0, 0, 130})
	rl.DrawRectangleRec(r, {18, 22, 34, 255})
	rl.DrawRectangleLinesEx(r, max(1, px(1)), {70, 84, 110, 255})

	cx := i32(r.x) + pxi(18)
	cy := i32(r.y) + pxi(14)
	rl.DrawText(window_title(kind), cx, cy, pxi(18), rl.RAYWHITE)
	if gui_button({r.x + r.width - px(86), r.y + px(12), px(70), px(26)}, "Close") {
		open_window = .None
	}
	cy += pxi(32)

	content_top := cy
	content_bottom := i32(r.y + r.height) - pxi(14)
	region_h := max(content_bottom - content_top, pxi(40))
	max_scroll := max(0, window_content_h[kind] - region_h)
	window_scroll[kind] = clamp(window_scroll[kind], 0, max_scroll)
	scroll := window_scroll[kind]

	rl.BeginScissorMode(i32(r.x), content_top, i32(r.width), region_h)
	ui_clip_on = true
	ui_clip = {r.x, f32(content_top), r.width, f32(region_h)}
	start_cy := content_top - scroll
	end_cy := start_cy
	#partial switch kind {
	case .Colonies:
		end_cy = draw_colonies_tab(g, cx, start_cy, sel, player_id)
	case .Fleets:
		end_cy = draw_fleets_tab(g, cx, start_cy, sel, player_id)
	case .Mutation:
		end_cy = draw_mutation_tab(g, cx, start_cy, player_id)
	}
	ui_clip_on = false
	rl.EndScissorMode()
	window_content_h[kind] = end_cy - start_cy

	if window_content_h[kind] > region_h {
		track_x := i32(r.x + r.width) - pxi(6)
		bar_h := max(region_h * region_h / window_content_h[kind], pxi(24))
		bar_y := content_top + (region_h - bar_h) * scroll / max(1, max_scroll)
		rl.DrawRectangle(track_x, content_top, pxi(3), region_h, {30, 36, 50, 255})
		rl.DrawRectangle(track_x, bar_y, pxi(3), bar_h, {90, 104, 130, 255})
	}

	// Click outside dismisses; ignore the bottom bar (its buttons already
	// switch/close windows themselves).
	if rl.IsMouseButtonPressed(.LEFT) {
		mp := rl.GetMousePosition()
		over_bottom_bar := mp.y >= f32(sh - bottombar_h()) && mp.x < f32(sw - panel_w())
		if !rl.CheckCollisionPointRec(mp, r) && !over_bottom_bar {
			open_window = .None
		}
	}
}

// Contextual default mode: ships of the selected fleet, else planets of the
// selected star, else usage hints.
draw_galaxy_tab :: proc(g: Api_Game, cx, cy_in: i32, sel: ^Selection, player_id: int) -> i32 {
	cy := cy_in

	if sel.has_fleet {
		if sel.move_targeting {
			rl.DrawText("click a star to move", cx, cy, pxi(14), rl.YELLOW)
			cy += pxi(20)
		}
		return draw_fleet_inspector(g, cx, cy, sel, player_id, sel.fleet_id)
	}

	if sel.has_star {
		for s in g.galaxy.stars {
			if s.id != sel.star_id do continue
			rl.DrawText(fmt.ctprintf("%s", s.name), cx, cy, pxi(18), rl.RAYWHITE)
			cy += pxi(26)
			if len(s.planets) == 0 {
				rl.DrawText("(no planets)", cx, cy, pxi(15), rl.GRAY)
				cy += pxi(20)
			}
			for p in s.planets {
				name := p.name
				if name == "" do name = "?"
				row := rl.Rectangle{f32(cx), f32(cy), f32(pxi(268)), f32(pxi(18))}
				own := p.owner == player_id
				hover := own && ui_hover(row)
				if hover do rl.DrawRectangleRec(row, {34, 42, 60, 255})
				col := own ? (hover ? rl.YELLOW : rl.Color{170, 200, 255, 255}) : rl.LIGHTGRAY
				line := fmt.ctprintf("%s  %s  pop %d  %s", name, env_name(p.environment), planet_units(p), player_name_for(g, p.owner))
				rl.DrawText(line, cx, cy, pxi(13), col)
				if hover && rl.IsMouseButtonPressed(.LEFT) {
					// Owned planet: drill into the management view.
					sel.planet_id = p.id
					sel.has_planet = true
					active_tab = .Planet
				}
				cy += pxi(19)
				cy = draw_star_planet_actions(g, s, p, cx, cy, player_id)
			}
			// Own fleets parked at this star, as a shortcut into the inspector.
			for f in g.fleets {
				if f.owner != player_id || f.star_id != s.id || !fleet_in_orbit(f) do continue
				row := rl.Rectangle{f32(cx), f32(cy), f32(pxi(268)), f32(pxi(18))}
				hover := ui_hover(row)
				if hover do rl.DrawRectangleRec(row, {34, 42, 60, 255})
				rl.DrawText(fmt.ctprintf("fleet #%d  %d ships in orbit", f.id, len(f.ships)), cx, cy, pxi(13), hover ? rl.YELLOW : rl.Color{180, 200, 220, 255})
				if hover && rl.IsMouseButtonPressed(.LEFT) {
					sel.fleet_id = f.id
					sel.has_fleet = true
					fleet_select(g, f.id)
				}
				cy += pxi(19)
			}
			break
		}
		return cy
	}

	rl.DrawText("click a star to see its planets,", cx, cy, pxi(14), rl.GRAY); cy += pxi(18)
	rl.DrawText("or a fleet to see its ships", cx, cy, pxi(14), rl.GRAY); cy += pxi(24)
	rl.DrawText("right-drag pan, wheel zoom", cx, cy, pxi(13), rl.GRAY); cy += pxi(17)
	rl.DrawText("1..9 quick-select fleet + move", cx, cy, pxi(13), rl.GRAY); cy += pxi(17)
	rl.DrawText("E or button to end turn", cx, cy, pxi(13), rl.GRAY); cy += pxi(17)
	return cy
}

// Inline Colonize / Build Planet buttons under a star-panel planet row.
// Mirrors the web client's colonize + artificial-planet modals: colonizing
// needs an orbiting colony ship or a completed Colony Base in this system;
// artificial planets need a completed Artificial Planet project anywhere.
draw_star_planet_actions :: proc(g: Api_Game, star: Api_Star, p: Api_Planet, cx, cy_in: i32, player_id: int) -> i32 {
	cy := cy_in

	if planet_suitable_for_colonization(p) {
		if colonize_queued_for_planet(p.id) {
			if gui_button({f32(cx + pxi(12)), f32(cy), f32(pxi(180)), f32(pxi(20))}, "Cancel Colonize") {
				cancel_colonize_planet(p.id)
			}
			return cy + pxi(24)
		}
		// Prefer an orbiting colony-ship fleet; fall back to a completed
		// colony base on a planet of this same system.
		fleet_id := 0
		for f in g.fleets {
			if f.owner == player_id && f.star_id == star.id && fleet_in_orbit(f) && fleet_has_ship_type(f, "colony_ship") {
				fleet_id = f.id
				break
			}
		}
		has_base := false
		if fleet_id == 0 {
			for src_id in pending_ids_for_player(g.pending_colony_bases, player_id) {
				if src, ok := star_of_planet(g, src_id); ok && src.id == star.id {
					has_base = true
					break
				}
			}
		}
		if fleet_id != 0 || has_base {
			if gui_button({f32(cx + pxi(12)), f32(cy), f32(pxi(180)), f32(pxi(20))}, "Colonize") {
				enqueue_colonize(p.id, fleet_id)
			}
			cy += pxi(24)
		}
	} else if planet_artificial_target(p) {
		if artificial_planet_queued_for_target(p.id) {
			if gui_button({f32(cx + pxi(12)), f32(cy), f32(pxi(180)), f32(pxi(20))}, "Cancel Build Planet") {
				cancel_artificial_planet_target(p.id)
			}
			return cy + pxi(24)
		}
		// Any completed (and not yet re-targeted) Artificial Planet project
		// can target this world.
		source_id := 0
		for src_id in pending_ids_for_player(g.pending_artificial_planets, player_id) {
			if !artificial_planet_queued_for_source(src_id) {
				source_id = src_id
				break
			}
		}
		if source_id != 0 {
			if gui_button({f32(cx + pxi(12)), f32(cy), f32(pxi(180)), f32(pxi(20))}, "Build Artificial Planet") {
				enqueue_artificial_planet(source_id, p.id)
			}
			cy += pxi(24)
		}
	}
	return cy
}

draw_planet_tab :: proc(g: Api_Game, cx, cy_in: i32, sel: ^Selection, player_id: int, act: ^Panel_Action) -> i32 {
	cy := cy_in

	if !sel.has_planet {
		rl.DrawText("Planet", cx, cy, pxi(20), rl.RAYWHITE); cy += pxi(28)
		rl.DrawText("click a star to select a planet", cx, cy, pxi(14), rl.GRAY)
		return cy + pxi(18)
	}

	if gui_button({f32(cx), f32(cy), f32(pxi(70)), f32(pxi(20))}, "< Star") {
		active_tab = .Galaxy
	}
	cy += pxi(26)

	planet, ok := find_planet(g, sel.planet_id)
	if !ok {
		rl.DrawText("Planet not found", cx, cy, pxi(16), rl.RED)
		sel.has_planet = false
		return cy + pxi(20)
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
	return cy
}

draw_colonies_tab :: proc(g: Api_Game, cx, cy_in: i32, sel: ^Selection, player_id: int) -> i32 {
	cy := cy_in

	planets := owned_planets(g, player_id, context.temp_allocator)
	rl.DrawText(fmt.ctprintf("Your Colonies (%d)", len(planets)), cx, cy, pxi(16), rl.RAYWHITE); cy += pxi(22)

	if len(planets) == 0 {
		rl.DrawText("(no colonies yet)", cx, cy, pxi(14), rl.GRAY)
		return cy + pxi(18)
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
		hover := ui_hover(row_rec)

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

		// Click to jump to the planet's management view (in the sidebar).
		if hover && rl.IsMouseButtonPressed(.LEFT) {
			sel.planet_id = p.id
			sel.has_planet = true
			// Also select the parent star
			if s, sok := star_of_planet(g, p.id); sok {
				sel.star_id = s.id
				sel.has_star = true
			}
			active_tab = .Planet
			open_window = .None
		}

		cy += row_h
	}
	return cy
}

player_name_for :: proc(g: Api_Game, id: int) -> string {
	if id == 0 do return "-"
	for p in g.players {
		if p.id == id do return p.name
	}
	return "?"
}

// Draw the top bar with empire-wide stats (treasury, income, food, stardate)
// and the Research button. Returns whether the Research button was clicked
// (toggles the research popup).
draw_top_bar :: proc(g: Api_Game, player_id: int) -> (research_clicked: bool) {
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
	stardate := g.current_turn / 10.0
	rl.DrawText(fmt.ctprintf("Star date: %.1f", stardate), cx, i32(f32(th) - f32(fs_big) - px(2)), fs_big, rl.LIGHTGRAY)

	// Stats (right-aligned, flowing left)
	sx := f32(sw) - px(16)

	// End turn hint
	rl.DrawText("[E] End Turn", i32(sx - px(110)), i32(f32(th) - f32(fs) - px(4)), fs, {120, 140, 160, 255})

	// Stardate label on right side
	rl.DrawText(fmt.ctprintf("Turn %.1f", g.current_turn), i32(sx - px(200)), i32(f32(th) - f32(fs) - px(4)), fs, {120, 140, 160, 255})

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

	// Research button (two lines: title + output/ETA), after the stardate.
	research_clicked = draw_research_button(g, player_id)
	return
}

