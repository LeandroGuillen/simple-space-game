# External AI Player Plan

AI players should run outside the game server and interact with the game as normal player slots. The game server should not know whether a slot is controlled by a human, an AI process, or another kind of client.

## Runtime Shape

Future AI process inputs:

- `--server`: base HTTP URL, for example `http://127.0.0.1:8080`
- `--game`: game ID
- `--player`: player ID/slot to control

Example:

```bash
go run ./cmd/ai --server http://127.0.0.1:8080 --game <game-id> --player 2
```

## API Surface

The AI should use the same public API surface as a human client:

- `GET /games/:gameID` with `Game-Id` and `Player-Id` headers to load visible state.
- `GET /gameupdates?game_id=<game-id>&player_id=<player-id>` for websocket updates.
- `POST /turn` with `Game-Id` and `Player-Id` headers to submit orders.

The AI should not call game internals, mutate server state directly, or rely on server-side hooks. Sharing DTO packages can be considered if it avoids duplicating command payload types, but the AI should remain a client of the HTTP/WebSocket contract.

## Turn Loop

1. Load visible game state.
2. If this player ID is in `turn_status.ready_player_ids`, wait for a websocket update.
3. If this player ID is in `turn_status.waiting_player_ids`, compute commands from visible state.
4. Submit commands to `POST /turn`.
5. If `advanced=false`, wait for other slots.
6. If `advanced=true`, process the updated game state and repeat.

## Initial Policy Ideas

The first AI policy can be intentionally simple:

- Choose a research project when none is selected.
- Adjust colony labor to avoid starvation when possible.
- Build early infrastructure when available.
- Queue expansion projects when useful.
- Colonize valid pending targets.
- Move scouts or colony ships only when fuel and visibility rules are clear through the public state.

These are implementation notes only. No AI binary or AI policy is part of the multiplayer server foundation.
