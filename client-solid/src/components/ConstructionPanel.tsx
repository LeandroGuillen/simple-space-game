import { createMemo, createSignal, For, Show } from 'solid-js';

import type { BuildingViewModel, PlanetViewModel } from '../lib/types/domain';
import { useAppContext } from '../state/app-context';

type ConstructionPanelProps = {
  planet: PlanetViewModel;
};

const TRADE_GOODS: BuildingViewModel = {
  id: 0,
  name: 'Trade Goods',
  description: 'Placeholder economic project. For now, idle production is stockpiled while the colony waits for a real project.',
  icon: '💱',
  cost: 0,
  maintenance: 0,
  effects: [],
};

export function ConstructionPanel(props: ConstructionPanelProps) {
  const { state, changeBuilding } = useAppContext();
  const [isChoosing, setIsChoosing] = createSignal(false);
  const [selectedProjectId, setSelectedProjectId] = createSignal<number | null>(null);

  const builtBuildingIds = createMemo(() => new Set(props.planet.buildings.map((building) => building.id)));
  const availableProjects = createMemo(() => [
    TRADE_GOODS,
    ...(state.game?.resources.buildingCollection.buildings.filter((project) => !builtBuildingIds().has(project.id)) ?? []),
  ]);
  const pendingMoves = createMemo(() =>
    state.pendingCommands.movePopulation.filter((cmd) => cmd.planetId === props.planet.id),
  );
  const activeProject = createMemo(() => props.planet.currentProject ?? TRADE_GOODS);
  const selectedProject = createMemo(
    () => availableProjects().find((project) => project.id === (selectedProjectId() ?? activeProject().id)) ?? TRADE_GOODS,
  );
  const projectedWorkers = createMemo(() => {
    let workers = props.planet.industry.workers;
    for (const move of pendingMoves()) {
      if (move.to === 'workers') {
        workers += move.count;
      }
      if (move.from === 'workers') {
        workers -= move.count;
      }
    }
    return Math.max(0, workers);
  });
  const industryPerTurn = createMemo(
    () => projectedWorkers() * props.planet.industry.produced_per_worker + props.planet.industry.produced_per_building,
  );
  const completedProduction = createMemo(() => {
    if (!props.planet.isBuilding || !props.planet.currentProject) {
      return 0;
    }
    return Math.max(0, props.planet.currentProject.cost - props.planet.buildBucket);
  });
  const progressPercent = createMemo(() => {
    if (!props.planet.isBuilding || !props.planet.currentProject?.cost) {
      return 0;
    }
    return Math.min(100, Math.round((completedProduction() / props.planet.currentProject.cost) * 100));
  });
  const turnsRemaining = createMemo(() => {
    if (!props.planet.isBuilding || industryPerTurn() <= 0) {
      return null;
    }
    return Math.max(1, Math.ceil(props.planet.buildBucket / industryPerTurn()));
  });

  async function selectProject(project: BuildingViewModel) {
    if (project.id !== 0 && builtBuildingIds().has(project.id)) {
      return;
    }
    setSelectedProjectId(project.id);
    await changeBuilding(props.planet.id, project.id);
  }

  return (
    <section class='panel construction-panel'>
      <button class='construction-widget' type='button' onClick={() => setIsChoosing(!isChoosing())}>
        <span class='construction-widget-icon'>{activeProject().icon}</span>
        <span class='construction-widget-body'>
          <span class='eyebrow'>Construction</span>
          <strong>{activeProject().name}</strong>
          <Show
            when={props.planet.isBuilding && props.planet.currentProject}
            fallback={<span class='construction-muted'>No active build — Trade Goods placeholder</span>}
          >
            <span class='construction-muted'>
              {props.planet.buildBucket} PP left · {industryPerTurn()} PP/turn · {turnsRemaining() ? `~${turnsRemaining()} turns` : 'no production'}
            </span>
          </Show>
        </span>
        <span class='construction-widget-action'>{isChoosing() ? 'Hide' : 'Change'}</span>
      </button>

      <Show when={props.planet.isBuilding && props.planet.currentProject}>
        <div class='construction-progress' aria-label={`Construction progress ${progressPercent()}%`}>
          <span style={{ width: `${progressPercent()}%` }} />
        </div>
        <div class='construction-progress-meta'>
          <span>{completedProduction()} / {props.planet.currentProject?.cost ?? 0} PP invested</span>
          <span>{progressPercent()}%</span>
        </div>
      </Show>

      <Show when={isChoosing()}>
        <div class='construction-chooser'>
          <div class='construction-project-list'>
            <For each={availableProjects()}>
              {(project) => (
                <button
                  type='button'
                  class={`construction-project-option ${selectedProject().id === project.id ? 'is-selected' : ''}`}
                  disabled={state.isBusy}
                  onClick={() => void selectProject(project)}
                >
                  <span>{project.icon}</span>
                  <span>{project.name}</span>
                </button>
              )}
            </For>
          </div>

          <article class='construction-project-details'>
            <div class='construction-project-title'>
              <span class='construction-detail-icon'>{selectedProject().icon}</span>
              <div>
                <h3>{selectedProject().name}</h3>
                <p>{selectedProject().description || 'No project description available yet.'}</p>
              </div>
            </div>
            <dl class='construction-facts'>
              <div>
                <dt>Cost</dt>
                <dd>{selectedProject().cost === 0 ? '—' : `${selectedProject().cost} PP`}</dd>
              </div>
              <div>
                <dt>Maintenance</dt>
                <dd>{selectedProject().maintenance} BC/turn</dd>
              </div>
              <div>
                <dt>ETA</dt>
                <dd>
                  {selectedProject().cost > 0 && industryPerTurn() > 0
                    ? `~${Math.ceil(selectedProject().cost / industryPerTurn())} turns · ${industryPerTurn()} PP/turn`
                    : `— · ${industryPerTurn()} PP/turn`}
                </dd>
              </div>
            </dl>
          </article>
        </div>
      </Show>

    </section>
  );
}
