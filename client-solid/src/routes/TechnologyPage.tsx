import { For, Show, createMemo, createSignal } from "solid-js";

import type {
  BuildingViewModel,
  TechLevelViewModel,
  TechSubjectViewModel,
  TechViewModel,
} from "../lib/types/domain";
import { researchLevelUnlocksAllTechs } from "../lib/research";
import { useAppContext } from "../state/app-context";

function arePreviousLevelsComplete(
  subject: TechSubjectViewModel,
  levelIndex: number,
  researchedTechs: string[],
  traitIds: string[] | undefined,
): boolean {
  for (let i = 0; i < levelIndex; i++) {
    const level = subject.levels[i];
    const allTechs = researchLevelUnlocksAllTechs(level, traitIds);
    
    if (allTechs) {
      if (!level.availableTechs.every((tech) => researchedTechs.includes(tech))) {
        return false;
      }
    } else {
      if (!level.availableTechs.some((tech) => researchedTechs.includes(tech))) {
        return false;
      }
    }
  }
  return true;
}

type KnownTechEntry = {
  name: string;
  description: string;
  cost: number;
  maintenance: number;
  bonuses: string[];
  known: boolean;
  unavailable: boolean;
  source: "Building" | "Technology";
};

type KnownTechLevel = {
  level: TechLevelViewModel;
  techs: KnownTechEntry[];
};

type KnownTechSubject = {
  subject: TechSubjectViewModel;
  levels: KnownTechLevel[];
};

export function TechnologyPage() {
  const { state, currentPlayer } = useAppContext();
  const [knownOnly, setKnownOnly] = createSignal(true);

  const knownTree = createMemo<KnownTechSubject[]>(() => {
    const game = state.game;
    const player = currentPlayer();
    if (!game || !player) {
      return [];
    }

    const knownNames = new Set(player.researchedTechs.map((name) => name.toLowerCase()));
    const techByName = new Map(
      game.resources.techCollection.techs.map((tech) => [
        tech.name.toLowerCase(),
        tech,
      ]),
    );
    const buildingByName = new Map(
      game.resources.buildingCollection.buildings.map((building) => [
        building.name.toLowerCase(),
        building,
      ]),
    );

    return game.resources.techTree.subjects
      .map((subject) => ({
        subject,
        levels: subject.levels
          .map((level, levelIndex) => ({
            level,
            techs: level.availableTechs
              .filter(
                (techName) =>
                  !knownOnly() || knownNames.has(techName.toLowerCase()),
              )
              .map((techName) =>
                knownTechEntry(
                  techName,
                  knownNames.has(techName.toLowerCase()),
                  !arePreviousLevelsComplete(
                    subject,
                    levelIndex,
                    player.researchedTechs,
                    player.raceTraitIds,
                  ) ||
                    (!researchLevelUnlocksAllTechs(level, player.raceTraitIds) &&
                      level.availableTechs.some((levelTech) =>
                        knownNames.has(levelTech.toLowerCase()),
                      ) &&
                      !knownNames.has(techName.toLowerCase())),
                  techByName.get(techName.toLowerCase()),
                  findUnlockedBuilding(
                    techByName.get(techName.toLowerCase()),
                    buildingByName,
                  ) ?? buildingByName.get(techName.toLowerCase()),
                ),
              ),
          }))
          .filter((level) => level.techs.length > 0),
      }))
      .filter((subject) => subject.levels.length > 0);
  });

  const knownTechCount = createMemo(() =>
    knownTree().reduce(
      (count, subject) =>
        count +
        subject.levels.reduce(
          (levelCount, level) =>
            levelCount + level.techs.filter((tech) => tech.known).length,
          0,
        ),
      0,
    ),
  );

  return (
    <section class="panel technology-page">
      <div class="section-header">
        <div>
          <p class="eyebrow">Technology</p>
          <h2>Technology catalog</h2>
        </div>
        <div class="technology-header-tools">
          <label class="technology-toggle">
            <input
              type="checkbox"
              checked={knownOnly()}
              onChange={(event) => setKnownOnly(event.currentTarget.checked)}
            />
            <span aria-hidden="true" />
            <strong>Known only</strong>
          </label>
          <div class="technology-count">
            <span>Known</span>
            <strong>{knownTechCount()}</strong>
          </div>
        </div>
      </div>

      <Show
        when={knownTree().length > 0}
        fallback={
          <p class="empty-copy">
            {knownOnly()
              ? "No researched technologies are known yet."
              : "No technologies are available in the current catalog."}
          </p>
        }
      >
        <div class="technology-tree">
          <For each={knownTree()}>
            {(subject) => (
              <section class="technology-subject">
                <h3>{subject.subject.name}</h3>
                <div class="technology-level-list">
                  <For each={subject.levels}>
                    {(level) => (
                      <section class="technology-level">
                        <div class="technology-level-header">
                          <span>{level.level.name}</span>
                          <strong>{level.level.cost}RP</strong>
                        </div>
                        <div class="technology-tech-list">
                          <For each={level.techs}>
                            {(tech) => (
                              <article
                                class="technology-tech"
                                classList={{
                                  "is-unknown": !tech.known,
                                  "is-unavailable": tech.unavailable,
                                }}
                              >
                                <div class="technology-tech-header">
                                  <h4>{tech.name}</h4>
                                  <span>
                                    {tech.known
                                      ? tech.source
                                      : tech.unavailable
                                        ? "Unavailable"
                                        : "Unknown"}
                                  </span>
                                </div>
                                <Show when={tech.description}>
                                  <p>{tech.description}</p>
                                </Show>
                                <dl class="technology-tech-facts">
                                  <Show when={tech.cost > 0}>
                                    <div>
                                      <dt>Cost</dt>
                                      <dd>{tech.cost} PP</dd>
                                    </div>
                                  </Show>
                                  <Show when={tech.maintenance > 0}>
                                    <div>
                                      <dt>Maintenance</dt>
                                      <dd>{tech.maintenance} BC/turn</dd>
                                    </div>
                                  </Show>
                                  <For each={tech.bonuses}>
                                    {(bonus) => (
                                      <div>
                                        <dt>Bonus</dt>
                                        <dd>{bonus}</dd>
                                      </div>
                                    )}
                                  </For>
                                </dl>
                              </article>
                            )}
                          </For>
                        </div>
                      </section>
                    )}
                  </For>
                </div>
              </section>
            )}
          </For>
        </div>
      </Show>
    </section>
  );
}

function knownTechEntry(
  name: string,
  known: boolean,
  unavailable: boolean,
  tech: TechViewModel | undefined,
  building: BuildingViewModel | undefined,
): KnownTechEntry {
  return {
    name,
    description: normalizeDescription(tech?.description || building?.description || ""),
    cost: building?.cost || tech?.cost || 0,
    maintenance: tech?.maintenance || building?.maintenance || 0,
    bonuses: building?.modifiers.map(formatBonus) ?? tech?.grants.modifiers.map(formatBonus) ?? [],
    known,
    unavailable,
    source: building ? "Building" : "Technology",
  };
}

function findUnlockedBuilding(
  tech: TechViewModel | undefined,
  buildingByName: Map<string, BuildingViewModel>,
) {
  const buildingName = tech?.grants.unlocks.buildings[0];
  return buildingName ? buildingByName.get(buildingName.toLowerCase()) : undefined;
}

function formatBonus(effect: { name: string; description?: string; bonus?: number; value?: number }) {
  const value = effect.bonus ?? effect.value ?? 0;
  const sign = value > 0 ? "+" : "";
  return `${effect.description || humanizeModifierName(effect.name)} ${sign}${value}`;
}

function humanizeModifierName(name: string) {
  return name
    .split("_")
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

function normalizeDescription(description: string) {
  const trimmed = description.trim();
  return trimmed.toLowerCase() === "no description provided" ? "" : trimmed;
}
