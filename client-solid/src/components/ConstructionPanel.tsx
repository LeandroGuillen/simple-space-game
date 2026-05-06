import { createMemo, createSignal, For, onCleanup, onMount, Show } from 'solid-js';

import type { BuildingViewModel, PlanetViewModel } from '../lib/types/domain';
import { getPlanetIndustryForWorkers } from '../lib/utils';
import { useAppContext } from '../state/app-context';

type ConstructionPanelProps = {
  planet: PlanetViewModel;
};

const TRADE_GOODS: BuildingViewModel = {
  id: 0,
  name: 'Trade Goods',
  description: 'Converts surplus colony output into treasury income.',
  icon: '💱',
  cost: 0,
  maintenance: 0,
  effects: [],
};

export function ConstructionPanel(props: ConstructionPanelProps) {
  const { state, changeBuilding } = useAppContext();
  const [isChoosing, setIsChoosing] = createSignal(false);
  const [selectedProjectId, setSelectedProjectId] = createSignal<number | null>(null);
  let panelRef: HTMLElement | undefined;

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
  const projectedFarmers = createMemo(() => {
    let farmers = props.planet.food.farmers;
    for (const move of pendingMoves()) {
      if (move.to === 'farmers') {
        farmers += move.count;
      }
      if (move.from === 'farmers') {
        farmers -= move.count;
      }
    }
    return Math.max(0, farmers);
  });
  const industryPerTurn = createMemo(() => getPlanetIndustryForWorkers(props.planet, projectedWorkers()));
  const foodPerTurn = createMemo(
    () => projectedFarmers() * props.planet.food.produced_per_farmer + props.planet.food.produced_per_building,
  );
  const foodSurplus = createMemo(() => Math.max(0, foodPerTurn() - Math.floor(props.planet.population / 1000)));
  const tradeGoodsIncome = createMemo(() => Math.floor(industryPerTurn() / 2) + Math.floor(foodSurplus() / 2));
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

  function cancelSelection() {
    setIsChoosing(false);
    setSelectedProjectId(null);
  }

  function openChooser() {
    setSelectedProjectId(activeProject().id);
    setIsChoosing(true);
  }

  function toggleChooser() {
    if (isChoosing()) {
      cancelSelection();
      return;
    }
    openChooser();
  }

  function selectProject(project: BuildingViewModel) {
    if (project.id !== 0 && builtBuildingIds().has(project.id)) {
      return;
    }
    setSelectedProjectId(project.id);
  }

  async function buildSelectedProject() {
    await changeBuilding(props.planet.id, selectedProject().id);
    setIsChoosing(false);
  }

  async function buildProject(project: BuildingViewModel) {
    if (project.id !== 0 && builtBuildingIds().has(project.id)) {
      return;
    }
    setSelectedProjectId(project.id);
    await changeBuilding(props.planet.id, project.id);
    setIsChoosing(false);
  }

  onMount(() => {
    const handlePointerDown = (event: PointerEvent) => {
      if (!isChoosing() || panelRef?.contains(event.target as Node)) {
        return;
      }
      cancelSelection();
    };
    document.addEventListener('pointerdown', handlePointerDown);
    onCleanup(() => document.removeEventListener('pointerdown', handlePointerDown));
  });

  return (
    <section class='construction-panel' ref={panelRef}>
      <button class='construction-widget' type='button' onClick={toggleChooser}>
        <span class='construction-widget-icon'>{activeProject().icon}</span>
        <span class='construction-widget-body'>
          <span class='eyebrow'>Construction</span>
          <strong>{activeProject().name}</strong>
          <Show
            when={props.planet.isBuilding && props.planet.currentProject}
            fallback={<span class='construction-muted'>Producing treasury income</span>}
          >
            <span class='construction-muted'>
              {props.planet.buildBucket} PP left · {industryPerTurn()} PP/turn · {turnsRemaining() ? `~${turnsRemaining()} turns` : 'no production'}
            </span>
          </Show>
        </span>
        <span class='construction-widget-action'>{isChoosing() ? 'Cancel' : 'Change'}</span>
      </button>

      <div class='construction-progress' aria-label={`Construction progress ${progressPercent()}%`}>
        <span style={{ width: `${progressPercent()}%` }} />
      </div>
      <div class='construction-progress-meta'>
        <Show
          when={props.planet.isBuilding && props.planet.currentProject}
          fallback={<span>Trade Goods producing {tradeGoodsIncome()} BC</span>}
        >
          <span>{completedProduction()} / {props.planet.currentProject?.cost ?? 0} PP invested</span>
          <span>{progressPercent()}%</span>
        </Show>
      </div>

      <Show when={isChoosing()}>
        <div class='construction-chooser'>
          <div class='construction-project-list'>
            <For each={availableProjects()}>
              {(project) => (
                <button
                  type='button'
                  class={`construction-project-option ${selectedProject().id === project.id ? 'is-selected' : ''}`}
                  disabled={state.isBusy}
                  onClick={() => selectProject(project)}
                  onDblClick={() => void buildProject(project)}
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
                <dt>Maintenance</dt>
                <dd>{selectedProject().maintenance} BC/turn</dd>
              </div>
            </dl>
            <div class='construction-chooser-actions'>
              <button class='primary-button' type='button' disabled={state.isBusy} onClick={() => void buildSelectedProject()}>
                BUILD
              </button>
            </div>
          </article>
        </div>
      </Show>

    </section>
  );
}
