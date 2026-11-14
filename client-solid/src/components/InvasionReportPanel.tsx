import { For, Show } from "solid-js";

import { UNKNOWN_EMPIRE } from "../lib/diplomacy";
import type { ApiInvasionReport } from "../lib/types/api";

type InvasionReportPanelProps = {
  reports: ApiInvasionReport[];
  playerId: number;
  playerNames: Map<number, string>;
};

export function InvasionReportPanel(props: InvasionReportPanelProps) {
  const visibleReports = () => props.reports.filter((report) => report.attacker_owner_id === props.playerId || report.defender_owner_id === props.playerId);
  const getPlayerName = (playerId: number) => props.playerNames.get(playerId) || UNKNOWN_EMPIRE;

  return (
    <Show when={visibleReports().length > 0}>
      <div class="invasion-reports">
        <h3>Invasion Reports</h3>
        <For each={visibleReports()}>
          {(report) => (
            <div class="invasion-report">
              <div class="invasion-report-header">
                <h4>
                  {report.cancelled ? "🚫" : report.success ? "🎯" : "❌"} Invasion of {report.planet_name}
                </h4>
                <div class="invasion-participants">
                  <span class="attacker">
                    {getPlayerName(report.attacker_owner_id)} attacked
                  </span>
                  <span class="defender">
                    {getPlayerName(report.defender_owner_id)}'s colony
                  </span>
                </div>
              </div>

              <div class="invasion-outcome">
                <Show when={report.cancelled}>
                  <div class="invasion-cancelled">
                    Invasion cancelled — orbit lost before landing
                  </div>
                </Show>
                <Show when={!report.cancelled && report.planet_captured}>
                  <div class="planet-captured">
                    ✅ Planet captured by {getPlayerName(report.attacker_owner_id)}
                  </div>
                  <Show when={report.capture_effects}>
                    {(effects) => <div class="capture-effects">
                      <div>Population: {effects().population_race || "Unknown"} remains on the planet.</div>
                      <div>Conquest unrest: {Math.round(effects().unrest_penalty * 100)}% morale penalty.</div>
                      <div>Buildings destroyed: {(effects().destroyed_buildings ?? []).join(", ") || "None"}.</div>
                      <div>Former empire buildings removed: {(effects().removed_buildings ?? []).join(", ") || "None"}.</div>
                      <div>Construction project and queue cleared.</div>
                    </div>}
                  </Show>
                </Show>
                <Show when={!report.cancelled && !report.planet_captured}>
                  <div class="invasion-failed">
                    ⚔️ Invasion repelled by defenders
                  </div>
                </Show>
              </div>

              <div class="invasion-forces">
                <div class="initial-forces">
                  <div class="force attacker-force">
                    <h5>Attacking Force</h5>
                    <div class="unit-count">
                      Marines: {(report.initial_attacker.marines ?? []).length}
                    </div>
                    <div class="unit-count">
                      Armored: {(report.initial_attacker.armored ?? []).length}
                    </div>
                  </div>
                  
                  <div class="force defender-force">
                    <h5>Defending Force</h5>
                    <div class="unit-count">
                      Marines: {(report.initial_defender.marines ?? []).length}
                    </div>
                    <div class="unit-count">
                      Armored: {(report.initial_defender.armored ?? []).length}
                    </div>
                  </div>
                </div>

                <Show when={(report.rounds ?? []).length > 0}>
                  <div class="battle-summary">
                    <h5>Battle Summary</h5>
                    <div class="casualty-summary">
                      Attacker losses: {(report.rounds ?? []).reduce((sum, r) => sum + r.attacker_losses, 0)} units
                    </div>
                    <div class="casualty-summary">
                      Defender losses: {(report.rounds ?? []).reduce((sum, r) => sum + r.defender_losses, 0)} units
                    </div>
                    <div class="rounds-fought">
                      Combat lasted {(report.rounds ?? []).length} round{(report.rounds ?? []).length !== 1 ? 's' : ''}
                    </div>
                  </div>
                </Show>
              </div>
            </div>
          )}
        </For>
      </div>
    </Show>
  );
}
