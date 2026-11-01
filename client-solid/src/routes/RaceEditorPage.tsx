import { useNavigate } from "@solidjs/router";
import { createMemo, createSignal, For, onCleanup, onMount, Show } from "solid-js";
import type { Accessor } from "solid-js";

import { ApiClient } from "../lib/api";
import { getCustomRaceDraft, setCustomRaceDraft } from "../lib/custom-race-storage";
import {
  defaultGovernment,
  fallbackGovernments,
  isRaceTraitBlocked,
  raceNegativePicks,
  RACE_EDITOR_LEFT_COLUMN,
  RACE_EDITOR_MIDDLE_COLUMN,
  RACE_EDITOR_RIGHT_COLUMN,
  raceRemainingPicks,
  partialRaceTraitDetails,
  raceTraitDefinitions,
  raceTraitGroupsForCategories,
  swapExclusiveRaceTrait,
  validateCustomRaceDesign,
  validatePlayableCustomRaceDesign,
} from "../lib/race-design";
import type { RaceTraitDefinition } from "../lib/race-design";
import type { ApiGovernment, RaceDesign, RaceTraitId } from "../lib/types/api";
import { useAppContext } from "../state/app-context";

function RaceEditorFoldChevron() {
  return (
    <svg
      class="race-editor-sidebar-fold-chevron"
      width="10"
      height="10"
      viewBox="0 0 10 10"
      fill="none"
      aria-hidden="true"
    >
      <path
        class="race-editor-sidebar-fold-chevron-right"
        d="M3.25 2.25 6.75 5 3.25 7.75"
        stroke="currentColor"
        stroke-width="1.35"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
      <path
        class="race-editor-sidebar-fold-chevron-down"
        d="M2.25 3.25 5 6.75 7.75 3.25"
        stroke="currentColor"
        stroke-width="1.35"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
    </svg>
  );
}

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

function DuplicateIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
      <rect x="4.5" y="4.5" width="7" height="7" rx="1" stroke="currentColor" stroke-width="1.2" />
      <path
        d="M2.5 9.5V3.5C2.5 2.95 2.95 2.5 3.5 2.5H9.5"
        stroke="currentColor"
        stroke-width="1.2"
        stroke-linecap="round"
      />
    </svg>
  );
}

function TrashIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
      <path
        d="M3 4h8M5.5 4V3h3v1M5.25 6v3.5M8.75 6v3.5M4.25 4l.5 7h4.5l.5-7"
        stroke="currentColor"
        stroke-width="1.2"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
    </svg>
  );
}

const SAVE_NOTICE_MS = 3000;
const NAME_AUTOSAVE_MS = 1000;

type RaceTraitGroup = { category: string; traits: RaceTraitDefinition[] };

function RaceEditorListItem(props: {
  race: RaceDesign;
  selected: boolean;
  confirmDelete: boolean;
  actionsDisabled: boolean;
  onSelect: () => void;
  onCopy: () => void;
  onDelete?: () => void;
}) {
  return (
    <div
      class="race-editor-row"
      classList={{ "is-selected": props.selected, "is-confirm-delete": props.confirmDelete }}
    >
      <button
        type="button"
        class="race-editor-option construction-project-option"
        classList={{ "is-selected": props.selected }}
        onClick={props.onSelect}
      >
        <span class="race-editor-option-marker" />
        <span class="race-editor-option-name">{props.race.name}</span>
      </button>
      <div class="race-editor-row-actions">
        <button
          type="button"
          class="race-editor-row-action"
          title="Duplicate race"
          aria-label={`Duplicate ${props.race.name}`}
          disabled={props.actionsDisabled}
          onClick={(event) => {
            event.stopPropagation();
            props.onCopy();
          }}
        >
          <DuplicateIcon />
        </button>
        <Show when={props.onDelete}>
          <button
            type="button"
            class="race-editor-row-action race-editor-row-action--danger"
            title={props.confirmDelete ? "Confirm delete" : "Delete race"}
            aria-label={props.confirmDelete ? `Confirm delete ${props.race.name}` : `Delete ${props.race.name}`}
            disabled={props.actionsDisabled}
            onClick={(event) => {
              event.stopPropagation();
              props.onDelete?.();
            }}
          >
            <TrashIcon />
          </button>
        </Show>
      </div>
    </div>
  );
}

function RaceEditorTraitSection(props: {
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
                  disabled: props.readOnly() || (!trait.enabled && !selected()) || blocked(),
                }}
                disabled={props.readOnly() || (!trait.enabled && !selected()) || blocked()}
                aria-pressed={selected()}
                onClick={() => props.onToggleTrait(trait.id)}
                title={!trait.enabled ? trait.description : undefined}
              >
                <span class="race-trait-main">
                  <span class="race-trait-name">{trait.name} <Show when={partialRaceTraitDetails[trait.id]}><span class="race-trait-partial" title={partialRaceTraitDetails[trait.id]}>Partially implemented</span></Show></span>
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

function RaceEditorGovernmentSection(props: {
  readOnly: Accessor<boolean>;
  governments: Accessor<ApiGovernment[]>;
  government: Accessor<string>;
  onSelectGovernment: (name: string) => void;
}) {
  return (
    <section class="race-editor-section" aria-label="Governments">
      <h2>Governments</h2>
      <div class="race-trait-list">
        <For each={props.governments()}>
          {(item) => {
            const selected = () => props.government() === item.name;
            return (
              <button
                type="button"
                class="race-trait"
                classList={{ selected: selected(), disabled: props.readOnly() }}
                disabled={props.readOnly()}
                aria-pressed={selected()}
                onClick={() => !props.readOnly() && props.onSelectGovernment(item.name)}
              >
                <span class="race-trait-main">
                  <span class="race-trait-name">{item.name}</span>
                  <span class="race-trait-description">{item.description ?? ""}</span>
                </span>
                <span classList={{ "race-trait-cost": true, negative: item.pick_cost < 0 }}>
                  {item.pick_cost > 0 ? `+${item.pick_cost}` : item.pick_cost}
                </span>
              </button>
            );
          }}
        </For>
      </div>
    </section>
  );
}

export function RaceEditorPage() {
  const navigate = useNavigate();
  const { state } = useAppContext();
  const [allRaces, setAllRaces] = createSignal<RaceDesign[]>([]);
  const [listFocus, setListFocus] = createSignal<string | null>(null);
  const [isDraft, setIsDraft] = createSignal(false);
  const [name, setName] = createSignal("Custom");
  const [government, setGovernment] = createSignal("Dictatorship");
  const [governments, setGovernments] = createSignal<ApiGovernment[]>(fallbackGovernments);
  const [traitIds, setTraitIds] = createSignal<RaceTraitId[]>([]);
  const [saveError, setSaveError] = createSignal("");
  const [saveNotice, setSaveNotice] = createSignal("");
  const [confirmDeleteName, setConfirmDeleteName] = createSignal<string | null>(null);
  const [isSaving, setIsSaving] = createSignal(false);

  let suppressAutoSave = 0;
  let nameAutosaveTimer: ReturnType<typeof setTimeout> | undefined;
  let saveNoticeTimer: ReturnType<typeof setTimeout> | undefined;

  onCleanup(() => {
    clearTimeout(nameAutosaveTimer);
    clearTimeout(saveNoticeTimer);
  });

  const client = createMemo(() =>
    state.session.serverUrl ? new ApiClient(state.session.serverUrl, state.session.playerId) : null,
  );

  const readOnly = createMemo(() => {
    if (isDraft()) return false;
    const focus = listFocus();
    if (!focus) return false;
    const race = allRaces().find((r) => r.name === focus);
    return Boolean(race?.built_in);
  });

  const design = createMemo(() => ({ name: name().trim(), government: government(), trait_ids: traitIds() }));
  const errors = createMemo(() => validateCustomRaceDesign(design()));
  const playableErrors = createMemo(() => validatePlayableCustomRaceDesign(design(), governments()));
  const leftColumnGroups = createMemo(() => raceTraitGroupsForCategories(RACE_EDITOR_LEFT_COLUMN));
  const middleColumnGroups = createMemo(() => raceTraitGroupsForCategories(RACE_EDITOR_MIDDLE_COLUMN));
  const rightColumnGroups = createMemo(() => raceTraitGroupsForCategories(RACE_EDITOR_RIGHT_COLUMN));

  const showSavedNotice = () => {
    setSaveNotice("Race saved");
    clearTimeout(saveNoticeTimer);
    saveNoticeTimer = setTimeout(() => setSaveNotice(""), SAVE_NOTICE_MS);
  };

  const loadFormFromRace = (race: RaceDesign) => {
    suppressAutoSave += 1;
    setName(race.name);
    setGovernment(race.government || defaultGovernment(governments()));
    setTraitIds([...(race.trait_ids ?? [])] as RaceTraitId[]);
    queueMicrotask(() => {
      suppressAutoSave = Math.max(0, suppressAutoSave - 1);
    });
  };

  const scheduleAutoSave = (immediate: boolean) => {
    if (suppressAutoSave > 0 || readOnly() || errors().length > 0 || isSaving()) {
      return;
    }
    clearTimeout(nameAutosaveTimer);
    if (immediate) {
      void persistCurrentRace();
      return;
    }
    nameAutosaveTimer = setTimeout(() => {
      void persistCurrentRace();
    }, NAME_AUTOSAVE_MS);
  };

  const refreshAll = async (): Promise<RaceDesign[]> => {
    if (!client()) return [];
    try {
      const races = await client()!.getRaces();
      setAllRaces(races);
      return races;
    } catch {
      setSaveError("Unable to load races.");
      return [];
    }
  };

  const refreshGovernments = async () => {
    if (!client()) return;
    try {
      const fetched = await client()!.getGovernments();
      if (fetched.length > 0) {
        setGovernments(fetched);
        setGovernment((current) => current || defaultGovernment(fetched));
      }
    } catch {
      setGovernments(fallbackGovernments);
    }
  };

  onMount(() => {
    void (async () => {
      await refreshGovernments();
      const races = await refreshAll();
      const draft = getCustomRaceDraft();
      if (draft?.name.trim()) {
        setIsDraft(true);
        setListFocus(null);
        loadFormFromRace(draft);
        return;
      }
      const firstBuiltin = races.find((r) => r.built_in);
      if (firstBuiltin) {
        setListFocus(firstBuiltin.name);
        setIsDraft(false);
        loadFormFromRace(firstBuiltin);
      }
    })();
  });

  const selectRow = (race: RaceDesign) => {
    setListFocus(race.name);
    setIsDraft(false);
    loadFormFromRace(race);
    setSaveError("");
    setConfirmDeleteName(null);
  };

  const handleNew = () => {
    const taken = new Set(allRaces().map((r) => r.name));
    let candidate = "Custom";
    if (taken.has(candidate)) {
      for (let i = 2; i < 100; i += 1) {
        const next = `Custom ${i}`;
        if (!taken.has(next)) {
          candidate = next;
          break;
        }
      }
    }
    setIsDraft(true);
    setListFocus(null);
    setName(candidate);
    setGovernment(defaultGovernment(governments()));
    setTraitIds([]);
    setSaveError("");
    setConfirmDeleteName(null);
  };

  const copyRace = (race: RaceDesign) => {
    suppressAutoSave += 1;
    setIsDraft(true);
    setListFocus(null);
    setName(nextCopyName(race.name, allRaces()));
    setGovernment(race.government || defaultGovernment(governments()));
    setTraitIds([...(race.trait_ids ?? [])] as RaceTraitId[]);
    setSaveError("");
    setConfirmDeleteName(null);
    queueMicrotask(() => {
      suppressAutoSave = Math.max(0, suppressAutoSave - 1);
      scheduleAutoSave(true);
    });
  };

  const toggleTrait = (id: string) => {
    if (readOnly()) return;
    const trait = raceTraitDefinitions.find((item) => item.id === id);
    if (!trait) return;
    if (traitIds().includes(id as RaceTraitId)) {
      setTraitIds((prev) => prev.filter((traitId) => traitId !== id));
      scheduleAutoSave(true);
      return;
    }
    if (!trait.enabled || isRaceTraitBlocked(trait)) return;
    setTraitIds((prev) => swapExclusiveRaceTrait(trait, prev));
    scheduleAutoSave(true);
  };

  const updateGovernment = (nextGovernment: string) => {
    if (readOnly()) return;
    setGovernment(nextGovernment);
    scheduleAutoSave(true);
  };

  const persistCurrentRace = async () => {
    if (readOnly() || errors().length > 0 || isSaving()) {
      return null;
    }
    setIsSaving(true);
    setSaveError("");
    try {
      const focus = listFocus();
      const trimmed = name().trim();
      const row = focus ? allRaces().find((r) => r.name === focus) : undefined;
      const renaming = Boolean(!isDraft() && focus && row && !row.built_in && trimmed !== focus);
      const saved = client()
        ? await client()!.saveCustomRace(design(), renaming ? { renameFrom: focus! } : undefined)
        : design();
      setCustomRaceDraft(saved);
      setIsDraft(false);
      setListFocus(saved.name);
      await refreshAll();
      showSavedNotice();
      return saved;
    } catch (error) {
      setSaveError(error instanceof Error ? error.message : "Unable to save race.");
      return null;
    } finally {
      setIsSaving(false);
    }
  };

  const deleteRace = async (race: RaceDesign) => {
    if (race.built_in) return;
    if (confirmDeleteName() !== race.name) {
      setConfirmDeleteName(race.name);
      return;
    }
    if (!client()) return;
    setIsSaving(true);
    setSaveError("");
    try {
      await client()!.deleteCustomRace(race.name);
      setConfirmDeleteName(null);
      const updated = await refreshAll();
      const deletingFocused = listFocus() === race.name || (isDraft() && name().trim() === race.name);
      if (deletingFocused) {
        const builtin = updated.find((r) => r.built_in);
        if (builtin) {
          selectRow(builtin);
        } else {
          handleNew();
        }
      }
    } catch (error) {
      setSaveError(error instanceof Error ? error.message : "Unable to delete race.");
    } finally {
      setIsSaving(false);
    }
  };

  const rowSelected = (race: RaceDesign) => !isDraft() && listFocus() === race.name;
  const builtInRaces = createMemo(() => allRaces().filter((race) => race.built_in));
  const customRaces = createMemo(() => allRaces().filter((race) => !race.built_in));

  return (
    <section class="race-editor-page">
      <div class="race-editor-shell">
        <header class="race-editor-header">
          <div class="race-editor-header-main">
            <button
              class="race-editor-back"
              type="button"
              onClick={() => navigate("/")}
              aria-label="Back to home"
            >
              <BackArrowIcon />
            </button>
            <h1>Race Editor</h1>
          </div>
        </header>

        <Show when={saveNotice()}>
          <p class="race-editor-save-notice" role="status">
            {saveNotice()}
          </p>
        </Show>

        <div class="race-editor-body">
          <aside class="race-editor-sidebar" aria-label="Race list">
            <details class="race-editor-sidebar-fold" open>
              <summary class="race-editor-sidebar-fold-summary">
                <span class="race-editor-sidebar-fold-heading">
                  <RaceEditorFoldChevron />
                  <span class="race-editor-sidebar-fold-title">Built-in races</span>
                </span>
              </summary>
              <div class="race-editor-list">
                <For each={builtInRaces()}>
                  {(race) => (
                    <RaceEditorListItem
                      race={race}
                      selected={rowSelected(race)}
                      confirmDelete={false}
                      actionsDisabled={!client() || isSaving()}
                      onSelect={() => selectRow(race)}
                      onCopy={() => copyRace(race)}
                    />
                  )}
                </For>
              </div>
            </details>

            <details class="race-editor-sidebar-fold" open>
              <summary class="race-editor-sidebar-fold-summary">
                <span class="race-editor-sidebar-fold-heading">
                  <RaceEditorFoldChevron />
                  <span class="race-editor-sidebar-fold-title">Custom races</span>
                </span>
                <button
                  type="button"
                  class="race-editor-sidebar-new-race"
                  disabled={!client() || isSaving()}
                  onClick={(event) => {
                    event.preventDefault();
                    event.stopPropagation();
                    handleNew();
                  }}
                >
                  New race
                </button>
              </summary>
              <div class="race-editor-list">
                <Show when={isDraft()}>
                  <div class="race-editor-draft-pill">Unsaved draft</div>
                </Show>
                <Show
                  when={customRaces().length > 0}
                  fallback={
                    <Show when={!isDraft()}>
                      <p class="race-editor-sidebar-empty">No custom races yet.</p>
                    </Show>
                  }
                >
                  <For each={customRaces()}>
                    {(race) => (
                      <RaceEditorListItem
                        race={race}
                        selected={rowSelected(race)}
                        confirmDelete={confirmDeleteName() === race.name}
                        actionsDisabled={!client() || isSaving()}
                        onSelect={() => selectRow(race)}
                        onCopy={() => copyRace(race)}
                        onDelete={() => void deleteRace(race)}
                      />
                    )}
                  </For>
                </Show>
              </div>
            </details>
          </aside>

          <main class="race-editor-main" classList={{ "race-editor-main--readonly": readOnly() }}>
            <div class="race-editor-toolbar">
              <label class="race-editor-name-field">
                <span>Race name</span>
                <input
                  value={name()}
                  maxlength={32}
                  disabled={readOnly()}
                  onInput={(event) => {
                    setName(event.currentTarget.value);
                    scheduleAutoSave(false);
                  }}
                />
              </label>
              <div class="race-editor-picks" aria-label="Race design picks">
                <div>
                  <span>Remaining</span>
                  <strong
                    classList={{ danger: raceRemainingPicks(traitIds(), government(), governments()) < 0 }}
                  >
                    {raceRemainingPicks(traitIds(), government(), governments())}
                  </strong>
                </div>
                <div>
                  <span>Negative</span>
                  <strong classList={{ danger: raceNegativePicks(traitIds(), government(), governments()) > 10 }}>
                    {raceNegativePicks(traitIds(), government(), governments())}/10
                  </strong>
                </div>
              </div>
            </div>

            <div class="race-editor-columns">
              <div class="race-editor-column">
                <For each={leftColumnGroups()}>
                  {(group) => (
                    <RaceEditorTraitSection
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
                    <RaceEditorTraitSection
                      group={group}
                      readOnly={readOnly}
                      traitIds={traitIds}
                      onToggleTrait={toggleTrait}
                    />
                  )}
                </For>
                <RaceEditorGovernmentSection
                  readOnly={readOnly}
                  governments={governments}
                  government={government}
                  onSelectGovernment={updateGovernment}
                />
              </div>
              <div class="race-editor-column">
                <For each={rightColumnGroups()}>
                  {(group) => (
                    <RaceEditorTraitSection
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
              <Show when={errors().length === 0 && playableErrors().length > 0}>
                <ul class="race-editor-warnings">
                  <For each={playableErrors()}>{(error) => <li>Unplayable: {error}</li>}</For>
                </ul>
              </Show>
              <Show when={readOnly()}>
                <p class="race-editor-readonly-note">
                  Built-in races are read-only. Duplicate one from the list to create a custom race.
                </p>
              </Show>
              <Show when={saveError()}>
                <p class="race-editor-save-error">{saveError()}</p>
              </Show>
            </div>
          </main>
        </div>
      </div>
    </section>
  );
}

function nextCopyName(baseName: string, races: RaceDesign[]) {
  const existing = new Set(races.map((race) => race.name));
  const base = `${baseName} Copy`;
  if (!existing.has(base)) return base;
  for (let index = 2; index < 100; index += 1) {
    const candidate = `${base} ${index}`;
    if (!existing.has(candidate)) return candidate;
  }
  return base;
}
