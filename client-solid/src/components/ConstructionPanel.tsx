import { createEffect, createMemo, createSignal, For, onCleanup, onMount, Show } from 'solid-js';
import { Portal } from 'solid-js/web';

import type { BuildingViewModel, PlanetViewModel, ShipDesignViewModel } from '../lib/types/domain';
import { getPlanetIndustryForWorkers, findStar, getSuitablePlanetsForColonization } from '../lib/utils';
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
  grants: {
    unlocks: {
      buildings: [],
      hullSizes: [],
      drives: [],
      armors: [],
      computers: [],
      fuelCells: [],
      shields: [],
      beams: [],
      shipSystems: [],
    },
    modifiers: [],
  },
  modifiers: [],
};

export function ConstructionPanel(props: ConstructionPanelProps) {
  const { state, currentPlayer, changeBuilding, changeBuildQueue } = useAppContext();
  const [isChoosing, setIsChoosing] = createSignal(false);
  const [selectedProjectId, setSelectedProjectId] = createSignal<number | null>(null);
  const [isAutobuildEnabled, setIsAutobuildEnabled] = createSignal(false);
  const [queuedProjects, setQueuedProjects] = createSignal<BuildingViewModel[]>([]);
  const [draggedQueueIndex, setDraggedQueueIndex] = createSignal<number | null>(null);
  const [dragOverQueueIndex, setDragOverQueueIndex] = createSignal<number | null>(null);
  const [isQueueDragging, setIsQueueDragging] = createSignal(false);
  const [hoveredProjectId, setHoveredProjectId] = createSignal<number | null>(null);
  let modalRef: HTMLElement | undefined;
  let projectClickTimer: ReturnType<typeof setTimeout> | undefined;
  let dragClickTimer: ReturnType<typeof setTimeout> | undefined;

  const builtBuildingIds = createMemo(() => new Set(
    props.planet.buildings
      .filter((building) => building.name !== 'Colony Base')
      .map((building) => building.id),
  ));
  const availableBuildingNames = createMemo(() => new Set(currentPlayer()?.available.buildings ?? []));
  const buildableProjects = createMemo(() =>
    state.game?.resources.buildingCollection.buildings.filter(
      (project) => availableBuildingNames().has(project.name) && !builtBuildingIds().has(project.id) && project.name !== 'Colony Base',
    ) ?? [],
  );
  const colonyBaseProject = createMemo(() =>
    state.game?.resources.buildingCollection.buildings.find(
      (project) => project.name === 'Colony Base' && availableBuildingNames().has(project.name),
    ) ?? null,
  );
  const availableProjects = createMemo(() => {
    const projects = [TRADE_GOODS, ...buildableProjects()];
    if (colonyBaseProject()) {
      projects.push(colonyBaseProject()!);
    }
    return projects;
  });
  const buildingListProjects = createMemo(() => [TRADE_GOODS, ...buildableProjects()]);
  const projectByName = createMemo(() => new Map(availableProjects().map((project) => [project.name.toLowerCase(), project])));
  const pendingMoves = createMemo(() =>
    state.pendingCommands.movePopulation.filter((cmd) => cmd.planetId === props.planet.id),
  );
  const pendingBuild = createMemo(() =>
    state.pendingCommands.build.find((cmd) => cmd.planetId === props.planet.id),
  );
  const pendingQueuedProjects = createMemo(() => {
    const command = pendingBuild();
    const queueNames = command?.build_queue ?? (command?.building ? [command.building] : []);
    return queueNames
      .map((name) => projectByName().get(name.toLowerCase()) ?? null)
      .filter((project): project is BuildingViewModel => Boolean(project));
  });
  const activeProject = createMemo(() => {
    const pendingProject = availableProjects().find((project) => project.name === pendingBuild()?.building);
    return pendingProject ?? props.planet.currentProject ?? TRADE_GOODS;
  });
  const selectedProject = createMemo(
    () => availableProjects().find((project) => project.id === (selectedProjectId() ?? activeProject().id)) ?? TRADE_GOODS,
  );
  const previewProject = createMemo(
    () => availableProjects().find((project) => project.id === hoveredProjectId()) ?? selectedProject(),
  );
  const queueSlots = createMemo(() => {
    const queued = queuedProjects().slice(0, 10);
    return Array.from({ length: 10 }, (_, index) => queued[index] ?? null);
  });
  const researchedTechs = createMemo(() => new Set(currentPlayer()?.researchedTechs ?? []));
  const availableHullSizes = createMemo(() => new Set(currentPlayer()?.available.hullSizes ?? []));
  const currentStar = createMemo(() => findStar(state.game, props.planet.parentStar));
  const suitablePlanetsForColonization = createMemo(() => {
    const star = currentStar();
    if (!star) return [];
    return getSuitablePlanetsForColonization(star);
  });
  const hasSuitablePlanetsForColonization = createMemo(() => suitablePlanetsForColonization().length > 0);
  const shipDesigns = createMemo(() => {
    const designs = [
      ...(currentPlayer()?.shipDesigns ?? []),
      ...(state.game?.resources.shipDesignCollection.shipDesigns ?? []),
    ];
    const seenHullSizes = new Set<string>();
    const uniqueByHull: ShipDesignViewModel[] = [];
    for (const design of designs) {
      const hullName = design.hullSize.name;
      if (availableHullSizes().size > 0 && !availableHullSizes().has(hullName)) {
        continue;
      }
      if (!seenHullSizes.has(hullName)) {
        seenHullSizes.add(hullName);
        uniqueByHull.push(design);
      }
      if (uniqueByHull.length >= 6) {
        break;
      }
    }
    return Array.from({ length: 6 }, (_, index) => uniqueByHull[index] ?? null);
  });
  const sectionOneCommands = createMemo(() => {
    const commands = [
      specialCommand('Colony Base', 'Colony Base'),
      specialCommand('Freighter Fleet', 'Freighters'),
      specialCommand('Colony Ship', 'Colony Ship'),
      specialCommand('Outpost', 'Outpost Ship'),
      specialCommand('Transport', 'Transport'),
    ];
    return commands
      .map((command) => {
        if (command.unlockName === 'Colony Base') {
          const hasTech = command.isAvailable;
          const hasPlanets = hasSuitablePlanetsForColonization();
          const isAvailable = hasTech && hasPlanets;
          return { ...command, isAvailable, hasSuitablePlanets: hasPlanets, hasTech };
        }
        return command;
      })
      .filter((command) => command.isAvailable || command.unlockName === 'Colony Base');
  });
  const androidCommands = createMemo(() =>
    [
      specialCommand('Android Farmers', 'Android Farmers'),
      specialCommand('Android Workers', 'Android Workers'),
      specialCommand('Android Scientists', 'Android Scientists'),
    ].filter((command) => command.isAvailable).slice(0, 3),
  );
  const spyCommands = createMemo(() =>
    [specialCommand('Spy', 'Spy')].filter((command) => command.isAvailable),
  );
  const personnelCommands = createMemo(() => [...androidCommands(), ...spyCommands()]);
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
    setDraggedQueueIndex(null);
    setDragOverQueueIndex(null);
    setHoveredProjectId(null);
  }

  function openChooser() {
    setSelectedProjectId(activeProject().id);
    setQueuedProjects(pendingQueuedProjects().length > 0 ? pendingQueuedProjects() : props.planet.buildQueue);
    setIsChoosing(true);
  }

  function toggleChooser() {
    if (isChoosing()) {
      cancelSelection();
      return;
    }
    openChooser();
  }

  function updateQueuedProjects(projects: BuildingViewModel[]) {
    const nextProjects = projects.slice(0, 10);
    setQueuedProjects(nextProjects);
    changeBuildQueue(props.planet.id, nextProjects.map((project) => project.name));
  }

  function addProjectToQueue(project: BuildingViewModel): BuildingViewModel[] {
    // Colony Base can always be queued (even if already built - multiple are allowed)
    const isColonyBase = project.name === 'Colony Base';
    if (!isColonyBase && project.id !== 0 && builtBuildingIds().has(project.id)) {
      return queuedProjects();
    }
    setSelectedProjectId(project.id);
    const projects = queuedProjects();
    if (projects.length >= 10) {
      return projects;
    }
    // Allow multiple Colony Bases but prevent duplicate real buildings
    if (!isColonyBase && project.id !== 0 && projects.some((queued) => queued.id === project.id)) {
      return projects;
    }
    const nextProjects = [...projects, project];
    updateQueuedProjects(nextProjects);
    return nextProjects;
  }

  function removeProjectFromQueue(index: number) {
    if (isQueueDragging()) {
      return;
    }
    updateQueuedProjects(queuedProjects().filter((_, projectIndex) => projectIndex !== index));
  }

  function specialCommand(label: string, unlockName: string) {
    const project = projectByName().get(unlockName.toLowerCase()) ?? null;
    return {
      label,
      unlockName,
      project,
      isAvailable:
        Boolean(project) ||
        researchedTechs().has(unlockName) ||
        availableBuildingNames().has(unlockName),
    };
  }

  function confirmProject(project: BuildingViewModel, projects = queuedProjects()) {
    const nextProjects = projects.length > 0 ? projects : [project];
    changeBuildQueue(props.planet.id, nextProjects.map((queuedProject) => queuedProject.name));
    setIsChoosing(false);
  }

  function confirmQueue() {
    if (queuedProjects().length === 0) {
      const nextProjects = addProjectToQueue(selectedProject());
      confirmProject(selectedProject(), nextProjects);
      return;
    }
    confirmProject(queuedProjects()[0] ?? selectedProject());
  }

  function handleProjectClick(project: BuildingViewModel, event: MouseEvent) {
    if (event.detail !== 1) {
      return;
    }
    if (projectClickTimer) {
      clearTimeout(projectClickTimer);
    }
    projectClickTimer = setTimeout(() => {
      addProjectToQueue(project);
      projectClickTimer = undefined;
    }, 180);
  }

  function handleProjectDoubleClick(project: BuildingViewModel) {
    if (projectClickTimer) {
      clearTimeout(projectClickTimer);
      projectClickTimer = undefined;
    }
    const nextProjects = addProjectToQueue(project);
    confirmProject(project, nextProjects);
  }

  function reorderQueue(toIndex: number) {
    const fromIndex = draggedQueueIndex();
    if (fromIndex === null || fromIndex === toIndex) {
      setDraggedQueueIndex(null);
      return;
    }
    const projects = queuedProjects();
    let nextQueuedProjects = projects;
    {
      if (!projects[fromIndex]) {
        setDraggedQueueIndex(null);
        setDragOverQueueIndex(null);
        return;
      }
      const nextProjects = [...projects];
      const [movedProject] = nextProjects.splice(fromIndex, 1);
      if (!movedProject) {
        setDraggedQueueIndex(null);
        setDragOverQueueIndex(null);
        return;
      }
      nextProjects.splice(Math.min(toIndex, nextProjects.length), 0, movedProject);
      nextQueuedProjects = nextProjects;
    }
    updateQueuedProjects(nextQueuedProjects);
    setDraggedQueueIndex(null);
    setDragOverQueueIndex(null);
  }

  function moveQueueItemDuringDrag(toIndex: number) {
    const fromIndex = draggedQueueIndex();
    if (fromIndex === null || fromIndex === toIndex) {
      setDragOverQueueIndex(toIndex);
      return;
    }
    const projects = queuedProjects();
    let nextQueuedProjects = projects;
    {
      if (!projects[fromIndex]) {
        return;
      }
      const nextProjects = [...projects];
      const [movedProject] = nextProjects.splice(fromIndex, 1);
      if (!movedProject) {
        return;
      }
      nextProjects.splice(toIndex, 0, movedProject);
      nextQueuedProjects = nextProjects;
    }
    updateQueuedProjects(nextQueuedProjects);
    setDraggedQueueIndex(toIndex);
    setDragOverQueueIndex(toIndex);
  }

  function startQueueDrag(index: number) {
    if (!queuedProjects()[index]) {
      return;
    }
    if (dragClickTimer) {
      clearTimeout(dragClickTimer);
    }
    setIsQueueDragging(true);
    setDraggedQueueIndex(index);
    setDragOverQueueIndex(index);
  }

  function endQueueDrag() {
    setDraggedQueueIndex(null);
    setDragOverQueueIndex(null);
    if (dragClickTimer) {
      clearTimeout(dragClickTimer);
    }
    dragClickTimer = setTimeout(() => {
      setIsQueueDragging(false);
      dragClickTimer = undefined;
    }, 120);
  }

  createEffect(() => {
    if (
      !isAutobuildEnabled() ||
      state.isBusy ||
      props.planet.isBuilding ||
      pendingBuild() ||
      buildableProjects().length === 0
    ) {
      return;
    }
    const nextProjects = buildableProjects();
    const randomProject = nextProjects[Math.floor(Math.random() * nextProjects.length)];
    if (randomProject) {
      changeBuilding(props.planet.id, randomProject.id);
    }
  });

  onCleanup(() => {
    if (projectClickTimer) {
      clearTimeout(projectClickTimer);
    }
    if (dragClickTimer) {
      clearTimeout(dragClickTimer);
    }
  });

  onMount(() => {
    const handlePointerDown = (event: PointerEvent) => {
      if (!isChoosing() || modalRef?.contains(event.target as Node)) {
        return;
      }
      cancelSelection();
    };
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        cancelSelection();
      }
    };
    document.addEventListener('pointerdown', handlePointerDown);
    document.addEventListener('keydown', handleKeyDown);
    onCleanup(() => {
      document.removeEventListener('pointerdown', handlePointerDown);
      document.removeEventListener('keydown', handleKeyDown);
    });
  });

  return (
    <section class='construction-panel'>
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
        <Portal>
          <div class='construction-modal-backdrop'>
            <section
              class='construction-modal'
              role='dialog'
              aria-modal='true'
              aria-labelledby='construction-modal-title'
              ref={modalRef}
            >
              <header class='construction-modal-header'>
                <div>
                  <p class='eyebrow'>Construction</p>
                  <h2 id='construction-modal-title'>{props.planet.name}</h2>
                </div>
                <button class='secondary-button construction-close-button' type='button' onClick={cancelSelection}>
                  Close
                </button>
              </header>

              <div class='construction-modal-grid'>
                <aside class='construction-modal-sidebar'>
                  <h3>Buildings</h3>
                  <div class='construction-project-list'>
                    <For each={buildingListProjects()}>
                      {(project) => (
                        <button
                          type='button'
                          class={`construction-project-option ${selectedProject().id === project.id ? 'is-selected' : ''}`}
                          disabled={state.isBusy}
                          onMouseEnter={() => setHoveredProjectId(project.id)}
                          onMouseLeave={() => setHoveredProjectId(null)}
                          onClick={(event) => handleProjectClick(project, event)}
                          onDblClick={() => handleProjectDoubleClick(project)}
                        >
                          <span class='construction-project-name'>{project.name}</span>
                          <span class='construction-project-cost'>{project.cost > 0 ? `${project.cost} PP` : 'Now'}</span>
                        </button>
                      )}
                    </For>
                  </div>
                </aside>

                <main class='construction-queue-panel'>
                  <article class='construction-project-details'>
                    <div class='construction-project-title'>
                      <div>
                        <h3>{previewProject().name}</h3>
                        <p>{previewProject().description || 'No project description available yet.'}</p>
                      </div>
                    </div>
                    <dl class='construction-facts'>
                      <div>
                        <dt>Cost</dt>
                        <dd>{previewProject().cost} PP</dd>
                      </div>
                      <div>
                        <dt>Maintenance</dt>
                        <dd>{previewProject().maintenance} BC</dd>
                      </div>
                      <div>
                        <dt>Output</dt>
                        <dd>{industryPerTurn()} PP/turn</dd>
                      </div>
                    </dl>
                    <div class='construction-chooser-actions'>
                      <label class='construction-autobuild'>
                        <span>Autobuild</span>
                        <span class='construction-autobuild-switch'>
                          <input
                            type='checkbox'
                            checked={isAutobuildEnabled()}
                            onChange={(event) => setIsAutobuildEnabled(event.currentTarget.checked)}
                          />
                          <span aria-hidden='true' />
                        </span>
                      </label>
                      <button class='primary-button' type='button' disabled={state.isBusy} onClick={confirmQueue}>
                        Build
                      </button>
                    </div>
                  </article>

                  <section class='construction-queue'>
                    <div class='construction-column-header'>
                      <h3>Queue</h3>
                      <span>{queueSlots().filter(Boolean).length}/10</span>
                    </div>
                    <div class='construction-queue-slots'>
                      <For each={queueSlots()}>
                        {(project, index) => (
                          <div
                            class={`construction-queue-slot ${project ? 'is-filled' : ''} ${draggedQueueIndex() === index() ? 'is-dragging' : ''} ${dragOverQueueIndex() === index() ? 'is-drop-target' : ''}`}
                            draggable={Boolean(project)}
                            onClick={() => project && removeProjectFromQueue(index())}
                            onMouseEnter={() => project && setHoveredProjectId(project.id)}
                            onMouseLeave={() => project && setHoveredProjectId(null)}
                            onDragStart={() => startQueueDrag(index())}
                            onDragEnd={endQueueDrag}
                            onDragOver={(event) => {
                              event.preventDefault();
                              moveQueueItemDuringDrag(index());
                            }}
                            onDrop={() => reorderQueue(index())}
                          >
                            <span class='construction-queue-index'>{index() + 1}</span>
                            <span class='construction-queue-name'>{project?.name ?? 'Empty'}</span>
                            <span class='construction-queue-cost'>{project && project.cost > 0 ? `${project.cost} PP` : ''}</span>
                          </div>
                        )}
                      </For>
                    </div>
                  </section>
                </main>

                <aside class='construction-modal-sidebar construction-specials'>
                  <Show when={sectionOneCommands().length > 0}>
                    <section class='construction-special-section'>
                      <h3>Fleet Orders</h3>
                      <div class='construction-special-list'>
                        <For each={sectionOneCommands()}>
                          {(command) => {
                            const isColonyBase = command.unlockName === 'Colony Base';
                            const hasSuitablePlanets = (command as any).hasSuitablePlanets;
                            const hasTech = (command as any).hasTech;
                            const colonyBaseDisabled = isColonyBase && !hasSuitablePlanets;
                            const isDisabled = isColonyBase 
                              ? (state.isBusy || colonyBaseDisabled || !hasTech)
                              : (state.isBusy || !command.project);
                            return (
                              <button
                                type='button'
                                class='construction-special-option'
                                disabled={isDisabled}
                                onMouseEnter={() => !isColonyBase && command.project && setHoveredProjectId(command.project.id)}
                                onMouseLeave={() => !isColonyBase && command.project && setHoveredProjectId(null)}
                                onClick={(event) => {
                                  if (isColonyBase) {
                                    if (!colonyBaseDisabled && hasTech) {
                                      const project = colonyBaseProject();
                                      if (project) {
                                        addProjectToQueue(project);
                                      }
                                    }
                                  } else if (command.project) {
                                    handleProjectClick(command.project, event);
                                  }
                                }}
                                onDblClick={() => {
                                  if (isColonyBase) {
                                    if (!colonyBaseDisabled && hasTech) {
                                      const project = colonyBaseProject();
                                      if (project) {
                                        addProjectToQueue(project);
                                        confirmProject(project);
                                      }
                                    }
                                  } else if (command.project) {
                                    handleProjectDoubleClick(command.project);
                                  }
                                }}
                              >
                                <span>{isColonyBase && !hasSuitablePlanets ? 'No suitable planets' : command.label}</span>
                              </button>
                            );
                          }}
                        </For>
                      </div>
                    </section>
                  </Show>

                  <Show when={shipDesigns().some(Boolean)}>
                    <section class='construction-special-section'>
                      <h3>Ship Designs</h3>
                      <div class='construction-design-slots'>
                        <For each={shipDesigns()}>
                          {(design) => (
                            <button type='button' class='construction-design-slot' disabled>
                              <Show when={design} fallback={<span>Empty</span>}>
                                {(item) => (
                                  <>
                                    <span>{item().name}</span>
                                    <strong>{item().hullSize.name}</strong>
                                  </>
                                )}
                              </Show>
                            </button>
                          )}
                        </For>
                      </div>
                    </section>
                  </Show>

                  <Show when={personnelCommands().length > 0}>
                    <section class='construction-special-section'>
                      <h3>Personnel</h3>
                      <div class='construction-special-list'>
                        <For each={personnelCommands()}>
                          {(command) => (
                            <button
                              type='button'
                              class='construction-special-option'
                              disabled={state.isBusy || !command.project}
                              onMouseEnter={() => command.project && setHoveredProjectId(command.project.id)}
                              onMouseLeave={() => command.project && setHoveredProjectId(null)}
                              onClick={(event) => command.project && handleProjectClick(command.project, event)}
                              onDblClick={() => command.project && handleProjectDoubleClick(command.project)}
                            >
                              <span>{command.label}</span>
                            </button>
                          )}
                        </For>
                      </div>
                    </section>
                  </Show>
                </aside>
              </div>
            </section>
          </div>
        </Portal>
      </Show>

    </section>
  );
}
