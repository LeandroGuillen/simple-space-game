import { createMemo, createSignal, For } from "solid-js";
import type { MovePopulationCommand } from "../lib/types/commands";
import type { PlanetViewModel } from "../lib/types/domain";
import { raceTraitById } from "../lib/race-design";
import {
  getMoraleAdjustedLaborOutput,
  getPlanetIndustryForWorkers,
  getPlanetPollutionDivisor,
  getPlanetPollutionForWorkers,
  getPlanetPollutionSize,
} from "../lib/utils";
import { useAppContext } from "../state/app-context";

type PopCategory = "farmers" | "workers" | "scientists";

type OutputBreakdown = {
  total: number;
  fromPopulation: number;
  fromBuildings: number;
};

type BuildingContribution = {
  buildingName: string;
  label: string;
  total: number;
};

type RaceContribution = {
  raceName: string;
  traitName: string;
  label: string;
  total: number;
};

const POP_LABELS: Record<PopCategory, string> = {
  farmers: "Farmer",
  workers: "Worker",
  scientists: "Scientist",
};

const POP_COLORS: Record<PopCategory, string> = {
  farmers: "#b59a58",
  workers: "#5a8cb8",
  scientists: "#9eb8a8",
};

function formatSigned(value: number) {
  return `${value > 0 ? "+" : ""}${value}`;
}

type PopulationPanelProps = {
  planet: PlanetViewModel;
};

export function PopulationPanel(props: PopulationPanelProps) {
  const { state, addMovePopulationCommand } = useAppContext();
  const [draggedCount, setDraggedCount] = createSignal(0);
  const [dragSource, setDragSource] = createSignal<PopCategory | null>(null);
  const pendingMoves = createMemo(() =>
    state.pendingCommands.movePopulation.filter(
      (cmd) => cmd.planetId === props.planet.id,
    ),
  );

  const isDragging = () => draggedCount() > 0;

  const getPopCount = (cat: PopCategory) => {
    let base = 0;
    switch (cat) {
      case "farmers":
        base = props.planet.food.farmers;
        break;
      case "workers":
        base = props.planet.industry.workers;
        break;
      case "scientists":
        base = props.planet.science.scientists;
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

  const getOutputIcons = (
    count: number,
    tensIcon: string,
    onesIcon: string,
  ): string => {
    const tens = Math.floor(count / 10);
    const ones = count % 10;
    const icons: string[] = [];
    for (let i = 0; i < tens; i++) icons.push(tensIcon);
    for (let i = 0; i < ones; i++) icons.push(onesIcon);
    return icons.join(" ");
  };

  const getBuildingModifierBonus = (modifierName: string) =>
    props.planet.buildings.reduce(
      (sum, building) =>
        sum +
        building.modifiers.reduce(
          (modifierSum, modifier) =>
            modifierSum + (modifier.name === modifierName ? modifier.bonus : 0),
          0,
        ),
      0,
    );

  const owner = createMemo(() =>
    state.game?.players.find((player) => player.id === props.planet.owner) ?? null,
  );

  const getRaceContributions = (cat: PopCategory): RaceContribution[] => {
    const player = owner();
    if (!player) {
      return [];
    }
    const workerCount = getPopCount(cat);
    const contributions: RaceContribution[] = [];
    for (const traitId of player.raceTraitIds) {
      const trait = raceTraitById.get(traitId);
      if (!trait) {
        continue;
      }
      for (const modifier of player.raceGrants.modifiers) {
        if (cat === "farmers" && modifier.name === "food_per_farmer" && trait.category === "Farming") {
          contributions.push({
            raceName: player.raceName,
            traitName: trait.name,
            label: `${formatSigned(modifier.value)} food per farmer × ${workerCount}`,
            total: modifier.value * workerCount,
          });
        }
        if (cat === "workers" && modifier.name === "industry_per_worker" && trait.category === "Production") {
          contributions.push({
            raceName: player.raceName,
            traitName: trait.name,
            label: `${formatSigned(modifier.value)} production per worker × ${workerCount}`,
            total: modifier.value * workerCount,
          });
        }
        if (cat === "scientists" && modifier.name === "science_per_scientist" && trait.category === "Research") {
          contributions.push({
            raceName: player.raceName,
            traitName: trait.name,
            label: `${formatSigned(modifier.value)} research per scientist × ${workerCount}`,
            total: modifier.value * workerCount,
          });
        }
      }
    }
    return contributions;
  };

  const getOutputBreakdown = (cat: PopCategory): OutputBreakdown => {
    if (cat === "farmers") {
      const raceBonus = getRaceContributions("farmers").reduce(
        (sum, contribution) => sum + contribution.total / Math.max(1, getPopCount("farmers")),
        0,
      );
      const fromPopulation = getMoraleAdjustedLaborOutput(
        getPopCount("farmers"),
        Math.max(0, props.planet.food.produced_per_farmer - raceBonus),
        props.planet,
      );
      const fromBuildings = props.planet.food.produced_per_building +
        getMoraleAdjustedLaborOutput(getPopCount("farmers"), raceBonus, props.planet);
      return {
        total: fromPopulation + fromBuildings,
        fromPopulation,
        fromBuildings,
      };
    }
    if (cat === "workers") {
      const perWorkerBuildingBonus = getBuildingModifierBonus("industry_per_worker");
      const perWorkerRaceBonus = getRaceContributions("workers").reduce(
        (sum, contribution) => sum + contribution.total / Math.max(1, getPopCount("workers")),
        0,
      );
      const basePerWorker = Math.max(
        0,
        props.planet.industry.produced_per_worker - perWorkerBuildingBonus - perWorkerRaceBonus,
      );
      const fromPopulation = getMoraleAdjustedLaborOutput(
        getPopCount("workers"),
        basePerWorker,
        props.planet,
      );
      const fromBuildings =
        props.planet.industry.produced_per_building +
        getMoraleAdjustedLaborOutput(
          getPopCount("workers"),
          perWorkerBuildingBonus + perWorkerRaceBonus,
          props.planet,
        ) +
        Math.floor(props.planet.population / 1000) *
          props.planet.industry.produced_per_colonist;
      return {
        total: fromPopulation + fromBuildings,
        fromPopulation,
        fromBuildings,
      };
    }
    const perScientistBuildingBonus = getBuildingModifierBonus("science_per_scientist");
    const perScientistRaceBonus = getRaceContributions("scientists").reduce(
      (sum, contribution) => sum + contribution.total / Math.max(1, getPopCount("scientists")),
      0,
    );
    const basePerScientist = Math.max(
      0,
      props.planet.science.produced_per_scientist - perScientistBuildingBonus - perScientistRaceBonus,
    );
    const fromPopulation = getMoraleAdjustedLaborOutput(
      getPopCount("scientists"),
      basePerScientist,
      props.planet,
    );
    const fromBuildings =
      props.planet.science.produced_per_building +
      getMoraleAdjustedLaborOutput(
        getPopCount("scientists"),
        perScientistBuildingBonus + perScientistRaceBonus,
        props.planet,
      );
    return {
      total: fromPopulation + fromBuildings,
      fromPopulation,
      fromBuildings,
    };
  };

  const renderRaceContributionRows = (cat: PopCategory) => {
    const contributions = getRaceContributions(cat);
    if (contributions.length === 0) {
      return (
        <div class="pop-output-tooltip-row pop-output-tooltip-row--detail">
          <span>No race trait bonuses</span>
          <strong>0</strong>
        </div>
      );
    }

    return (
      <>
        <For each={contributions}>
          {(contribution) => (
            <div class="pop-output-tooltip-row pop-output-tooltip-row--detail">
              <span>
                {contribution.raceName} trait: {contribution.traitName} ({contribution.label})
              </span>
              <strong>{formatSigned(contribution.total)}</strong>
            </div>
          )}
        </For>
      </>
    );
  };

  const getBuildingContributions = (
    cat: PopCategory,
  ): BuildingContribution[] => {
    const workerCount = getPopCount(cat);
    const contributions: BuildingContribution[] = [];
    for (const building of props.planet.buildings) {
      for (const modifier of building.modifiers) {
        if (cat === "farmers" && modifier.name === "food_flat") {
          contributions.push({
            buildingName: building.name,
            label: "flat food",
            total: modifier.bonus,
          });
        }
        if (cat === "workers" && modifier.name === "industry_per_worker") {
          contributions.push({
            buildingName: building.name,
            label: `+${modifier.bonus} per worker × ${workerCount}`,
            total: modifier.bonus * workerCount,
          });
        }
        if (cat === "workers" && modifier.name === "industry_flat") {
          contributions.push({
            buildingName: building.name,
            label: "flat industry",
            total: modifier.bonus,
          });
        }
        if (cat === "workers" && modifier.name === "industry_per_colonist") {
          const colonistCount = Math.floor(props.planet.population / 1000);
          contributions.push({
            buildingName: building.name,
            label: `+${modifier.bonus} per colonist × ${colonistCount}`,
            total: modifier.bonus * colonistCount,
          });
        }
        if (cat === "scientists" && modifier.name === "science_per_scientist") {
          contributions.push({
            buildingName: building.name,
            label: `+${modifier.bonus} per scientist × ${workerCount}`,
            total: modifier.bonus * workerCount,
          });
        }
        if (cat === "scientists" && modifier.name === "science_flat") {
          contributions.push({
            buildingName: building.name,
            label: "flat science",
            total: modifier.bonus,
          });
        }
      }
    }
    return contributions;
  };

  const renderBuildingContributionRows = (cat: PopCategory) => {
    const contributions = getBuildingContributions(cat);
    if (contributions.length === 0) {
      return (
        <div class="pop-output-tooltip-row pop-output-tooltip-row--detail">
          <span>No building bonuses</span>
          <strong>0</strong>
        </div>
      );
    }

    return (
      <>
        <For each={contributions}>
          {(contribution) => (
            <div class="pop-output-tooltip-row pop-output-tooltip-row--detail">
              <span>
                {contribution.buildingName}: {contribution.label}
              </span>
              <strong>+{contribution.total}</strong>
            </div>
          )}
        </For>
      </>
    );
  };

  const getIndustryOutput = () =>
    getOutputIcons(
      getPlanetIndustryForWorkers(props.planet, getPopCount("workers")),
      "⚡",
      "🔧",
    );
  const getScienceOutput = () =>
    getOutputIcons(getOutputBreakdown("scientists").total, "🔬", "🧫");
  const getTotalPopUnits = () =>
    getPopCount("farmers") + getPopCount("workers") + getPopCount("scientists");
  const getPollutionDivisor = () => getPlanetPollutionDivisor(props.planet);
  const getPollutionPlanetSize = () => getPlanetPollutionSize(props.planet);
  const getPollution = () =>
    getPlanetPollutionForWorkers(props.planet, getPopCount("workers"));
  const renderPollutionIcons = () => {
    const pollution = getPollution();
    if (pollution === 0) {
      return null;
    }
    const tens = Math.floor(pollution / 10);
    const ones = pollution % 10;
    return (
      <>
        <For each={Array.from({ length: tens })}>
          {() => <span class="pollution-output-icon">🌫</span>}
        </For>
        <For each={Array.from({ length: ones })}>
          {() => <span class="pollution-output-icon">☁</span>}
        </For>
      </>
    );
  };

  const renderFoodOutput = () => {
    const produced = getOutputBreakdown("farmers").total;
    const required = getTotalPopUnits();
    const consumed = Math.min(produced, required);
    const surplus = Math.max(0, produced - required);
    const deficit = Math.max(0, required - produced);
    const renderFoodIcons = (count: number, outlined = false) => {
      const tens = Math.floor(count / 10);
      const ones = count % 10;
      return (
        <>
          <For each={Array.from({ length: tens })}>
            {() => (
              <span
                class={`food-output-icon ${outlined ? "food-output-icon--deficit" : ""}`}
                data-icon="🌾"
              >
                🌾
              </span>
            )}
          </For>
          <For each={Array.from({ length: ones })}>
            {() => (
              <span
                class={`food-output-icon ${outlined ? "food-output-icon--deficit" : ""}`}
                data-icon="🌱"
              >
                🌱
              </span>
            )}
          </For>
        </>
      );
    };

    return (
      <div class="food-output-icons">
        {renderFoodIcons(consumed)}
        {surplus > 0 && <span class="food-output-gap" aria-hidden="true" />}
        {renderFoodIcons(surplus)}
        {renderFoodIcons(deficit, true)}
      </div>
    );
  };

  const handleDragStart = (
    e: DragEvent,
    category: PopCategory,
    index: number,
  ) => {
    const count = getPopCount(category);
    const toDrag = index === count - 1 ? 1 : count - index;
    setDraggedCount(toDrag);
    setDragSource(category);
    const label = POP_LABELS[category];
    const ghost = document.createElement("div");
    ghost.className = "drag-ghost";
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
    e.dataTransfer!.effectAllowed = "move";
  };

  const handleDragEnd = () => {
    setDragSource(null);
    setDraggedCount(0);
  };

  const handleDrop = (e: DragEvent, to: PopCategory) => {
    e.preventDefault();
    const dropZone = e.currentTarget as HTMLDivElement | null;
    dropZone?.classList.remove("drag-over");
    const source = dragSource();
    const count = draggedCount();
    if (!source || count === 0 || source === to) return;
    const cmd: MovePopulationCommand = {
      planetId: props.planet.id,
      from: source,
      to,
      count,
    };
    addMovePopulationCommand(cmd);
    setDragSource(null);
    setDraggedCount(0);
  };

  return (
    <section class="panel">
      <p class="eyebrow">Population</p>

      <div class={`pop-grid ${isDragging() ? "is-dragging" : ""}`}>
        <div class={`pop-row ${isDragging() ? "pop-row--drag-active" : ""}`}>
          <span class="pop-stat-label">Food</span>
          <div class="pop-output pop-output--food">
            {renderFoodOutput()}
            <div class="pop-output-tooltip">
              <div class="pop-output-tooltip-row pop-output-tooltip-row--total">
                <span>Total food produced:</span>
                <strong>{getOutputBreakdown("farmers").total}</strong>
              </div>
              <div class="pop-output-tooltip-row">
                <span>From farmers:</span>
                <strong>{getOutputBreakdown("farmers").fromPopulation}</strong>
              </div>
              <div class="pop-output-tooltip-row">
                <span>From buildings:</span>
                <strong>{getOutputBreakdown("farmers").fromBuildings}</strong>
              </div>
              {renderRaceContributionRows("farmers")}
              {renderBuildingContributionRows("farmers")}
            </div>
          </div>
          <div
            class="pop-units-row"
            onDragOver={(e) => {
              e.preventDefault();
              e.currentTarget.classList.add("drag-over");
            }}
            onDragLeave={(e) => e.currentTarget.classList.remove("drag-over")}
            onDrop={(e) => handleDrop(e, "farmers")}
          >
            {Array.from({ length: getPopCount("farmers") }).map((_, idx) => (
              <button
                class="pop-unit"
                style={{ background: POP_COLORS.farmers }}
                draggable={true}
                onDragStart={(e) => handleDragStart(e, "farmers", idx)}
                onDragEnd={handleDragEnd}
              />
            ))}
          </div>
        </div>

        <div class={`pop-row ${isDragging() ? "pop-row--drag-active" : ""}`}>
          <span class="pop-stat-label">Industry</span>
          <div class="pop-output">
            <span>{getIndustryOutput()}</span>
            {getPollution() > 0 && (
              <span
                class="pollution-output-icons"
                aria-label={`${getPollution()} pollution`}
              >
                {renderPollutionIcons()}
              </span>
            )}
            <div class="pop-output-tooltip">
              <div class="pop-output-tooltip-row pop-output-tooltip-row--total">
                <span>Final industry output:</span>
                <strong>
                  {getPlanetIndustryForWorkers(
                    props.planet,
                    getPopCount("workers"),
                  )}
                </strong>
              </div>
              <div class="pop-output-tooltip-row">
                <span>Gross industry produced:</span>
                <strong>{getOutputBreakdown("workers").total}</strong>
              </div>
              <div class="pop-output-tooltip-row">
                <span>Pollution:</span>
                <strong>-{getPollution()}</strong>
              </div>
              <div class="pop-output-tooltip-row">
                <span>From workers:</span>
                <strong>{getOutputBreakdown("workers").fromPopulation}</strong>
              </div>
              <div class="pop-output-tooltip-row">
                <span>From buildings:</span>
                <strong>{getOutputBreakdown("workers").fromBuildings}</strong>
              </div>
              {renderRaceContributionRows("workers")}
              <div class="pop-output-tooltip-row pop-output-tooltip-row--detail">
                <span>Planet absorption:</span>
                <strong>{getPollutionPlanetSize()}</strong>
              </div>
              <div class="pop-output-tooltip-row pop-output-tooltip-row--detail">
                <span>Cleanup divisor:</span>
                <strong>{getPollutionDivisor()}×</strong>
              </div>
              {renderBuildingContributionRows("workers")}
            </div>
          </div>
          <div
            class="pop-units-row"
            onDragOver={(e) => {
              e.preventDefault();
              e.currentTarget.classList.add("drag-over");
            }}
            onDragLeave={(e) => e.currentTarget.classList.remove("drag-over")}
            onDrop={(e) => handleDrop(e, "workers")}
          >
            {Array.from({ length: getPopCount("workers") }).map((_, idx) => (
              <button
                class="pop-unit"
                style={{ background: POP_COLORS.workers }}
                draggable={true}
                onDragStart={(e) => handleDragStart(e, "workers", idx)}
                onDragEnd={handleDragEnd}
              />
            ))}
          </div>
        </div>

        <div class={`pop-row ${isDragging() ? "pop-row--drag-active" : ""}`}>
          <span class="pop-stat-label">Science</span>
          <div class="pop-output">
            {getScienceOutput()}
            <div class="pop-output-tooltip">
              <div class="pop-output-tooltip-row pop-output-tooltip-row--total">
                <span>Total science produced:</span>
                <strong>{getOutputBreakdown("scientists").total}</strong>
              </div>
              <div class="pop-output-tooltip-row">
                <span>From scientists:</span>
                <strong>
                  {getOutputBreakdown("scientists").fromPopulation}
                </strong>
              </div>
              <div class="pop-output-tooltip-row">
                <span>From buildings:</span>
                <strong>
                  {getOutputBreakdown("scientists").fromBuildings}
                </strong>
              </div>
              {renderRaceContributionRows("scientists")}
              {renderBuildingContributionRows("scientists")}
            </div>
          </div>
          <div
            class="pop-units-row"
            onDragOver={(e) => {
              e.preventDefault();
              e.currentTarget.classList.add("drag-over");
            }}
            onDragLeave={(e) => e.currentTarget.classList.remove("drag-over")}
            onDrop={(e) => handleDrop(e, "scientists")}
          >
            {Array.from({ length: getPopCount("scientists") }).map((_, idx) => (
              <button
                class="pop-unit"
                style={{ background: POP_COLORS.scientists }}
                draggable={true}
                onDragStart={(e) => handleDragStart(e, "scientists", idx)}
                onDragEnd={handleDragEnd}
              />
            ))}
          </div>
        </div>
      </div>

      <div class="list-block">
        <p class="eyebrow">Installed Buildings</p>
        <ul class="compact-list">
          {props.planet.buildings.map((building) => (
            <li>
              <strong>{building.name}</strong>
              <span>{building.maintenance} maintenance</span>
            </li>
          ))}
          {props.planet.buildings.length === 0 && <li>No structures yet.</li>}
        </ul>
      </div>
    </section>
  );
}
