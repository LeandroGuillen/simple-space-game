package main

import "core:encoding/base64"
import "core:encoding/json"
import "core:fmt"
import "core:math/rand"
import "core:net"
import "core:strconv"
import "core:strings"
import "core:thread"
import "core:time"

// Minimal RFC 6455 WebSocket client, text frames only.
// 1. HTTP/1.1 GET with Upgrade headers -> expect 101.
// 2. From then on, read framed messages. Server -> client frames are
//    unmasked; client -> server frames must be masked, but we never
//    write back, so we only implement the read side.
//
// The server (gorilla/websocket) sends each game state as a complete
// text JSON frame. We assume fin=1 on every frame for the MVP.

WS_Args :: struct {
	host:      string,
	port:      int,
	path:      string, // includes ?game_id=…&player_id=…
	game_id:   string,
	player_id: int,
	gen:       int,     // session generation this reader belongs to
}

ws_start :: proc(c: ^Api_Client, game_id: string, gen: int) {
	args := new(WS_Args)
	args.host = c.host
	args.port = c.port
	args.game_id = game_id
	args.player_id = c.player_id
	args.gen = gen

	pid_buf := make([]byte, 16)
	pid_s := strconv.write_int(pid_buf, i64(c.player_id), 10)
	args.path = fmt.aprintf("/gameupdates?game_id=%s&player_id=%s", game_id, pid_s)

	thread.create_and_start_with_poly_data(args, ws_reader_thread, self_cleanup = true)
}

@(private="file")
ws_reader_thread :: proc(args: ^WS_Args) {
	defer free(args)

	// Reconnect loop: keep (re)dialing while this reader's session
	// generation is still live. After each successful handshake, pull a
	// REST snapshot so state missed while disconnected is recovered (the
	// server only pushes on changes).
	first := true
	for ws_gen_is_current(args.gen) {
		if !first do time.sleep(2 * time.Second)
		first = false

		sock, ok := ws_handshake(args.host, args.port, args.path)
		if !ok {
			fmt.eprintln("ws: handshake failed, retrying...")
			continue
		}

		client := Api_Client{host = args.host, port = args.port, player_id = args.player_id}
		if game, gok := api_get_game(&client, args.game_id); gok {
			ws_set_game(game, args.gen)
		}

		ws_frame_loop(sock, args)
		net.close(sock)
		if ws_gen_is_current(args.gen) do fmt.eprintln("ws: connection closed, reconnecting...")
	}
}

@(private="file")
ws_frame_loop :: proc(sock: net.TCP_Socket, args: ^WS_Args) {
	rx := make([dynamic]byte, 0, 1 << 16)
	defer delete(rx)

	for {
		payload, frame_ok := ws_read_text_frame(sock, &rx)
		if !frame_ok do return
		if len(payload) == 0 do continue

		// The server sends {"game_update": {...}} on every state change.
		msg: Api_Game_Update
		err := json.unmarshal(payload, &msg)
		if err != nil {
			fmt.eprintfln("ws: decode error: %v", err)
			continue
		}
		ws_set_game(msg.game_update, args.gen)
		// NOTE: This leaks the old snapshot's allocations because we
		// reuse the default allocator for everything. For an MVP that's
		// fine — the deltas are small JSON objects, not hundreds of MB.
	}
}

@(private="file")
ws_handshake :: proc(host: string, port: int, path: string) -> (net.TCP_Socket, bool) {
	sock, derr := net.dial_tcp_from_hostname_with_port_override(host, port)
	if derr != nil {
		fmt.eprintfln("ws: dial failed: %v", derr)
		return 0, false
	}

	key_bytes: [16]byte
	_ = rand.read(key_bytes[:])
	key_b64 := base64.encode(key_bytes[:], allocator = context.temp_allocator)

	b := strings.builder_make(context.temp_allocator)
	fmt.sbprintf(&b, "GET %s HTTP/1.1\r\n", path)
	fmt.sbprintf(&b, "Host: %s:%d\r\n", host, port)
	strings.write_string(&b, "Upgrade: websocket\r\n")
	strings.write_string(&b, "Connection: Upgrade\r\n")
	fmt.sbprintf(&b, "Sec-WebSocket-Key: %s\r\n", key_b64)
	strings.write_string(&b, "Sec-WebSocket-Version: 13\r\n")
	strings.write_string(&b, "\r\n")

	if !send_all(sock, transmute([]byte)strings.to_string(b)) {
		net.close(sock)
		return 0, false
	}

	// Read the handshake response one byte at a time so we never spill
	// into the first WS frame. Slow, but only runs once per connection
	// and the response is only a few hundred bytes.
	head := make([dynamic]byte, 0, 512, context.temp_allocator)
	one: [1]byte
	for {
		n, rerr := net.recv_tcp(sock, one[:])
		if n == 0 || (rerr != nil && rerr != .None) {
			fmt.eprintln("ws: lost connection during handshake")
			net.close(sock)
			return 0, false
		}
		append(&head, one[0])
		if find_header_end(head[:]) >= 0 do break
	}

	end := find_header_end(head[:])
	resp_head := string(head[:end])
	line_end := strings.index_byte(resp_head, '\n')
	if line_end < 0 {
		net.close(sock)
		return 0, false
	}
	line := strings.trim_right(resp_head[:line_end], "\r")
	parts := strings.split(line, " ", context.temp_allocator)
	if len(parts) < 2 {
		net.close(sock)
		return 0, false
	}
	code, _ := strconv.parse_int(parts[1])
	if code != 101 {
		fmt.eprintfln("ws: expected 101, got %d: %s", code, resp_head)
		net.close(sock)
		return 0, false
	}
	return sock, true
}

@(private="file")
find_header_end :: proc(data: []byte) -> int {
	for i := 0; i + 3 < len(data); i += 1 {
		if data[i] == '\r' && data[i + 1] == '\n' && data[i + 2] == '\r' && data[i + 3] == '\n' {
			return i + 4
		}
	}
	return -1
}

@(private="file")
ws_read_text_frame :: proc(sock: net.TCP_Socket, scratch: ^[dynamic]byte) -> ([]byte, bool) {
	// Assembles a complete WS message from possibly multiple frames.
	// Gorilla fragments game-state updates above ~4KB, so we must
	// concatenate continuation frames. Pings/pongs in between are
	// ignored (we never write pongs back — fine for localhost).
	clear(scratch)
	first_opcode: u8 = 0xFF // sentinel: not yet seen
	for {
		hdr: [2]byte
		if !read_exact(sock, hdr[:]) do return nil, false
		fin := (hdr[0] & 0x80) != 0
		opcode := hdr[0] & 0x0F
		masked := (hdr[1] & 0x80) != 0
		len7 := int(hdr[1] & 0x7F)

		payload_len := len7
		switch len7 {
		case 126:
			ext: [2]byte
			if !read_exact(sock, ext[:]) do return nil, false
			payload_len = int(u16(ext[0]) << 8 | u16(ext[1]))
		case 127:
			ext: [8]byte
			if !read_exact(sock, ext[:]) do return nil, false
			v: u64 = 0
			for i := 0; i < 8; i += 1 do v = (v << 8) | u64(ext[i])
			payload_len = int(v)
		}

		mask: [4]byte
		if masked {
			if !read_exact(sock, mask[:]) do return nil, false
		}

		// Control frames (high bit of opcode set: 0x8/0x9/0xA) must not
		// be fragmented and never carry application data; read past
		// their payload and continue.
		is_control := (opcode & 0x8) != 0
		if is_control {
			ctrl := make([]byte, payload_len, context.temp_allocator)
			if payload_len > 0 {
				if !read_exact(sock, ctrl) do return nil, false
				if masked do for i in 0 ..< payload_len do ctrl[i] ~= mask[i & 3]
			}
			if opcode == 0x8 do return nil, false // close
			continue                              // ping/pong: ignore
		}

		// Data frame: opcode 0x1 (text), 0x2 (binary), or 0x0 (continuation).
		if first_opcode == 0xFF do first_opcode = opcode

		off := len(scratch)
		if payload_len > 0 {
			resize(scratch, off + payload_len)
			if !read_exact(sock, scratch[off:off + payload_len]) do return nil, false
			if masked do for i in 0 ..< payload_len do scratch[off + i] ~= mask[i & 3]
		}

		if fin do break
	}

	if first_opcode != 0x1 {
		// We only handle text messages — game updates are JSON text.
		fmt.eprintfln("ws: ignoring non-text message (opcode=0x%X)", first_opcode)
		return ws_read_text_frame(sock, scratch)
	}
	return scratch[:], true
}

@(private="file")
read_exact :: proc(sock: net.TCP_Socket, dst: []byte) -> bool {
	remaining := dst
	for len(remaining) > 0 {
		n, err := net.recv_tcp(sock, remaining)
		if err != nil && err != .None do return false
		if n == 0 do return false
		remaining = remaining[n:]
	}
	return true
}

