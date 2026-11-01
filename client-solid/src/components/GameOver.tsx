import { A } from "@solidjs/router";
import { createMemo, Show } from "solid-js";
import type { GameViewModel } from "../lib/types/domain";
import { formatStardate } from "../lib/utils";

export function GameOver(props: { game: GameViewModel; playerId: number }) {
  const player = createMemo(() => props.game.players.find((p) => p.id === props.playerId));
  const colonies = createMemo(() => props.game.galaxy.stars.flatMap((s) => s.planets).filter((p) => p.owner === props.playerId));
  const ships = createMemo(() => props.game.fleets.filter((f) => f.ownerId === props.playerId).reduce((total, f) => total + f.ships.length, 0));
  const title = createMemo(() => {
    if (props.game.winnerId === props.playerId) return "Victory";
    if (player()?.eliminated) return "Defeat";
    return "Game Over";
  });
  const winner = createMemo(() => props.game.players.find((p) => p.id === props.game.winnerId));
  return (
    <section class="panel game-over" aria-labelledby="game-over-title">
      <p class="eyebrow">Empire report</p>
      <h1 id="game-over-title">{title()}</h1>
      <Show when={winner()} fallback={<p>Your empire can no longer continue.</p>}>
        {(winningPlayer) => <p>{winningPlayer().name} won by conquest.</p>}
      </Show>
      <dl class="game-over-stats">
        <div><dt>Turns played</dt><dd>{props.game.turnsPlayed ?? 0}</dd></div>
        <div><dt>Stardate</dt><dd>{formatStardate(props.game.currentTurn)}</dd></div>
        <div><dt>Colonies</dt><dd>{colonies().length}</dd></div>
        <div><dt>Population</dt><dd>{colonies().reduce((sum, p) => sum + p.population, 0).toLocaleString()}</dd></div>
        <div><dt>Ships</dt><dd>{ships()}</dd></div>
        <div><dt>Technologies</dt><dd>{player()?.researchedTechs.length ?? 0}</dd></div>
      </dl>
      <A class="primary-button" href="/">Return to setup</A>
    </section>
  );
}
