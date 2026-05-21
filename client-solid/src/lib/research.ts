import type { TechLevelViewModel } from "./types/domain";

export function playerHasRaceTrait(
  traitIds: string[] | undefined,
  traitId: string,
): boolean {
  return (traitIds ?? []).includes(traitId);
}

/** True when completing a level grants every tech at that breakthrough (MoO2 Creative or force_all levels). */
export function researchLevelUnlocksAllTechs(
  level: Pick<TechLevelViewModel, "forceAll">,
  traitIds: string[] | undefined,
): boolean {
  return level.forceAll || playerHasRaceTrait(traitIds, "creative");
}
