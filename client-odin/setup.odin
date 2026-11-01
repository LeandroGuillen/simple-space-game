package main

import "core:fmt"
import "core:strings"

import rl "vendor:raylib"

// Setup / join screen and a couple of tiny immediate-mode widgets shared
// with the in-game UI. Replaces the old "auto-join the first game as
// player 1" behaviour: the player now picks a game + a player slot, or
// creates a new game.

// Option lists for the create-game form. Values match what the server
// accepts (server/handler/game_controller.go + game/galaxy_utils.go).
setup_races := [?]string {
	"human", "alkari", "bulrathi", "darloks", "elerians",
	"gnolams", "klackons", "meklars", "mrrshan", "psilons",
}
setup_sizes  := [?]string{"tiny", "medium", "huge"}
setup_colors := [?]string{"#007AFF", "#FF3B30", "#34C759", "#FF9500", "#AF52DE", "#FFD60A"}

Setup_State :: struct {
	games:        []Api_Game_List_Item,
	games_loaded: bool,
	sel_game:     int, // index into games, -1 = none
	sel_player:   int, // chosen player id, 0 = none
	status:       string,
	// create-game form
	show_create:  bool,
	name_buf:     [48]u8,
	name_len:     int,
	race_idx:     int,
	size_idx:     int,
	color_idx:    int,
	num_stars:    int,
	num_players:  int,
}

setup_init :: proc(s: ^Setup_State, c: ^Api_Client) {
	s.sel_game = -1
	s.sel_player = 0
	s.num_stars = 20
	s.num_players = 2
	s.size_idx = 1 // medium
	name := "Commander"
	copy(s.name_buf[:], name)
	s.name_len = len(name)
	setup_refresh(s, c)
}

setup_refresh :: proc(s: ^Setup_State, c: ^Api_Client) {
	// NOTE: leaks the previous list's allocations on each refresh. Fine for
	// the handful of refreshes a setup session does.
	games, ok := api_get_games(c)
	s.games = games
	s.games_loaded = ok
	if !ok {
		s.status = "failed to load games"
	} else if len(games) == 0 {
		s.status = "no games yet - create one"
	}
}

// Draws and updates the setup screen for one frame. Returns joined=true
// (with the chosen game_id + player_id) once the player commits. Sizes go
// through px()/pxi() and columns are positioned from the live width, so the
// screen scales and reflows with the window.
setup_screen :: proc(s: ^Setup_State, c: ^Api_Client) -> (game_id: string, player_id: int, joined: bool) {
	m := px(40)
	rl.DrawText("SIMPLE SPACE GAME - Odin Client", i32(m), pxi(28), pxi(26), rl.RAYWHITE)
	rl.DrawText("select a game and player, or create a new game", i32(m), pxi(66), pxi(16), {150, 165, 190, 255})

	if s.show_create {
		return setup_draw_create(s, c)
	}

	vw := f32(rl.GetScreenWidth())
	col_w := vw * 0.5 - px(60)
	rx := vw * 0.5 + px(20)
	mp := rl.GetMousePosition()

	// --- Games list (left column) ---
	rl.DrawText("GAMES", i32(m), pxi(108), pxi(16), {130, 150, 180, 255})
	list_y := px(136)
	row_h := px(62)
	if !s.games_loaded || len(s.games) == 0 {
		rl.DrawText("(no games)", i32(m), i32(list_y) + pxi(6), pxi(16), rl.GRAY)
	}
	for g, i in s.games {
		rec := rl.Rectangle{m, list_y + f32(i) * row_h, col_w, row_h - px(8)}
		hover := rl.CheckCollisionPointRec(mp, rec)
		bg := rl.Color{26, 32, 46, 255}
		if i == s.sel_game do bg = {44, 60, 92, 255}
		else if hover do bg = {34, 42, 60, 255}
		rl.DrawRectangleRec(rec, bg)
		rl.DrawRectangleLinesEx(rec, max(1, px(1)), {70, 84, 110, 255})

		short := g.game_id
		if len(short) > 8 do short = short[:8]
		title := fmt.ctprintf("%s   turn %.1f   %d stars", short, g.current_turn, g.num_stars)
		rl.DrawText(title, i32(rec.x) + pxi(12), i32(rec.y) + pxi(9), pxi(16), rl.RAYWHITE)
		sub := fmt.ctprintf("%d player(s)", len(g.players))
		rl.DrawText(sub, i32(rec.x) + pxi(12), i32(rec.y) + pxi(33), pxi(14), {160, 175, 200, 255})

		if hover && rl.IsMouseButtonPressed(.LEFT) {
			s.sel_game = i
			s.sel_player = 0 // reset player choice when switching games
		}
	}

	// --- Right column: players of the selected game ---
	rl.DrawText("PLAYERS", i32(rx), pxi(108), pxi(16), {130, 150, 180, 255})
	py := px(136)
	if s.sel_game >= 0 && s.sel_game < len(s.games) {
		for p in s.games[s.sel_game].players {
			rec := rl.Rectangle{rx, py, col_w, px(44)}
			hover := rl.CheckCollisionPointRec(mp, rec)
			bg := rl.Color{26, 32, 46, 255}
			if p.id == s.sel_player do bg = {44, 60, 92, 255}
			else if hover do bg = {34, 42, 60, 255}
			rl.DrawRectangleRec(rec, bg)
			rl.DrawRectangleLinesEx(rec, max(1, px(1)), {70, 84, 110, 255})
			swatch := rl.Rectangle{rec.x + px(8), rec.y + px(12), px(20), px(20)}
			rl.DrawRectangleRec(swatch, parse_hex_color(p.color))
			label := fmt.ctprintf("#%d  %s  (%s)", p.id, p.name, p.race_name)
			rl.DrawText(label, i32(rec.x) + pxi(40), i32(rec.y) + pxi(13), pxi(15), rl.RAYWHITE)
			if hover && rl.IsMouseButtonPressed(.LEFT) do s.sel_player = p.id
			py += px(52)
		}
	} else {
		rl.DrawText("(select a game)", i32(rx), i32(py) + pxi(6), pxi(15), rl.GRAY)
	}

	// --- Action buttons (anchored near the bottom of the live window) ---
	by := f32(rl.GetScreenHeight()) - px(110)
	if gui_button({m, by, px(150), px(44)}, "Refresh") do setup_refresh(s, c)
	if gui_button({m + px(166), by, px(170), px(44)}, "New Game") {
		s.show_create = true
	}
	can_join := s.sel_game >= 0 && s.sel_player > 0
	if gui_button({rx, by, col_w, px(48)}, "Join", can_join) && can_join {
		gid := strings.clone(s.games[s.sel_game].game_id)
		return gid, s.sel_player, true
	}

	if s.status != "" {
		rl.DrawText(fmt.ctprintf("%s", s.status), i32(m), i32(by) + pxi(56), pxi(14), {200, 180, 120, 255})
	}
	return "", 0, false
}

setup_draw_create :: proc(s: ^Setup_State, c: ^Api_Client) -> (game_id: string, player_id: int, joined: bool) {
	rl.DrawText("NEW GAME", i32(px(40)), pxi(108), pxi(20), {130, 150, 180, 255})

	x := px(40)
	y := px(152)
	label_w := pxi(160)
	row := px(48)

	// Player name — text field (always focused while this form is open).
	text_input_update(s.name_buf[:], &s.name_len)
	rl.DrawText("Player name", i32(x), i32(y) + pxi(8), pxi(15), {170, 185, 210, 255})
	field := rl.Rectangle{x + f32(label_w), y, px(320), px(34)}
	rl.DrawRectangleRec(field, {20, 26, 38, 255})
	rl.DrawRectangleLinesEx(field, max(1, px(1)), {90, 130, 200, 255})
	name_c := fmt.ctprintf("%s_", string(s.name_buf[:s.name_len]))
	rl.DrawText(name_c, i32(field.x) + pxi(8), i32(field.y) + pxi(9), pxi(16), rl.RAYWHITE)
	y += px(52)

	// Cycling enum rows: left-click = next, right-click = previous.
	s.race_idx = cycle_row(x, y, label_w, "Race", setup_races[:], s.race_idx);        y += row
	s.size_idx = cycle_row(x, y, label_w, "Galaxy size", setup_sizes[:], s.size_idx);  y += row
	s.color_idx = cycle_row(x, y, label_w, "Color", setup_colors[:], s.color_idx);     y += row

	// Numeric steppers.
	s.num_stars = stepper_row(x, y, label_w, "Stars", s.num_stars, 5, 5, 100);   y += row
	s.num_players = stepper_row(x, y, label_w, "Players", s.num_players, 1, 1, 8); y += px(60)

	if gui_button({x, y, px(150), px(46)}, "Create") {
		name := strings.trim_space(string(s.name_buf[:s.name_len]))
		if name == "" do name = "Commander"
		req := New_Game_Request {
			player_name  = name,
			player_color = setup_colors[s.color_idx],
			race         = setup_races[s.race_idx],
			galaxy_size  = setup_sizes[s.size_idx],
			num_stars    = s.num_stars,
			players      = s.num_players,
		}
		if gid, ok := api_create_game(c, req); ok {
			s.show_create = false
			setup_refresh(s, c)
			// Select the freshly created game + its first (human) player.
			for g, i in s.games {
				if g.game_id == gid {
					s.sel_game = i
					if len(g.players) > 0 do s.sel_player = g.players[0].id
					break
				}
			}
			s.status = "game created - press Join"
		} else {
			s.status = "create failed (see console)"
		}
	}
	if gui_button({x + px(166), y, px(150), px(46)}, "Cancel") {
		s.show_create = false
	}
	return "", 0, false
}

// --- Tiny immediate-mode widgets (shared with ui.odin) ---

// Optional clip rectangle for widgets drawn inside a scissored/scrolling
// region: while active, buttons outside it neither hover nor click (the
// scissor already hides their pixels; this hides their hitboxes too).
ui_clip_on: bool
ui_clip: rl.Rectangle

// Optional modal rectangle: while a modal (e.g. the research popup) is open,
// only widgets inside it respond. Top-level chrome that must stay clickable
// (the popup's own opener button) checks hover manually instead.
ui_modal_on: bool
ui_modal_rect: rl.Rectangle

// Hit-test for hand-rolled hoverable rows and gui_button: collision with
// `rec`, limited by the active clip and modal regions.
ui_hover :: proc(rec: rl.Rectangle) -> bool {
	mp := rl.GetMousePosition()
	if !rl.CheckCollisionPointRec(mp, rec) do return false
	if ui_clip_on && !rl.CheckCollisionPointRec(mp, ui_clip) do return false
	if ui_modal_on && !rl.CheckCollisionPointRec(mp, ui_modal_rect) do return false
	return true
}

gui_button :: proc(rec: rl.Rectangle, label: cstring, enabled := true) -> bool {
	hover := ui_hover(rec)
	bg := rl.Color{40, 48, 66, 255}
	if !enabled {
		bg = {28, 32, 42, 255}
	} else if hover {
		bg = {58, 70, 96, 255}
	}
	rl.DrawRectangleRec(rec, bg)
	rl.DrawRectangleLinesEx(rec, max(1, px(1)), {90, 104, 130, 255})
	fs := pxi(16)
	tw := rl.MeasureText(label, fs)
	tx := i32(rec.x) + (i32(rec.width) - tw) / 2
	ty := i32(rec.y) + (i32(rec.height) - fs) / 2
	col := enabled ? rl.RAYWHITE : rl.Color{120, 128, 140, 255}
	rl.DrawText(label, tx, ty, fs, col)
	return enabled && hover && rl.IsMouseButtonPressed(.LEFT)
}

// A labelled value that cycles through `options` on click. Returns the new
// index. Left-click advances, right-click goes back.
cycle_row :: proc(x, y: f32, label_w: i32, label: cstring, options: []string, idx: int) -> int {
	rl.DrawText(label, i32(x), i32(y) + pxi(8), pxi(15), {170, 185, 210, 255})
	rec := rl.Rectangle{x + f32(label_w), y, px(220), px(34)}
	val := options[idx] if len(options) > 0 else ""
	if gui_button(rec, fmt.ctprintf("< %s >", val)) {
		return (idx + 1) % len(options)
	}
	if rl.CheckCollisionPointRec(rl.GetMousePosition(), rec) && rl.IsMouseButtonPressed(.RIGHT) {
		return (idx - 1 + len(options)) % len(options)
	}
	return idx
}

// A labelled integer with [-]/[+] steppers, clamped to [lo, hi].
stepper_row :: proc(x, y: f32, label_w: i32, label: cstring, value, step, lo, hi: int) -> int {
	rl.DrawText(label, i32(x), i32(y) + pxi(8), pxi(15), {170, 185, 210, 255})
	v := value
	if gui_button({x + f32(label_w), y, px(36), px(34)}, "-") do v -= step
	box := rl.Rectangle{x + f32(label_w) + px(40), y, px(80), px(34)}
	rl.DrawRectangleRec(box, {20, 26, 38, 255})
	rl.DrawRectangleLinesEx(box, max(1, px(1)), {70, 84, 110, 255})
	num := fmt.ctprintf("%d", value)
	rl.DrawText(num, i32(box.x) + (pxi(80) - rl.MeasureText(num, pxi(16))) / 2, i32(box.y) + pxi(9), pxi(16), rl.RAYWHITE)
	if gui_button({box.x + px(84), y, px(36), px(34)}, "+") do v += step
	return clamp(v, lo, hi)
}

text_input_update :: proc(buf: []u8, length: ^int) {
	for {
		ch := rl.GetCharPressed()
		if ch == 0 do break
		if ch >= 32 && ch < 127 && length^ < len(buf) {
			buf[length^] = u8(ch)
			length^ += 1
		}
	}
	if rl.IsKeyPressed(.BACKSPACE) && length^ > 0 {
		length^ -= 1
	}
}
