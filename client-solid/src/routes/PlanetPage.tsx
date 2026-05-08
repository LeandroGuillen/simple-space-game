import { useParams } from '@solidjs/router';
import { createMemo, For, Show } from 'solid-js';
import { ConstructionPanel } from '../components/ConstructionPanel';
import { PlanetVisual } from '../components/PlanetVisual';
import { PopulationPanel } from '../components/PopulationPanel';
import { findPlanet, getPlanetProjectedIncome, getPlanetProjectedPopulationGrowth, labelForEnvironment, labelForGravity, labelForMinerals, labelForSize } from '../lib/utils';
import { useAppContext } from '../state/app-context';

export function PlanetPage() {
  const params = useParams();
  const { state } = useAppContext();
  const planet = () => findPlanet(state.game, Number(params.planetId));
  const projectedPopGrowth = createMemo(() => {
    const current = planet();
    return current ? getPlanetProjectedPopulationGrowth(current, state.pendingCommands) : 0;
  });
  const incomeBreakdown = createMemo(() => {
    const current = planet();
    if (!current) {
      return { taxes: 0, buildings: 0, tradeGoods: 0, maintenance: 0, total: 0, bonus: 0 };
    }

    return getPlanetProjectedIncome(current, state.pendingCommands);
  });
  const incomeIcons = createMemo(() => {
    const total = Math.max(0, incomeBreakdown().total);
    const bags = Math.floor(total / 10);
    const coins = total % 10;
    return [
      ...Array.from({ length: bags }, () => '💰'),
      ...Array.from({ length: coins }, () => '🪙'),
    ];
  });

  return (
    <Show when={planet()} fallback={<section class='panel'>Planet not found.</section>}>
      {(currentPlanet) => (
        <section class='page-grid'>
          <section class='panel'>
            <div class='feature-header'>
              <PlanetVisual environment={currentPlanet().environment} size='large' />
              <div class='planet-details'>
                <h2>{currentPlanet().name}</h2>
                <p class='lede'>
                  {labelForEnvironment(currentPlanet().environment)} · {labelForSize(currentPlanet().size)} ·{' '}
                  {labelForGravity(currentPlanet().gravity)} gravity · {labelForMinerals(currentPlanet().mineralClass)}
                </p>
                <article class='metric-card'>
                  <span>Population</span>
                  <strong>
                    {currentPlanet().population}/{currentPlanet().maxPopulation}
                  </strong>
                  <Show when={projectedPopGrowth() >= 0}>
                    <span>+{projectedPopGrowth()} this turn</span>
                  </Show>
                  <Show when={projectedPopGrowth() < 0}>
                    <span class='starving-warning'>{projectedPopGrowth()} this turn</span>
                  </Show>
                </article>
                <article
                  class='metric-card income-metric'
                  classList={{ 'income-metric--negative': incomeBreakdown().total < 0 }}
                  aria-label={`Income ${incomeBreakdown().total} BC`}
                >
                  <span>Income</span>
                  <div class='income-value'>
                    <Show
                      when={incomeBreakdown().total < 0}
                      fallback={
                        <For each={incomeIcons()}>
                          {(icon) => <span class='pop-stat-icon income-icon'>{icon}</span>}
                        </For>
                      }
                    >
                      <strong>{incomeBreakdown().total} BC</strong>
                    </Show>
                  </div>
                  <div class='pop-output-tooltip income-tooltip'>
                    <div class='pop-output-tooltip-row pop-output-tooltip-row--total'>
                      <span>Total income:</span>
                      <strong>{incomeBreakdown().total} BC</strong>
                    </div>
                    <div class='pop-output-tooltip-row'>
                      <span>Taxes:</span>
                      <strong>{incomeBreakdown().taxes} BC</strong>
                    </div>
                    <div class='pop-output-tooltip-row'>
                      <span>Buildings:</span>
                      <strong>{incomeBreakdown().buildings} BC</strong>
                    </div>
                    <div class='pop-output-tooltip-row'>
                      <span>Trade Goods:</span>
                      <strong>{incomeBreakdown().tradeGoods} BC</strong>
                    </div>
                    <Show when={incomeBreakdown().bonus !== 0}>
                      <div class='pop-output-tooltip-row'>
                        <span>Income bonus:</span>
                        <strong>+{Math.round(incomeBreakdown().bonus * 100)}%</strong>
                      </div>
                    </Show>
                    <div class='pop-output-tooltip-row'>
                      <span>Maintenance:</span>
                      <strong>-{incomeBreakdown().maintenance} BC</strong>
                    </div>
                  </div>
                </article>
                <article class='metric-card'>
                  <span>Morale</span>
                  <div class='pop-stat-icons'>
                    <span class='pop-stat-icon'>😊</span>
                    <span class='pop-stat-icon'>😊</span>
                    <span class='pop-stat-icon'>😊</span>
                    <span class='pop-stat-icon'>😐</span>
                  </div>
                </article>
              </div>
              <ConstructionPanel planet={currentPlanet()} />
            </div>
          </section>

          <PopulationPanel planet={currentPlanet()} />
        </section>
      )}
    </Show>
  );
}
