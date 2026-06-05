package main

import "core:fmt"
import "core:mem"
import "core:net"
import "core:strconv"
import "core:strings"

// Minimal HTTP/1.1 client built directly on core:net TCP.
// Localhost-only, no TLS. Uses Connection: close so each request opens
// a fresh socket and reads until EOF — simple to get right, fine for the
// volume this client generates.

Http_Header :: struct {
	name:  string,
	value: string,
}

Http_Response :: struct {
	status_code: int,
	body:        []byte, // allocated in the allocator passed to http_request
}

http_get :: proc(host: string, port: int, path: string, extra_headers: []Http_Header = nil, allocator := context.allocator) -> (Http_Response, bool) {
	return http_request(host, port, "GET", path, extra_headers, nil, allocator)
}

http_post_json :: proc(host: string, port: int, path: string, body: []byte, extra_headers: []Http_Header = nil, allocator := context.allocator) -> (Http_Response, bool) {
	return http_request(host, port, "POST", path, extra_headers, body, allocator)
}

http_request :: proc(host: string, port: int, method: string, path: string, extra_headers: []Http_Header, body: []byte, allocator: mem.Allocator) -> (Http_Response, bool) {
	sock, derr := net.dial_tcp_from_hostname_with_port_override(host, port)
	if derr != nil {
		fmt.eprintfln("http: dial %s:%d failed: %v", host, port, derr)
		return {}, false
	}
	defer net.close(sock)

	b := strings.builder_make(context.temp_allocator)
	fmt.sbprintf(&b, "%s %s HTTP/1.1\r\n", method, path)
	fmt.sbprintf(&b, "Host: %s:%d\r\n", host, port)
	strings.write_string(&b, "Connection: close\r\n")
	if body != nil {
		strings.write_string(&b, "Content-Type: application/json\r\n")
		fmt.sbprintf(&b, "Content-Length: %d\r\n", len(body))
	}
	for h in extra_headers {
		fmt.sbprintf(&b, "%s: %s\r\n", h.name, h.value)
	}
	strings.write_string(&b, "\r\n")
	if body != nil {
		strings.write_bytes(&b, body)
	}

	if !send_all(sock, transmute([]byte)strings.to_string(b)) {
		fmt.eprintln("http: send failed")
		return {}, false
	}

	raw := make([dynamic]byte, 0, 4096, context.temp_allocator)
	buf: [4096]byte
	for {
		n, rerr := net.recv_tcp(sock, buf[:])
		if n > 0 do append(&raw, ..buf[:n])
		if n == 0 do break // EOF
		if rerr != nil && rerr != .None do break
	}

	return parse_http_response(raw[:], allocator)
}

send_all :: proc(sock: net.TCP_Socket, data: []byte) -> bool {
	remaining := data
	for len(remaining) > 0 {
		n, err := net.send_tcp(sock, remaining)
		if err != nil && err != .None do return false
		if n == 0 do return false
		remaining = remaining[n:]
	}
	return true
}

@(private="file")
parse_http_response :: proc(data: []byte, allocator: mem.Allocator) -> (Http_Response, bool) {
	sep := -1
	for i := 0; i + 3 < len(data); i += 1 {
		if data[i] == '\r' && data[i + 1] == '\n' && data[i + 2] == '\r' && data[i + 3] == '\n' {
			sep = i
			break
		}
	}
	if sep < 0 do return {}, false

	head := string(data[:sep])
	body := data[sep + 4:]

	line_end := strings.index_byte(head, '\n')
	if line_end < 0 do return {}, false
	line := strings.trim_right(head[:line_end], "\r")
	parts := strings.split(line, " ", context.temp_allocator)
	if len(parts) < 2 do return {}, false
	code, ok := strconv.parse_int(parts[1])
	if !ok do return {}, false

	rest := head[line_end + 1:]
	lower := strings.to_lower(rest, context.temp_allocator)

	if strings.contains(lower, "transfer-encoding: chunked") {
		decoded, ok := decode_chunked(body, allocator)
		if !ok do return {}, false
		return Http_Response{status_code = code, body = decoded}, true
	}

	body_copy := make([]byte, len(body), allocator)
	copy(body_copy, body)
	return Http_Response{status_code = code, body = body_copy}, true
}

@(private="file")
decode_chunked :: proc(data: []byte, allocator: mem.Allocator) -> ([]byte, bool) {
	out := make([dynamic]byte, 0, len(data), allocator)
	cursor := 0
	for cursor < len(data) {
		// chunk-size line ends at first \r\n; the line may contain ;ext
		line_end := -1
		for i := cursor; i + 1 < len(data); i += 1 {
			if data[i] == '\r' && data[i + 1] == '\n' {
				line_end = i
				break
			}
		}
		if line_end < 0 do return nil, false
		line := string(data[cursor:line_end])
		// strip any ";extension" suffix
		if semi := strings.index_byte(line, ';'); semi >= 0 do line = line[:semi]
		line = strings.trim_space(line)
		size, ok := strconv.parse_int(line, 16)
		if !ok do return nil, false
		cursor = line_end + 2
		if size == 0 do break
		if cursor + size > len(data) do return nil, false
		append(&out, ..data[cursor:cursor + size])
		cursor += size
		// trailing \r\n
		if cursor + 1 >= len(data) || data[cursor] != '\r' || data[cursor + 1] != '\n' do return nil, false
		cursor += 2
	}
	return out[:], true
}
