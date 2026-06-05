import { useNavigate } from "@solidjs/router";
import { createEffect, createMemo, createSignal, For, onMount, Show } from "solid-js";
import type { Accessor } from "solid-js";

import {
  fallbackGovernments,
  isRaceTraitBlocked,
  mutationNegativePicks,
  MUTATION_PICK_BUDGET,
  RACE_EDITOR_LEFT_COLUMN,
  RACE_EDITOR_MIDDLE_COLUMN,
  RACE_EDITOR_RIGHT_COLUMN,
  mutationRemainingPicks,
  racePickCost,
  raceTraitDefinitions,
  raceTraitGroupsForCategories,
  swapExclusiveRaceTrait,
  validateCustomRaceDesign,
} from "../lib/race-design";
import type { RaceTraitDefinition } from "../lib/race-design";
import type { ApiGovernment, RaceTraitId } from "../lib/types/api";
import { useAppContext } from "../state/app-context";

function BackArrowIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 18 18" fill="none" aria-hidden="true">
      <path
        d="M11 14L5 9l6-5"
        stroke="currentColor"
        stroke-width="1.6"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
    </svg>
  );
}

type RaceTraitGroup = { category: string; traits: RaceTraitDefinition[] };

function MutationTraitSection(props: {
  group: RaceTraitGroup;
  readOnly: Accessor<boolean>;
  traitIds: Accessor<RaceTraitId[]>;
  onToggleTrait: (id: string) => void;
}) {
  return (
    <section class="race-editor-section">
      <h2>{props.group.category}</h2>
      <div class="race-trait-list">
        <For each={props.group.traits}>
          {(trait) => {
            const selected = () => props.traitIds().includes(trait.id as RaceTraitId);
            const blocked = () => isRaceTraitBlocked(trait) && !selected();
            return (
              <button
                type="button"
                class="race-trait"
                classList={{
                  selected: selected(),
                  disabled: props.readOnly() || !trait.enabled || blocked(),
                }}
                disabled={props.readOnly() || !trait.enabled || blocked()}
                aria-pressed={selected()}
                onClick={() => props.onToggleTrait(trait.id)}
                title={!trait.enabled ? trait.description : undefined}
              >
                <span class="race-trait-main">
                  <span class="race-trait-name">{trait.name}</span>
                  <span class="race-trait-description">{trait.description}</span>
                </span>
                <span classList={{ "race-trait-cost": true, negative: trait.pickCost < 0 }}>
                  {trait.pickCost > 0 ? `+${trait.pickCost}` : trait.pickCost}
                </span>
              </button>
            );
          }}
        </For>
      </div>
    </section>
  );
}

export function MutationPage() {
  const navigate = useNavigate();
  const { state, mutateRace } = useAppContext();
  const [governments] = createSignal<ApiGovernment[]>(fallbackGovernments);
  const [traitIds, setTraitIds] = createSignal<RaceTraitId[]>([]);
  const [saveError, setSaveError] = createSignal("");
  const [isSubmitting, setIsSubmitting] = createSignal(false);

  const currentPlayer = createMemo(() => {
    const playerId = state.session.playerId;
    return state.game?.players.find((p) => p.id === playerId) ?? null;
  });

  const canMutate = createMemo(() => {
    const player = currentPlayer();
    if (!player) return false;
    // Can only mutate if they have the tech, haven't used it yet, and have points
    return (player.evolutionaryMutationPoints ?? 0) > 0 && !player.hasUsedEvolutionaryMutation;
  });

  const readOnly = createMemo(() => !canMutate() || isSubmitting());

  const design = createMemo(() => ({
    name: currentPlayer()?.raceName ?? "",
    government: currentPlayer()?.raceGovernment ?? "Dictatorship",
    trait_ids: traitIds(),
  }));

  const errors = createMemo(() => validateCustomRaceDesign(design()));
  const leftColumnGroups = createMemo(() => raceTraitGroupsForCategories(RACE_EDITOR_LEFT_COLUMN));
  const middleColumnGroups = createMemo(() => raceTraitGroupsForCategories(RACE_EDITOR_MIDDLE_COLUMN));
  const rightColumnGroups = createMemo(() => raceTraitGroupsForCategories(RACE_EDITOR_RIGHT_COLUMN));

  // Calculate original race pick cost (what the race was designed with)
  const originalRacePickCost = createMemo(() => {
    const player = currentPlayer();
    if (!player?.raceTraitIds) return 0;
    return racePickCost(player.raceTraitIds);
  });

  // Current picks being used in the mutation editor
  const currentPickCost = createMemo(() => {
    return racePickCost(traitIds());
  });

  // Base points are 10, mutation tech adds +4 = 14 total
  const BASE_PICKS = 10;
  const MUTATION_BONUS = 4;

  onMount(() => {
    // Load current race traits
    const player = currentPlayer();
    if (player?.raceTraitIds) {
      setTraitIds([...player.raceTraitIds] as RaceTraitId[]);
    }
  });

  const toggleTrait = (id: string) => {
    if (readOnly()) return;
    const trait = raceTraitDefinitions.find((item) => item.id === id);
    if (!trait || !trait.enabled) return;
    if (traitIds().includes(id as RaceTraitId)) {
      setTraitIds((prev) => prev.filter((traitId) => traitId !== id));
      return;
    }
    if (isRaceTraitBlocked(trait)) return;
    setTraitIds((prev) => swapExclusiveRaceTrait(trait, prev));
  };

  const handleSubmit = async () => {
    if (readOnly()) return;
    if (errors().length > 0) {
      setSaveError(errors().join("; "));
      return;
    }

    setIsSubmitting(true);
    setSaveError("");

    try {
      // Queue the mutation command - it will be sent with the next turn
      mutateRace(traitIds());
      navigate("/galaxy");
    } catch (error) {
      setSaveError(error instanceof Error ? error.message : "Unable to save mutation.");
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleCancel = () => {
    navigate("/galaxy");
  };

  // Check if mutation is unavailable (no points or already used)
  const mutationUnavailable = createMemo(() => {
    const player = currentPlayer();
    if (!player) return true;
    return (player.evolutionaryMutationPoints ?? 0) <= 0 || player.hasUsedEvolutionaryMutation;
  });

  // Navigate away if mutation is unavailable
  createEffect(() => {
    if (mutationUnavailable() && !isSubmitting()) {
      navigate("/galaxy");
    }
  });

  const remainingPicks = createMemo(() =>
    mutationRemainingPicks(traitIds(), design().government, governments())
  );

  return (
    <section class="race-editor-page">
      <div class="race-editor-shell">
        <header class="race-editor-header">
          <div class="race-editor-header-main">
            <button
              class="race-editor-back"
              type="button"
              onClick={handleCancel}
              aria-label="Back to galaxy"
            >
              <BackArrowIcon />
            </button>
            <h1>Evolutionary Mutation</h1>
          </div>
          <div class="race-editor-header-actions">
            <button
              type="button"
              class="secondary-button"
              onClick={handleCancel}
              disabled={isSubmitting()}
            >
              Cancel
            </button>
            <button
              type="button"
              class="primary-button"
              onClick={handleSubmit}
              disabled={readOnly() || errors().length > 0 || isSubmitting()}
            >
              {isSubmitting() ? "Applying..." : "Apply Mutation"}
            </button>
          </div>
        </header>

        <Show when={mutationUnavailable()}>
          <div class="status-banner info" role="status">
            {currentPlayer()?.hasUsedEvolutionaryMutation
              ? "Evolutionary Mutation has already been used. Redirecting..."
              : "No mutation points available. Research Evolutionary Mutation to unlock. Redirecting..."}
          </div>
        </Show>

        <Show when={!mutationUnavailable()}>
          <div class="race-editor-body race-editor-body--single">
            <main class="race-editor-main">
              <div class="race-editor-toolbar">
                <div class="race-editor-info">
                  <div class="race-editor-info-item">
                    <span>Race</span>
                    <strong>{currentPlayer()?.raceName}</strong>
                  </div>
                  <div class="race-editor-info-item">
                    <span>Government</span>
                    <strong>{currentPlayer()?.raceGovernment}</strong>
                  </div>
                </div>
                <div class="race-editor-picks mutation-picks-display" aria-label="Race mutation picks">
                  <div class="mutation-pick-budget">
                    <span>Budget</span>
                    <div class="pick-budget-breakdown">
                      <span class="pick-budget-base" title="Base race design points">{BASE_PICKS}</span>
                      <span class="pick-budget-plus">+</span>
                      <span class="pick-budget-bonus" title="Evolutionary Mutation tech bonus">{MUTATION_BONUS}</span>
                      <span class="pick-budget-equals">=</span>
                      <span class="pick-budget-total">{MUTATION_PICK_BUDGET}</span>
                    </div>
                  </div>
                  <div>
                    <span>Used</span>
                    <strong classList={{ danger: currentPickCost() > MUTATION_PICK_BUDGET }}>
                      {currentPickCost()}
                    </strong>
                  </div>
                  <div>
                    <span>Remaining</span>
                    <strong
                      classList={{ danger: remainingPicks() < 0, warning: remainingPicks() > 0 }}
                    >
                      {remainingPicks()}
                    </strong>
                  </div>
                  <div>
                    <span>Negative</span>
                    <strong classList={{ danger: mutationNegativePicks(traitIds(), design().government, governments()) > 10 }}>
                      {mutationNegativePicks(traitIds(), design().government, governments())}/10
                    </strong>
                  </div>
                </div>
              </div>

              <div class="race-editor-description">
                <p>
                  One-time opportunity to reconfigure your race traits using up to <strong>{MUTATION_PICK_BUDGET}</strong> pick points
                  (<strong>{BASE_PICKS}</strong> base + <strong>{MUTATION_BONUS}</strong> bonus from tech).
                </p>
                <p class="race-editor-note mutation-warning">
                  <strong>One-time use:</strong> Once applied, you cannot mutate again. Unused points are lost.
                  Race name and government cannot be changed.
                </p>
              </div>

              <div class="race-editor-columns">
                <div class="race-editor-column">
                  <For each={leftColumnGroups()}>
                    {(group) => (
                      <MutationTraitSection
                        group={group}
                        readOnly={readOnly}
                        traitIds={traitIds}
                        onToggleTrait={toggleTrait}
                      />
                    )}
                  </For>
                </div>
                <div class="race-editor-column">
                  <For each={middleColumnGroups()}>
                    {(group) => (
                      <MutationTraitSection
                        group={group}
                        readOnly={readOnly}
                        traitIds={traitIds}
                        onToggleTrait={toggleTrait}
                      />
                    )}
                  </For>
                </div>
                <div class="race-editor-column">
                  <For each={rightColumnGroups()}>
                    {(group) => (
                      <MutationTraitSection
                        group={group}
                        readOnly={readOnly}
                        traitIds={traitIds}
                        onToggleTrait={toggleTrait}
                      />
                    )}
                  </For>
                </div>
              </div>

              <div class="race-editor-messages" aria-live="polite">
                <Show when={errors().length > 0}>
                  <ul class="race-editor-errors">
                    <For each={errors()}>{(error) => <li>{error}</li>}</For>
                  </ul>
                </Show>
                <Show when={saveError()}>
                  <p class="race-editor-save-error">{saveError()}</p>
                </Show>
              </div>
            </main>
          </div>
        </Show>
      </div>
    </section>
  );
}
