import type { CustomRaceDesign } from "./types/api";
import { DEFAULT_GOVERNMENT, isRaceTraitId } from "./race-design";

const CUSTOM_RACE_DRAFT_KEY = "client-solid.customRaceDraft";
const SETUP_FORM_OPEN_KEY = "client-solid.setupFormOpen";

export function getCustomRaceDraft(): CustomRaceDesign | null {
  try {
    const raw = window.sessionStorage.getItem(CUSTOM_RACE_DRAFT_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as Partial<CustomRaceDesign>;
    const name = typeof parsed.name === "string" ? parsed.name : "";
    const government = typeof parsed.government === "string" && parsed.government ? parsed.government : DEFAULT_GOVERNMENT;
    const traitIds = Array.isArray(parsed.trait_ids)
      ? parsed.trait_ids.filter((id): id is NonNullable<CustomRaceDesign["trait_ids"]>[number] => typeof id === "string" && isRaceTraitId(id))
      : [];
    return { name, government, trait_ids: traitIds, playable: parsed.playable === true };
  } catch {
    return null;
  }
}

export function setCustomRaceDraft(design: CustomRaceDesign) {
  window.sessionStorage.setItem(CUSTOM_RACE_DRAFT_KEY, JSON.stringify(design));
}

export function clearCustomRaceDraft() {
  window.sessionStorage.removeItem(CUSTOM_RACE_DRAFT_KEY);
}

export function markSetupFormOpen() {
  window.sessionStorage.setItem(SETUP_FORM_OPEN_KEY, "1");
}

export function consumeSetupFormOpen() {
  const shouldOpen = window.sessionStorage.getItem(SETUP_FORM_OPEN_KEY) === "1";
  window.sessionStorage.removeItem(SETUP_FORM_OPEN_KEY);
  return shouldOpen;
}
