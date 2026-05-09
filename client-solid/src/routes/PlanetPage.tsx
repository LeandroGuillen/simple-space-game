import { A, useParams } from '@solidjs/router';
import { createMemo, For, Show } from 'solid-js';
import { ConstructionPanel } from '../components/ConstructionPanel';
import { PlanetVisual } from '../components/PlanetVisual';
import { PopulationPanel } from '../components/PopulationPanel';
import { findPlanet, findStar, getPlanetProjectedIncome, getPlanetProjectedPopulationGrowth, getPrimaryPlayer, labelForEnvironment, labelForGravity, labelForMinerals, labelForSize } from '../lib/utils';
import { useAppContext } from '../state/app-context';

export function PlanetPage() {
  const params = useParams();
  const { state } = useAppContext();
  const planet = () => findPlanet(state.game, Number(params.planetId));
  const populationUnits = (population: number) => Math.floor(population / 1000);
  const hasPopulationCapacity = (maxPopulation: number) => populationUnits(maxPopulation) > 0;
  const primaryPlayer = createMemo(() => getPrimaryPlayer(state.game));
  const playerColorById = createMemo(
    () =>
      new Map(
        (state.game?.players ?? []).map((player) => [player.id, player.color]),
      ),
  );
  const planetIconSize = (size: number) => `${[14, 16, 21, 26, 31][size - 1] ?? 22}px`;
  const planetOwnerColor = (owner: number) => playerColorById().get(owner) ?? 'var(--text-mute)';
  const canVisitPlanet = (owner: number) => owner === primaryPlayer()?.id;
  const star = createMemo(() => {
    const current = planet();
    return current ? findStar(state.game, current.parentStar) : null;
  });
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
              <Show when={star()?.planets.length}>
                <nav class='system-planet-nav' aria-label={`${star()?.name ?? 'System'} planets`}>
                  <For each={star()?.planets ?? []}>
                    {(systemPlanet) => (
                      <Show
                        when={canVisitPlanet(systemPlanet.owner)}
                        fallback={
                          <span
                            class='system-planet-link system-planet-link--disabled'
                            title={`${systemPlanet.name}: ${labelForMinerals(systemPlanet.mineralClass)}, ${populationUnits(systemPlanet.population)}/${populationUnits(systemPlanet.maxPopulation)} population`}
                          >
                            <span
                              class='system-planet-icon'
                              style={{ '--planet-nav-icon-size': planetIconSize(systemPlanet.size) }}
                            >
                              <PlanetVisual environment={systemPlanet.environment} size='small' />
                            </span>
                            <span class='system-planet-meta'>
                              <strong style={{ color: systemPlanet.owner > 0 ? planetOwnerColor(systemPlanet.owner) : 'var(--text-mute)' }}>
                                {systemPlanet.name}
                              </strong>
                              <span>{labelForMinerals(systemPlanet.mineralClass)}</span>
                              <Show when={hasPopulationCapacity(systemPlanet.maxPopulation)}>
                                <span>{populationUnits(systemPlanet.population)}/{populationUnits(systemPlanet.maxPopulation)}</span>
                              </Show>
                            </span>
                          </span>
                        }
                      >
                        <A
                          class='system-planet-link'
                          classList={{ 'system-planet-link--active': systemPlanet.id === currentPlanet().id }}
                          href={`/stars/${systemPlanet.parentStar}/planets/${systemPlanet.id}`}
                          aria-current={systemPlanet.id === currentPlanet().id ? 'page' : undefined}
                          title={`${systemPlanet.name}: ${labelForMinerals(systemPlanet.mineralClass)}, ${populationUnits(systemPlanet.population)}/${populationUnits(systemPlanet.maxPopulation)} population`}
                        >
                          <span
                            class='system-planet-icon'
                            style={{ '--planet-nav-icon-size': planetIconSize(systemPlanet.size) }}
                          >
                            <PlanetVisual environment={systemPlanet.environment} size='small' />
                          </span>
                          <span class='system-planet-meta'>
                            <strong style={{ color: planetOwnerColor(systemPlanet.owner) }}>{systemPlanet.name}</strong>
                            <span>{labelForMinerals(systemPlanet.mineralClass)}</span>
                            <Show when={hasPopulationCapacity(systemPlanet.maxPopulation)}>
                              <span>{populationUnits(systemPlanet.population)}/{populationUnits(systemPlanet.maxPopulation)}</span>
                            </Show>
                          </span>
                        </A>
                      </Show>
                    )}
                  </For>
                </nav>
              </Show>
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
