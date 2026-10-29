# ssg-odin

Native Odin client for Simple Space Game — an MVP alternative to `client-solid`.

## Requirements

- Odin (dev-2026-05 or newer)
- raylib via `vendor:raylib` (already bundled with Odin's tree)
- The Go server running on `localhost:8080`

## Build / run

```sh
./build.sh                          # produces ./ssg-odin
./ssg-odin m1                       # smoke-test: print /status and /games
./ssg-odin endturn [game-id]        # POST /turn with no commands (round-trip test)
./ssg-odin ui [game-id]             # full UI; opens the setup/join screen,
                                    # or fast-paths into [game-id] if given
odin test .                         # @(test) procs (e.g. invasion report labels)
```

The UI opens on a **setup screen**: pick a game and a player slot and press
Join, or create a new game from the in-app form. You can still pass a game id
on the CLI to skip straight in (as the default player).

If you'd rather create a game out-of-band, use the Solid client or `curl`:

```sh
curl -s -X POST http://localhost:8080/games -H 'Content-Type: application/json' -d '{
  "player_name": "Test", "player_color": "#007AFF",
  "num_stars": 20, "difficulty": "medium",
  "galaxy_size": "tiny", "galaxy_age": "average",
  "players": 2, "tech_level": "average", "race": "human"
}'
```

## UI controls

- right-drag — pan
- mouse wheel — zoom (around cursor)
- left-click — select nearest star
- `1`..`9` — select the corresponding fleet from your fleet list
- with a fleet selected, left-click a star to **queue** a fleet move
- `E` or the **End Turn** button — submit all queued commands as one `/turn`
- panel tabs switch screens (only Galaxy is live in Phase 0)
- **Menu** button (panel top-right) — leave the game and return to setup

## Layout

Flat package (`package main`); subdivided into:

- `main.odin`   — entry, mode dispatch
- `http.odin`   — HTTP/1.1 client over `core:net` TCP (chunked supported)
- `ws.odin`     — RFC 6455 WebSocket client; handshake + frame reader
                  with continuation-frame assembly; runs on its own thread
- `api.odin`    — typed wrappers over the endpoints used by the client
- `types.odin`  — JSON-mapped structs mirroring the server's wire format
- `ui.odin`     — raylib window, galaxy rendering, side panel, input

The plan called for subpackages (`net/`, `model/`, `ui/`) but flat keeps
the import boilerplate down for an MVP. Easy to split later.

## What's notably missing

- TLS — works only with plaintext local server
- Auth — server has none, so neither does the client
- Race / ship / research / colony screens
- Population transfers, build queues, colonize, exploration commands
- Reconnecting WebSocket; if the connection drops the client just stops
  receiving updates until restart
