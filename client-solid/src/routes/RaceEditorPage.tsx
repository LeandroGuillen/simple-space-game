import { useNavigate } from "@solidjs/router";
import { createMemo, createSignal, For, onMount, Show } from "solid-js";
import type { Accessor } from "solid-js";

import { ApiClient } from "../lib/api";
import { getCustomRaceDraft, setCustomRaceDraft } from "../lib/custom-race-storage";
import {
  defaultGovernment,
  fallbackGovernments,
  isRaceTraitBlocked,
  raceNegativePicks,
  raceRemainingPicks,
  raceTraitDefinitions,
  swapExclusiveRaceTrait,
  validateCustomRaceDesign,
  validatePlayableCustomRaceDesign,
} from "../lib/race-design";
import type { RaceTraitDefinition } from "../lib/race-design";
import type { ApiGovernment, RaceDesign, RaceTraitId } from "../lib/types/api";
import { useAppContext } from "../state/app-context";

const categoryOrder = ["Population", "Farming", "Production", "Research", "Homeworld", "Money", "Future Traits"];

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

function traitCountForCategory(category: string) {
  return raceTraitDefinitions.filter((t) => t.category === category).length;
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

function RaceEditorGovernmentSection(props: {
  readOnly: Accessor<boolean>;
  governments: Accessor<ApiGovernment[]>;
  government: Accessor<string>;
  onSelectGovernment: (name: string) => void;
}) {
  return (
    <section class="race-editor-section" aria-label="Government">
      <h2>Government</h2>
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
  const [saveMessage, setSaveMessage] = createSignal("");
  const [confirmDelete, setConfirmDelete] = createSignal("");
  const [isSaving, setIsSaving] = createSignal(false);

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

  const showSaveAsNew = createMemo(() => {
    if (readOnly() || !client() || isDraft()) return false;
    const focus = listFocus();
    if (!focus) return false;
    const race = allRaces().find((r) => r.name === focus);
    if (!race || race.built_in) return false;
    return name().trim() !== focus && name().trim().length > 0;
  });

  const design = createMemo(() => ({ name: name().trim(), government: government(), trait_ids: traitIds() }));
  const errors = createMemo(() => validateCustomRaceDesign(design()));
  const playableErrors = createMemo(() => validatePlayableCustomRaceDesign(design(), governments()));
  const groupedTraits = createMemo(() =>
    categoryOrder.map((category) => ({
      category,
      traits: raceTraitDefinitions.filter((trait) => trait.category === category),
    })),
  );

  const tripleTraitGroups = createMemo(() =>
    groupedTraits().filter((g) => traitCountForCategory(g.category) === 3),
  );

  const homeworldGroup = createMemo(() => groupedTraits().find((g) => g.category === "Homeworld"));

  const futureTraitGroup = createMemo(() => groupedTraits().find((g) => g.category === "Future Traits"));

  const loadFormFromRace = (race: RaceDesign) => {
    setName(race.name);
    setGovernment(race.government || defaultGovernment(governments()));
    setTraitIds([...(race.trait_ids ?? [])] as RaceTraitId[]);
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
    setSaveMessage("");
    setConfirmDelete("");
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
    setSaveMessage("");
    setConfirmDelete("");
  };

  const handleCopy = () => {
    const next = nextCopyName(name(), allRaces());
    setIsDraft(true);
    setListFocus(null);
    setName(next);
    setSaveError("");
    setSaveMessage("");
    setConfirmDelete("");
  };

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

  const saveCurrentRace = async (opts?: { renameFrom?: string; saveAsNew?: boolean }) => {
    if (readOnly()) return null;
    setIsSaving(true);
    setSaveError("");
    setSaveMessage("");
    try {
      const saved = client() ? await client()!.saveCustomRace(design(), opts) : design();
      setCustomRaceDraft(saved);
      setSaveMessage(`Saved ${saved.name}.`);
      setIsDraft(false);
      setListFocus(saved.name);
      await refreshAll();
      return saved;
    } catch (error) {
      setSaveError(error instanceof Error ? error.message : "Unable to save race.");
      return null;
    } finally {
      setIsSaving(false);
    }
  };

  const handleSave = async () => {
    if (errors().length > 0 || readOnly()) return;
    const focus = listFocus();
    const trimmed = name().trim();
    const row = focus ? allRaces().find((r) => r.name === focus) : undefined;
    const renaming = Boolean(!isDraft() && focus && row && !row.built_in && trimmed !== focus);
    await saveCurrentRace(renaming ? { renameFrom: focus! } : undefined);
  };

  const handleSaveAsNew = async () => {
    if (errors().length > 0 || readOnly() || !showSaveAsNew()) return;
    await saveCurrentRace({ saveAsNew: true });
  };

  const handleDelete = async () => {
    if (readOnly() || isDraft()) return;
    const nm = listFocus();
    if (!nm) return;
    const race = allRaces().find((r) => r.name === nm);
    if (!race || race.built_in) return;
    if (confirmDelete() !== nm) {
      setConfirmDelete(nm);
      return;
    }
    if (!client()) return;
    setIsSaving(true);
    setSaveError("");
    setSaveMessage("");
    try {
      await client()!.deleteCustomRace(nm);
      setConfirmDelete("");
      const updated = await refreshAll();
      const builtin = updated.find((r) => r.built_in);
      if (builtin) {
        selectRow(builtin);
      } else {
        handleNew();
      }
    } catch (error) {
      setSaveError(error instanceof Error ? error.message : "Unable to delete race.");
    } finally {
      setIsSaving(false);
    }
  };

  const rowSelected = (race: RaceDesign) => !isDraft() && listFocus() === race.name;

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
          <div class="race-editor-header-actions">
            <button class="link-button" type="button" onClick={handleNew} disabled={!client() || isSaving()}>
              New
            </button>
            <button class="link-button" type="button" onClick={handleCopy} disabled={!client() || isSaving()}>
              Copy
            </button>
            <button class="secondary-button" type="button" onClick={() => void handleSave()} disabled={errors().length > 0 || readOnly() || isSaving()}>
              Save
            </button>
            <Show when={showSaveAsNew()}>
              <button
                class="secondary-button"
                type="button"
                onClick={() => void handleSaveAsNew()}
                disabled={errors().length > 0 || readOnly() || isSaving()}
              >
                Save as new
              </button>
            </Show>
            <button
              class="link-button"
              type="button"
              classList={{ "race-editor-delete": true }}
              onClick={() => void handleDelete()}
              disabled={readOnly() || isDraft() || isSaving() || !listFocus()}
            >
              {confirmDelete() === listFocus() ? "Confirm delete" : "Delete"}
            </button>
          </div>
        </header>

        <div class="race-editor-body">
          <aside class="race-editor-sidebar" aria-label="Race list">
            <h2 class="race-editor-sidebar-title">Races</h2>
            <div class="race-editor-list">
              <Show when={isDraft()}>
                <div class="race-editor-draft-pill">Unsaved draft</div>
              </Show>
              <For each={allRaces()}>
                {(race) => (
                  <button
                    type="button"
                    class="race-editor-option construction-project-option"
                    classList={{ "is-selected": rowSelected(race) }}
                    onClick={() => selectRow(race)}
                  >
                    <span class="race-editor-option-marker" />
                    <span class="race-editor-option-main">
                      <span class="race-editor-option-name">
                        {race.name}
                        {race.built_in ? <span class="race-editor-badge">Built-in</span> : null}
                      </span>
                      <span class="race-editor-option-meta">
                        {race.playable === false ? "Unplayable" : "Playable"} · {race.government || "—"}
                      </span>
                    </span>
                  </button>
                )}
              </For>
            </div>
          </aside>

          <main class="race-editor-main" classList={{ "race-editor-main--readonly": readOnly() }}>
            <div class="race-editor-toolbar">
              <label class="race-editor-name-field">
                <span>Race name</span>
                <input
                  value={name()}
                  maxlength={32}
                  disabled={readOnly()}
                  onInput={(event) => setName(event.currentTarget.value)}
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

            <div class="race-editor-traits-stack">
              <div class="race-editor-traits-grid">
                <For each={tripleTraitGroups()}>
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
              <div class="race-editor-traits-grid race-editor-traits-grid--homeworld-pair">
                {homeworldGroup() != null && (
                  <RaceEditorTraitSection
                    group={homeworldGroup()!}
                    readOnly={readOnly}
                    traitIds={traitIds}
                    onToggleTrait={toggleTrait}
                  />
                )}
                <RaceEditorGovernmentSection
                  readOnly={readOnly}
                  governments={governments}
                  government={government}
                  onSelectGovernment={setGovernment}
                />
              </div>
              {futureTraitGroup() != null && (
                <RaceEditorTraitSection
                  group={futureTraitGroup()!}
                  readOnly={readOnly}
                  traitIds={traitIds}
                  onToggleTrait={toggleTrait}
                />
              )}
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
                  Built-in races are read-only. Use Copy to edit a derived custom race.
                </p>
              </Show>
              <Show when={saveError()}>
                <p class="race-editor-save-error">{saveError()}</p>
              </Show>
              <Show when={saveMessage()}>
                <p class="race-editor-save-message">{saveMessage()}</p>
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
