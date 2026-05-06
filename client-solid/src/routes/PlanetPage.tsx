import { useParams } from '@solidjs/router';
import { createMemo, Show } from 'solid-js';
import { ConstructionPanel } from '../components/ConstructionPanel';
import { PlanetVisual } from '../components/PlanetVisual';
import { PopulationPanel } from '../components/PopulationPanel';
import { findPlanet, labelForEnvironment, labelForGravity, labelForMinerals, labelForSize } from '../lib/utils';
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
                <article class='metric-card'>
                  <span>Income</span>
                  <strong>50 BC</strong>
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
            </div>
          </section>

          <ConstructionPanel planet={currentPlanet()} />
          <PopulationPanel planet={currentPlanet()} />
        </section>
      )}
    </Show>
  );
}
