package main

import "core:fmt"
import "core:os"

// SSG Odin client — MVP that walks through the milestones from the plan.
//
// Run modes (first CLI arg):
//   m1               GET /status and /games, print results, exit
//   ui [game-id]     full UI; auto-joins given game id or the first one listed
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
	resp, ok := api_end_turn(c, game_id, Turn_Commands_Body{})
	if !ok {
		fmt.eprintln("endturn: request failed")
		return
	}
	fmt.printfln("endturn: ok=%v advanced=%v ready=%v waiting=%v",
		resp.ok, resp.advanced,
		resp.turn_status.ready_player_ids,
		resp.turn_status.waiting_player_ids)
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
			"  game_id=%s  turn=%d  stars=%d  players=%d",
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
