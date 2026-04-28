import { useParams } from '@solidjs/router';
import { createMemo, createSignal, onMount, Show } from 'solid-js';

import { PlanetVisual } from '../components/PlanetVisual';
import { findPlanet, labelForEnvironment, labelForGravity, labelForMinerals, labelForSize } from '../lib/utils';
import { useAppContext } from '../state/app-context';

type PopCategory = 'farmers' | 'workers' | 'scientists' | 'marines';

const POP_LABELS: Record<PopCategory, string> = {
  farmers: 'Farmer',
  workers: 'Worker',
  scientists: 'Scientist',
  marines: 'Marine',
};
const POP_COLORS: Record<PopCategory, string> = {
  farmers: '#b59a58',
  workers: '#5a8cb8',
  scientists: '#9eb8a8',
  marines: '#a88c5a',
};

function getOutputIcons(count: number, type: 'food' | 'industry' | 'science'): string {
  const tens = Math.floor(count / 10);
  const ones = count % 10;
  const icons: string[] = [];
  if (type === 'food') {
    for (let i = 0; i < tens; i++) icons.push('🌾');
    for (let i = 0; i < ones; i++) icons.push('🌱');
  } else if (type === 'industry') {
    for (let i = 0; i < tens; i++) icons.push('⚡');
    for (let i = 0; i < ones; i++) icons.push('🔧');
  } else if (type === 'science') {
    for (let i = 0; i < tens; i++) icons.push('🧫');
    for (let i = 0; i < ones; i++) icons.push('🔬');
  }
  return icons.join(' ');
}

export function PlanetPage() {
  const params = useParams();
  const { state, refreshGame, addPopulationChange } = useAppContext();
  const [_raggedCount, setDraggedCount] = createSignal(0);
  const [_dragSource, setDragSource] = createSignal<PopCategory | null>(null);

  onMount(() => {
    if (state.session.selectedGameId) {
      void refreshGame();
    }
  });

  const planet = createMemo(() => findPlanet(state.game, Number(params.planetId)));

  const pendingChanges = createMemo(() => {
    const p = planet();
    if (!p) return [];
    return state.pendingPlanetChanges[p.id] ?? [];
  });

  const getPopCount = (cat: PopCategory) => {
    const p = planet();
    if (!p) return 0;
    let base = 0;
    switch (cat) {
      case 'farmers':
        base = p.farmers;
        break;
      case 'workers':
        base = p.workers;
        break;
      case 'scientists':
        base = p.scientists;
        break;
      case 'marines':
        base = p.marines;
        break;
    }
    for (const change of pendingChanges()) {
      if (change.to === cat) {
        base += change.count;
      }
      if (change.from === cat) {
        base -= change.count;
      }
    }
    return Math.max(0, base);
  };

  const isDragging = () => _raggedCount() > 0;

  const handleDragStart = (e: DragEvent, category: PopCategory, index: number) => {
    const count = getPopCount(category);
    const toDrag = index === count - 1 ? 1 : count - index;
    setDraggedCount(toDrag);
    setDragSource(category);
    const label = POP_LABELS[category];
    const ghost = document.createElement('div');
    ghost.className = 'drag-ghost';
    ghost.textContent = `${toDrag} ${label}`;
    ghost.style.cssText = `
      position: absolute;
      top: -1000px;
      padding: 6px 12px;
      background: var(--accent-soft);
      border: 1px solid var(--accent-strong);
      border-radius: 4px;
      font-size: 13px;
      color: var(--accent);
      font-weight: 500;
      pointer-events: none;
    `;
    document.body.appendChild(ghost);
    e.dataTransfer?.setDragImage(ghost, 30, 15);
    requestAnimationFrame(() => document.body.removeChild(ghost));
    e.dataTransfer!.effectAllowed = 'move';
  };

  const handleDragEnd = () => {
    setDragSource(null);
    setDraggedCount(0);
  };

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

          <section class='panel'>
            <p class='eyebrow'>Population</p>

            <div class={`pop-grid ${isDragging() ? 'is-dragging' : ''}`}>
              <div class='pop-stats-row'>
                <div class='pop-stat'>
                  <span class='pop-stat-label'>Income</span>
                  <div class='pop-stat-icons'>
                    <span class='pop-stat-icon income-icon'>🪙</span>
                    <span class='pop-stat-icon income-icon'>🪙</span>
                    <span class='pop-stat-icon income-icon'>🪙</span>
                    <span class='pop-stat-icon income-icon'>💰</span>
                    <span class='pop-stat-icon income-icon'>💰</span>
                  </div>
                </div>
              </div>
              {(['farmers', 'workers', 'scientists'] as PopCategory[]).map((cat) => (
                <div class={`pop-row ${isDragging() ? 'pop-row--drag-active' : ''}`}>
                  <span class='pop-stat-label'>
                    {cat === 'farmers' && 'Food'}
                    {cat === 'workers' && 'Industry'}
                    {cat === 'scientists' && 'Science'}
                  </span>
                  <div class='pop-output'>
                    {cat === 'farmers' && getOutputIcons(getPopCount('farmers'), 'food')}
                    {cat === 'workers' && getOutputIcons(getPopCount('workers'), 'industry')}
                    {cat === 'scientists' && getOutputIcons(getPopCount('scientists'), 'science')}
                  </div>
                  <div
                    class='pop-units-row'
                    onDragOver={(e) => {
                      e.preventDefault();
                      e.currentTarget.classList.add('drag-over');
                    }}
                    onDragLeave={(e) => e.currentTarget.classList.remove('drag-over')}
                    onDrop={(e) => {
                      e.preventDefault();
                      e.currentTarget.classList.remove('drag-over');
                      const source = _dragSource();
                      const count = _raggedCount();
                      if (!source || count === 0 || source === cat) return;
                      const p = planet();
                      if (!p) return;
                      addPopulationChange(p.id, source, cat, count);
                      setDragSource(null);
                      setDraggedCount(0);
                    }}
                  >
                    {Array.from({ length: getPopCount(cat) }).map((_, idx) => (
                      <button
                        class='pop-unit'
                        style={{ background: POP_COLORS[cat] }}
                        draggable={true}
                        onDragStart={(e) => handleDragStart(e, cat, idx)}
                        onDragEnd={handleDragEnd}
                      />
                    ))}
                  </div>
                </div>
              ))}
            </div>

            <div class='list-block'>
              <p class='eyebrow'>Installed Buildings</p>
              <ul class='compact-list'>
                {currentPlanet().buildings.map((building) => (
                  <li>
                    <strong>{building.name}</strong>
                    <span>{building.maintenance} maintenance</span>
                  </li>
                ))}
                <Show when={currentPlanet().buildings.length === 0}>
                  <li>No structures yet.</li>
                </Show>
              </ul>
            </div>
          </section>
        </section>
      )}
    </Show>
  );
}
