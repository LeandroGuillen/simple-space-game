# Simple Space Game

A turn-based space strategy game with a Go server, a Solid web client, and an optional Odin client and AI manager. The project is hosted at [github.com/LeandroGuillen/simple-space-game](https://github.com/LeandroGuillen/simple-space-game).

## Run locally

In separate terminals from the repository root:

```sh
cd server && go run ./cmd/server
cd client-solid && npm install && npm run dev
cd server && go run ./cmd/ai
```

Open the Vite URL printed by the client. The game server listens on port **8080** by default; the optional AI manager uses **8090**. Change the server port with `go run ./cmd/server -port 8081` and the AI manager address with `go run ./cmd/ai -listen :8091`.

Scenario files are read from the directory selected by `SSG_SAVES_DIR`; see [server/game/saves_path.go](server/game/saves_path.go) for the default search locations. See [client-odin/README.md](client-odin/README.md) for the native client.

## Verify

Run `go test ./...` from `server/`, and `npm test` and `npm run build` from `client-solid/`.
