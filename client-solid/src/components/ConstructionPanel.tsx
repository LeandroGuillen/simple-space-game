import {
  createEffect,
  createMemo,
  createSignal,
  For,
  onCleanup,
  onMount,
  Show,
} from "solid-js";
import { Portal } from "solid-js/web";

import type {
  BuildingViewModel,
  PlanetViewModel,
  ShipDesignViewModel,
  TechViewModel,
} from "../lib/types/domain";
import { toggleQueuedConstructionProject } from "../lib/construction";
import {
  formatInteger,
  getBuyProductionCost,
  getPlanetIndustryForWorkers,
  getPlanetProjectedPopulationGrowth,
  findStar,
  getArtificialPlanetTargets,
  getSuitablePlanetsForColonization,
} from "../lib/utils";
import { useAppContext } from "../state/app-context";

type ConstructionPanelProps = {
  planet: PlanetViewModel;
  compact?: boolean;
};

const TRADE_GOODS: BuildingViewModel = {
  id: 0,
  name: "Trade Goods",
  description: "Converts surplus colony output into treasury income.",
  icon: "💱",
  cost: 0,
  maintenance: 0,
  unique: false,
  upgrades: [],
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

const HOUSING: BuildingViewModel = {
  id: -1,
  name: "Housing",
  description: "Converts colony production into population growth.",
  icon: "🏘️",
  cost: 0,
  maintenance: 0,
  unique: false,
  upgrades: [],
  grants: TRADE_GOODS.grants,
  modifiers: [],
};

function hasSuperiorBuilding(
  project: BuildingViewModel,
  builtBuildings: BuildingViewModel[],
) {
  return builtBuildings.some((builtBuilding) =>
    builtBuilding.upgrades.includes(project.name),
  );
}

function isSpecialConstructionProject(projectName: string) {
  return (
    projectName === "Colony Base" ||
    projectName === "Colony Ship" ||
    projectName === "Freighters" ||
    projectName === "Artificial Planet Construction" ||
    projectName === "Android Farmers" ||
    projectName === "Android Workers" ||
    projectName === "Android Scientists"
  );
}

export function ConstructionPanel(props: ConstructionPanelProps) {
  const {
    state,
    currentPlayer,
    changeBuildQueue,
    setPlanetAutobuild,
    buyProduction,
    cancelBuyProduction,
  } = useAppContext();
  const [isChoosing, setIsChoosing] = createSignal(false);
  const [selectedProjectId, setSelectedProjectId] = createSignal<number | null>(
    null,
  );
  const [queuedProjects, setQueuedProjects] = createSignal<BuildingViewModel[]>(
    [],
  );
  const isAutobuildEnabled = createMemo(
    () => props.planet.autobuild,
  );
  const [draggedQueueIndex, setDraggedQueueIndex] = createSignal<number | null>(
    null,
  );
  const [dragOverQueueIndex, setDragOverQueueIndex] = createSignal<
    number | null
  >(null);
  const [isQueueDragging, setIsQueueDragging] = createSignal(false);
  const [hoveredProjectId, setHoveredProjectId] = createSignal<number | null>(
    null,
  );
  let modalRef: HTMLElement | undefined;
  let dragClickTimer: ReturnType<typeof setTimeout> | undefined;

  const builtBuildingIds = createMemo(
    () =>
      new Set(
        props.planet.buildings
          .filter((building) => building.name !== "Colony Base")
          .map((building) => building.id),
      ),
  );
  const builtBuildingNames = createMemo(
    () => new Set(props.planet.buildings.map((building) => building.name)),
  );
  const availableBuildingNames = createMemo(
    () => new Set(currentPlayer()?.available.buildings ?? []),
  );
  const uniqueBuildingsLockedElsewhere = createMemo(() => {
    const playerId = currentPlayer()?.id;
    if (!playerId) return new Set<string>();
    const locked = new Set<string>();
    for (const star of state.game?.galaxy.stars ?? []) {
      for (const planet of star.planets) {
        if (planet.owner !== playerId || planet.id === props.planet.id) {
          continue;
        }
        for (const building of planet.buildings) {
          if (building.unique) {
            locked.add(building.name);
          }
        }
        if (planet.currentProject?.unique) {
          locked.add(planet.currentProject.name);
        }
        for (const queued of planet.buildQueue) {
          if (queued.unique) {
            locked.add(queued.name);
          }
        }
      }
    }
    return locked;
  });
  const buildableProjects = createMemo(
    () =>
      state.game?.resources.buildingCollection.buildings.filter(
        (project) =>
          availableBuildingNames().has(project.name) &&
          !builtBuildingIds().has(project.id) &&
          !hasSuperiorBuilding(project, props.planet.buildings) &&
          !isSpecialConstructionProject(project.name) &&
          (!project.unique ||
            !uniqueBuildingsLockedElsewhere().has(project.name)),
      ) ?? [],
  );
  const colonyBaseProject = createMemo(
    () =>
      state.game?.resources.buildingCollection.buildings.find(
        (project) =>
          project.name === "Colony Base" &&
          availableBuildingNames().has(project.name),
      ) ?? null,
  );
  const colonyShipProject = createMemo(
    () =>
      state.game?.resources.buildingCollection.buildings.find(
        (project) =>
          project.name === "Colony Ship" &&
          availableBuildingNames().has(project.name),
      ) ?? null,
  );
  const freightersProject = createMemo(
    () =>
      state.game?.resources.buildingCollection.buildings.find(
        (project) =>
          project.name === "Freighters" &&
          availableBuildingNames().has(project.name),
      ) ?? null,
  );
  const artificialPlanetProject = createMemo(
    () =>
      state.game?.resources.buildingCollection.buildings.find(
        (project) =>
          project.name === "Artificial Planet Construction" &&
          availableBuildingNames().has(project.name),
      ) ?? null,
  );
  const availableHullSizes = createMemo(
    () => new Set(currentPlayer()?.available.hullSizes ?? []),
  );
  const availableDrives = createMemo(
    () => new Set(currentPlayer()?.available.drives ?? []),
  );
  const availableArmors = createMemo(
    () => new Set(currentPlayer()?.available.armors ?? []),
  );
  const availableComputers = createMemo(
    () => new Set(currentPlayer()?.available.computers ?? []),
  );
  const availableFuelCells = createMemo(
    () => new Set(currentPlayer()?.available.fuelCells ?? []),
  );
  const availableShields = createMemo(
    () => new Set(currentPlayer()?.available.shields ?? []),
  );
  const shipDesigns = createMemo(() => {
    const designs = [
      ...(currentPlayer()?.shipDesigns ?? []),
      ...(state.game?.resources.shipDesignCollection.shipDesigns ?? []),
    ];
    const seenHullSizes = new Set<string>();
    const uniqueByHull: ShipDesignViewModel[] = [];
    for (const design of designs) {
      const hullName = design.hullSize.name;
      if (
        availableHullSizes().size > 0 &&
        !availableHullSizes().has(hullName)
      ) {
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
  const buildableShipDesigns = createMemo(() =>
    shipDesigns().filter(
      (design): design is ShipDesignViewModel =>
        Boolean(design) && canBuildShipDesign(design),
    ),
  );
  const shipDesignProjects = createMemo(() =>
    buildableShipDesigns().map((design) => shipDesignProject(design)),
  );
  const availableProjects = createMemo(() => {
    const projects = [TRADE_GOODS, HOUSING, ...buildableProjects(), ...shipDesignProjects()];
    if (colonyBaseProject()) {
      projects.push(colonyBaseProject()!);
    }
    if (colonyShipProject()) {
      projects.push(colonyShipProject()!);
    }
    if (freightersProject()) {
      projects.push(freightersProject()!);
    }
    if (artificialPlanetProject()) {
      projects.push(artificialPlanetProject()!);
    }
    return projects;
  });
  const buildingListProjects = createMemo(() => [
    TRADE_GOODS,
    HOUSING,
    ...buildableProjects(),
  ]);
  const projectByName = createMemo(
    () =>
      new Map(
        availableProjects().map((project) => [
          project.name.toLowerCase(),
          project,
        ]),
      ),
  );
  const pendingMoves = createMemo(() =>
    state.pendingCommands.movePopulation.filter(
      (cmd) => cmd.planetId === props.planet.id,
    ),
  );
  const pendingBuild = createMemo(() =>
    state.pendingCommands.build.find((cmd) => cmd.planetId === props.planet.id),
  );
  const pendingQueuedProjects = createMemo(() => {
    const command = pendingBuild();
    const queueNames = command
      ? [command.building, ...(command.build_queue ?? [])]
      : [];
    return queueNames
      .filter(Boolean)
      .map((name) => projectByName().get(name.toLowerCase()) ?? null)
      .filter((project): project is BuildingViewModel => Boolean(project));
  });
  createEffect(() => {
    if (!isChoosing() || pendingBuild()) {
      return;
    }
    setQueuedProjects(
      [activeProject(), ...props.planet.buildQueue].slice(0, 10),
    );
  });
  const activeProject = createMemo(() => {
    const pendingProject = availableProjects().find(
      (project) => project.name === pendingBuild()?.building,
    );
    return pendingProject ?? props.planet.currentProject ?? TRADE_GOODS;
  });
  const selectedProject = createMemo(
    () =>
      availableProjects().find(
        (project) => project.id === (selectedProjectId() ?? activeProject().id),
      ) ?? TRADE_GOODS,
  );
  const previewProject = createMemo(
    () =>
      availableProjects().find(
        (project) => project.id === hoveredProjectId(),
      ) ?? selectedProject(),
  );
  const queueSlots = createMemo(() => {
    const queued = queuedProjects().slice(0, 10);
    return Array.from({ length: 10 }, (_, index) => queued[index] ?? null);
  });
  const queuedProjectIds = createMemo(
    () => new Set(queuedProjects().map((project) => project.id)),
  );
  const researchedTechs = createMemo(
    () => new Set(currentPlayer()?.researchedTechs ?? []),
  );
  const currentStar = createMemo(() =>
    findStar(state.game, props.planet.parentStar),
  );
  const suitablePlanetsForColonization = createMemo(() => {
    const star = currentStar();
    if (!star) return [];
    return getSuitablePlanetsForColonization(star);
  });
  const hasSuitablePlanetsForColonization = createMemo(
    () => suitablePlanetsForColonization().length > 0,
  );
  const hasArtificialPlanetTargets = createMemo(() => {
    const star = currentStar();
    return Boolean(star && getArtificialPlanetTargets(star).length > 0);
  });
  const sectionOneCommands = createMemo(() => {
    const commands = [
      specialCommand("Colony Base", "Colony Base"),
      specialCommand("Artificial Planet", "Artificial Planet Construction"),
      specialCommand("Freighter Fleet", "Freighters"),
      specialCommand("Colony Ship", "Colony Ship"),
      specialCommand("Outpost", "Outpost Ship"),
      specialCommand("Transport", "Transport"),
    ];
    return commands
      .map((command) => {
        if (command.unlockName === "Colony Base") {
          const hasTech = command.isAvailable;
          const hasPlanets = hasSuitablePlanetsForColonization();
          const isAvailable = hasTech && hasPlanets;
          return {
            ...command,
            isAvailable,
            hasSuitablePlanets: hasPlanets,
            hasTech,
          };
        }
        if (command.unlockName === "Artificial Planet Construction") {
          return {
            ...command,
            isAvailable: command.isAvailable && hasArtificialPlanetTargets(),
          };
        }
        return command;
      })
      .filter(
        (command) =>
          command.isAvailable || command.unlockName === "Colony Base",
      );
  });
  const androidCommands = createMemo(() =>
    [
      specialCommand("Android Farmers", "Android Farmers"),
      specialCommand("Android Workers", "Android Workers"),
      specialCommand("Android Scientists", "Android Scientists"),
    ]
      .filter((command) => command.isAvailable)
      .slice(0, 3),
  );
  const spyCommands = createMemo(() =>
    [specialCommand("Spy", "Spy")].filter((command) => command.isAvailable),
  );
  const personnelCommands = createMemo(() => [
    ...androidCommands(),
    ...spyCommands(),
  ]);
  const projectedWorkers = createMemo(() => {
    let workers = props.planet.industry.workers;
    for (const move of pendingMoves()) {
      if (move.to === "workers") {
        workers += move.count;
      }
      if (move.from === "workers") {
        workers -= move.count;
      }
    }
    return Math.max(0, workers);
  });
  const industryPerTurn = createMemo(() =>
    getPlanetIndustryForWorkers(props.planet, projectedWorkers()),
  );
  const housingGrowthProjection = createMemo(() =>
    getPlanetProjectedPopulationGrowth(props.planet, state.pendingCommands),
  );
  function effectiveProjectCost(project: BuildingViewModel) {
    const upgradeCredit = project.upgrades.reduce((credit, upgradedName) => {
      if (!builtBuildingNames().has(upgradedName)) {
        return credit;
      }
      const upgradedBuilding =
        state.game?.resources.buildingCollection.buildings.find(
          (building) => building.name === upgradedName,
        );
      return Math.max(credit, upgradedBuilding?.cost ?? 0);
    }, 0);
    const baseCost = Math.max(0, project.cost - upgradeCredit);
    if (!project.name.endsWith(" Ship") && !isShipDesignProject(project)) {
      return baseCost;
    }
    return Math.ceil(baseCost * shipCostMultiplier());
  }
  const activeGovernment = createMemo(() => {
    const techs = state.game?.resources.techCollection.techs ?? [];
    let active = techs.find((tech) => tech.type === "government" && tech.name === (currentPlayer()?.raceGovernment || "Dictatorship"));
    if (!active) {
      active = techs.find((tech) => tech.type === "government" && tech.default);
    }
    if (!active) return null;
    const researched = researchedTechs();
    const seen = new Set([active.name]);
    let changed = true;
    while (changed) {
      changed = false;
      for (const tech of techs) {
        if (tech.type !== "government" || tech.upgrades.length === 0 || !researched.has(tech.name)) {
          continue;
        }
        if (tech.upgrades.includes(active.name)) {
          if (seen.has(tech.name)) {
            return active;
          }
          seen.add(tech.name);
          active = tech;
          changed = true;
          break;
        }
      }
    }
    return active;
  });
  const shipCostMultiplier = createMemo(() => governmentModifierValue(activeGovernment(), "ship_cost_multiplier", 1));
  const currentInvestedProduction = createMemo(() => {
    const stackedProduction = props.planet.industryStack;
    if (!props.planet.isBuilding || !props.planet.currentProject) {
      return stackedProduction;
    }
    return Math.max(
      0,
      effectiveProjectCost(props.planet.currentProject) -
        props.planet.buildBucket +
        stackedProduction,
    );
  });
  const displayedBuildBucket = createMemo(() => {
    const project = activeProject();
    if (project.id === TRADE_GOODS.id || project.id === HOUSING.id) {
      return 0;
    }
    if (
      pendingBuild()?.building &&
      project.name !== props.planet.currentProject?.name
    ) {
      return Math.max(
        0,
        effectiveProjectCost(project) - currentInvestedProduction(),
      );
    }
    return Math.max(0, props.planet.buildBucket - props.planet.industryStack);
  });
  const completedProduction = createMemo(() => {
    const project = activeProject();
    if (project.id === TRADE_GOODS.id || project.id === HOUSING.id) {
      return 0;
    }
    return Math.max(0, effectiveProjectCost(project) - displayedBuildBucket());
  });
  const progressPercent = createMemo(() => {
    const project = activeProject();
    const projectCost = effectiveProjectCost(project);
    if (project.id === TRADE_GOODS.id || project.id === HOUSING.id || !projectCost) {
      return 0;
    }
    return Math.min(
      100,
      Math.round((completedProduction() / projectCost) * 100),
    );
  });
  const turnsRemaining = createMemo(() => {
    if (activeProject().id === TRADE_GOODS.id || activeProject().id === HOUSING.id || industryPerTurn() <= 0) {
      return null;
    }
    return Math.max(1, Math.ceil(displayedBuildBucket() / industryPerTurn()));
  });
  const isRepeatableProject = (projectName: string) =>
    projectName === "Colony Base" ||
    projectName === "Colony Ship" ||
    projectName === "Freighters" ||
    projectName === "Artificial Planet Construction" ||
    isShipDesignProject(projectByName().get(projectName.toLowerCase()) ?? null);

  const isBuyPending = createMemo(() =>
    state.pendingCommands.buyProduction.some(
      (command) => command.planetId === props.planet.id,
    ),
  );
  const canBuyProject = createMemo(() => {
    const project = activeProject();
    if (project.id === TRADE_GOODS.id || project.id === HOUSING.id) return false;
    return effectiveProjectCost(project) > 0 && displayedBuildBucket() > 0;
  });
  const buyCost = createMemo(() =>
    canBuyProject()
      ? getBuyProductionCost(
          effectiveProjectCost(activeProject()),
          displayedBuildBucket(),
        )
      : 0,
  );
  const playerTreasury = createMemo(() => currentPlayer()?.treasury ?? 0);
  const canAffordBuy = createMemo(() => playerTreasury() >= buyCost());
  const buyRateLabel = createMemo(() => {
    const cost = effectiveProjectCost(activeProject());
    const invested = Math.max(0, cost - displayedBuildBucket());
    return invested * 2 >= cost ? "2 BC/PP" : "4 BC/PP";
  });
  const buyTooltip = createMemo(() => {
    if (!canBuyProject()) {
      return `${activeProject().name} cannot be bought`;
    }
    if (isBuyPending()) {
      return `Buy queued: ${formatInteger(buyCost())} BC · click to cancel`;
    }
    if (!canAffordBuy()) {
      return `Need ${formatInteger(buyCost())} BC (treasury: ${formatInteger(playerTreasury())} BC)`;
    }
    return `Pay ${formatInteger(buyCost())} BC to finish next turn (${buyRateLabel()})`;
  });
  function handleBuyClick() {
    if (!canBuyProject()) return;
    if (isBuyPending()) {
      cancelBuyProduction(props.planet.id);
      return;
    }
    if (!canAffordBuy()) return;
    buyProduction(props.planet.id);
  }

  function cancelSelection() {
    setIsChoosing(false);
    setSelectedProjectId(null);
    setDraggedQueueIndex(null);
    setDragOverQueueIndex(null);
    setHoveredProjectId(null);
  }

  function openChooser() {
    setSelectedProjectId(activeProject().id);
    setQueuedProjects(
      pendingQueuedProjects().length > 0
        ? pendingQueuedProjects()
        : [activeProject(), ...props.planet.buildQueue].slice(0, 10),
    );
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
    changeBuildQueue(
      props.planet.id,
      nextProjects.map((project) => project.name),
    );
  }

  function addProjectToQueue(project: BuildingViewModel): BuildingViewModel[] {
    const isRepeatable = isRepeatableProject(project.name);
    if (
      !isRepeatable &&
      project.id !== 0 &&
      builtBuildingIds().has(project.id)
    ) {
      return queuedProjects();
    }
    setSelectedProjectId(project.id);
    const currentProjects = queuedProjects();
    const nextProjects = toggleQueuedConstructionProject(currentProjects, project, {
      isRepeatable,
      terminalProjectIds: [TRADE_GOODS.id, HOUSING.id],
    });
    updateQueuedProjects(nextProjects);
    return nextProjects;
  }

  function removeProjectFromQueue(index: number) {
    if (isQueueDragging()) {
      return;
    }
    updateQueuedProjects(
      queuedProjects().filter((_, projectIndex) => projectIndex !== index),
    );
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

  function canBuildShipDesign(design: ShipDesignViewModel) {
    return (
      design.hullSize.name === "Frigate" &&
      availableHullSizes().has(design.hullSize.name) &&
      availableDrives().has(design.drive.name) &&
      availableArmors().has(design.armor.name) &&
      availableComputers().has(design.computer.name) &&
      availableFuelCells().has(design.fuelCell.name) &&
      availableShields().has(design.shield.name)
    );
  }

  function shipDesignProject(design: ShipDesignViewModel): BuildingViewModel {
    return {
      id: shipDesignProjectId(design.name),
      name: design.name,
      description: `${design.hullSize.name} ship design`,
      icon: "🚢",
      cost: 100,
      maintenance: 0,
      unique: false,
      upgrades: [],
      grants: TRADE_GOODS.grants,
      modifiers: [],
    };
  }

  function isShipDesignProject(project: BuildingViewModel | null) {
    return Boolean(project && project.id <= -10000);
  }

  function shipDesignProjectId(name: string) {
    let hash = 0;
    for (let index = 0; index < name.length; index += 1) {
      hash += name.charCodeAt(index) * (index + 1);
    }
    return -10000 - hash;
  }

  function handleProjectClick(project: BuildingViewModel) {
    addProjectToQueue(project);
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
      nextProjects.splice(
        Math.min(toIndex, nextProjects.length),
        0,
        movedProject,
      );
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

  onCleanup(() => {
    if (dragClickTimer) {
      clearTimeout(dragClickTimer);
    }
  });

  onMount(() => {
    const handlePointerDown = (event: PointerEvent) => {
      const target = event.target as Element | null;
      if (
        !isChoosing() ||
        modalRef?.contains(event.target as Node) ||
        target?.closest(".construction-modal")
      ) {
        return;
      }
      cancelSelection();
    };
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        cancelSelection();
      }
    };
    document.addEventListener("pointerdown", handlePointerDown);
    document.addEventListener("keydown", handleKeyDown);
    onCleanup(() => {
      document.removeEventListener("pointerdown", handlePointerDown);
      document.removeEventListener("keydown", handleKeyDown);
    });
  });

  return (
    <section
      class="construction-panel"
      classList={{ "construction-panel--compact": props.compact }}
    >
      <button class="construction-widget" type="button" onClick={toggleChooser}>
        <span class="construction-widget-icon">{activeProject().icon}</span>
        <span class="construction-widget-body">
          <strong>{activeProject().name}</strong>
          <span
            class="construction-progress"
            aria-label={`Construction progress ${progressPercent()}%`}
          >
            <span style={{ width: `${progressPercent()}%` }} />
          </span>
        </span>
      </button>

      <div class="construction-status-row">
        <div class="construction-progress-wrap">
          <Show when={props.planet.isBuilding && props.planet.currentProject}>
            <div class="construction-progress-meta">
              <span>
                {completedProduction()} / {effectiveProjectCost(activeProject())} PP invested
                {turnsRemaining()
                  ? ` · ~${turnsRemaining()} turns`
                  : " · no production"}
              </span>
            </div>
          </Show>
        </div>
        <Show when={canBuyProject()}>
          <button
            type="button"
            class="construction-buy-button"
            classList={{ "is-pending": isBuyPending() }}
            onClick={handleBuyClick}
            disabled={state.isBusy || (!isBuyPending() && !canAffordBuy())}
            title={buyTooltip()}
          >
            <span class="construction-buy-button-label">
              {isBuyPending() ? "Buying" : "Buy"}
            </span>
            <span class="construction-buy-button-cost">
              {formatInteger(buyCost())} BC
            </span>
          </button>
        </Show>
        <Show when={!props.compact}>
          <label class="construction-autobuild construction-autobuild--header">
            <span>Autobuild</span>
            <span class="construction-autobuild-switch">
              <input
                type="checkbox"
                checked={isAutobuildEnabled()}
                aria-label={`Autobuild ${props.planet.name}`}
                onChange={(event) =>
                  setPlanetAutobuild(
                    props.planet.id,
                    event.currentTarget.checked,
                  )
                }
              />
              <span aria-hidden="true" />
            </span>
          </label>
        </Show>
      </div>

      <Show when={isChoosing()}>
        <Portal>
          <div class="construction-modal-backdrop">
            <section
              class="construction-modal"
              role="dialog"
              aria-modal="true"
              aria-labelledby="construction-modal-title"
              ref={modalRef}
            >
              <header class="construction-modal-header">
                <div>
                  <p class="eyebrow">Construction</p>
                  <h2 id="construction-modal-title">{props.planet.name}</h2>
                </div>
                <button
                  class="secondary-button construction-close-button"
                  type="button"
                  onClick={cancelSelection}
                >
                  Close
                </button>
              </header>

              <div class="construction-modal-grid">
                <aside class="construction-modal-sidebar">
                  <h3>Buildings</h3>
                  <div class="construction-project-list">
                    <For each={buildingListProjects()}>
                      {(project) => (
                        <button
                          type="button"
                          class="construction-project-option"
                          classList={{
                            "is-selected": selectedProject().id === project.id,
                            "is-queued": queuedProjectIds().has(project.id),
                          }}
                          disabled={state.isBusy}
                          onMouseEnter={() => setHoveredProjectId(project.id)}
                          onMouseLeave={() => setHoveredProjectId(null)}
                          onClick={() => handleProjectClick(project)}
                        >
                          <span
                            class="construction-project-queue-marker"
                            aria-hidden="true"
                          />
                          <span class="construction-project-name">
                            {project.name}
                          </span>
                          <span class="construction-project-cost">
                            {project.cost > 0 ? `${effectiveProjectCost(project)} PP` : "Now"}
                          </span>
                        </button>
                      )}
                    </For>
                  </div>
                </aside>

                <main class="construction-queue-panel">
                  <article class="construction-project-details">
                    <div class="construction-project-title">
                      <div>
                        <h3>{previewProject().name}</h3>
                        <p>
                          {previewProject().description ||
                            "No project description available yet."}
                        </p>
                      </div>
                    </div>
                    <dl class="construction-facts">
                      <div>
                        <dt>Cost</dt>
                        <dd>{effectiveProjectCost(previewProject())} PP</dd>
                      </div>
                      <div>
                        <dt>Maintenance</dt>
                        <dd>{previewProject().maintenance} BC</dd>
                      </div>
                      <div>
                        <dt>Output</dt>
                        <dd>
                          {previewProject().id === HOUSING.id
                            ? `${housingGrowthProjection()} population growth`
                            : `${industryPerTurn()} PP/turn`}
                        </dd>
                      </div>
                    </dl>
                  </article>

                  <section class="construction-queue">
                    <div class="construction-column-header">
                      <h3>Queue</h3>
                      <span>{queueSlots().filter(Boolean).length}/10</span>
                    </div>
                    <div class="construction-queue-slots">
                      <For each={queueSlots()}>
                        {(project, index) => (
                          <div
                            class={`construction-queue-slot ${project ? "is-filled" : ""} ${draggedQueueIndex() === index() ? "is-dragging" : ""} ${dragOverQueueIndex() === index() ? "is-drop-target" : ""}`}
                            draggable={Boolean(project)}
                            onClick={() =>
                              project && removeProjectFromQueue(index())
                            }
                            onMouseEnter={() =>
                              project && setHoveredProjectId(project.id)
                            }
                            onMouseLeave={() =>
                              project && setHoveredProjectId(null)
                            }
                            onDragStart={() => startQueueDrag(index())}
                            onDragEnd={endQueueDrag}
                            onDragOver={(event) => {
                              event.preventDefault();
                              moveQueueItemDuringDrag(index());
                            }}
                            onDrop={() => reorderQueue(index())}
                          >
                            <span class="construction-queue-index">
                              {index() + 1}
                            </span>
                            <span class="construction-queue-name">
                              {project?.name ?? "Empty"}
                            </span>
                            <span class="construction-queue-cost">
                              {project && project.cost > 0
                                ? `${effectiveProjectCost(project)} PP`
                                : ""}
                            </span>
                          </div>
                        )}
                      </For>
                    </div>
                  </section>
                </main>

                <aside class="construction-modal-sidebar construction-specials">
                  <Show when={sectionOneCommands().length > 0}>
                    <section class="construction-special-section">
                      <div class="construction-special-list">
                        <For each={sectionOneCommands()}>
                          {(command) => {
                            const isColonyBase =
                              command.unlockName === "Colony Base";
                            const hasSuitablePlanets = (command as any)
                              .hasSuitablePlanets;
                            const hasTech = (command as any).hasTech;
                            const colonyBaseDisabled =
                              isColonyBase && !hasSuitablePlanets;
                            const isDisabled = isColonyBase
                              ? state.isBusy || colonyBaseDisabled || !hasTech
                              : state.isBusy || !command.project;
                            return (
                              <button
                                type="button"
                                class="construction-special-option"
                                disabled={isDisabled}
                                onMouseEnter={() =>
                                  !isColonyBase &&
                                  command.project &&
                                  setHoveredProjectId(command.project.id)
                                }
                                onMouseLeave={() =>
                                  !isColonyBase &&
                                  command.project &&
                                  setHoveredProjectId(null)
                                }
                                onClick={() => {
                                  if (isColonyBase) {
                                    if (!colonyBaseDisabled && hasTech) {
                                      const project = colonyBaseProject();
                                      if (project) {
                                        addProjectToQueue(project);
                                      }
                                    }
                                  } else if (command.project) {
                                    handleProjectClick(command.project);
                                  }
                                }}
                              >
                                <span>
                                  {isColonyBase && !hasSuitablePlanets
                                    ? "No suitable planets for Colony Base"
                                    : command.label}
                                </span>
                              </button>
                            );
                          }}
                        </For>
                      </div>
                    </section>
                  </Show>

                  <Show when={shipDesigns().some(Boolean)}>
                    <section class="construction-special-section">
                      <div class="construction-design-slots">
                        <For each={shipDesigns()}>
                          {(design) => (
                            <button
                              type="button"
                              class="construction-design-slot"
                              disabled={
                                state.isBusy ||
                                !design ||
                                !canBuildShipDesign(design)
                              }
                              onMouseEnter={() => {
                                if (!design || !canBuildShipDesign(design)) {
                                  return;
                                }
                                const project = projectByName().get(
                                  design.name.toLowerCase(),
                                );
                                if (project) {
                                  setHoveredProjectId(project.id);
                                }
                              }}
                              onMouseLeave={() => setHoveredProjectId(null)}
                              onClick={() => {
                                if (!design) return;
                                const project = projectByName().get(
                                  design.name.toLowerCase(),
                                );
                                if (project) {
                                  handleProjectClick(project);
                                }
                              }}
                            >
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
                    <section class="construction-special-section">
                      <div class="construction-special-list">
                        <For each={personnelCommands()}>
                          {(command) => (
                            <button
                              type="button"
                              class="construction-special-option"
                              disabled={state.isBusy || !command.project}
                              onMouseEnter={() =>
                                command.project &&
                                setHoveredProjectId(command.project.id)
                              }
                              onMouseLeave={() =>
                                command.project && setHoveredProjectId(null)
                              }
                              onClick={() =>
                                command.project &&
                                handleProjectClick(command.project)
                              }
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

function governmentModifierValue(government: TechViewModel | null, modifierName: string, fallback: number) {
  const modifier = government?.grants.modifiers.find((item) => item.name === modifierName);
  return modifier?.value ?? fallback;
}
