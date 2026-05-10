import { useNavigate } from "@solidjs/router";
import { createMemo, createSignal, For, onMount, Show } from "solid-js";

import { getCustomRaceDraft, markSetupFormOpen, setCustomRaceDraft } from "../lib/custom-race-storage";
import { ApiClient } from "../lib/api";
import {
  isRaceTraitBlocked,
  raceNegativePicks,
  raceRemainingPicks,
  raceTraitDefinitions,
  selectedRaceTraits,
  swapExclusiveRaceTrait,
  validateCustomRaceDesign,
  validatePlayableCustomRaceDesign,
} from "../lib/race-design";
import type { RaceTraitId } from "../lib/types/api";
import type { CustomRaceDesign } from "../lib/types/api";
import { useAppContext } from "../state/app-context";

const categoryOrder = ["Population", "Farming", "Production", "Research", "Money", "Homeworld", "Future Traits"];

export function CustomRacePage() {
  const navigate = useNavigate();
  const { state } = useAppContext();
  const [name, setName] = createSignal("Custom");
  const [traitIds, setTraitIds] = createSignal<RaceTraitId[]>([]);
  const [savedRaces, setSavedRaces] = createSignal<CustomRaceDesign[]>([]);
  const [saveError, setSaveError] = createSignal("");
  const [saveMessage, setSaveMessage] = createSignal("");
  const [confirmDelete, setConfirmDelete] = createSignal("");
  const [isSaving, setIsSaving] = createSignal(false);
  const client = createMemo(() =>
    state.session.serverUrl ? new ApiClient(state.session.serverUrl, state.session.playerId) : null,
  );

  onMount(() => {
    const draft = getCustomRaceDraft();
    if (!draft) return;
    setName(draft.name || "Custom");
    setTraitIds(draft.trait_ids);
  });

  onMount(() => {
    void refreshSavedRaces();
  });

  const design = createMemo(() => ({ name: name().trim(), trait_ids: traitIds() }));
  const selectedTraits = createMemo(() => selectedRaceTraits(traitIds()));
  const errors = createMemo(() => validateCustomRaceDesign(design()));
  const playableErrors = createMemo(() => validatePlayableCustomRaceDesign(design()));
  const groupedTraits = createMemo(() =>
    categoryOrder.map((category) => ({
      category,
      traits: raceTraitDefinitions.filter((trait) => trait.category === category),
    })),
  );

  const toggleTrait = (id: string) => {
    const trait = raceTraitDefinitions.find((item) => item.id === id);
    if (!trait || !trait.enabled) return;
    if (traitIds().includes(id as RaceTraitId)) {
      setTraitIds((prev) => prev.filter((traitId) => traitId !== id));
      return;
    }
    if (isRaceTraitBlocked(trait)) return;
    setTraitIds((prev) => swapExclusiveRaceTrait(trait, prev));
  };

  const handleClear = () => {
    setName("Custom");
    setTraitIds([]);
  };

  const refreshSavedRaces = async () => {
    if (!client()) return;
    try {
      setSavedRaces(await client()!.getCustomRaces());
    } catch {
      setSaveError("Unable to load saved races.");
    }
  };

  const loadSavedRace = (race: CustomRaceDesign) => {
    setName(race.name);
    setTraitIds([...(race.trait_ids ?? [])]);
    setSaveError("");
    setSaveMessage("");
    setConfirmDelete("");
  };

  const copySavedRace = (race: CustomRaceDesign) => {
    setName(nextCopyName(race.name, savedRaces()));
    setTraitIds([...(race.trait_ids ?? [])]);
    setSaveError("");
    setSaveMessage("");
    setConfirmDelete("");
  };

  const deleteSavedRace = async (race: CustomRaceDesign) => {
    if (!client()) return;
    if (confirmDelete() !== race.name) {
      setConfirmDelete(race.name);
      return;
    }
    setIsSaving(true);
    setSaveError("");
    setSaveMessage("");
    try {
      await client()!.deleteCustomRace(race.name);
      setConfirmDelete("");
      await refreshSavedRaces();
    } catch (error) {
      setSaveError(error instanceof Error ? error.message : "Unable to delete race.");
    } finally {
      setIsSaving(false);
    }
  };

  const handleCancel = () => {
    navigate("/setup");
  };

  const saveCurrentRace = async () => {
    setIsSaving(true);
    setSaveError("");
    setSaveMessage("");
    try {
      const saved = client() ? await client()!.saveCustomRace(design()) : design();
      setCustomRaceDraft(saved);
      setSaveMessage(`Saved ${saved.name}.`);
      await refreshSavedRaces();
      return saved;
    } catch (error) {
      setSaveError(error instanceof Error ? error.message : "Unable to save race.");
      return null;
    } finally {
      setIsSaving(false);
    }
  };

  const handleSave = async () => {
    if (errors().length > 0) return;
    await saveCurrentRace();
  };

  const handleUse = async () => {
    if (errors().length > 0) return;
    const saved = await saveCurrentRace();
    if (!saved) return;
    markSetupFormOpen();
    navigate("/setup");
  };

  return (
    <section class="custom-race-page">
      <div class="custom-race-shell">
        <header class="custom-race-toolbar">
          <label class="custom-race-name">
            <span>Race Name</span>
            <input
              value={name()}
              maxlength={32}
              onInput={(event) => setName(event.currentTarget.value)}
              autofocus
            />
          </label>
          <div class="custom-race-picks" aria-label="Race design picks">
            <div>
              <span>Remaining</span>
              <strong classList={{ danger: raceRemainingPicks(traitIds()) < 0 }}>{raceRemainingPicks(traitIds())}</strong>
            </div>
            <div>
              <span>Negative</span>
              <strong classList={{ danger: raceNegativePicks(traitIds()) > 10 }}>{raceNegativePicks(traitIds())}/10</strong>
            </div>
          </div>
          <div class="custom-race-actions">
            <button class="link-button" type="button" onClick={handleClear}>Clear</button>
            <button class="link-button" type="button" onClick={handleCancel}>Cancel</button>
            <button class="secondary-button" type="button" onClick={() => void handleSave()} disabled={errors().length > 0 || isSaving()}>Save</button>
            <button class="primary-button" type="button" onClick={() => void handleUse()} disabled={errors().length > 0 || isSaving()}>Use</button>
          </div>
        </header>

        <div class="custom-race-grid">
          <main class="custom-race-traits" aria-label="Race traits">
            <For each={groupedTraits()}>
              {(group) => (
                <section class="custom-race-section">
                  <h2>{group.category}</h2>
                  <div class="race-trait-list">
                    <For each={group.traits}>
                      {(trait) => {
                        const selected = () => traitIds().includes(trait.id as RaceTraitId);
                        const blocked = () => isRaceTraitBlocked(trait) && !selected();
                        return (
                          <button
                            type="button"
                            class="race-trait"
                            classList={{
                              selected: selected(),
                              disabled: !trait.enabled || blocked(),
                            }}
                            disabled={!trait.enabled || blocked()}
                            aria-pressed={selected()}
                            onClick={() => toggleTrait(trait.id)}
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
              )}
            </For>
          </main>

          <aside class="custom-race-summary" aria-label="Custom race summary">
            <Show when={savedRaces().length > 0}>
              <section class="custom-race-saved">
                <h2>Saved Races</h2>
                <ul>
                  <For each={savedRaces()}>
                    {(saved) => (
                      <li>
                        <div class="custom-race-saved-main">
                          <strong>{saved.name}</strong>
                          <span classList={{ unplayable: saved.playable === false }}>
                            {saved.playable === false ? "Unplayable" : "Playable"} · {(saved.trait_ids ?? []).length} traits
                          </span>
                        </div>
                        <div class="custom-race-saved-actions">
                          <button type="button" class="link-button" onClick={() => loadSavedRace(saved)}>
                            Edit
                          </button>
                          <button type="button" class="link-button" onClick={() => copySavedRace(saved)}>
                            Copy
                          </button>
                          <button
                            type="button"
                            class="custom-race-delete-text"
                            onClick={() => void deleteSavedRace(saved)}
                            disabled={isSaving()}
                            title="Delete saved race"
                          >
                            {confirmDelete() === saved.name ? "Confirm" : "Delete"}
                          </button>
                        </div>
                      </li>
                    )}
                  </For>
                </ul>
              </section>
            </Show>
            <h2>Summary</h2>
            <Show
              when={selectedTraits().length > 0}
              fallback={<p class="custom-race-empty">No traits selected</p>}
            >
              <ul class="custom-race-selected">
                <For each={selectedTraits()}>
                  {(trait) => (
                    <li>
                      <span>{trait.name}</span>
                      <strong classList={{ negative: trait.pickCost < 0 }}>
                        {trait.pickCost > 0 ? `+${trait.pickCost}` : trait.pickCost}
                      </strong>
                    </li>
                  )}
                </For>
              </ul>
            </Show>
            <dl class="custom-race-totals">
              <div>
                <dt>Positive Picks</dt>
                <dd>{selectedTraits().reduce((total, trait) => total + Math.max(0, trait.pickCost), 0)}</dd>
              </div>
              <div>
                <dt>Disadvantages</dt>
                <dd>{raceNegativePicks(traitIds())}</dd>
              </div>
            </dl>
            <Show when={errors().length > 0}>
              <ul class="custom-race-errors">
                <For each={errors()}>{(error) => <li>{error}</li>}</For>
              </ul>
            </Show>
            <Show when={errors().length === 0 && playableErrors().length > 0}>
              <ul class="custom-race-warnings">
                <For each={playableErrors()}>{(error) => <li>Unplayable: {error}</li>}</For>
              </ul>
            </Show>
            <Show when={saveError()}>
              <p class="custom-race-save-error">{saveError()}</p>
            </Show>
            <Show when={saveMessage()}>
              <p class="custom-race-save-message">{saveMessage()}</p>
            </Show>
          </aside>
        </div>
      </div>
    </section>
  );
}

function nextCopyName(name: string, races: CustomRaceDesign[]) {
  const existing = new Set(races.map((race) => race.name));
  const base = `${name} Copy`;
  if (!existing.has(base)) return base;
  for (let index = 2; index < 100; index += 1) {
    const candidate = `${base} ${index}`;
    if (!existing.has(candidate)) return candidate;
  }
  return base;
}
