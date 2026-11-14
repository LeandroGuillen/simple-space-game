import { createMemo } from "solid-js";

import { DiplomacyContacts } from "../components/DiplomacyContacts";
import { diplomacyContacts } from "../lib/diplomacy";
import { useAppContext } from "../state/app-context";

export function DiplomacyPage() {
  const { state, declareWar, cancelDeclareWar } = useAppContext();
  const queuedWarIds = createMemo(() => (state.pendingCommands.declareWar ?? []).map((command) => command.player_id));
  const rows = createMemo(() => diplomacyContacts(state.game, state.session.playerId));

  return (
    <section class="page-grid">
      <section class="panel">
        <div class="fleet-list-header">
          <div>
            <p class="eyebrow">Diplomacy</p>
            <h2>Known empires</h2>
          </div>
        </div>
        <DiplomacyContacts
          rows={rows()}
          queuedWarIds={queuedWarIds()}
          onDeclareWar={declareWar}
          onCancelDeclareWar={cancelDeclareWar}
        />
      </section>
    </section>
  );
}
