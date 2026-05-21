import type { TechLevelViewModel } from "./types/domain";

export function playerHasRaceTrait(
  traitIds: string[] | undefined,
  traitId: string,
): boolean {
  return (traitIds ?? []).includes(traitId);
}

export function tradeGoodsMultiplier(traitIds: string[] | undefined): number {
  return playerHasRaceTrait(traitIds, "fantastic_traders") ? 1.5 : 1;
}

/** True when completing a level grants every tech at that breakthrough (MoO2 Creative or force_all levels). Uncreative overrides force_all. */
export function researchLevelUnlocksAllTechs(
  level: Pick<TechLevelViewModel, "forceAll">,
  traitIds: string[] | undefined,
): boolean {
  if (playerHasRaceTrait(traitIds, "uncreative")) {
    return false;
  }
  return level.forceAll || playerHasRaceTrait(traitIds, "creative");
}
