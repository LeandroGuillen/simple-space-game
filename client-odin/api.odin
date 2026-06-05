package main

import "core:encoding/json"
import "core:fmt"
import "core:strconv"

// Typed wrappers over the HTTP client. Mirrors the endpoints used by
// client-solid/src/lib/api.ts.

Api_Client :: struct {
	host:      string,
	port:      int,
	player_id: int,
}

new_api_client :: proc(host := "localhost", port := 8080, player_id := 1) -> Api_Client {
	return {host = host, port = port, player_id = player_id}
}

@(private="file")
game_headers :: proc(c: ^Api_Client, game_id: string) -> []Http_Header {
	headers := make([]Http_Header, 2, context.temp_allocator)
	pid_buf := make([]byte, 16, context.temp_allocator)
	pid_s := strconv.write_int(pid_buf, i64(c.player_id), 10)
	headers[0] = {"Game-Id", game_id}
	headers[1] = {"Player-Id", pid_s}
	return headers
}

api_get_status :: proc(c: ^Api_Client) -> (Api_Status, bool) {
	resp, ok := http_get(c.host, c.port, "/status", allocator = context.temp_allocator)
	if !ok do return {}, false
	if resp.status_code != 200 {
		fmt.eprintfln("api: /status %d: %s", resp.status_code, string(resp.body))
		return {}, false
	}
	out: Api_Status
	if err := json.unmarshal(resp.body, &out, allocator = context.temp_allocator); err != nil {
		fmt.eprintfln("api: /status decode: %v", err)
		return {}, false
	}
	return out, true
}

api_get_games :: proc(c: ^Api_Client, allocator := context.allocator) -> ([]Api_Game_List_Item, bool) {
	resp, ok := http_get(c.host, c.port, "/games", allocator = allocator)
	if !ok do return nil, false
	if resp.status_code != 200 {
		fmt.eprintfln("api: /games %d", resp.status_code)
		return nil, false
	}
	wrapper: Api_Game_List_Response
	if err := json.unmarshal(resp.body, &wrapper, allocator = allocator); err != nil {
		fmt.eprintfln("api: /games decode: %v", err)
		return nil, false
	}
	return wrapper.game_list, true
}

api_get_game :: proc(c: ^Api_Client, game_id: string, allocator := context.allocator) -> (Api_Game, bool) {
	path := fmt.tprintf("/games/%s", game_id)
	resp, ok := http_get(c.host, c.port, path, game_headers(c, game_id), allocator)
	if !ok do return {}, false
	if resp.status_code != 200 {
		fmt.eprintfln("api: GET /games/%s -> %d: %s", game_id, resp.status_code, string(resp.body))
		return {}, false
	}
	out: Api_Game
	if err := json.unmarshal(resp.body, &out, allocator = allocator); err != nil {
		fmt.eprintfln("api: /games/{id} decode: %v", err)
		return {}, false
	}
	return out, true
}

api_end_turn :: proc(c: ^Api_Client, game_id: string, commands: Turn_Commands_Body) -> (Api_Turn_Response, bool) {
	req := Turn_Request{game_id = game_id, commands = commands}
	body, merr := json.marshal(req, allocator = context.temp_allocator)
	if merr != nil {
		fmt.eprintfln("api: marshal turn: %v", merr)
		return {}, false
	}
	resp, ok := http_post_json(c.host, c.port, "/turn", body, game_headers(c, game_id), context.temp_allocator)
	if !ok do return {}, false
	if resp.status_code != 200 {
		fmt.eprintfln("api: POST /turn -> %d: %s", resp.status_code, string(resp.body))
		return {}, false
	}
	out: Api_Turn_Response
	if err := json.unmarshal(resp.body, &out, allocator = context.temp_allocator); err != nil {
		fmt.eprintfln("api: /turn decode: %v", err)
		return {}, false
	}
	return out, true
}
