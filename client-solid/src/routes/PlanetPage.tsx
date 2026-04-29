import { useParams } from '@solidjs/router';
import { createMemo, createSignal, Show } from 'solid-js';
import { PlanetVisual } from '../components/PlanetVisual';
import type { MovePopulationCommand } from '../lib/types/commands';
import { findPlanet, labelForEnvironment, labelForGravity, labelForMinerals, labelForSize } from '../lib/utils';
import { useAppContext } from '../state/app-context';

type PopCategory = 'farmers' | 'workers' | 'scientists';

const POP_LABELS: Record<PopCategory, string> = {
  farmers: 'Farmer',
  workers: 'Worker',
  scientists: 'Scientist',
};
const POP_COLORS: Record<PopCategory, string> = {
  farmers: '#b59a58',
  workers: '#5a8cb8',
  scientists: '#9eb8a8',
};

export function PlanetPage() {
  const params = useParams();
  const { state, addMovePopulationCommand } = useAppContext();
  const [_raggedCount, setDraggedCount] = createSignal(0);
  const [_dragSource, setDragSource] = createSignal<PopCategory | null>(null);

  const planet = createMemo(() => findPlanet(state.game, Number(params.planetId)));

  const pendingMoves = createMemo(() => {
    const p = planet();
    if (!p) return [];
    return state.pendingCommands.movePopulation.filter((cmd) => cmd.planetId === p.id);
  });

  const getOutputIcons = (count: number, tensIcon: string, onesIcon: string): string => {
    const tens = Math.floor(count / 10);
    const ones = count % 10;
    const icons: string[] = [];
    for (let i = 0; i < tens; i++) icons.push(tensIcon);
    for (let i = 0; i < ones; i++) icons.push(onesIcon);
    return icons.join(' ');
  };

  const getFoodOutput = (): string => {
    const p = planet();
    if (!p) return '';
    let count = p.food.farmers * p.food.produced_per_farmer;
    for (const move of pendingMoves()) {
      if (move.to === 'farmers') count += move.count * p.food.produced_per_farmer;
      if (move.from === 'farmers') count -= move.count * p.food.produced_per_farmer;
    }
    return getOutputIcons(Math.max(0, count), '🌾', '🌱');
  };

  const getIndustryOutput = (): string => {
    const p = planet();
    if (!p) return '';
    let count = p.industry.workers * p.industry.produced_per_worker;
    for (const move of pendingMoves()) {
      if (move.to === 'workers') count += move.count * p.industry.produced_per_worker;
      if (move.from === 'workers') count -= move.count * p.industry.produced_per_worker;
    }
    return getOutputIcons(Math.max(0, count), '⚡', '🔧');
  };

  const getScienceOutput = (): string => {
    const p = planet();
    if (!p) return '';
    let count = p.science.scientists * p.science.produced_per_scientist;
    for (const move of pendingMoves()) {
      if (move.to === 'scientists') count += move.count * p.science.produced_per_scientist;
      if (move.from === 'scientists') count -= move.count * p.science.produced_per_scientist;
    }
    return getOutputIcons(Math.max(0, count), '🔬', '🧫');
  };

  const getPopCount = (cat: PopCategory) => {
    const p = planet();
    if (!p) return 0;
    let base = 0;
    switch (cat) {
      case 'farmers':
        base = p.food.farmers;
        break;
      case 'workers':
        base = p.industry.workers;
        break;
      case 'scientists':
        base = p.science.scientists;
        break;
    }
    for (const move of pendingMoves()) {
      if (move.to === cat) {
        base += move.count;
      }
      if (move.from === cat) {
        base -= move.count;
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
                    {cat === 'farmers' && getFoodOutput()}
                    {cat === 'workers' && getIndustryOutput()}
                    {cat === 'scientists' && getScienceOutput()}
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
                      const cmd: MovePopulationCommand = {
                        planetId: p.id,
                        from: source,
                        to: cat,
                        count,
                      };
                      addMovePopulationCommand(cmd);
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
