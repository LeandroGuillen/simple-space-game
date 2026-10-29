# Space Combat (MoO2-style tactical) — Plan & Progress

Decisions made with the user (2026-06-11):

- **Architecture**: server-authoritative interactive tactical combat. Server holds
  battle state, validates each move/fire command; the Solid client renders with
  **Phaser 3** (latest) and sends orders.
- **v1 scope**: square grid, per-ship initiative order, movement, beam weapons with
  range dissipation, armor/structure damage, Done/Wait/Auto buttons. (Shields/facing,
  missiles, boarding, retreat are later passes — show disabled buttons for flavor.)
- **Opponents**: tactical screen only for battles involving the human player
  (player 1, marked `is_human` on the server). AI-vs-AI battles keep using the
  existing instant auto-resolve in `server/game/battle.go`.
- **Art**: procedural placeholders in Phaser (hull silhouettes by size, player
  colors, beam lines, particle explosions, starfield). No sprite assets.

## Design

### Server

- `model/tactical_battle.go`: `TacticalBattle` (grid 14x10, round, ships, turn
  order, event log, status) + `TacticalShip` (grid pos, combat speed = drive
  CombatSpeed fallback hull BaseSpeed, beams, ShipStatus, fleet provenance) +
  `BattleEvent` (seq-numbered: move/fire/destroyed/round/finish — drives client
  animation).
- `game/tactical_battle.go`: creation from contested star fleets (reuses
  `materializeBattleShip`-style design lookup), action application
  (`move|fire|done|wait|auto`), simple AI for non-human sides (close distance,
  fire everything in range), finish → apply survivors back to fleets (like
  `applyBattleSurvivors`) + append `BattleReport` + remove pending battle.
- `game/battle.go` `resolveStarBattles`: stars whose fight involves a human
  player produce a pending `TacticalBattle` instead of auto-resolve.
- `SubmitTurn`: rejected for a human player with a pending battle ("resolve
  battle first"). Safety: `AdvanceTurn` auto-resolves any leftover pending
  battles before processing (covers Odin client / stale states).
- `Game` JSON gains `pending_battles` (light summaries). Full state via REST:
  - `GET  /battles/:battleID` (player-id header) → state + events since `?since=seq`
  - `POST /battles/:battleID/action` `{type, ship_id, x, y, target_id}` →
    applies human action, runs AI activations until next human ship or battle
    end, returns new events + state. Broadcasts game update when battle ends.
- Combat math reuses `GetChanceToHit`, `GetDamage` (dissipation), `FireOn`-style
  damage application, race attack/defense bonuses. Distance = Chebyshev.
- Round cap 50 → draw, all ships survive (mirrors auto-resolve cap behavior).
- `model.Player.IsHuman` set for player 1 in `NewGame` (and scenario loads keep
  whatever the save has; default false).

### Client (client-solid)

- `npm install phaser` (v3.x latest).
- Route `/battle/:battleId` → `BattlePage.tsx`: full-screen, no AppShell chrome
  context — Phaser canvas + bottom HUD bar styled after MoO2 reference
  (`references/battle.png`): left ship portrait/name panel, weapons list,
  center buttons (AUTO, WAIT, DONE active; SCAN, BOARD, RETREAT disabled),
  right "Systems" readout (armor/structure/speed/beams remaining).
- `lib/battle-api.ts` (or extend `api.ts`): getBattle, postBattleAction.
- Phaser `BattleScene`: starfield, subtle grid, procedural ship sprites
  (size/class + player color), HP bars, active-ship highlight, movement range
  overlay, click-to-move (animated), click-enemy-to-fire (beam flash, damage
  popup, explosion on kill). Consumes server event log sequentially.
- Hook into game flow: after end-turn response / game update with a
  `pending_battles` entry involving the player → navigate to battle screen.
  After battle finishes → back to `/galaxy` (and refresh game).

### Known v1 simplifications (documented, future passes)

- ~~Ground invasions still resolve during AdvanceTurn even if orbit battle is
  pending (sequencing quirk).~~ **Fixed (v0.2.13)**: Invade orders at a star
  with an active pending tactical battle move to `postponed_invasions` and
  resolve (or cancel) only after the battle finishes, and only if the invader
  still `holdsOrbit` (has parked ships; no rival armed ships remain). AI-vs-AI
  auto-resolve uses the same orbit check in the same turn. Turn-ephemeral
  `pending_invasions` are unchanged (cleared on hydrate / AdvanceTurn start /
  failed advance); postponed entries persist with the pending battle and are
  pruned on hydrate / rejected on upload if the battle/fleet/planet is gone.
- Odin client cannot fight tactically; its turn submit is rejected while a
  battle is pending — it would need an "auto" call (server supports `auto`).
- No shields/facing, missiles, boarding, retreat, ship systems damage.
- Multi-side (3+ owners) battles: supported by the model; AI plays all
  non-human sides.

## Tasks

- [x] Explore codebase, clarify requirements with user
- [x] Server: `model/tactical_battle.go` (state + events + JSON wire shapes)
- [x] Server: `game/tactical_battle.go` creation + turn order + grid placement
- [x] Server: actions (move/fire/done/wait/auto) + AI activations + finish/apply-survivors
- [x] Server: `resolveStarBattles` split (human → pending, AI-only → auto), `IsHuman` flag
- [x] Server: SubmitTurn guard + AdvanceTurn leftover auto-resolve
- [x] Server: REST controller + routes + full battles in game JSON (`pending_battles`)
- [x] Server: tests (8 new in `game/tactical_battle_test.go`, all `go test ./...` green)
- [x] Client: install phaser, types, api methods
- [x] Client: BattlePage route + HUD layout (buttons per reference image)
- [x] Client: Phaser BattleScene (grid, ships, movement, beams, explosions)
- [x] Client: event-log animation consumption + battle-end flow back to galaxy
- [x] Client: navigation hook when pending battle appears
- [x] `go test ./...`, `npm run build`, `npm test` green
- [x] Manual verification (scenario with two hostile fleets at one star)

## Status log

- 2026-06-11: Plan agreed with user, file created. Implementation starting.
- 2026-06-11: Server-side complete (8 tests green). Client-side implementation found complete by previous dev. Fixed pre-existing TS build errors (construction.test.ts, GalaxyPage.tsx, MutationPage.tsx). All tests and build green.
- 2026-06-11: Verified `battledemo` scenario generates correct game state. Battle triggers on turn advance: 3 human Laser Gunships (speed 10, 3×Laser) vs 2 AI Fusion Raiders (speed 8, 2×Fusion) on 14×10 grid. All tasks complete.
- 2026-10-29 (v0.2.13): Ground invasions wait for orbit. `holdsOrbit` + `postponed_invasions`; release from `finishTacticalBattle`; cancelled InvasionReport on orbit loss.
