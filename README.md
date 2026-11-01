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

Scenario files are read from `server/data/scenarios` by default. Set `SSG_SAVES_DIR` to use a different scenario directory, for example `SSG_SAVES_DIR=/path/to/scenarios go run ./cmd/server` from `server/`. See [client-odin/README.md](client-odin/README.md) for the native client.

## Winning and elimination

In games with multiple player slots, an empire is eliminated when it has no colonies or colony ships. It no longer submits turns. The last remaining empire wins once it has a colony; simultaneous elimination ends the game without a winner. Outcomes wait for pending tactical battles to resolve. A one-slot game remains open as a sandbox until the player is eliminated. Saves made before turn counting was added count turns from when they are loaded.

## Galaxy sizes

The web client creates Tiny, Medium and Huge galaxies with 15, 40 and 100 stars. Tiny's 240×180 map uses a randomized, jittered layout with at least 47 pixels between stars (scaled below the usual 60 pixels per parsec so the map remains readable). Medium and Huge use 60-pixel minimum spacing at their standard star counts.

## Verify

Run `go vet ./...` and `go test -race ./...` from `server/`, and `npm ci`, `npm test` and `npm run build` from `client-solid/`. GitHub Actions runs these checks on pushes to `master` and pull requests targeting `master`.
