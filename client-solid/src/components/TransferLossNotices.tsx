import { For, Show } from "solid-js";
import type { TransferLossNotice } from "../lib/types/domain";

export function formatTransferLoss(notice: TransferLossNotice): string {
  const source = notice.sourcePlanetName || `Planet #${notice.sourcePlanetId}`;
  const destination = notice.destinationPlanetName || `Planet #${notice.destinationPlanetId}`;
  return `${notice.populationLost.toLocaleString()} colonists were lost in transit from ${source} to ${destination}.`;
}

export function TransferLossNotices(props: { notices: TransferLossNotice[]; playerId: number }) {
  const ownNotices = () => props.notices.filter((notice) => notice.owner === props.playerId);
  return (
    <Show when={ownNotices().length > 0}>
      <section class="transfer-loss-notices" role="status" aria-live="polite" aria-label="Population transfer losses">
        <strong>Population transfer losses</strong>
        <For each={ownNotices()}>{(notice) => <p>{formatTransferLoss(notice)}</p>}</For>
      </section>
    </Show>
  );
}
