import { useParams } from '@solidjs/router';
import { createMemo, For, Show } from 'solid-js';
import { ConstructionPanel } from '../components/ConstructionPanel';
import { PlanetVisual } from '../components/PlanetVisual';
import { PopulationPanel } from '../components/PopulationPanel';
import { findPlanet, getPlanetIndustryForWorkers, labelForEnvironment, labelForGravity, labelForMinerals, labelForSize } from '../lib/utils';
import { useAppContext } from '../state/app-context';

export function PlanetPage() {
  const params = useParams();
  const { state } = useAppContext();
  const planet = () => findPlanet(state.game, Number(params.planetId));
  const projectedPopGrowth = createMemo(() => {
    const current = planet();
    if (!current || current.population <= 0) {
      return 0;
    }

    let farmers = current.food.farmers;
    for (const move of state.pendingCommands.movePopulation.filter((cmd) => cmd.planetId === current.id)) {
      if (move.to === 'farmers') {
        farmers += move.count;
      }
      if (move.from === 'farmers') {
        farmers -= move.count;
      }
    }
    farmers = Math.max(0, farmers);

    const planetCapacity = Math.floor(current.maxPopulation / 1000);
    const popUnits = Math.floor(current.population / 1000);
    const freeSpace = planetCapacity - popUnits;
    const basicIncrement = planetCapacity > 0 && freeSpace > 0
      ? Math.floor(Math.sqrt((2000 * popUnits * freeSpace) / planetCapacity))
      : 0;
    const foodProduced = farmers * current.food.produced_per_farmer + current.food.produced_per_building;
    const foodLack = popUnits - foodProduced;
    const foodLackPenalty = foodLack > 0 ? 50 * foodLack : 0;

    return Math.floor(basicIncrement - foodLackPenalty);
  });
  const incomeBreakdown = createMemo(() => {
    const current = planet();
    if (!current) {
      return { taxes: 0, buildings: 0, tradeGoods: 0, maintenance: 0, total: 0 };
    }

    let farmers = current.food.farmers;
    let workers = current.industry.workers;
    for (const move of state.pendingCommands.movePopulation.filter((cmd) => cmd.planetId === current.id)) {
      if (move.to === 'farmers') {
        farmers += move.count;
      }
      if (move.from === 'farmers') {
        farmers -= move.count;
      }
      if (move.to === 'workers') {
        workers += move.count;
      }
      if (move.from === 'workers') {
        workers -= move.count;
      }
    }

    const hasPendingBuild = state.pendingCommands.build.some((cmd) => cmd.planetId === current.id);
    const food = Math.max(0, farmers) * current.food.produced_per_farmer + current.food.produced_per_building;
    const foodSurplus = Math.max(0, food - Math.floor(current.population / 1000));
    const industry = getPlanetIndustryForWorkers(current, workers);
    const tradeGoods = current.isBuilding || hasPendingBuild ? 0 : Math.floor(foodSurplus / 2) + Math.floor(industry / 2);
    const total = current.income.taxes + current.income.buildings + tradeGoods - current.income.maintenance;

    return {
      taxes: current.income.taxes,
      buildings: current.income.buildings,
      tradeGoods,
      maintenance: current.income.maintenance,
      total,
    };
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
                <article class='metric-card income-metric' aria-label={`Income ${incomeBreakdown().total} BC`}>
                  <span>Income</span>
                  <div class='income-value'>
                    <For each={incomeIcons()}>
                      {(icon) => <span class='pop-stat-icon income-icon'>{icon}</span>}
                    </For>
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
