package main

import "core:fmt"
import "core:os"

// SSG Odin client — MVP that walks through the milestones from the plan.
//
// Run modes (first CLI arg):
//   m1               GET /status and /games, print results, exit
//   endturn [id]     POST an empty /turn (readies the default player)
//   wiretest [id]    print the /turn body for one of each command kind
//   ui [game-id]     full UI; setup screen, or fast-path into the given game
//   (no arg)         same as ui

main :: proc() {
	mode := "ui"
	game_id_arg := ""
	if len(os.args) >= 2 do mode = os.args[1]
	if len(os.args) >= 3 do game_id_arg = os.args[2]

	client := new_api_client()

	switch mode {
	case "m1":
		run_m1(&client)
	case "endturn":
		run_endturn(&client, game_id_arg)
	case "wiretest":
		run_wiretest(game_id_arg)
	case "ui":
		run_ui(&client, game_id_arg)
	case:
		fmt.eprintfln("unknown mode %q (expected: m1, endturn, ui)", mode)
		os.exit(2)
	}
}

run_endturn :: proc(c: ^Api_Client, game_id_arg: string) {
	game_id := game_id_arg
	if game_id == "" {
		games, ok := api_get_games(c)
		if !ok || len(games) == 0 {
			fmt.eprintln("endturn: no games")
			return
		}
		game_id = games[0].game_id
	}
	fmt.printfln("endturn: posting empty commands to game %s", game_id)
	resp, ok := api_end_turn(c, game_id)
	if !ok {
		fmt.eprintln("endturn: request failed")
		return
	}
	fmt.printfln(
		"endturn: ok=%v advanced=%v ready=%v waiting=%v",
		resp.ok,
		resp.advanced,
		resp.turn_status.ready_player_ids,
		resp.turn_status.waiting_player_ids,
	)
}

// Prints the /turn body produced by one of each command kind, so the wire
// format can be eyeballed / diffed against the web client without a UI.
run_wiretest :: proc(game_id_arg: string) {
	game_id := game_id_arg
	if game_id == "" do game_id = "test-game"

	enqueue_fleet_move(7, 3)
	enqueue_fleet_move_ships(8, 4, []int{0, 2})
	enqueue_build(11, "Factory")
	set_planet_allocation(11, 3, 2, 1)
	toggle_buy_production(11)
	enqueue_colonize(21, 9)
	enqueue_colonize(22)
	enqueue_artificial_planet(11, 23)
	enqueue_invade(10, 31, []int{1, 2})
	set_select_research("Deuterium Fuel Cells")
	set_mutate_race([]string{"creative", "food_plus_1"})

	body := build_turn_body(game_id)
	fmt.printfln("%s", string(body))
	clear_pending()
}

run_m1 :: proc(c: ^Api_Client) {
	fmt.println("== GET /status ==")
	status, ok := api_get_status(c)
	if !ok {
		fmt.println("  failed")
	} else {
		fmt.printfln("  ok=%v", status.ok)
	}

	fmt.println("== GET /games ==")
	games, gok := api_get_games(c)
	if !gok {
		fmt.println("  failed")
		return
	}
	if len(games) == 0 {
		fmt.println("  (no games)")
		return
	}
	for g in games {
		fmt.printfln(
			"  game_id=%s  turn=%.1f  stars=%d  players=%d",
			g.game_id,
			g.current_turn,
			g.num_stars,
			len(g.players),
		)
		for p in g.players {
			fmt.printfln("    - id=%d %q color=%s stars=%d", p.id, p.name, p.color, p.num_stars)
		}
	}
}
