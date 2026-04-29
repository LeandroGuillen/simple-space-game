import { useParams } from '@solidjs/router';
import { Show } from 'solid-js';
import { PopulationPanel } from '../components/PopulationPanel';
import { PlanetVisual } from '../components/PlanetVisual';
import { findPlanet, labelForEnvironment, labelForGravity, labelForMinerals, labelForSize } from '../lib/utils';
import { useAppContext } from '../state/app-context';

export function PlanetPage() {
  const params = useParams();
  const { state } = useAppContext();
  const planet = () => findPlanet(state.game, Number(params.planetId));

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

              <button class='construction-button square-button'>
                <span class='construction-icon'>🏗️</span>
                <span class='construction-name'>Command Center</span>
                <div class='construction-progress-display'>
                  <span class='construction-progress-completion'>45%</span>
                  <span class='construction-progress-turns'>~3 turns</span>
                </div>
              </button>
            </div>
          </section>

          <PopulationPanel planet={currentPlanet()} />
        </section>
      )}
    </Show>
  );
}
