import type { ApiMissile } from "./types/api";
import type { ShipDesignForm } from "./types/domain";

/** Editor damage column for a missile rack, e.g. "8 (5 shots/battle)". */
export function formatMissileDamage(missile: Pick<ApiMissile, "damage" | "ammo">): string {
  return `${missile.damage} (${missile.ammo} shots/battle)`;
}

/** The weapon currently in a design's one slot (beam or missile), or "". */
export function selectedWeaponName(form: Pick<ShipDesignForm, "beam" | "missile">): string {
  return form.beam || form.missile || "";
}

/**
 * Puts a weapon in the design's single slot. A name from the missile catalog
 * becomes the missile rack and clears the beam; anything else is a beam and
 * clears the missile. "" empties the slot.
 */
export function withWeapon<T extends Pick<ShipDesignForm, "beam" | "missile">>(
  form: T,
  name: string,
  missiles: readonly Pick<ApiMissile, "name">[] | undefined,
): T {
  if (!name) return { ...form, beam: "", missile: "" };
  if ((missiles ?? []).some((missile) => missile.name === name)) {
    return { ...form, beam: "", missile: name };
  }
  return { ...form, beam: name, missile: "" };
}
