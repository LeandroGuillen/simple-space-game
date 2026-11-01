import { createMemo, For, Show } from "solid-js";
import { partialRaceTraitDetails, raceTotalPickCost, raceTraitById } from "../lib/race-design";
import type { ApiGovernment, RaceDesign } from "../lib/types/api";

const categoryFor = (category: string) => {
  if (category === "Ship Defense" || category === "Ship Attack") return "Ship";
  if (category === "Special Abilities") return "Special";
  return category;
};

export function RacePreview(props: {
  race: RaceDesign | null;
  governments: ApiGovernment[];
  selected: boolean;
  onEdit: (race: RaceDesign) => void;
}) {
  const government = createMemo(() => props.governments.find((g) => g.name === props.race?.government));
  const traitGroups = createMemo(() => {
    const groups = new Map<string, { id: string; name: string; description: string; partial?: string }[]>();
    for (const id of props.race?.trait_ids ?? []) {
      const trait = raceTraitById.get(id);
      const category = trait ? categoryFor(trait.category) : "Other";
      const entries = groups.get(category) ?? [];
      entries.push({ id, name: trait?.name ?? id, description: trait?.description ?? "Trait details unavailable.", partial: partialRaceTraitDetails[id] });
      groups.set(category, entries);
    }
    return Array.from(groups, ([category, traits]) => ({ category, traits }));
  });
  return (
    <aside class="race-preview" aria-label="Race preview" aria-live="polite">
      <Show when={props.race} fallback={<p class="race-preview-empty">Select or focus a race to see its details.</p>}>
        {(race) => <>
          <p class="race-preview-eyebrow">{props.selected ? "Selected race" : "Preview"}</p>
          <h3>{race().name}</h3>
          <p class="race-preview-government"><strong>{race().government}</strong><br />{government()?.description || "Government details unavailable."}</p>
          <Show when={traitGroups().length > 0} fallback={<p class="race-preview-empty">No race traits.</p>}>
            <div class="race-preview-groups">
              <For each={traitGroups()}>{(group) => <section>
                <h4>{group.category}</h4>
                <ul><For each={group.traits}>{(trait) => <li>
                  <span class="race-preview-chip">{trait.name}</span>
                  <Show when={trait.partial}><span class="race-preview-partial" title={trait.partial}>Partially implemented</span></Show>
                  <span class="race-preview-description">{trait.description}</span>
                </li>}</For></ul>
              </section>}</For>
            </div>
          </Show>
          <Show when={!race().built_in}>
            <p class="race-preview-picks"><Show when={government()} fallback="Pick total unavailable.">Picks used: {raceTotalPickCost(race().trait_ids, race().government, props.governments)}</Show></p>
            <button class="link-button race-preview-edit" type="button" onClick={() => props.onEdit(race())}>Edit in Race Editor</button>
          </Show>
        </>}
      </Show>
    </aside>
  );
}
